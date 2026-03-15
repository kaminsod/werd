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
		if r.Header.Get("X-Webhook-Secret") != "key" {
			t.Fatal("missing secret")
		}
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	s := NewSender(server.URL, "key")
	err := s.Send(context.Background(), IngestPayload{
		ProjectID: "p1", SourceType: "hn", SourceID: "hn_100",
		Title: "Show HN", Content: "", URL: "https://news.ycombinator.com/item?id=100", Severity: "low",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received.SourceType != "hn" {
		t.Fatalf("wrong source_type: %s", received.SourceType)
	}
}
