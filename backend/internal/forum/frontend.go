package forum

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

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
