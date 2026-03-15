package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RedditReader implements PlatformReader for Reddit.
type RedditReader struct {
	authHost string
	apiHost  string
	httpCli  *http.Client
}

func NewRedditReader() *RedditReader {
	return &RedditReader{
		authHost: defaultRedditAuthHost,
		apiHost:  defaultRedditAPIHost,
		httpCli:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (r *RedditReader) GetReplies(ctx context.Context, platformPostID, sinceID string, credentials json.RawMessage) ([]PlatformReply, error) {
	var creds RedditCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, fmt.Errorf("reddit reader: invalid credentials: %w", err)
	}

	// Get an access token.
	reddit := &Reddit{authHost: r.authHost, apiHost: r.apiHost, httpCli: r.httpCli}
	token, err := reddit.getAccessToken(ctx, &creds)
	if err != nil {
		return nil, fmt.Errorf("reddit reader: auth failed: %w", err)
	}

	// Extract article ID from fullname (e.g., "t3_abc123" → "abc123").
	articleID := strings.TrimPrefix(platformPostID, "t3_")
	if articleID == "" {
		return nil, fmt.Errorf("reddit reader: invalid post ID: %s", platformPostID)
	}

	// Fetch comments.
	url := fmt.Sprintf("%s/comments/%s.json?sort=new&limit=100&raw_json=1", r.apiHost, articleID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", creds.UserAgent)

	resp, err := r.httpCli.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reddit reader: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reddit reader: status %d: %s", resp.StatusCode, string(body))
	}

	// Reddit returns an array of 2 listings: [post, comments].
	var listings []struct {
		Data struct {
			Children []struct {
				Data struct {
					Name      string  `json:"name"`
					Author    string  `json:"author"`
					Body      string  `json:"body"`
					Permalink string  `json:"permalink"`
					CreatedUTC float64 `json:"created_utc"`
					ParentID  string  `json:"parent_id"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &listings); err != nil {
		return nil, fmt.Errorf("reddit reader: parsing response: %w", err)
	}

	if len(listings) < 2 {
		return nil, nil
	}

	var replies []PlatformReply
	seenSince := false
	for _, child := range listings[1].Data.Children {
		c := child.Data
		if c.Name == "" || c.Author == "" {
			continue // "more" placeholder or deleted
		}
		if sinceID != "" && c.Name == sinceID {
			seenSince = true
			continue
		}
		if sinceID != "" && !seenSince {
			// Comments are sorted newest first — if we haven't reached sinceID yet, include this one.
		}
		replies = append(replies, PlatformReply{
			ID:        c.Name,
			Author:    c.Author,
			Content:   c.Body,
			URL:       fmt.Sprintf("https://reddit.com%s", c.Permalink),
			CreatedAt: time.Unix(int64(c.CreatedUTC), 0),
			ParentID:  c.ParentID,
		})
	}

	// Filter to only replies newer than sinceID.
	if sinceID != "" {
		var filtered []PlatformReply
		for _, r := range replies {
			if r.ID == sinceID {
				break
			}
			filtered = append(filtered, r)
		}
		replies = filtered
	}

	return replies, nil
}
