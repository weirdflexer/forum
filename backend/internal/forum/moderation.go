package forum

import (
	"net/http"
	"strings"
	"time"
)

func (s *Server) modReports(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.require(r, "moderator"); err != nil {
		return err
	}
	page, err := pagination(r)
	if err != nil {
		return err
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "open"
	}
	if status != "open" && status != "dismissed" && status != "resolved" {
		return problem(400, "validation", "Неизвестный статус жалобы.")
	}
	var total int
	if err = s.db.QueryRow(r.Context(), "SELECT count(*) FROM reports WHERE status=$1", status).Scan(&total); err != nil {
		return err
	}
	rows, err := s.db.Query(r.Context(), `SELECT r.id,r.post_id,r.reason,r.comment,r.status,r.decision_reason,r.created_at,p.body,p.status,p.topic_id,p.number,t.title FROM reports r JOIN posts p ON p.id=r.post_id JOIN topics t ON t.id=p.topic_id WHERE r.status=$1 ORDER BY r.created_at,r.id LIMIT 20 OFFSET $2`, status, (page-1)*20)
	if err != nil {
		return err
	}
	defer rows.Close()
	type report struct {
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
	items := []report{}
	for rows.Next() {
		var x report
		if err = rows.Scan(&x.ID, &x.PostID, &x.Reason, &x.Comment, &x.Status, &x.DecisionReason, &x.CreatedAt, &x.Body, &x.PostStatus, &x.TopicID, &x.Number, &x.Title); err != nil {
			return err
		}
		items = append(items, x)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"items": items, "page": page, "total": total, "page_size": 20})
}
func (s *Server) decideReport(w http.ResponseWriter, r *http.Request) error {
	staff, err := s.require(r, "moderator")
	if err != nil {
		return err
	}
	var in struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason"`
	}
	if err = decode(w, r, &in); err != nil {
		return err
	}
	in.Reason = strings.TrimSpace(in.Reason)
	if (in.Decision != "hide" && in.Decision != "dismiss") || !validText(in.Reason, 1, 1000) {
		return problem(400, "validation", "Укажите решение и причину (1–1000 символов).")
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(r.Context())
	var pid, status string
	if err = tx.QueryRow(r.Context(), "SELECT post_id,status FROM reports WHERE id=$1 FOR UPDATE", r.PathValue("id")).Scan(&pid, &status); err != nil {
		return err
	}
	if status != "open" {
		return problem(409, "already_resolved", "Жалоба уже обработана.")
	}
	status = "dismissed"
	if in.Decision == "hide" {
		status = "resolved"
		if _, err = tx.Exec(r.Context(), "UPDATE posts SET status='hidden' WHERE id=$1 AND status='visible'", pid); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(r.Context(), "UPDATE reports SET status=$2,decision_reason=$3,resolved_at=now() WHERE id=$1", r.PathValue("id"), status, in.Reason); err != nil {
		return err
	}
	if _, err = tx.Exec(r.Context(), "INSERT INTO moderation_actions(staff_id,post_id,action,reason) VALUES($1,$2,$3,$4)", staff.ID, pid, in.Decision, in.Reason); err != nil {
		return err
	}
	if err = tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 200, map[string]string{"status": status})
}
func (s *Server) changeTopic(w http.ResponseWriter, r *http.Request) error {
	staff, err := s.require(r, "moderator")
	if err != nil {
		return err
	}
	var in struct {
		Status string `json:"status"`
		Reason string `json:"reason"`
	}
	if err = decode(w, r, &in); err != nil {
		return err
	}
	in.Reason = strings.TrimSpace(in.Reason)
	if (in.Status != "open" && in.Status != "closed") || !validText(in.Reason, 1, 1000) {
		return problem(400, "validation", "Укажите статус и причину (1–1000 символов).")
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(r.Context())
	tag, err := tx.Exec(r.Context(), "UPDATE topics SET status=$2 WHERE id=$1", r.PathValue("id"), in.Status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return problem(404, "not_found", "Тема не найдена.")
	}
	if _, err = tx.Exec(r.Context(), "INSERT INTO moderation_actions(staff_id,topic_id,action,reason) VALUES($1,$2,$3,$4)", staff.ID, r.PathValue("id"), in.Status, in.Reason); err != nil {
		return err
	}
	if err = tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 200, map[string]string{"status": in.Status})
}
func (s *Server) modActions(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.require(r, "moderator"); err != nil {
		return err
	}
	page, err := pagination(r)
	if err != nil {
		return err
	}
	var total int
	if err = s.db.QueryRow(r.Context(), "SELECT count(*) FROM moderation_actions").Scan(&total); err != nil {
		return err
	}
	rows, err := s.db.Query(r.Context(), `SELECT m.id,a.login,m.action,m.reason,m.created_at,COALESCE(m.topic_id,p.topic_id)::text FROM moderation_actions m JOIN staff_accounts a ON a.id=m.staff_id LEFT JOIN posts p ON p.id=m.post_id ORDER BY m.created_at DESC,m.id DESC LIMIT 20 OFFSET $1`, (page-1)*20)
	if err != nil {
		return err
	}
	defer rows.Close()
	type action struct {
		ID        string    `json:"id"`
		Login     string    `json:"login"`
		Action    string    `json:"action"`
		Reason    string    `json:"reason"`
		CreatedAt time.Time `json:"created_at"`
		TopicID   string    `json:"topic_id"`
	}
	items := []action{}
	for rows.Next() {
		var x action
		if err = rows.Scan(&x.ID, &x.Login, &x.Action, &x.Reason, &x.CreatedAt, &x.TopicID); err != nil {
			return err
		}
		items = append(items, x)
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"items": items, "total": total, "page": page, "page_size": 20})
}

