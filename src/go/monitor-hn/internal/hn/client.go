package hn

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://hacker-news.firebaseio.com/v0"

type Item struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
	Text  string `json:"text"`
	By    string `json:"by"`
	Time  int64  `json:"time"`
	Type  string `json:"type"`
	Score int    `json:"score"`
}

type Client struct {
	baseURL string
	httpCli *http.Client
}

func NewClient() *Client {
	return &Client{
		baseURL: defaultBaseURL,
		httpCli: &http.Client{Timeout: 10 * time.Second},
	}
}

// NewClientWithURL creates a client with a custom base URL (for testing).
func NewClientWithURL(baseURL string) *Client {
	return &Client{baseURL: baseURL, httpCli: &http.Client{Timeout: 10 * time.Second}}
}

// GetNewStoryIDs returns the IDs of the newest stories (up to 500).
func (c *Client) GetNewStoryIDs(ctx context.Context) ([]int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/newstories.json", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpCli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching new stories: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("new stories (%d): %s", resp.StatusCode, string(body))
	}

	var ids []int
	if err := json.Unmarshal(body, &ids); err != nil {
		return nil, fmt.Errorf("parsing story IDs: %w", err)
	}
	return ids, nil
}

// GetItem fetches a single HN item by ID.
func (c *Client) GetItem(ctx context.Context, id int) (*Item, error) {
	url := fmt.Sprintf("%s/item/%d.json", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpCli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching item %d: %w", id, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("item %d (%d): %s", id, resp.StatusCode, string(body))
	}

	var item Item
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, fmt.Errorf("parsing item %d: %w", id, err)
	}

	// Deleted/dead items have no title.
	if item.Title == "" && item.ID == 0 {
		return nil, nil
	}

	return &item, nil
}
