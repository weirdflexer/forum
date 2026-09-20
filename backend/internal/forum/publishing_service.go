package forum

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Keep the field order stable: normalized request JSON is part of persisted
// idempotency hashes, including keys created before these named input types.
type createTopicInput struct {
	SectionID string `json:"section_id"`
	Title     string `json:"title"`
	Body      string `json:"body"`
}

type createdTopic struct {
	FirstPostID string `json:"first_post_id"`
	TopicID     string `json:"topic_id"`
}

type createPostInput struct {
	Body string `json:"body"`
}

type createdPost struct {
	Number int    `json:"number"`
	Page   int    `json:"page"`
	PostID string `json:"post_id"`
}

type createReportInput struct {
	PostID  string `json:"post_id"`
	Reason  string `json:"reason"`
	Comment string `json:"comment"`
}

type createdReport struct {
	ReportID string `json:"report_id"`
}

func (s *Server) publishTopic(ctx context.Context, req publicationRequest, in createTopicInput) (publicationResult[createdTopic], error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Body = strings.TrimSpace(in.Body)
	if !uuidPattern.MatchString(in.SectionID) || !validText(in.Title, 5, 120) || !validText(in.Body, 1, 5000) {
		return publicationResult[createdTopic]{}, problem(400, "validation", "Выберите раздел. Заголовок: 5–120 символов; сообщение: 1–5000.")
	}
	return publishOnce(ctx, s, req, requestHash(req.Path, in), func(tx pgx.Tx) (createdTopic, error) {
		var out createdTopic
		var archived bool
		if err := tx.QueryRow(ctx, "SELECT is_archived FROM sections WHERE id=$1 FOR SHARE", in.SectionID).Scan(&archived); err != nil {
			return out, err
		}
		if archived {
			return out, problem(409, "section_archived", "Раздел в архиве: новые темы недоступны.")
		}
		if err := sessionLimit(ctx, tx, req.SessionID, "topic", 1, 60); err != nil {
			return out, err
		}
		if err := tx.QueryRow(ctx, "INSERT INTO topics(section_id,author_session_id,title) VALUES($1,$2,$3) RETURNING id", in.SectionID, req.SessionID, in.Title).Scan(&out.TopicID); err != nil {
			return out, err
		}
		if err := tx.QueryRow(ctx, "INSERT INTO posts(topic_id,author_session_id,number,body) VALUES($1,$2,1,$3) RETURNING id", out.TopicID, req.SessionID, in.Body).Scan(&out.FirstPostID); err != nil {
			return out, err
		}
		return out, nil
	})
}

func (s *Server) publishPost(ctx context.Context, req publicationRequest, topicID string, in createPostInput) (publicationResult[createdPost], error) {
	in.Body = strings.TrimSpace(in.Body)
	if !validText(in.Body, 1, 5000) {
		return publicationResult[createdPost]{}, problem(400, "validation", "Сообщение: от 1 до 5000 символов.")
	}
	return publishOnce(ctx, s, req, requestHash(req.Path, in), func(tx pgx.Tx) (createdPost, error) {
		var out createdPost
		var status string
		// Serialize writers to a topic before allocating its next post number.
		if err := tx.QueryRow(ctx, "SELECT status FROM topics WHERE id=$1 FOR UPDATE", topicID).Scan(&status); err != nil {
			return out, err
		}
		if status == "closed" {
			return out, problem(409, "topic_closed", "Тема закрыта для новых ответов.")
		}
		if err := sessionLimit(ctx, tx, req.SessionID, "post", 1, 10); err != nil {
			return out, err
		}
		if err := tx.QueryRow(ctx, "INSERT INTO posts(topic_id,author_session_id,number,body) SELECT $1,$2,COALESCE(max(number),0)+1,$3 FROM posts WHERE topic_id=$1 RETURNING id,number", topicID, req.SessionID, in.Body).Scan(&out.PostID, &out.Number); err != nil {
			return out, err
		}
		if _, err := tx.Exec(ctx, "UPDATE topics SET last_activity_at=now() WHERE id=$1", topicID); err != nil {
			return out, err
		}
		out.Page = (out.Number-1)/20 + 1
		return out, nil
	})
}

func (s *Server) removeOwnPost(ctx context.Context, sessionID, postID string) error {
	tx, err := s.beginAnonymousWrite(ctx, sessionID)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var owner string
	if err = tx.QueryRow(ctx, "SELECT COALESCE(author_session_id::text,'') FROM posts WHERE id=$1 FOR UPDATE", postID).Scan(&owner); err != nil {
		return err
	}
	if owner != sessionID {
		return problem(403, "not_owner", "Можно удалить только сообщение своей текущей сессии.")
	}
	if _, err = tx.Exec(ctx, "UPDATE posts SET body='',status='deleted' WHERE id=$1", postID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Server) submitReport(ctx context.Context, sessionID string, in createReportInput) (createdReport, error) {
	var out createdReport
	in.Comment = strings.TrimSpace(in.Comment)
	validReason := in.Reason == "spam" || in.Reason == "abuse" || in.Reason == "personal_data" || in.Reason == "other"
	if !uuidPattern.MatchString(in.PostID) || !validReason || !validText(in.Comment, 0, 1000) {
		return out, problem(400, "validation", "Выберите причину. Комментарий — до 1000 символов.")
	}
	tx, err := s.beginAnonymousWrite(ctx, sessionID)
	if err != nil {
		return out, err
	}
	defer tx.Rollback(ctx)
	var status string
	if err = tx.QueryRow(ctx, "SELECT status FROM posts WHERE id=$1 FOR SHARE", in.PostID).Scan(&status); err != nil {
		return out, err
	}
	if status != "visible" {
		return out, problem(409, "post_unavailable", "Сообщение уже недоступно.")
	}
	var exists bool
	if err = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM reports WHERE post_id=$1 AND reporter_session_id=$2 AND status='open')", in.PostID, sessionID).Scan(&exists); err != nil {
		return out, err
	}
	if exists {
		return out, problem(409, "duplicate_report", "Ваша жалоба уже ожидает рассмотрения.")
	}
	if err = sessionLimit(ctx, tx, sessionID, "report", 3, 60); err != nil {
		return out, err
	}
	if err = tx.QueryRow(ctx, "INSERT INTO reports(post_id,reporter_session_id,reason,comment) VALUES($1,$2,$3,$4) RETURNING id", in.PostID, sessionID, in.Reason, in.Comment).Scan(&out.ReportID); err != nil {
		return out, err
	}
	if err = tx.Commit(ctx); err != nil {
		return out, err
	}
	return out, nil
}

// Every anonymous mutation first locks the session row, serializing publication,
// rate-limit checks and duplicate-report checks for the same visitor.
func (s *Server) beginAnonymousWrite(ctx context.Context, sessionID string) (pgx.Tx, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	var active bool
	err = tx.QueryRow(ctx, "SELECT true FROM anonymous_sessions WHERE id=$1 AND expires_at>now() AND revoked_at IS NULL FOR UPDATE", sessionID).Scan(&active)
	if err != nil {
		_ = tx.Rollback(ctx)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, problem(403, "session_expired", "Сессия завершена. Обновите страницу.")
		}
		return nil, err
	}
	return tx, nil
}
