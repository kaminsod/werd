package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// BlueskyUserMonitor monitors a Bluesky user's feed for new posts.
type BlueskyUserMonitor struct {
	host    string
	httpCli *http.Client
}

func NewBlueskyUserMonitor() *BlueskyUserMonitor {
	return &BlueskyUserMonitor{
		host:    defaultBskyHost,
		httpCli: &http.Client{Timeout: 15 * time.Second},
	}
}

func (m *BlueskyUserMonitor) Poll(ctx context.Context, config, watermark, credentials json.RawMessage) ([]MonitoredItem, json.RawMessage, error) {
	var cfg struct {
		Handle string `json:"handle"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, watermark, fmt.Errorf("bluesky user: invalid config: %w", err)
	}
	if cfg.Handle == "" {
		return nil, watermark, fmt.Errorf("bluesky user: handle required")
	}

	var creds BlueskyCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, watermark, fmt.Errorf("bluesky user: invalid credentials: %w", err)
	}

	// Authenticate.
	bsky := &Bluesky{host: m.host, httpCli: m.httpCli}
	session, err := bsky.createSession(ctx, &creds)
	if err != nil {
		return nil, watermark, fmt.Errorf("bluesky user: auth failed: %w", err)
	}

	// Parse watermark.
	var wm struct {
		LastSeenURI string `json:"last_seen_uri"`
	}
	json.Unmarshal(watermark, &wm)

	// Fetch author feed.
	params := url.Values{
		"actor":  {cfg.Handle},
		"limit":  {"50"},
		"filter": {"posts_no_replies"},
	}
	reqURL := fmt.Sprintf("%s/xrpc/app.bsky.feed.getAuthorFeed?%s", m.host, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, watermark, err
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessJwt)

	resp, err := m.httpCli.Do(req)
	if err != nil {
		return nil, watermark, fmt.Errorf("bluesky user: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, watermark, fmt.Errorf("bluesky user: status %d: %s", resp.StatusCode, string(body))
	}

	var feedResp struct {
		Feed []struct {
			Post struct {
				URI    string `json:"uri"`
				CID    string `json:"cid"`
				Author struct {
					Handle      string `json:"handle"`
					DisplayName string `json:"displayName"`
				} `json:"author"`
				Record struct {
					Text      string `json:"text"`
					CreatedAt string `json:"createdAt"`
				} `json:"record"`
				IndexedAt string `json:"indexedAt"`
			} `json:"post"`
		} `json:"feed"`
	}
	if err := json.Unmarshal(body, &feedResp); err != nil {
		return nil, watermark, fmt.Errorf("bluesky user: parsing: %w", err)
	}

	var items []MonitoredItem
	newLastSeenURI := wm.LastSeenURI

	for _, entry := range feedResp.Feed {
		post := entry.Post

		// Only include posts from the target user (skip reposts of others).
		if post.Author.Handle != cfg.Handle {
			continue
		}

		// Stop at previously seen post.
		if wm.LastSeenURI != "" && post.URI == wm.LastSeenURI {
			break
		}

		if newLastSeenURI == "" {
			newLastSeenURI = post.URI
		}

		title := fmt.Sprintf("Post by @%s", post.Author.Handle)
		if post.Author.DisplayName != "" {
			title = fmt.Sprintf("Post by %s (@%s)", post.Author.DisplayName, post.Author.Handle)
		}

		webURL := bsky.atURIToWebURL(post.URI, post.Author.Handle)

		items = append(items, MonitoredItem{
			SourceID:  fmt.Sprintf("bsky_post_%s", post.CID),
			Title:     title,
			Content:   post.Record.Text,
			URL:       webURL,
			Author:    post.Author.Handle,
			CreatedAt: parseTime(post.Record.CreatedAt),
		})
	}

	newWM, _ := json.Marshal(map[string]string{"last_seen_uri": newLastSeenURI})
	return items, newWM, nil
}
