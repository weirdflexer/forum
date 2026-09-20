package forum

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type requestState struct {
	Anon, Staff identity
	RequestID   string
}
type stateKey struct{}

func state(r *http.Request) *requestState { return r.Context().Value(stateKey{}).(*requestState) }

func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		st := &requestState{RequestID: token()[:16]}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		r = r.WithContext(context.WithValue(ctx, stateKey{}, st))
		w.Header().Set("X-Request-ID", st.RequestID)
		s.securityHeaders(w)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		defer func() {
			if rec := recover(); rec != nil {
				s.fail(w, r, fmt.Errorf("panic recovered"))
			}
		}()
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
			if r.Header.Get("Origin") != s.cfg.Origin {
				s.fail(w, r, problem(403, "origin", "Запрос с другого сайта запрещён."))
				return
			}
			if err := s.networkLimit(w, r, "write", s.cfg.NetworkWriteLimit); err != nil {
				s.fail(w, r, err)
				return
			}
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if err := s.identify(r, st); err != nil {
				s.fail(w, r, err)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) securityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; font-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
	if s.cfg.SecureCookies {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000")
	}
}
