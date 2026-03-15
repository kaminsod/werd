package reddit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListNewPosts_Success(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "tok", "token_type": "bearer", "expires_in": 3600,
		})
	}))
	defer authServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"children": []any{
					map[string]any{"data": map[string]any{
						"id": "abc", "name": "t3_abc", "title": "Test Post",
						"selftext": "body", "author": "user", "permalink": "/r/test/comments/abc/test/",
						"subreddit": "test", "created_utc": 1234567890.0,
					}},
				},
			},
		})
	}))
	defer apiServer.Close()

	c := NewClientWithHosts("id", "secret", "user", "pass", "ua", authServer.URL, apiServer.URL)
	posts, err := c.ListNewPosts(context.Background(), "test", 25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(posts) != 1 {
		t.Fatalf("expected 1 post, got %d", len(posts))
	}
	if posts[0].Fullname != "t3_abc" {
		t.Fatalf("wrong fullname: %s", posts[0].Fullname)
	}
	if posts[0].Title != "Test Post" {
		t.Fatalf("wrong title: %s", posts[0].Title)
	}
}

func TestGetToken_CachesToken(t *testing.T) {
	calls := 0
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "tok", "token_type": "bearer", "expires_in": 3600,
		})
	}))
	defer authServer.Close()

	c := NewClientWithHosts("id", "secret", "user", "pass", "ua", authServer.URL, "")

	ctx := context.Background()
	_, err := c.getToken(ctx)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	_, err = c.getToken(ctx)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}

	if calls != 1 {
		t.Fatalf("expected 1 token request (cached), got %d", calls)
	}
}
