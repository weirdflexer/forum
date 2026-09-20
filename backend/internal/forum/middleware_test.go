package forum

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMiddlewareRequestContextAndHeaders(t *testing.T) {
	for _, secure := range []bool{false, true} {
		app := &Server{cfg: Config{SecureCookies: secure}}
		called := false
		h := app.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			if state(r).RequestID == "" || state(r).RequestID != w.Header().Get("X-Request-ID") {
				t.Fatal("request ID is missing or differs between context and response")
			}
			deadline, ok := r.Context().Deadline()
			if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > 10*time.Second {
				t.Fatal("request has no bounded 10-second context")
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/session", nil))
		if !called || rec.Code != http.StatusNoContent {
			t.Fatal("middleware did not reach the handler")
		}
		for name, want := range map[string]string{
			"X-Content-Type-Options": "nosniff", "Referrer-Policy": "no-referrer",
			"X-Frame-Options": "DENY", "Cache-Control": "no-store",
		} {
			if got := rec.Header().Get(name); got != want {
				t.Errorf("%s = %q, want %q", name, got, want)
			}
		}
		if !strings.Contains(rec.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
			t.Fatal("missing frame restriction in Content Security Policy")
		}
		if got := rec.Header().Get("Strict-Transport-Security"); (got != "") != secure {
			t.Fatalf("HSTS=%q for SecureCookies=%t", got, secure)
		}
	}
}

func TestMiddlewareWriteGuards(t *testing.T) {
	app := &Server{cfg: Config{Origin: "https://forum.example", Secret: "test", NetworkWriteLimit: 1}}
	calls := 0
	h := app.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusNoContent)
	}))
	for _, tc := range []struct {
		name, origin string
		status       int
	}{
		{"missing Origin", "", http.StatusForbidden},
		{"foreign Origin", "https://other.example", http.StatusForbidden},
		{"accepted write", app.cfg.Origin, http.StatusNoContent},
		{"write rate exceeded", app.cfg.Origin, http.StatusTooManyRequests},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/api/v1/sessions", nil)
			r.Header.Set("Origin", tc.origin)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, r)
			if rec.Code != tc.status {
				t.Fatalf("response=%d %s; want %d", rec.Code, rec.Body.String(), tc.status)
			}
			if tc.status == http.StatusTooManyRequests && rec.Header().Get("Retry-After") == "" {
				t.Fatal("rate-limited write has no Retry-After")
			}
		})
	}
	if calls != 1 {
		t.Fatalf("handler called %d times, want only the one permitted write", calls)
	}
}

func TestMiddlewareRecoversPanic(t *testing.T) {
	app := &Server{}
	h := app.middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("private implementation detail")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/session", nil))
	var body apiError
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusInternalServerError || body.Code != "internal_error" || body.RequestID != rec.Header().Get("X-Request-ID") {
		t.Fatalf("panic response=%d %#v", rec.Code, body)
	}
	if strings.Contains(rec.Body.String(), "private implementation detail") {
		t.Fatal("panic detail leaked to client")
	}
}
