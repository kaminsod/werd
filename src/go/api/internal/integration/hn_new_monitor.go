package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// HNNewMonitor monitors all new HN stories without keyword filtering.
// Keyword filtering is handled by processing rules in the pipeline.
type HNNewMonitor struct {
	reader *HNReader
}

func NewHNNewMonitor() *HNNewMonitor {
	return &HNNewMonitor{reader: NewHNReader()}
}

func (m *HNNewMonitor) Poll(ctx context.Context, config, watermark, _ json.RawMessage) ([]MonitoredItem, json.RawMessage, error) {
	var wm struct {
		MaxSeenID int `json:"max_seen_id"`
	}
	json.Unmarshal(watermark, &wm)

	// Fetch new story IDs.
	ids, err := m.reader.fetchNewStoryIDs(ctx)
	if err != nil {
		return nil, watermark, fmt.Errorf("hn new: %w", err)
	}

	// Filter to new IDs only.
	var newIDs []int
	maxID := wm.MaxSeenID
	for _, id := range ids {
		if id > wm.MaxSeenID {
			newIDs = append(newIDs, id)
		}
		if id > maxID {
			maxID = id
		}
	}

	if len(newIDs) == 0 {
		newWM, _ := json.Marshal(map[string]int{"max_seen_id": maxID})
		return nil, newWM, nil
	}

	// Cap per poll to avoid flooding.
	if len(newIDs) > 50 {
		newIDs = newIDs[:50]
	}

	// Fetch items concurrently (max 5).
	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)
	var mu sync.Mutex
	var items []MonitoredItem

	for _, id := range newIDs {
		sem <- struct{}{}
		go func(storyID int) {
			defer func() { <-sem }()

			item, err := m.reader.fetchItem(ctx, storyID)
			if err != nil || item == nil || item.Type != "story" {
				return
			}

			url := fmt.Sprintf("https://news.ycombinator.com/item?id=%d", item.ID)
			content := item.Text
			if content == "" && item.Title != "" {
				content = item.Title
			}

			mu.Lock()
			items = append(items, MonitoredItem{
				SourceID:  fmt.Sprintf("hn_%d", item.ID),
				Title:     item.Title,
				Content:   content,
				URL:       url,
				Author:    item.By,
				CreatedAt: time.Unix(item.Time, 0),
			})
			mu.Unlock()
		}(id)
	}

	// Wait for all goroutines.
	for i := 0; i < maxConcurrent; i++ {
		sem <- struct{}{}
	}

	newWM, _ := json.Marshal(map[string]int{"max_seen_id": maxID})
	return items, newWM, nil
}
