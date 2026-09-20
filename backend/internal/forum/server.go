package forum

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config contains HTTP, security, and frontend settings for the forum.
type Config struct {
	NetworkWriteLimit, NetworkSessionLimit int
	Origin, Secret, StaticDir              string
	SecureCookies                          bool
}

// Server serves the forum API and its built frontend using a shared database pool.
type Server struct {
	db        *pgxpool.Pool
	cfg       Config
	limits    limiter
	dummyHash string
}

// New constructs a server and applies the default per-network request limits.
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

// Handler assembles API routes, the frontend, and the shared request middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	routes := map[string]handler{
		"GET /health":                            s.health,
		"GET /api/v1/session":                    s.sessionInfo,
		"POST /api/v1/sessions":                  s.createSession,
		"DELETE /api/v1/session":                 s.endSession,
		"GET /api/v1/sections":                   s.sections,
		"GET /api/v1/topics":                     s.topics,
		"GET /api/v1/topics/{id}/posts":          s.posts,
		"POST /api/v1/topics":                    s.createTopic,
		"POST /api/v1/topics/{id}/posts":         s.createPost,
		"DELETE /api/v1/posts/{id}":              s.deletePost,
		"POST /api/v1/reports":                   s.createReport,
		"POST /api/v1/staff/session":             s.staffLogin,
		"DELETE /api/v1/staff/session":           s.staffLogout,
		"GET /api/v1/mod/reports":                s.modReports,
		"POST /api/v1/mod/reports/{id}/decision": s.decideReport,
		"PATCH /api/v1/mod/topics/{id}":          s.changeTopic,
		"GET /api/v1/mod/actions":                s.modActions,
		"POST /api/v1/admin/sections":            s.createSection,
		"PATCH /api/v1/admin/sections/{id}":      s.updateSection,
		"GET /api/v1/admin/staff":                s.listStaff,
		"POST /api/v1/admin/staff":               s.addStaff,
		"PATCH /api/v1/admin/staff/{id}/role":    s.changeRole,
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

func (s *Server) health(w http.ResponseWriter, r *http.Request) error {
	if err := s.db.Ping(r.Context()); err != nil {
		return problem(503, "unavailable", "База данных недоступна.")
	}
	return respond(w, 200, map[string]string{"status": "ok"})
}
