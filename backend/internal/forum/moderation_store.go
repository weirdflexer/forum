package forum

import (
	"context"
	"time"
)

type moderationReport struct {
	ID             string    `json:"id"`
	PostID         string    `json:"post_id"`
	Reason         string    `json:"reason"`
	Comment        string    `json:"comment"`
	Status         string    `json:"status"`
	DecisionReason string    `json:"decision_reason"`
	CreatedAt      time.Time `json:"created_at"`
	Body           string    `json:"body"`
	PostStatus     string    `json:"post_status"`
	TopicID        string    `json:"topic_id"`
	Number         int       `json:"number"`
	Title          string    `json:"topic_title"`
}

func (s *Server) queryModerationReports(ctx context.Context, status string, page int) ([]moderationReport, int, error) {
	var total int
	if err := s.db.QueryRow(ctx, "SELECT count(*) FROM reports WHERE status=$1", status).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT r.id,r.post_id,r.reason,r.comment,r.status,r.decision_reason,r.created_at,
		       p.body,p.status,p.topic_id,p.number,t.title
		FROM reports r
		JOIN posts p ON p.id=r.post_id
		JOIN topics t ON t.id=p.topic_id
		WHERE r.status=$1 ORDER BY r.created_at,r.id LIMIT 20 OFFSET $2`, status, (page-1)*20)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []moderationReport{}
	for rows.Next() {
		var x moderationReport
		if err := rows.Scan(&x.ID, &x.PostID, &x.Reason, &x.Comment, &x.Status, &x.DecisionReason, &x.CreatedAt, &x.Body, &x.PostStatus, &x.TopicID, &x.Number, &x.Title); err != nil {
			return nil, 0, err
		}
		items = append(items, x)
	}
	return items, total, rows.Err()
}

// applyReportDecision serializes competing decisions on one report and commits
// the post visibility change, report result and audit entry together.
func (s *Server) applyReportDecision(ctx context.Context, staffID, reportID string, in reportDecisionInput) (string, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var postID, status string
	if err := tx.QueryRow(ctx, "SELECT post_id,status FROM reports WHERE id=$1 FOR UPDATE", reportID).Scan(&postID, &status); err != nil {
		return "", err
	}
	if status != "open" {
		return "", problem(409, "already_resolved", "Жалоба уже обработана.")
	}
	status = "dismissed"
	if in.Decision == "hide" {
		status = "resolved"
		// Deleted posts must stay deleted, including their cleared body.
		if _, err := tx.Exec(ctx, "UPDATE posts SET status='hidden' WHERE id=$1 AND status='visible'", postID); err != nil {
			return "", err
		}
	}
	if _, err := tx.Exec(ctx, "UPDATE reports SET status=$2,decision_reason=$3,resolved_at=now() WHERE id=$1", reportID, status, in.Reason); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, "INSERT INTO moderation_actions(staff_id,post_id,action,reason) VALUES($1,$2,$3,$4)", staffID, postID, in.Decision, in.Reason); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return status, nil
}

func (s *Server) setTopicStatus(ctx context.Context, staffID, topicID string, in topicStatusInput) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, "UPDATE topics SET status=$2 WHERE id=$1", topicID, in.Status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return problem(404, "not_found", "Тема не найдена.")
	}
	if _, err := tx.Exec(ctx, "INSERT INTO moderation_actions(staff_id,topic_id,action,reason) VALUES($1,$2,$3,$4)", staffID, topicID, in.Status, in.Reason); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type moderationAction struct {
	ID        string    `json:"id"`
	Login     string    `json:"login"`
	Action    string    `json:"action"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
	TopicID   string    `json:"topic_id"`
}

func (s *Server) queryModerationActions(ctx context.Context, page int) ([]moderationAction, int, error) {
	var total int
	if err := s.db.QueryRow(ctx, "SELECT count(*) FROM moderation_actions").Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := s.db.Query(ctx, `
		SELECT m.id,a.login,m.action,m.reason,m.created_at,COALESCE(m.topic_id,p.topic_id)::text
		FROM moderation_actions m
		JOIN staff_accounts a ON a.id=m.staff_id
		LEFT JOIN posts p ON p.id=m.post_id
		ORDER BY m.created_at DESC,m.id DESC LIMIT 20 OFFSET $1`, (page-1)*20)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := []moderationAction{}
	for rows.Next() {
		var x moderationAction
		if err := rows.Scan(&x.ID, &x.Login, &x.Action, &x.Reason, &x.CreatedAt, &x.TopicID); err != nil {
			return nil, 0, err
		}
		items = append(items, x)
	}
	return items, total, rows.Err()
}
