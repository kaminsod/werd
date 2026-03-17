package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RedditThreadMonitor monitors a Reddit thread for new comments.
type RedditThreadMonitor struct {
	reader *RedditReader
}

func NewRedditThreadMonitor() *RedditThreadMonitor {
	return &RedditThreadMonitor{reader: NewRedditReader()}
}

func (m *RedditThreadMonitor) Poll(ctx context.Context, config, watermark, credentials json.RawMessage) ([]MonitoredItem, json.RawMessage, error) {
	var cfg struct {
		ThreadID  string `json:"thread_id"`
		Subreddit string `json:"subreddit"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, watermark, fmt.Errorf("reddit thread: invalid config: %w", err)
	}
	if cfg.ThreadID == "" {
		return nil, watermark, fmt.Errorf("reddit thread: thread_id required")
	}

	var wm struct {
		LastSeenID string `json:"last_seen_id"`
	}
	json.Unmarshal(watermark, &wm)

	replies, err := m.reader.GetReplies(ctx, cfg.ThreadID, wm.LastSeenID, credentials)
	if err != nil {
		return nil, watermark, err
	}

	items := make([]MonitoredItem, len(replies))
	newLastSeen := wm.LastSeenID
	for i, r := range replies {
		items[i] = MonitoredItem{
			SourceID:  r.ID,
			Title:     fmt.Sprintf("Comment by u/%s", r.Author),
			Content:   r.Content,
			URL:       r.URL,
			Author:    r.Author,
			CreatedAt: r.CreatedAt,
		}
		if newLastSeen == "" || r.ID > newLastSeen {
			newLastSeen = r.ID
		}
	}

	newWM, _ := json.Marshal(map[string]string{"last_seen_id": newLastSeen})
	return items, newWM, nil
}

// RedditSubredditMonitor monitors a subreddit for new posts.
type RedditSubredditMonitor struct {
	authHost string
	apiHost  string
	httpCli  *http.Client
}

func NewRedditSubredditMonitor() *RedditSubredditMonitor {
	return &RedditSubredditMonitor{
		authHost: defaultRedditAuthHost,
		apiHost:  defaultRedditAPIHost,
		httpCli:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (m *RedditSubredditMonitor) Poll(ctx context.Context, config, watermark, credentials json.RawMessage) ([]MonitoredItem, json.RawMessage, error) {
	var cfg struct {
		Subreddit string `json:"subreddit"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, watermark, fmt.Errorf("reddit subreddit: invalid config: %w", err)
	}
	if cfg.Subreddit == "" {
		return nil, watermark, fmt.Errorf("reddit subreddit: subreddit required")
	}

	var creds RedditCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, watermark, fmt.Errorf("reddit subreddit: invalid credentials: %w", err)
	}

	reddit := &Reddit{authHost: m.authHost, apiHost: m.apiHost, httpCli: m.httpCli}
	token, err := reddit.getAccessToken(ctx, &creds)
	if err != nil {
		return nil, watermark, fmt.Errorf("reddit subreddit: auth failed: %w", err)
	}

	// Fetch new posts.
	url := fmt.Sprintf("%s/r/%s/new.json?limit=25&raw_json=1", m.apiHost, cfg.Subreddit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, watermark, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", creds.UserAgent)

	resp, err := m.httpCli.Do(req)
	if err != nil {
		return nil, watermark, fmt.Errorf("reddit subreddit: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, watermark, fmt.Errorf("reddit subreddit: status %d", resp.StatusCode)
	}

	var listing struct {
		Data struct {
			Children []struct {
				Data struct {
					Name      string  `json:"name"`
					Title     string  `json:"title"`
					Selftext  string  `json:"selftext"`
					Author    string  `json:"author"`
					Permalink string  `json:"permalink"`
					URL       string  `json:"url"`
					CreatedUTC float64 `json:"created_utc"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, watermark, fmt.Errorf("reddit subreddit: parsing: %w", err)
	}

	var wm struct {
		LastSeenID string `json:"last_seen_id"`
	}
	json.Unmarshal(watermark, &wm)

	var items []MonitoredItem
	newLastSeen := wm.LastSeenID

	for _, child := range listing.Data.Children {
		post := child.Data
		if post.Name == "" {
			continue
		}
		if wm.LastSeenID != "" && post.Name == wm.LastSeenID {
			break // reached previously seen post
		}

		content := post.Selftext
		if len(content) > 2000 {
			content = content[:2000] + "..."
		}

		items = append(items, MonitoredItem{
			SourceID:  post.Name,
			Title:     post.Title,
			Content:   content,
			URL:       fmt.Sprintf("https://reddit.com%s", post.Permalink),
			Author:    post.Author,
			CreatedAt: time.Unix(int64(post.CreatedUTC), 0),
		})

		if newLastSeen == "" || post.Name > newLastSeen {
			newLastSeen = post.Name
		}
	}

	// Update watermark to newest post seen.
	if len(listing.Data.Children) > 0 {
		newest := listing.Data.Children[0].Data.Name
		if newest > newLastSeen {
			newLastSeen = newest
		}
	}

	newWM, _ := json.Marshal(map[string]string{"last_seen_id": newLastSeen})
	return items, newWM, nil
}
