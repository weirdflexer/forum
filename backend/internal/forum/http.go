package forum

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

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

func respond(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if status != 204 {
		return json.NewEncoder(w).Encode(v)
	}
	return nil
}
func decode(w http.ResponseWriter, r *http.Request, dst any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
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
	// Errors may be reused by callers; attach request details only to the response copy.
	out := *p
	out.RequestID = state(r).RequestID
	_ = respond(w, out.Status, &out)
}
