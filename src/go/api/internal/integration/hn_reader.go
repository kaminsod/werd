package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// HNReader implements PlatformReader for Hacker News.
type HNReader struct {
	baseURL string
	httpCli *http.Client
}

func NewHNReader() *HNReader {
	return &HNReader{
		baseURL: "https://hacker-news.firebaseio.com/v0",
		httpCli: &http.Client{Timeout: 10 * time.Second},
	}
}

type hnItem struct {
	ID     int    `json:"id"`
	By     string `json:"by"`
	Text   string `json:"text"`
	Time   int64  `json:"time"`
	Kids   []int  `json:"kids"`
	Type   string `json:"type"`
	Title  string `json:"title"`
	Parent int    `json:"parent"`
}

// resolveStoryTitle walks up the parent chain from a comment to find the root
// story's title. maxDepth caps the number of API calls to avoid runaway chains.
func (r *HNReader) resolveStoryTitle(ctx context.Context, item *hnItem, maxDepth int) string {
	current := item
	for i := 0; i < maxDepth; i++ {
		if current.Title != "" {
			return current.Title
		}
		if current.Parent == 0 {
			return ""
		}
		parent, err := r.fetchItem(ctx, current.Parent)
		if err != nil || parent == nil {
			return ""
		}
		current = parent
	}
	return ""
}

func (r *HNReader) GetReplies(ctx context.Context, platformPostID, sinceID string, _ json.RawMessage) ([]PlatformReply, error) {
	// Parse the item ID from "hn_12345" or just "12345".
	idStr := strings.TrimPrefix(platformPostID, "hn_")
	itemID, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, fmt.Errorf("hn reader: invalid post ID: %s", platformPostID)
	}

	// Fetch the parent item to get its kids.
	parent, err := r.fetchItem(ctx, itemID)
	if err != nil {
		return nil, fmt.Errorf("hn reader: fetching parent: %w", err)
	}
	if parent == nil || len(parent.Kids) == 0 {
		return nil, nil
	}

	sinceNum := 0
	if sinceID != "" {
		sinceStr := strings.TrimPrefix(sinceID, "hn_")
		sinceNum, _ = strconv.Atoi(sinceStr)
	}

	// Fetch child items (direct replies only, not recursive).
	var replies []PlatformReply
	for _, kidID := range parent.Kids {
		if sinceNum > 0 && kidID <= sinceNum {
			continue // already seen
		}

		kid, err := r.fetchItem(ctx, kidID)
		if err != nil || kid == nil {
			continue
		}
		if kid.Type != "comment" {
			continue
		}

		replies = append(replies, PlatformReply{
			ID:        fmt.Sprintf("hn_%d", kid.ID),
			Author:    kid.By,
			Content:   kid.Text,
			URL:       fmt.Sprintf("https://news.ycombinator.com/item?id=%d", kid.ID),
			CreatedAt: time.Unix(kid.Time, 0),
			ParentID:  fmt.Sprintf("hn_%d", itemID),
		})
	}

	return replies, nil
}

func (r *HNReader) fetchItem(ctx context.Context, id int) (*hnItem, error) {
	url := fmt.Sprintf("%s/item/%d.json", r.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.httpCli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var item hnItem
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, nil // deleted
	}

	return &item, nil
}

// hnUser represents the HN user API response.
type hnUser struct {
	ID        string `json:"id"`
	Submitted []int  `json:"submitted"`
}

// fetchUser fetches an HN user profile.
func (r *HNReader) fetchUser(ctx context.Context, username string) (*hnUser, error) {
	url := fmt.Sprintf("%s/user/%s.json", r.baseURL, username)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := r.httpCli.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user %s: status %d", username, resp.StatusCode)
	}

	var user hnUser
	if err := json.Unmarshal(body, &user); err != nil {
		return nil, err
	}
	return &user, nil
}
