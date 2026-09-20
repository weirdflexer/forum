package forum

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublicationHashCompatibility(t *testing.T) {
	// This is the pre-refactor encoding stored with accepted idempotency keys.
	body := `{"section_id":"section","title":"Название","body":"Текст"}`
	in := createTopicInput{SectionID: "section", Title: "Название", Body: "Текст"}
	if got := requestHash("/api/v1/topics", in); got != digest("/api/v1/topics"+body) {
		t.Fatal("refactor invalidated persisted topic idempotency hashes")
	}
	post := createPostInput{Body: "Ответ"}
	if got := requestHash("/api/v1/topics/one/posts", post); got != digest(`/api/v1/topics/one/posts{"body":"Ответ"}`) {
		t.Fatal("refactor invalidated persisted reply idempotency hashes")
	}
	if requestHash("/api/v1/topics/one/posts", post) == requestHash("/api/v1/topics/two/posts", post) {
		t.Fatal("idempotency hash must distinguish topics")
	}
}

func TestRespondPublicationPreservesSavedReply(t *testing.T) {
	saved := json.RawMessage(`{"post_id":"original","number":21,"page":2,"extra":"preserved"}`)
	out := publicationResult[createdPost]{
		Value:  createdPost{PostID: "must-not-be-returned"},
		Replay: &savedPublication{Status: http.StatusCreated, Body: saved},
	}
	w := httptest.NewRecorder()
	if err := respondPublication(w, out, nil); err != nil {
		t.Fatal(err)
	}
	if w.Code != http.StatusCreated || w.Header().Get("Idempotency-Replayed") != "true" {
		t.Fatal("missing replay response status/header")
	}
	if w.Body.String() != string(saved)+"\n" {
		t.Fatalf("changed saved response: %s", w.Body.String())
	}
}

func TestPublicationRateLimitKeepsHTTPMetadata(t *testing.T) {
	limited := &sessionRateLimitError{RetryAfter: 7, cause: problem(429, "rate_limit", "wait")}
	wrapped := fmt.Errorf("publish: %w", limited)
	w := httptest.NewRecorder()
	err := respondPublication(w, publicationResult[createdTopic]{}, wrapped)
	var api *apiError
	if !errors.As(err, &api) || api.Status != 429 || api.Code != "rate_limit" {
		t.Fatal("rate-limit API error was lost")
	}
	if w.Header().Get("Retry-After") != "7" || w.Body.Len() != 0 {
		t.Fatal("rate limit must set Retry-After without writing a success response")
	}
}
