package forum

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) lockSession(r *http.Request, tx pgx.Tx, id string) error {
	var ok bool
	err := tx.QueryRow(r.Context(), "SELECT true FROM anonymous_sessions WHERE id=$1 AND expires_at>now() AND revoked_at IS NULL FOR UPDATE", id).Scan(&ok)
	if errors.Is(err, pgx.ErrNoRows) {
		return problem(403, "session_expired", "Сессия завершена. Обновите страницу.")
	}
	return err
}
func (s *Server) sessionLimit(w http.ResponseWriter, r *http.Request, tx pgx.Tx, id, action string, max, seconds int) error {
	var count int
	var oldest *time.Time
	err := tx.QueryRow(r.Context(), "SELECT count(*),min(created_at) FROM rate_events WHERE session_id=$1 AND action=$2 AND created_at>now()-make_interval(secs=>$3)", id, action, seconds).Scan(&count, &oldest)
	if err != nil {
		return err
	}
	if count >= max {
		wait := seconds
		if oldest != nil {
			wait = int(time.Until(oldest.Add(time.Duration(seconds)*time.Second)).Seconds()) + 1
		}
		if wait < 1 {
			wait = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(wait))
		return problem(429, "rate_limit", fmt.Sprintf("Подождите %d сек. перед следующим действием.", wait))
	}
	_, err = tx.Exec(r.Context(), "INSERT INTO rate_events(session_id,action) VALUES($1,$2)", id, action)
	return err
}
func idemKey(r *http.Request) (string, error) {
	key := r.Header.Get("Idempotency-Key")
	if len(key) < 8 || len(key) > 128 {
		return "", problem(400, "idempotency_key", "Нужен Idempotency-Key длиной 8–128 символов.")
	}
	return key, nil
}
func replay(w http.ResponseWriter, r *http.Request, tx pgx.Tx, sid, key, hash string) (bool, error) {
	var oldHash string
	var body json.RawMessage
	var status int
	err := tx.QueryRow(r.Context(), "SELECT request_hash,response,status FROM idempotency_keys WHERE session_id=$1 AND key=$2 AND expires_at>now()", sid, key).Scan(&oldHash, &body, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if hash != oldHash {
		return true, problem(409, "idempotency_conflict", "Этот ключ уже использован для другого запроса.")
	}
	w.Header().Set("Idempotency-Replayed", "true")
	return true, respond(w, status, body)
}
func saveReply(r *http.Request, tx pgx.Tx, sid, key, hash string, result any) error {
	b, err := json.Marshal(result)
	if err != nil {
		return err
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO idempotency_keys(session_id,key,request_hash,response,status,expires_at) VALUES($1,$2,$3,$4,201,now()+interval '24 hours') ON CONFLICT(session_id,key) DO UPDATE SET request_hash=excluded.request_hash,response=excluded.response,status=201,expires_at=excluded.expires_at`, sid, key, hash, b)
	return err
}
func requestHash(r *http.Request, v any) string {
	b, _ := json.Marshal(v)
	return digest(r.URL.Path + string(b))
}
func (s *Server) createTopic(w http.ResponseWriter, r *http.Request) error {
	id, err := s.require(r, "anonymous")
	if err != nil {
		return err
	}
	key, err := idemKey(r)
	if err != nil {
		return err
	}
	var in struct {
		SectionID string `json:"section_id"`
		Title     string `json:"title"`
		Body      string `json:"body"`
	}
	if err = decode(w, r, &in); err != nil {
		return err
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Body = strings.TrimSpace(in.Body)
	if !uuidPattern.MatchString(in.SectionID) || !validText(in.Title, 5, 120) || !validText(in.Body, 1, 5000) {
		return problem(400, "validation", "Выберите раздел. Заголовок: 5–120 символов; сообщение: 1–5000.")
	}
	hash := requestHash(r, in)
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(r.Context())
	if err = s.lockSession(r, tx, id.ID); err != nil {
		return err
	}
	if done, e := replay(w, r, tx, id.ID, key, hash); done || e != nil {
		return e
	}
	var archived bool
	if err = tx.QueryRow(r.Context(), "SELECT is_archived FROM sections WHERE id=$1 FOR SHARE", in.SectionID).Scan(&archived); err != nil {
		return err
	}
	if archived {
		return problem(409, "section_archived", "Раздел в архиве: новые темы недоступны.")
	}
	if err = s.sessionLimit(w, r, tx, id.ID, "topic", 1, 60); err != nil {
		return err
	}
	var tid, pid string
	if err = tx.QueryRow(r.Context(), "INSERT INTO topics(section_id,author_session_id,title) VALUES($1,$2,$3) RETURNING id", in.SectionID, id.ID, in.Title).Scan(&tid); err != nil {
		return err
	}
	if err = tx.QueryRow(r.Context(), "INSERT INTO posts(topic_id,author_session_id,number,body) VALUES($1,$2,1,$3) RETURNING id", tid, id.ID, in.Body).Scan(&pid); err != nil {
		return err
	}
	out := map[string]string{"topic_id": tid, "first_post_id": pid}
	if err = saveReply(r, tx, id.ID, key, hash, out); err != nil {
		return err
	}
	if err = tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 201, out)
}
func (s *Server) createPost(w http.ResponseWriter, r *http.Request) error {
	id, err := s.require(r, "anonymous")
	if err != nil {
		return err
	}
	key, err := idemKey(r)
	if err != nil {
		return err
	}
	var in struct {
		Body string `json:"body"`
	}
	if err = decode(w, r, &in); err != nil {
		return err
	}
	in.Body = strings.TrimSpace(in.Body)
	if !validText(in.Body, 1, 5000) {
		return problem(400, "validation", "Сообщение: от 1 до 5000 символов.")
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(r.Context())
	if err = s.lockSession(r, tx, id.ID); err != nil {
		return err
	}
	hash := requestHash(r, in)
	if done, e := replay(w, r, tx, id.ID, key, hash); done || e != nil {
		return e
	}
	var status string
	if err = tx.QueryRow(r.Context(), "SELECT status FROM topics WHERE id=$1 FOR UPDATE", r.PathValue("id")).Scan(&status); err != nil {
		return err
	}
	if status == "closed" {
		return problem(409, "topic_closed", "Тема закрыта для новых ответов.")
	}
	if err = s.sessionLimit(w, r, tx, id.ID, "post", 1, 10); err != nil {
		return err
	}
	var pid string
	var number int
	err = tx.QueryRow(r.Context(), "INSERT INTO posts(topic_id,author_session_id,number,body) SELECT $1,$2,COALESCE(max(number),0)+1,$3 FROM posts WHERE topic_id=$1 RETURNING id,number", r.PathValue("id"), id.ID, in.Body).Scan(&pid, &number)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(r.Context(), "UPDATE topics SET last_activity_at=now() WHERE id=$1", r.PathValue("id")); err != nil {
		return err
	}
	out := map[string]any{"post_id": pid, "number": number, "page": (number-1)/20 + 1}
	if err = saveReply(r, tx, id.ID, key, hash, out); err != nil {
		return err
	}
	if err = tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 201, out)
}
func (s *Server) deletePost(w http.ResponseWriter, r *http.Request) error {
	id, err := s.require(r, "anonymous")
	if err != nil {
		return err
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(r.Context())
	if err = s.lockSession(r, tx, id.ID); err != nil {
		return err
	}
	var owner string
	if err = tx.QueryRow(r.Context(), "SELECT COALESCE(author_session_id::text,'') FROM posts WHERE id=$1 FOR UPDATE", r.PathValue("id")).Scan(&owner); err != nil {
		return err
	}
	if owner != id.ID {
		return problem(403, "not_owner", "Можно удалить только сообщение своей текущей сессии.")
	}
	if _, err = tx.Exec(r.Context(), "UPDATE posts SET body='',status='deleted' WHERE id=$1", r.PathValue("id")); err != nil {
		return err
	}
	if err = tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 204, nil)
}
func (s *Server) createReport(w http.ResponseWriter, r *http.Request) error {
	id, err := s.require(r, "anonymous")
	if err != nil {
		return err
	}
	var in struct {
		PostID  string `json:"post_id"`
		Reason  string `json:"reason"`
		Comment string `json:"comment"`
	}
	if err = decode(w, r, &in); err != nil {
		return err
	}
	in.Comment = strings.TrimSpace(in.Comment)
	reasons := map[string]bool{"spam": true, "abuse": true, "personal_data": true, "other": true}
	if !uuidPattern.MatchString(in.PostID) || !reasons[in.Reason] || !validText(in.Comment, 0, 1000) {
		return problem(400, "validation", "Выберите причину. Комментарий — до 1000 символов.")
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(r.Context())
	if err = s.lockSession(r, tx, id.ID); err != nil {
		return err
	}
	var status string
	if err = tx.QueryRow(r.Context(), "SELECT status FROM posts WHERE id=$1 FOR SHARE", in.PostID).Scan(&status); err != nil {
		return err
	}
	if status != "visible" {
		return problem(409, "post_unavailable", "Сообщение уже недоступно.")
	}
	var exists bool
	if err = tx.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM reports WHERE post_id=$1 AND reporter_session_id=$2 AND status='open')", in.PostID, id.ID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return problem(409, "duplicate_report", "Ваша жалоба уже ожидает рассмотрения.")
	}
	if err = s.sessionLimit(w, r, tx, id.ID, "report", 3, 60); err != nil {
		return err
	}
	var rid string
	if err = tx.QueryRow(r.Context(), "INSERT INTO reports(post_id,reporter_session_id,reason,comment) VALUES($1,$2,$3,$4) RETURNING id", in.PostID, id.ID, in.Reason, in.Comment).Scan(&rid); err != nil {
		return err
	}
	if err = tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 201, map[string]string{"report_id": rid})
}
