package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBlueskyParseCreds_Valid(t *testing.T) {
	b := NewBluesky("")
	raw := json.RawMessage(`{"identifier":"user.bsky.social","app_password":"xxxx-xxxx"}`)
	creds, err := b.parseCreds(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.Identifier != "user.bsky.social" {
		t.Fatalf("unexpected identifier: %s", creds.Identifier)
	}
}

func TestBlueskyParseCreds_MissingFields(t *testing.T) {
	b := NewBluesky("")
	tests := []struct {
		name string
		json string
	}{
		{"missing identifier", `{"app_password":"xxxx"}`},
		{"missing app_password", `{"identifier":"user.bsky.social"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := b.parseCreds(json.RawMessage(tt.json))
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestBlueskyValidateCredentials_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"did":       "did:plc:test123",
			"handle":    "user.bsky.social",
			"accessJwt": "test-jwt",
		})
	}))
	defer server.Close()

	b := NewBluesky(server.URL)
	creds := json.RawMessage(`{"identifier":"user.bsky.social","app_password":"xxxx-xxxx"}`)
	if err := b.ValidateCredentials(context.Background(), creds); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBlueskyValidateCredentials_InvalidCreds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"AuthenticationRequired"}`))
	}))
	defer server.Close()

	b := NewBluesky(server.URL)
	creds := json.RawMessage(`{"identifier":"user.bsky.social","app_password":"wrong"}`)
	if err := b.ValidateCredentials(context.Background(), creds); err == nil {
		t.Fatal("expected error")
	}
}

func TestBlueskyPublish_Success(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			// createSession
			json.NewEncoder(w).Encode(map[string]any{
				"did": "did:plc:test123", "handle": "user.bsky.social", "accessJwt": "jwt",
			})
		} else {
			// createRecord
			json.NewEncoder(w).Encode(map[string]any{
				"uri": "at://did:plc:test123/app.bsky.feed.post/rkey456",
				"cid": "cid123",
			})
		}
	}))
	defer server.Close()

	b := NewBluesky(server.URL)
	creds := json.RawMessage(`{"identifier":"user.bsky.social","app_password":"xxxx"}`)
	result, err := b.Publish(context.Background(), "Hello world", creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PlatformPostID != "at://did:plc:test123/app.bsky.feed.post/rkey456" {
		t.Fatalf("wrong post ID: %s", result.PlatformPostID)
	}
	if result.URL != "https://bsky.app/profile/user.bsky.social/post/rkey456" {
		t.Fatalf("wrong URL: %s", result.URL)
	}
}

func TestBlueskyATURIToWebURL(t *testing.T) {
	b := NewBluesky("")
	url := b.atURIToWebURL("at://did:plc:abc/app.bsky.feed.post/xyz123", "user.bsky.social")
	if url != "https://bsky.app/profile/user.bsky.social/post/xyz123" {
		t.Fatalf("wrong URL: %s", url)
	}
}
