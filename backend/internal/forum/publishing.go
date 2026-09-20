package forum

import (
	"errors"
	"net/http"
	"strconv"
)

func (s *Server) createTopic(w http.ResponseWriter, r *http.Request) error {
	id, err := s.require(r, "anonymous")
	if err != nil {
		return err
	}
	key, err := idemKey(r)
	if err != nil {
		return err
	}
	var in createTopicInput
	if err = decode(w, r, &in); err != nil {
		return err
	}
	out, err := s.publishTopic(r.Context(), publicationRequest{SessionID: id.ID, Key: key, Path: r.URL.Path}, in)
	return respondPublication(w, out, err)
}

func (s *Server) createPost(w http.ResponseWriter, r *http.Request) error {
	id, err := s.require(r, "anonymous")
	if err != nil {
		return err
	}
	key, err := idemKey(r)
	if err != nil {
		return err
	}
	var in createPostInput
	if err = decode(w, r, &in); err != nil {
		return err
	}
	out, err := s.publishPost(r.Context(), publicationRequest{SessionID: id.ID, Key: key, Path: r.URL.Path}, r.PathValue("id"), in)
	return respondPublication(w, out, err)
}

func (s *Server) deletePost(w http.ResponseWriter, r *http.Request) error {
	id, err := s.require(r, "anonymous")
	if err != nil {
		return err
	}
	if err = s.removeOwnPost(r.Context(), id.ID, r.PathValue("id")); err != nil {
		return err
	}
	return respond(w, http.StatusNoContent, nil)
}

func (s *Server) createReport(w http.ResponseWriter, r *http.Request) error {
	id, err := s.require(r, "anonymous")
	if err != nil {
		return err
	}
	var in createReportInput
	if err = decode(w, r, &in); err != nil {
		return err
	}
	out, err := s.submitReport(r.Context(), id.ID, in)
	if err != nil {
		return publishingError(w, err)
	}
	return respond(w, http.StatusCreated, out)
}

func idemKey(r *http.Request) (string, error) {
	key := r.Header.Get("Idempotency-Key")
	if len(key) < 8 || len(key) > 128 {
		return "", problem(400, "idempotency_key", "Нужен Idempotency-Key длиной 8–128 символов.")
	}
	return key, nil
}

func respondPublication[T any](w http.ResponseWriter, out publicationResult[T], err error) error {
	if err != nil {
		return publishingError(w, err)
	}
	if out.Replay != nil {
		w.Header().Set("Idempotency-Replayed", "true")
		return respond(w, out.Replay.Status, out.Replay.Body)
	}
	return respond(w, http.StatusCreated, out.Value)
}

func publishingError(w http.ResponseWriter, err error) error {
	var limited *sessionRateLimitError
	if errors.As(err, &limited) {
		w.Header().Set("Retry-After", strconv.Itoa(limited.RetryAfter))
	}
	return err
}
