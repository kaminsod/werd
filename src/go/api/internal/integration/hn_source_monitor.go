package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// HNThreadMonitor monitors an HN thread for new comments.
type HNThreadMonitor struct {
	reader *HNReader
}

func NewHNThreadMonitor() *HNThreadMonitor {
	return &HNThreadMonitor{reader: NewHNReader()}
}

func (m *HNThreadMonitor) Poll(ctx context.Context, config, watermark, _ json.RawMessage) ([]MonitoredItem, json.RawMessage, error) {
	var cfg struct {
		ItemID int `json:"item_id"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, watermark, fmt.Errorf("hn thread: invalid config: %w", err)
	}
	if cfg.ItemID == 0 {
		return nil, watermark, fmt.Errorf("hn thread: item_id required")
	}

	var wm struct {
		LastSeenID string `json:"last_seen_id"`
	}
	json.Unmarshal(watermark, &wm)

	postID := fmt.Sprintf("hn_%d", cfg.ItemID)
	replies, err := m.reader.GetReplies(ctx, postID, wm.LastSeenID, nil)
	if err != nil {
		return nil, watermark, err
	}

	items := make([]MonitoredItem, len(replies))
	newLastSeen := wm.LastSeenID
	for i, r := range replies {
		items[i] = MonitoredItem{
			SourceID:  r.ID,
			Title:     fmt.Sprintf("Comment by %s", r.Author),
			Content:   r.Content,
			URL:       r.URL,
			Author:    r.Author,
			CreatedAt: r.CreatedAt,
		}
		if r.ID > newLastSeen {
			newLastSeen = r.ID
		}
	}

	newWM, _ := json.Marshal(map[string]string{"last_seen_id": newLastSeen})
	return items, newWM, nil
}

// HNKeywordMonitor monitors HN new stories for keyword matches.
type HNKeywordMonitor struct {
	reader *HNReader
}

func NewHNKeywordMonitor() *HNKeywordMonitor {
	return &HNKeywordMonitor{reader: NewHNReader()}
}

func (m *HNKeywordMonitor) Poll(ctx context.Context, config, watermark, _ json.RawMessage) ([]MonitoredItem, json.RawMessage, error) {
	var wm struct {
		MaxSeenID int `json:"max_seen_id"`
	}
	json.Unmarshal(watermark, &wm)

	// Fetch new story IDs.
	ids, err := m.reader.fetchNewStoryIDs(ctx)
	if err != nil {
		return nil, watermark, fmt.Errorf("hn keywords: %w", err)
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

	// Cap per poll.
	if len(newIDs) > 50 {
		newIDs = newIDs[:50]
	}

	// Fetch items concurrently (max 5).
	// Keyword filtering is now handled by processing rules in the pipeline.
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

// fetchNewStoryIDs fetches the newest story IDs from HN.
func (r *HNReader) fetchNewStoryIDs(ctx context.Context) ([]int, error) {
	url := r.baseURL + "/newstories.json"
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

	var ids []int
	json.Unmarshal(body, &ids)
	return ids, nil
}

// Helper for watermark ID parsing.
func init() {
	_ = strconv.Atoi
}
