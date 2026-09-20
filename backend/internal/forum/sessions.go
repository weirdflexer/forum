package forum

import (
	"errors"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
)

func (s *Server) sessionInfo(w http.ResponseWriter, r *http.Request) error {
	st := state(r)
	out := map[string]any{"active": st.Anon.ID != "", "csrf_token": "", "staff": nil}
	if st.Anon.ID != "" {
		out["csrf_token"] = s.csrf(st.Anon.Raw)
	}
	if st.Staff.ID != "" {
		out["staff"] = map[string]string{"login": st.Staff.Login, "role": st.Staff.Role, "csrf_token": s.csrf(st.Staff.Raw)}
	}
	return respond(w, 200, out)
}
func (s *Server) createSession(w http.ResponseWriter, r *http.Request) error {
	if state(r).Anon.ID != "" {
		return s.sessionInfo(w, r)
	}
	if err := s.networkLimit(w, r, "session", s.cfg.NetworkSessionLimit); err != nil {
		return err
	}
	raw := token()
	var id string
	if err := s.db.QueryRow(r.Context(), "INSERT INTO anonymous_sessions(token_hash,expires_at) VALUES($1,now()+interval '30 days') RETURNING id", digest(raw)).Scan(&id); err != nil {
		return err
	}
	s.cookie(w, "forum_session", raw, 30*86400)
	state(r).Anon = identity{ID: id, Raw: raw}
	return s.sessionInfo(w, r)
}
func (s *Server) endSession(w http.ResponseWriter, r *http.Request) error {
	id, err := s.require(r, "anonymous")
	if err != nil {
		return err
	}
	if _, err = s.db.Exec(r.Context(), "UPDATE anonymous_sessions SET revoked_at=now() WHERE id=$1", id.ID); err != nil {
		return err
	}
	s.cookie(w, "forum_session", "", -1)
	return respond(w, 204, nil)
}
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
