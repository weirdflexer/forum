package forum

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (s *Server) staffLogin(w http.ResponseWriter, r *http.Request) error {
	if err := s.networkLimit(w, r, "login", 5); err != nil {
		return err
	}
	var in struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := decode(w, r, &in); err != nil {
		return err
	}
	if len(in.Password) > 128 {
		return problem(400, "validation", "Пароль слишком длинный.")
	}
	var id, hash, role string
	in.Login = strings.TrimSpace(in.Login)
	err := s.db.QueryRow(r.Context(), "SELECT id,password_hash,role FROM staff_accounts WHERE login=$1 AND is_active", in.Login).Scan(&id, &hash, &role)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if hash == "" {
		hash = s.dummyHash
	}
	valid := VerifyPassword(in.Password, hash)
	if !valid || id == "" {
		return problem(401, "login_failed", "Неверный логин или пароль.")
	}
	raw := token()
	tx, err := s.db.Begin(r.Context())
	if err != nil {
		return err
	}
	defer tx.Rollback(r.Context())
	if state(r).Staff.Raw != "" {
		if _, err = tx.Exec(r.Context(), "DELETE FROM staff_sessions WHERE token_hash=$1", digest(state(r).Staff.Raw)); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(r.Context(), "INSERT INTO staff_sessions(staff_id,token_hash,expires_at) VALUES($1,$2,now()+interval '12 hours')", id, digest(raw)); err != nil {
		return err
	}
	if err = tx.Commit(r.Context()); err != nil {
		return err
	}
	s.cookie(w, "forum_staff", raw, 12*3600)
	state(r).Staff = identity{ID: id, Raw: raw, Login: in.Login, Role: role}
	return s.sessionInfo(w, r)
}
func (s *Server) staffLogout(w http.ResponseWriter, r *http.Request) error {
	id, err := s.require(r, "moderator")
	if err != nil {
		return err
	}
	if _, err = s.db.Exec(r.Context(), "DELETE FROM staff_sessions WHERE token_hash=$1", digest(id.Raw)); err != nil {
		return err
	}
	s.cookie(w, "forum_staff", "", -1)
	return respond(w, 204, nil)
}
