package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBrowserAdapter_ValidateCredentials_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/actions/validate" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer server.Close()

	adapter := NewBrowserAdapter(server.URL, "x", "secret")
	creds := json.RawMessage(`{"username":"user","password":"pass"}`)
	if err := adapter.ValidateCredentials(context.Background(), creds); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBrowserAdapter_ValidateCredentials_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "login failed"})
	}))
	defer server.Close()

	adapter := NewBrowserAdapter(server.URL, "x", "secret")
	creds := json.RawMessage(`{"username":"user","password":"wrong"}`)
	err := adapter.ValidateCredentials(context.Background(), creds)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "login failed") {
		t.Fatalf("expected error to contain 'login failed', got: %v", err)
	}
}

func TestBrowserAdapter_Publish_TextPost(t *testing.T) {
	var receivedReq browserPublishRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/actions/publish" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"post_id": "abc123",
			"url":     "https://example.com/post/abc123",
		})
	}))
	defer server.Close()

	adapter := NewBrowserAdapter(server.URL, "x", "secret")
	creds := json.RawMessage(`{"username":"user","password":"pass"}`)
	result, err := adapter.Publish(context.Background(), PublishContent{
		Title:    "My Title",
		Body:     "Body text",
		PostType: "text",
	}, creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedContent := "My Title\nBody text"
	if receivedReq.Content != expectedContent {
		t.Fatalf("expected content=%q, got %q", expectedContent, receivedReq.Content)
	}
	if result.PlatformPostID != "abc123" {
		t.Fatalf("expected PlatformPostID='abc123', got %q", result.PlatformPostID)
	}
	if result.URL != "https://example.com/post/abc123" {
		t.Fatalf("expected URL='https://example.com/post/abc123', got %q", result.URL)
	}
}

func TestBrowserAdapter_Publish_LinkPost(t *testing.T) {
	var receivedReq browserPublishRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"post_id": "link456",
			"url":     "https://example.com/post/link456",
		})
	}))
	defer server.Close()

	adapter := NewBrowserAdapter(server.URL, "x", "secret")
	creds := json.RawMessage(`{"username":"user","password":"pass"}`)
	_, err := adapter.Publish(context.Background(), PublishContent{
		Title:    "My Title",
		URL:      "https://example.com",
		PostType: "link",
	}, creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedContent := "My Title\nhttps://example.com"
	if receivedReq.Content != expectedContent {
		t.Fatalf("expected content=%q, got %q", expectedContent, receivedReq.Content)
	}
}

func TestBrowserAdapter_Publish_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"success": false, "error": "compose failed"})
	}))
	defer server.Close()

	adapter := NewBrowserAdapter(server.URL, "x", "secret")
	creds := json.RawMessage(`{"username":"user","password":"pass"}`)
	_, err := adapter.Publish(context.Background(), PublishContent{
		Title:    "Title",
		Body:     "Body",
		PostType: "text",
	}, creds)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "compose failed") {
		t.Fatalf("expected error to contain 'compose failed', got: %v", err)
	}
}

func TestBrowserAdapter_CreateAccount_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/actions/create-account" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"success":  true,
			"username": "testuser",
			"credentials": map[string]any{
				"username": "testuser",
				"password": "pass123",
			},
		})
	}))
	defer server.Close()

	adapter := NewBrowserAdapter(server.URL, "x", "secret")
	resp, err := adapter.CreateAccount(context.Background(), "test@example.com", "testuser", "pass123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Fatal("expected Success=true")
	}
	if resp.Username != "testuser" {
		t.Fatalf("expected Username='testuser', got %q", resp.Username)
	}
	if resp.Credentials["username"] != "testuser" {
		t.Fatalf("expected credentials username='testuser', got %v", resp.Credentials["username"])
	}
	if resp.Credentials["password"] != "pass123" {
		t.Fatalf("expected credentials password='pass123', got %v", resp.Credentials["password"])
	}
}

func TestBrowserAdapter_CreateAccount_Failure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   "username taken",
		})
	}))
	defer server.Close()

	adapter := NewBrowserAdapter(server.URL, "x", "secret")
	resp, err := adapter.CreateAccount(context.Background(), "test@example.com", "taken", "pass123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Fatal("expected Success=false")
	}
	if resp.Error != "username taken" {
		t.Fatalf("expected Error='username taken', got %q", resp.Error)
	}
}

func TestBrowserAdapter_AuthHeader(t *testing.T) {
	var capturedKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedKey = r.Header.Get("X-Internal-Key")
		json.NewEncoder(w).Encode(map[string]any{"success": true})
	}))
	defer server.Close()

	adapter := NewBrowserAdapter(server.URL, "x", "test-secret-key")
	creds := json.RawMessage(`{"username":"user","password":"pass"}`)
	if err := adapter.ValidateCredentials(context.Background(), creds); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedKey != "test-secret-key" {
		t.Fatalf("expected X-Internal-Key='test-secret-key', got %q", capturedKey)
	}
}

func TestBrowserAdapter_ServiceDown(t *testing.T) {
	adapter := NewBrowserAdapter("http://127.0.0.1:1", "x", "secret")
	creds := json.RawMessage(`{"username":"user","password":"pass"}`)
	err := adapter.ValidateCredentials(context.Background(), creds)
	if err == nil {
		t.Fatal("expected error for unreachable service")
	}
	if !strings.Contains(err.Error(), "unreachable") {
		t.Fatalf("expected error to contain 'unreachable', got: %v", err)
	}
}
