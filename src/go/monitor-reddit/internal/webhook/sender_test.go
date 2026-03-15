package webhook

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSend_Success(t *testing.T) {
	var received IngestPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Webhook-Secret") != "test-key" {
			t.Fatal("missing webhook secret")
		}
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	s := NewSender(server.URL, "test-key")
	err := s.Send(context.Background(), IngestPayload{
		ProjectID: "proj-1", SourceType: "reddit", SourceID: "t3_abc",
		Title: "Test", Content: "body", URL: "https://reddit.com/r/test/1", Severity: "low",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.SourceID != "t3_abc" {
		t.Fatalf("wrong source_id: %s", received.SourceID)
	}
}

func TestSend_AuthError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	s := NewSender(server.URL, "bad-key")
	err := s.Send(context.Background(), IngestPayload{ProjectID: "p", SourceType: "reddit", SourceID: "t3_x"})
	if err == nil {
		t.Fatal("expected error for 401")
	}
}

func TestSend_DuplicateOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // 200 = dedup update, not 201
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	s := NewSender(server.URL, "key")
	err := s.Send(context.Background(), IngestPayload{ProjectID: "p", SourceType: "reddit", SourceID: "t3_dup"})
	if err != nil {
		t.Fatalf("200 should not be an error: %v", err)
	}
}
