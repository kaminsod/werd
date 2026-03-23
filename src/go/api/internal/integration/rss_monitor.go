package integration

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mmcdole/gofeed"
)

// RSSMonitor fetches and parses RSS/Atom feeds for new items.
// Monitor key: "rss:default"
type RSSMonitor struct {
	rsshubURL string
	parser    *gofeed.Parser
}

func NewRSSMonitor(rsshubURL string) *RSSMonitor {
	fp := gofeed.NewParser()
	fp.Client = &http.Client{Timeout: 30 * time.Second}
	return &RSSMonitor{
		rsshubURL: strings.TrimRight(rsshubURL, "/"),
		parser:    fp,
	}
}

func (m *RSSMonitor) Poll(ctx context.Context, config, watermark, _ json.RawMessage) ([]MonitoredItem, json.RawMessage, error) {
	var cfg struct {
		Feeds []string `json:"feeds"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, watermark, fmt.Errorf("rss monitor: invalid config: %w", err)
	}
	if len(cfg.Feeds) == 0 {
		return nil, watermark, nil
	}

	var wm struct {
		LastSeen map[string]string `json:"last_seen"`
	}
	json.Unmarshal(watermark, &wm)
	if wm.LastSeen == nil {
		wm.LastSeen = make(map[string]string)
	}

	var allItems []MonitoredItem
	newLastSeen := make(map[string]string)
	for k, v := range wm.LastSeen {
		newLastSeen[k] = v
	}

	for _, feedURL := range cfg.Feeds {
		resolvedURL := feedURL
		if strings.HasPrefix(feedURL, "/") {
			resolvedURL = m.rsshubURL + feedURL
		}

		feed, err := m.parser.ParseURLWithContext(resolvedURL, ctx)
		if err != nil {
			log.Printf("rss monitor: error fetching %s: %v", resolvedURL, err)
			continue
		}

		lastGUID := wm.LastSeen[feedURL]
		var feedItems []MonitoredItem
		var newestGUID string

		for _, item := range feed.Items {
			guid := item.GUID
			if guid == "" {
				guid = item.Link
			}
			if guid == "" {
				continue
			}

			// Set newest GUID from first item (feeds are typically newest-first).
			if newestGUID == "" {
				newestGUID = guid
			}

			// Stop when we hit the previously seen GUID.
			if guid == lastGUID {
				break
			}

			content := item.Description
			if content == "" {
				content = item.Content
			}
			if len(content) > 2000 {
				content = content[:2000] + "..."
			}

			var createdAt time.Time
			if item.PublishedParsed != nil {
				createdAt = *item.PublishedParsed
			} else if item.UpdatedParsed != nil {
				createdAt = *item.UpdatedParsed
			} else {
				createdAt = time.Now()
			}

			feedHash := fmt.Sprintf("%x", sha256.Sum256([]byte(feedURL)))[:8]
			guidHash := fmt.Sprintf("%x", sha256.Sum256([]byte(guid)))[:12]

			feedItems = append(feedItems, MonitoredItem{
				SourceID:  fmt.Sprintf("rss_%s_%s", feedHash, guidHash),
				Title:     item.Title,
				Content:   content,
				URL:       item.Link,
				Author:    itemAuthor(item),
				CreatedAt: createdAt,
			})

			if len(feedItems) >= 25 {
				break
			}
		}

		if newestGUID != "" {
			newLastSeen[feedURL] = newestGUID
		}
		allItems = append(allItems, feedItems...)
	}

	// Cap total items.
	if len(allItems) > 50 {
		allItems = allItems[:50]
	}

	newWM, _ := json.Marshal(map[string]any{"last_seen": newLastSeen})
	return allItems, newWM, nil
}

func itemAuthor(item *gofeed.Item) string {
	if item.Author != nil && item.Author.Name != "" {
		return item.Author.Name
	}
	if len(item.Authors) > 0 && item.Authors[0].Name != "" {
		return item.Authors[0].Name
	}
	return ""
}
