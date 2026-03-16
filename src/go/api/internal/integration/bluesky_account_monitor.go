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

// BlueskyAccountMonitor monitors a Bluesky account's notifications for replies and mentions.
type BlueskyAccountMonitor struct {
	host    string
	httpCli *http.Client
}

func NewBlueskyAccountMonitor() *BlueskyAccountMonitor {
	return &BlueskyAccountMonitor{
		host:    defaultBskyHost,
		httpCli: &http.Client{Timeout: 15 * time.Second},
	}
}

func (m *BlueskyAccountMonitor) Poll(ctx context.Context, config, watermark, credentials json.RawMessage) ([]MonitoredItem, json.RawMessage, error) {
	var creds BlueskyCredentials
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return nil, watermark, fmt.Errorf("bluesky account: invalid credentials: %w", err)
	}

	// Authenticate.
	bsky := &Bluesky{host: m.host, httpCli: m.httpCli}
	session, err := bsky.createSession(ctx, &creds)
	if err != nil {
		return nil, watermark, fmt.Errorf("bluesky account: auth failed: %w", err)
	}

	// Parse watermark.
	var wm struct {
		LastSeen string `json:"last_seen"`
	}
	json.Unmarshal(watermark, &wm)

	// Fetch notifications.
	params := url.Values{"limit": {"50"}}
	reqURL := fmt.Sprintf("%s/xrpc/app.bsky.notification.listNotifications?%s", m.host, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, watermark, err
	}
	req.Header.Set("Authorization", "Bearer "+session.AccessJwt)

	resp, err := m.httpCli.Do(req)
	if err != nil {
		return nil, watermark, fmt.Errorf("bluesky account: request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, watermark, fmt.Errorf("bluesky account: status %d: %s", resp.StatusCode, string(body))
	}

	var notifResp struct {
		Notifications []struct {
			URI       string `json:"uri"`
			CID       string `json:"cid"`
			Reason    string `json:"reason"`
			IndexedAt string `json:"indexedAt"`
			Author    struct {
				Handle      string `json:"handle"`
				DisplayName string `json:"displayName"`
			} `json:"author"`
			Record struct {
				Text string `json:"text"`
			} `json:"record"`
		} `json:"notifications"`
	}
	if err := json.Unmarshal(body, &notifResp); err != nil {
		return nil, watermark, fmt.Errorf("bluesky account: parsing: %w", err)
	}

	var items []MonitoredItem
	newLastSeen := wm.LastSeen

	for _, notif := range notifResp.Notifications {
		// Only process replies and mentions.
		if notif.Reason != "reply" && notif.Reason != "mention" && notif.Reason != "quote" {
			continue
		}

		// Stop at previously seen notification.
		if wm.LastSeen != "" && notif.IndexedAt <= wm.LastSeen {
			break
		}

		if newLastSeen == "" || notif.IndexedAt > newLastSeen {
			newLastSeen = notif.IndexedAt
		}

		title := fmt.Sprintf("%s from @%s", notif.Reason, notif.Author.Handle)
		if notif.Author.DisplayName != "" {
			title = fmt.Sprintf("%s from %s (@%s)", notif.Reason, notif.Author.DisplayName, notif.Author.Handle)
		}

		webURL := bsky.atURIToWebURL(notif.URI, notif.Author.Handle)

		items = append(items, MonitoredItem{
			SourceID:  fmt.Sprintf("bsky_notif_%s", notif.CID),
			Title:     title,
			Content:   notif.Record.Text,
			URL:       webURL,
			Author:    notif.Author.Handle,
			CreatedAt: parseTime(notif.IndexedAt),
		})
	}

	newWM, _ := json.Marshal(map[string]string{"last_seen": newLastSeen})
	return items, newWM, nil
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}
