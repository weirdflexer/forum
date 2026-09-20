package forum

import (
	"crypto/subtle"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type identity struct{ ID, Raw, Role, Login string }

func (s *Server) identify(r *http.Request, st *requestState) error {
	if c, e := r.Cookie("forum_session"); e == nil && len(c.Value) == 43 {
		var id string
		e = s.db.QueryRow(r.Context(), "SELECT id FROM anonymous_sessions WHERE token_hash=$1 AND expires_at>now() AND revoked_at IS NULL", digest(c.Value)).Scan(&id)
		if e == nil {
			st.Anon = identity{ID: id, Raw: c.Value}
		} else if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
	}
	if c, e := r.Cookie("forum_staff"); e == nil && len(c.Value) == 43 {
		var id, login, role string
		e = s.db.QueryRow(r.Context(), "SELECT a.id,a.login,a.role FROM staff_sessions s JOIN staff_accounts a ON a.id=s.staff_id WHERE s.token_hash=$1 AND s.expires_at>now() AND a.is_active", digest(c.Value)).Scan(&id, &login, &role)
		if e == nil {
			st.Staff = identity{ID: id, Raw: c.Value, Login: login, Role: role}
		} else if !errors.Is(e, pgx.ErrNoRows) {
			return e
		}
	}
	return nil
}
func (s *Server) require(r *http.Request, role string) (identity, error) {
	id := state(r).Anon
	if role != "anonymous" {
		id = state(r).Staff
	}
	if id.ID == "" {
		return id, problem(403, "forbidden", "Для этого действия нужна действующая сессия и соответствующие права.")
	}
	if role == "admin" && id.Role != "admin" {
		return id, problem(403, "forbidden", "Действие доступно администратору.")
	}
	if r.Method != "GET" && r.Method != "HEAD" {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(s.csrf(id.Raw))) != 1 {
			return id, problem(403, "csrf", "Сессия обновилась. Повторите действие.")
		}
	}
	return id, nil
}
