package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLLMClient_Classify(t *testing.T) {
	// Mock OpenAI-compatible server.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json")
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization header with test-key")
		}

		// Verify request body.
		var reqBody map[string]any
		json.NewDecoder(r.Body).Decode(&reqBody)
		if reqBody["model"] != "test-model" {
			t.Errorf("expected model 'test-model', got %v", reqBody["model"])
		}

		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"content": `{"relevant": true, "severity": "high", "tags": ["security", "urgent"], "reason": "Contains security vulnerability discussion"}`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", "test-model")
	if client == nil {
		t.Fatal("expected non-nil client")
	}

	result, err := client.Classify(context.Background(), "Analyze this post for security issues", 200)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Relevant {
		t.Error("expected relevant=true")
	}
	if result.Severity != "high" {
		t.Errorf("expected severity 'high', got %q", result.Severity)
	}
	if len(result.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(result.Tags))
	}
	if result.Reason != "Contains security vulnerability discussion" {
		t.Errorf("unexpected reason: %q", result.Reason)
	}
}

func TestLLMClient_Classify_NotRelevant(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"content": `{"relevant": false, "severity": "low", "tags": [], "reason": "Not related to the project"}`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "", "model")
	result, err := client.Classify(context.Background(), "test prompt", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Relevant {
		t.Error("expected relevant=false")
	}
}

func TestLLMClient_Classify_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "", "model")
	_, err := client.Classify(context.Background(), "test", 100)
	if err == nil {
		t.Error("expected error on server error response")
	}
}

func TestLLMClient_Classify_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]string{
						"content": "This is not JSON",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "", "model")
	_, err := client.Classify(context.Background(), "test", 100)
	if err == nil {
		t.Error("expected error on invalid JSON response")
	}
}

func TestNewLLMClient_NilOnEmptyURL(t *testing.T) {
	client := NewLLMClient("", "key", "model")
	if client != nil {
		t.Error("expected nil client when URL is empty")
	}
}

func TestNewLLMClient_DefaultModel(t *testing.T) {
	client := NewLLMClient("http://localhost", "key", "")
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.model != "gpt-4o-mini" {
		t.Errorf("expected default model 'gpt-4o-mini', got %q", client.model)
	}
}
