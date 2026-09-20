package forum

import "net/http"

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
