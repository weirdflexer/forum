package forum

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	NetworkWriteLimit, NetworkSessionLimit int
	Origin, Secret, StaticDir              string
	SecureCookies                          bool
}
type Server struct {
	db        *pgxpool.Pool
	cfg       Config
	limits    limiter
	dummyHash string
}
type identity struct{ ID, Raw, Role, Login string }
type requestState struct {
	Anon, Staff identity
	RequestID   string
}
type stateKey struct{}

func state(r *http.Request) *requestState { return r.Context().Value(stateKey{}).(*requestState) }

type apiError struct {
	Status    int    `json:"-"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func (e *apiError) Error() string { return e.Code }
func problem(status int, code, message string) error {
	return &apiError{Status: status, Code: code, Message: message}
}

type handler func(http.ResponseWriter, *http.Request) error

func New(db *pgxpool.Pool, cfg Config) *Server {
	if cfg.NetworkWriteLimit <= 0 {
		cfg.NetworkWriteLimit = 120
	}
	if cfg.NetworkSessionLimit <= 0 {
		cfg.NetworkSessionLimit = 20
	}
	dummy, _ := HashPassword(token())
	return &Server{db: db, cfg: cfg, dummyHash: dummy}
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	routes := map[string]handler{
		"GET /health": s.health, "GET /api/v1/session": s.sessionInfo, "POST /api/v1/sessions": s.createSession, "DELETE /api/v1/session": s.endSession,
		"GET /api/v1/sections": s.sections, "GET /api/v1/topics": s.topics, "GET /api/v1/topics/{id}/posts": s.posts,
		"POST /api/v1/topics": s.createTopic, "POST /api/v1/topics/{id}/posts": s.createPost, "DELETE /api/v1/posts/{id}": s.deletePost, "POST /api/v1/reports": s.createReport,
		"POST /api/v1/staff/session": s.staffLogin, "DELETE /api/v1/staff/session": s.staffLogout,
		"GET /api/v1/mod/reports": s.modReports, "POST /api/v1/mod/reports/{id}/decision": s.decideReport, "PATCH /api/v1/mod/topics/{id}": s.changeTopic, "GET /api/v1/mod/actions": s.modActions,
		"POST /api/v1/admin/sections": s.createSection, "PATCH /api/v1/admin/sections/{id}": s.updateSection,
		"GET /api/v1/admin/staff": s.listStaff, "POST /api/v1/admin/staff": s.addStaff, "PATCH /api/v1/admin/staff/{id}/role": s.changeRole,
	}
	for path, h := range routes {
		mux.HandleFunc(path, s.wrap(h))
	}
	mux.HandleFunc("/api/", s.wrap(func(w http.ResponseWriter, r *http.Request) error {
		return problem(404, "not_found", "Метод API не найден.")
	}))
	mux.HandleFunc("/", s.serveFrontend)
	return s.middleware(mux)
}
func respond(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if status != 204 {
		return json.NewEncoder(w).Encode(v)
	}
	return nil
}
func decode(w http.ResponseWriter, r *http.Request, dst any) error {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		return problem(415, "content_type", "Используйте application/json.")
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32768)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return problem(400, "invalid_json", "Некорректные или лишние поля JSON.")
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return problem(400, "invalid_json", "Ожидается один JSON-объект.")
	}
	return nil
}
func (s *Server) wrap(h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if id := r.PathValue("id"); id != "" && !uuidPattern.MatchString(id) {
			s.fail(w, r, problem(400, "invalid_id", "Некорректный идентификатор."))
			return
		}
		if err := h(w, r); err != nil {
			s.fail(w, r, err)
		}
	}
}
func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	var p *apiError
	if errors.Is(err, pgx.ErrNoRows) {
		err = problem(404, "not_found", "Запись не найдена.")
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		err = problem(409, "conflict", "Такая запись уже существует.")
	}
	if !errors.As(err, &p) {
		slog.Error("request failed", "request_id", state(r).RequestID, "error_type", fmt.Sprintf("%T", err))
		p = &apiError{Status: 500, Code: "internal_error", Message: "Не удалось выполнить запрос. Попробуйте снова."}
	}
	p.RequestID = state(r).RequestID
	_ = respond(w, p.Status, p)
}
func (s *Server) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		st := &requestState{RequestID: token()[:16]}
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		r = r.WithContext(context.WithValue(ctx, stateKey{}, st))
		w.Header().Set("X-Request-ID", st.RequestID)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; font-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		if s.cfg.SecureCookies {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		}
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
func (s *Server) health(w http.ResponseWriter, r *http.Request) error {
	if err := s.db.Ping(r.Context()); err != nil {
		return problem(503, "unavailable", "База данных недоступна.")
	}
	return respond(w, 200, map[string]string{"status": "ok"})
}
func (s *Server) serveFrontend(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" && r.Method != "HEAD" {
		http.Error(w, "Method not allowed", 405)
		return
	}
	if s.cfg.StaticDir == "" {
		http.NotFound(w, r)
		return
	}
	clean := filepath.Clean("/" + r.URL.Path)
	path := filepath.Join(s.cfg.StaticDir, clean)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		http.ServeFile(w, r, path)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/assets/") {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(s.cfg.StaticDir, "index.html"))
}
