package forum

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestDecodeJSON(t *testing.T) {
	for _, tc := range []struct {
		name, contentType, body, code string
		status                        int
	}{
		{name: "object", contentType: "application/json", body: `{"name":"тема"}`},
		{name: "charset", contentType: "application/json; charset=utf-8", body: `{"name":"тема"}`},
		{name: "JSONP is not JSON", contentType: "application/jsonp", body: `{}`, code: "content_type", status: 415},
		{name: "suffix is not JSON", contentType: "application/json-invalid", body: `{}`, code: "content_type", status: 415},
		{name: "malformed media type", contentType: "application/json; charset", body: `{}`, code: "content_type", status: 415},
		{name: "missing media type", body: `{}`, code: "content_type", status: 415},
		{name: "unknown field", contentType: "application/json", body: `{"other":true}`, code: "invalid_json", status: 400},
		{name: "multiple objects", contentType: "application/json", body: `{} {}`, code: "invalid_json", status: 400},
		{name: "trailing null", contentType: "application/json", body: `{} null`, code: "invalid_json", status: 400},
		{name: "truncated object", contentType: "application/json", body: `{"name":`, code: "invalid_json", status: 400},
		{name: "empty body", contentType: "application/json", code: "invalid_json", status: 400},
		{name: "body limit", contentType: "application/json", body: `{"name":"` + strings.Repeat("x", 32768) + `"}`, code: "invalid_json", status: 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.contentType)
			var input struct {
				Name string `json:"name"`
			}
			err := decode(httptest.NewRecorder(), r, &input)
			if tc.code == "" {
				if err != nil || input.Name != "тема" {
					t.Fatalf("decode = %#v, %v", input, err)
				}
				return
			}
			var api *apiError
			if !errors.As(err, &api) || api.Code != tc.code || api.Status != tc.status {
				t.Fatalf("error = %#v; want %d %s", err, tc.status, tc.code)
			}
		})
	}
}

func TestHTTPErrorResponses(t *testing.T) {
	app := &Server{}
	shared := &apiError{Status: 403, Code: "forbidden", Message: "Нет доступа.", RequestID: "original"}
	for _, tc := range []struct {
		name   string
		err    error
		code   string
		status int
	}{
		{"application", shared, "forbidden", 403},
		{"missing row", fmt.Errorf("wrapped: %w", pgx.ErrNoRows), "not_found", 404},
		{"duplicate", &pgconn.PgError{Code: "23505", Detail: "private database detail"}, "conflict", 409},
		{"internal", errors.New("private database detail"), "internal_error", 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/v1/example", nil)
			r = r.WithContext(context.WithValue(r.Context(), stateKey{}, &requestState{RequestID: tc.name}))
			rec := httptest.NewRecorder()
			app.fail(rec, r, tc.err)
			var body apiError
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if rec.Code != tc.status || body.Code != tc.code || body.RequestID != tc.name {
				t.Fatalf("response = %d %#v", rec.Code, body)
			}
			if strings.Contains(rec.Body.String(), "private database detail") {
				t.Fatal("internal details leaked to the client")
			}
		})
	}
	if shared.RequestID != "original" {
		t.Fatal("responding with an error mutated the caller's shared error")
	}
}

func TestWrapRejectsInvalidID(t *testing.T) {
	app := &Server{}
	called := false
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/topics/{id}/posts", app.wrap(func(w http.ResponseWriter, r *http.Request) error {
		called = true
		return nil
	}))
	rec := httptest.NewRecorder()
	app.middleware(mux).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/topics/not-a-uuid/posts", nil))
	if called || rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"invalid_id"`) {
		t.Fatalf("invalid ID reached handler: called=%t, response=%d %s", called, rec.Code, rec.Body.String())
	}
}
