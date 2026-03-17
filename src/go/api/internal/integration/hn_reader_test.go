package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveStoryTitle_CommentChain(t *testing.T) {
	// Build a chain: comment 3 -> comment 2 -> story 1 (has title).
	items := map[int]hnItem{
		1: {ID: 1, Type: "story", Title: "Show HN: My Project"},
		2: {ID: 2, Type: "comment", Parent: 1},
		3: {ID: 3, Type: "comment", Parent: 2},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Parse item ID from /item/<id>.json
		var id int
		if _, err := fmt.Sscanf(r.URL.Path, "/item/%d.json", &id); err != nil {
			http.NotFound(w, r)
			return
		}
		item, ok := items[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(item)
	}))
	defer srv.Close()

	reader := &HNReader{baseURL: srv.URL, httpCli: srv.Client()}

	start := items[3]
	got := reader.resolveStoryTitle(context.Background(), &start, 10)
	if got != "Show HN: My Project" {
		t.Errorf("resolveStoryTitle = %q, want %q", got, "Show HN: My Project")
	}
}

func TestResolveStoryTitle_AlreadyHasTitle(t *testing.T) {
	item := &hnItem{ID: 1, Type: "story", Title: "Direct Title"}
	reader := &HNReader{baseURL: "http://unused", httpCli: http.DefaultClient}

	got := reader.resolveStoryTitle(context.Background(), item, 10)
	if got != "Direct Title" {
		t.Errorf("resolveStoryTitle = %q, want %q", got, "Direct Title")
	}
}

func TestResolveStoryTitle_NoParent(t *testing.T) {
	// Comment with no parent field — should return empty.
	item := &hnItem{ID: 5, Type: "comment", Parent: 0}
	reader := &HNReader{baseURL: "http://unused", httpCli: http.DefaultClient}

	got := reader.resolveStoryTitle(context.Background(), item, 10)
	if got != "" {
		t.Errorf("resolveStoryTitle = %q, want empty", got)
	}
}

func TestResolveStoryTitle_MaxDepthCap(t *testing.T) {
	// Build a long chain where the story is beyond maxDepth.
	items := make(map[int]hnItem)
	for i := 1; i <= 15; i++ {
		items[i] = hnItem{ID: i, Type: "comment", Parent: i - 1}
	}
	// The root at ID 0 won't be served, so the chain never resolves.

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var id int
		if _, err := fmt.Sscanf(r.URL.Path, "/item/%d.json", &id); err != nil {
			http.NotFound(w, r)
			return
		}
		item, ok := items[id]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(item)
	}))
	defer srv.Close()

	reader := &HNReader{baseURL: srv.URL, httpCli: srv.Client()}

	start := items[15]
	got := reader.resolveStoryTitle(context.Background(), &start, 3)
	if got != "" {
		t.Errorf("resolveStoryTitle with maxDepth=3 = %q, want empty", got)
	}
}
