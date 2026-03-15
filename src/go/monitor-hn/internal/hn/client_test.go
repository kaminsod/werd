package hn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetNewStoryIDs_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]int{100, 99, 98, 97})
	}))
	defer server.Close()

	c := NewClientWithURL(server.URL)
	ids, err := c.GetNewStoryIDs(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 4 || ids[0] != 100 {
		t.Fatalf("unexpected IDs: %v", ids)
	}
}

func TestGetItem_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"id": 100, "title": "Show HN: My Project", "url": "https://example.com",
			"by": "user", "time": 1234567890, "type": "story", "score": 42,
		})
	}))
	defer server.Close()

	c := NewClientWithURL(server.URL)
	item, err := c.GetItem(context.Background(), 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.Title != "Show HN: My Project" {
		t.Fatalf("wrong title: %s", item.Title)
	}
	if item.Score != 42 {
		t.Fatalf("wrong score: %d", item.Score)
	}
}

func TestGetItem_Deleted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("null"))
	}))
	defer server.Close()

	c := NewClientWithURL(server.URL)
	item, err := c.GetItem(context.Background(), 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item != nil {
		t.Fatal("expected nil for deleted item")
	}
}
