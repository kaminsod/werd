package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// RedditAccountMonitor monitors a Reddit account's inbox for messages, mentions, and replies.
type RedditAccountMonitor struct {
	authHost string
	apiHost  string
	httpCli  *http.Client
}

func NewRedditAccountMonitor() *RedditAccountMonitor {
	return &RedditAccountMonitor{
		authHost: defaultRedditAuthHost,
		apiHost:  defaultRedditAPIHost,
		httpCli:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (m *RedditAccountMonitor) Poll(ctx context.Context, config, watermark, credentials json.RawMessage) ([]MonitoredItem, json.RawMessage, error) {
	var cfg struct {
		CheckInbox    bool `json:"check_inbox"`
		CheckMentions bool `json:"check_mentions"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, watermark, fmt.Errorf("reddit account: invalid config: %w", err)
	}
	// Default both to true if not explicitly set.
	if !cfg.CheckInbox && !cfg.CheckMentions {
		cfg.CheckInbox = true
		cfg.CheckMentions = true
	}

	var creds RedditCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, watermark, fmt.Errorf("reddit account: invalid credentials: %w", err)
	}

	reddit := &Reddit{authHost: m.authHost, apiHost: m.apiHost, httpCli: m.httpCli}
	token, err := reddit.getAccessToken(ctx, &creds)
	if err != nil {
		return nil, watermark, fmt.Errorf("reddit account: auth failed: %w", err)
	}

	// Fetch inbox.
	url := fmt.Sprintf("%s/message/inbox.json?limit=25&raw_json=1", m.apiHost)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, watermark, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", creds.UserAgent)

	resp, err := m.httpCli.Do(req)
	if err != nil {
		return nil, watermark, fmt.Errorf("reddit account: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, watermark, fmt.Errorf("reddit account: status %d", resp.StatusCode)
	}

	var listing struct {
		Data struct {
			Children []struct {
				Kind string `json:"kind"`
				Data struct {
					Name       string  `json:"name"`
					Author     string  `json:"author"`
					Subject    string  `json:"subject"`
					Body       string  `json:"body"`
					Context    string  `json:"context"`
					CreatedUTC float64 `json:"created_utc"`
					WasComment bool    `json:"was_comment"`
				} `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, watermark, fmt.Errorf("reddit account: parsing: %w", err)
	}

	var wm struct {
		LastSeenID string `json:"last_seen_id"`
	}
	json.Unmarshal(watermark, &wm)

	var items []MonitoredItem
	newLastSeen := wm.LastSeenID

	for _, child := range listing.Data.Children {
		msg := child.Data
		if msg.Name == "" {
			continue
		}
		if wm.LastSeenID != "" && msg.Name == wm.LastSeenID {
			break
		}

		// Filter by type.
		isMention := msg.Subject == "username mention"
		isInbox := child.Kind == "t4" || msg.WasComment

		if isMention && !cfg.CheckMentions {
			continue
		}
		if !isMention && isInbox && !cfg.CheckInbox {
			continue
		}

		title := msg.Subject
		if msg.WasComment {
			title = fmt.Sprintf("Reply from u/%s", msg.Author)
		} else if isMention {
			title = fmt.Sprintf("Mention by u/%s", msg.Author)
		}

		itemURL := ""
		if msg.Context != "" {
			itemURL = fmt.Sprintf("https://reddit.com%s", msg.Context)
		}

		items = append(items, MonitoredItem{
			SourceID:  fmt.Sprintf("reddit_inbox_%s", msg.Name),
			Title:     title,
			Content:   msg.Body,
			URL:       itemURL,
			Author:    msg.Author,
			CreatedAt: time.Unix(int64(msg.CreatedUTC), 0),
		})

		if newLastSeen == "" {
			newLastSeen = msg.Name
		}
	}

	// Watermark is the newest item (first in list).
	if len(listing.Data.Children) > 0 {
		first := listing.Data.Children[0].Data.Name
		if first != "" {
			newLastSeen = first
		}
	}

	newWM, _ := json.Marshal(map[string]string{"last_seen_id": newLastSeen})
	return items, newWM, nil
}