type sectionInput struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Archived    bool   `json:"is_archived"`
}

func validateSection(in *sectionInput) error {
	in.Title = strings.TrimSpace(in.Title)
	in.Slug = strings.TrimSpace(in.Slug)
	in.Description = strings.TrimSpace(in.Description)
	if !slugPattern.MatchString(in.Slug) || !validText(in.Title, 2, 80) || !validText(in.Description, 0, 240) {
		return problem(400, "validation", "Название: 2–80 символов; адрес: 2–40 латинских букв, цифр или дефисов; описание: до 240.")
	}
	return nil
}
func (s *Server) createSection(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.require(r, "admin"); err != nil {
		return err
	}
	var in sectionInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if err := validateSection(&in); err != nil {
		return err
	}
	var id string
	if err := s.db.QueryRow(r.Context(), "INSERT INTO sections(slug,title,description,is_archived) VALUES($1,$2,$3,$4) RETURNING id", in.Slug, in.Title, in.Description, in.Archived).Scan(&id); err != nil {
		return err
	}
	return respond(w, 201, map[string]string{"id": id})
}
func (s *Server) updateSection(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.require(r, "admin"); err != nil {
		return err
	}
	var in sectionInput
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if err := validateSection(&in); err != nil {
		return err
	}
	tag, err := s.db.Exec(r.Context(), "UPDATE sections SET slug=$2,title=$3,description=$4,is_archived=$5 WHERE id=$1", r.PathValue("id"), in.Slug, in.Title, in.Description, in.Archived)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return problem(404, "not_found", "Раздел не найден.")
	}
	return respond(w, 204, nil)
}
func (s *Server) listStaff(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.require(r, "admin"); err != nil {
		return err
	}
	rows, err := s.db.Query(r.Context(), "SELECT id,login,role FROM staff_accounts WHERE is_active ORDER BY login")
	if err != nil {
		return err
	}
	defer rows.Close()
	items := []map[string]string{}
	for rows.Next() {
		var id, login, role string
		if err = rows.Scan(&id, &login, &role); err != nil {
			return err
		}
		items = append(items, map[string]string{"id": id, "login": login, "role": role})
	}
	if err = rows.Err(); err != nil {
		return err
	}
	return respond(w, 200, map[string]any{"items": items})
}
func (s *Server) addStaff(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.require(r, "admin"); err != nil {
		return err
	}
	var in struct {
		Login    string `json:"login"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if !loginPattern.MatchString(in.Login) || len(in.Password) < 12 || len(in.Password) > 128 || (in.Role != "moderator" && in.Role != "admin") {
		return problem(400, "validation", "Логин: 3–40 латинских символов; пароль: 12–128 байт; роль: moderator/admin.")
	}
	if err := CreateStaff(r.Context(), s.db, in.Login, in.Password, in.Role); err != nil {
		return err
	}
	return respond(w, 201, map[string]string{"login": in.Login})
}
func (s *Server) changeRole(w http.ResponseWriter, r *http.Request) error {
	if _, err := s.require(r, "admin"); err != nil {
		return err
	}
	var in struct {
		Role string `json:"role"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if in.Role != "admin" && in.Role != "moderator" {
		return problem(400, "validation", "Неизвестная роль.")
	}
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(r.Context())
	if _, err = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(72830403)"); err != nil {
		return err
	}
	var old string
	if err = tx.QueryRow(r.Context(), "SELECT role FROM staff_accounts WHERE id=$1 FOR UPDATE", r.PathValue("id")).Scan(&old); err != nil {
		return err
	}
	if old == "admin" && in.Role != "admin" {
		var count int
		if err = tx.QueryRow(r.Context(), "SELECT count(*) FROM staff_accounts WHERE role='admin' AND is_active").Scan(&count); err != nil {
			return err
		}
		if count <= 1 {
			return problem(409, "last_admin", "Нельзя убрать последнего администратора.")
		}
	}
	if _, err = tx.Exec(r.Context(), "UPDATE staff_accounts SET role=$2 WHERE id=$1", r.PathValue("id"), in.Role); err != nil {
		return err
	}
	if _, err = tx.Exec(r.Context(), "DELETE FROM staff_sessions WHERE staff_id=$1", r.PathValue("id")); err != nil {
		return err
	}
	if err = tx.Commit(r.Context()); err != nil {
		return err
	}
	return respond(w, 204, nil)
}
