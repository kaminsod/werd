package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRedditParseCreds_Valid(t *testing.T) {
	r := NewReddit()
	raw := json.RawMessage(`{"client_id":"id","client_secret":"secret","username":"user","password":"pass","user_agent":"ua","subreddit":"test"}`)
	creds, err := r.parseCreds(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if creds.ClientID != "id" || creds.Subreddit != "test" {
		t.Fatalf("unexpected creds: %+v", creds)
	}
}

func TestRedditParseCreds_MissingFields(t *testing.T) {
	r := NewReddit()
	tests := []struct {
		name string
		json string
	}{
		{"missing client_id", `{"client_secret":"s","username":"u","password":"p","user_agent":"ua","subreddit":"sr"}`},
		{"missing client_secret", `{"client_id":"i","username":"u","password":"p","user_agent":"ua","subreddit":"sr"}`},
		{"missing username", `{"client_id":"i","client_secret":"s","password":"p","user_agent":"ua","subreddit":"sr"}`},
		{"missing password", `{"client_id":"i","client_secret":"s","username":"u","user_agent":"ua","subreddit":"sr"}`},
		{"missing user_agent", `{"client_id":"i","client_secret":"s","username":"u","password":"p","subreddit":"sr"}`},
		{"missing subreddit", `{"client_id":"i","client_secret":"s","username":"u","password":"p","user_agent":"ua"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := r.parseCreds(json.RawMessage(tt.json))
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestRedditValidateCredentials_Success(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	}))
	defer authServer.Close()

	r := NewRedditWithHosts(authServer.URL, "")
	creds := json.RawMessage(`{"client_id":"id","client_secret":"secret","username":"user","password":"pass","user_agent":"test","subreddit":"test"}`)
	if err := r.ValidateCredentials(context.Background(), creds); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRedditValidateCredentials_InvalidCreds(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid_grant"}`))
	}))
	defer authServer.Close()

	r := NewRedditWithHosts(authServer.URL, "")
	creds := json.RawMessage(`{"client_id":"id","client_secret":"bad","username":"user","password":"wrong","user_agent":"test","subreddit":"test"}`)
	if err := r.ValidateCredentials(context.Background(), creds); err == nil {
		t.Fatal("expected error for invalid credentials")
	}
}

func TestRedditPublish_Success(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-token",
			"token_type":   "bearer",
			"expires_in":   3600,
		})
	}))
	defer authServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/submit" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("missing auth header")
		}
		r.ParseForm()
		if r.FormValue("sr") != "testsubreddit" {
			t.Fatalf("wrong subreddit: %s", r.FormValue("sr"))
		}
		if r.FormValue("title") != "My Title" {
			t.Fatalf("wrong title: %s", r.FormValue("title"))
		}
		if r.FormValue("text") != "Body content here" {
			t.Fatalf("wrong body: %s", r.FormValue("text"))
		}
		json.NewEncoder(w).Encode(map[string]any{
			"json": map[string]any{
				"errors": []any{},
				"data": map[string]any{
					"name": "t3_abc123",
					"url":  "https://reddit.com/r/testsubreddit/comments/abc123/my_title/",
				},
			},
		})
	}))
	defer apiServer.Close()

	adapter := NewRedditWithHosts(authServer.URL, apiServer.URL)
	creds := json.RawMessage(`{"client_id":"id","client_secret":"secret","username":"user","password":"pass","user_agent":"test","subreddit":"testsubreddit"}`)
	result, err := adapter.Publish(context.Background(), PublishContent{Title: "My Title", Body: "Body content here", PostType: "text"}, creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.PlatformPostID != "t3_abc123" {
		t.Fatalf("wrong post ID: %s", result.PlatformPostID)
	}
	if result.URL != "https://reddit.com/r/testsubreddit/comments/abc123/my_title/" {
		t.Fatalf("wrong URL: %s", result.URL)
	}
}

func TestRedditPublish_SingleLineContent(t *testing.T) {
	title, body := splitTitleBody("Just a title, no body")
	if title != "Just a title, no body" || body != "" {
		t.Fatalf("expected title='Just a title, no body' body='', got title='%s' body='%s'", title, body)
	}
}

func TestRedditPublish_EmptyContent(t *testing.T) {
	title, body := splitTitleBody("")
	if title != "Post from Werd" || body != "" {
		t.Fatalf("expected default title, got title='%s' body='%s'", title, body)
	}
}

func TestRedditPublish_SubmitError(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"access_token": "tok", "token_type": "bearer", "expires_in": 3600})
	}))
	defer authServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"json": map[string]any{
				"errors": []any{[]any{"SUBREDDIT_NOEXIST", "that subreddit doesn't exist", "sr"}},
				"data":   map[string]any{},
			},
		})
	}))
	defer apiServer.Close()

	adapter := NewRedditWithHosts(authServer.URL, apiServer.URL)
	creds := json.RawMessage(`{"client_id":"id","client_secret":"s","username":"u","password":"p","user_agent":"ua","subreddit":"nosub"}`)
	_, err := adapter.Publish(context.Background(), PublishContent{Title: "Title", Body: "Body", PostType: "text"}, creds)
	if err == nil {
		t.Fatal("expected error for Reddit API error response")
	}
}
