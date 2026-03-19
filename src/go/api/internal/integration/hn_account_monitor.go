package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

// HNAccountMonitor monitors an HN user's submissions for new replies.
type HNAccountMonitor struct {
	reader *HNReader
}

func NewHNAccountMonitor() *HNAccountMonitor {
	return &HNAccountMonitor{reader: NewHNReader()}
}

func (m *HNAccountMonitor) Poll(ctx context.Context, config, watermark, _ json.RawMessage) ([]MonitoredItem, json.RawMessage, error) {
	var cfg struct {
		Username string `json:"username"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, watermark, fmt.Errorf("hn account: invalid config: %w", err)
	}
	if cfg.Username == "" {
		return nil, watermark, fmt.Errorf("hn account: username required")
	}

	// Fetch user's submissions.
	user, err := m.reader.fetchUser(ctx, cfg.Username)
	if err != nil {
		return nil, watermark, fmt.Errorf("hn account: %w", err)
	}

	// Take the 20 most recent submissions.
	submissionIDs := user.Submitted
	if len(submissionIDs) > 20 {
		submissionIDs = submissionIDs[:20]
	}

	// Parse watermark: per-submission max kid ID.
	var wm struct {
		MaxKids map[string]int `json:"max_kids"`
	}
	json.Unmarshal(watermark, &wm)
	if wm.MaxKids == nil {
		wm.MaxKids = make(map[string]int)
	}

	// Check each submission for new kids.
	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)
	var mu sync.Mutex
	var items []MonitoredItem
	newMaxKids := make(map[string]int)

	for _, subID := range submissionIDs {
		subKey := fmt.Sprintf("%d", subID)
		sem <- struct{}{}

		go func(id int, key string) {
			defer func() { <-sem }()

			parent, err := m.reader.fetchItem(ctx, id)
			if err != nil || parent == nil {
				return
			}

			prevMax := wm.MaxKids[key]
			localMax := prevMax

			for _, kidID := range parent.Kids {
				if kidID <= prevMax {
					continue
				}
				if kidID > localMax {
					localMax = kidID
				}

				kid, err := m.reader.fetchItem(ctx, kidID)
				if err != nil || kid == nil || kid.Type != "comment" {
					continue
				}

				var title string
				if parent.Type == "story" || parent.Title != "" {
					// Parent is a story
					displayTitle := strings.TrimSpace(parent.Title)
					if displayTitle == "" {
						displayTitle = strings.TrimSpace(m.reader.resolveStoryTitle(ctx, parent, 10))
					}
					if displayTitle != "" {
						title = fmt.Sprintf("%s commented on \"%s\"", kid.By, displayTitle)
					} else {
						title = fmt.Sprintf("%s replied to %s", kid.By, parent.By)
					}
				} else {
					// Parent is a comment
					storyTitle := strings.TrimSpace(m.reader.resolveStoryTitle(ctx, parent, 10))
					if storyTitle != "" {
						title = fmt.Sprintf("%s replied to %s on \"%s\"", kid.By, parent.By, storyTitle)
					} else {
						title = fmt.Sprintf("%s replied to %s", kid.By, parent.By)
					}
				}

				mu.Lock()
				items = append(items, MonitoredItem{
					SourceID:  fmt.Sprintf("hn_reply_%d", kid.ID),
					Title:     title,
					Content:   kid.Text,
					URL:       fmt.Sprintf("https://news.ycombinator.com/item?id=%d", kid.ID),
					Author:    kid.By,
					CreatedAt: time.Unix(kid.Time, 0),
				})
				mu.Unlock()
			}

			mu.Lock()
			newMaxKids[key] = localMax
			mu.Unlock()
		}(subID, subKey)
	}

	// Wait for all goroutines.
	for i := 0; i < maxConcurrent; i++ {
		sem <- struct{}{}
	}

	// Merge new max kids with previous (keep entries for submissions we didn't check this time).
	for k, v := range wm.MaxKids {
		if _, exists := newMaxKids[k]; !exists {
			newMaxKids[k] = v
		}
	}

	newWM, _ := json.Marshal(map[string]any{"max_kids": newMaxKids})
	return items, newWM, nil
}

// hnUser and fetchUser are defined in hn_reader.go.
