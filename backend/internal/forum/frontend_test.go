package forum

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServeFrontend(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	for path, body := range map[string]string{"index.html": "<html>Forum shell</html>", "assets/app.css": "body{color:black}"} {
		if err := os.WriteFile(filepath.Join(dir, path), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	app := &Server{cfg: Config{StaticDir: dir}}
	for _, tc := range []struct {
		name, method, path, contains string
		status                       int
	}{
		{"SPA route", "GET", "/topics/example", "Forum shell", 200},
		{"asset", "GET", "/assets/app.css", "body{color:black}", 200},
		{"missing asset", "GET", "/assets/missing.js", "404", 404},
		{"head", "HEAD", "/login", "", 200},
		{"write", "POST", "/login", "Method not allowed", 405},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			app.serveFrontend(rec, httptest.NewRequest(tc.method, tc.path, nil))
			if rec.Code != tc.status || !strings.Contains(rec.Body.String(), tc.contains) {
				t.Fatalf("response=%d %q; want %d containing %q", rec.Code, rec.Body.String(), tc.status, tc.contains)
			}
			if tc.method == "HEAD" && rec.Body.Len() != 0 {
				t.Fatal("HEAD response contains a body")
			}
		})
	}
}
