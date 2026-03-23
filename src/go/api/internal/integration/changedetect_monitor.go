package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"time"
)

// ChangedetectMonitor polls changedetection.io history for new page changes.
// Monitor key: "web:default"
type ChangedetectMonitor struct {
	client *ChangedetectClient
}

func NewChangedetectMonitor(client *ChangedetectClient) *ChangedetectMonitor {
	return &ChangedetectMonitor{client: client}
}

func (m *ChangedetectMonitor) Poll(ctx context.Context, config, watermark, _ json.RawMessage) ([]MonitoredItem, json.RawMessage, error) {
	var cfg struct {
		URLs     []string `json:"urls"`
		WatchIDs []string `json:"watch_ids"`
	}
	if err := json.Unmarshal(config, &cfg); err != nil {
		return nil, watermark, fmt.Errorf("changedetect monitor: invalid config: %w", err)
	}
	if len(cfg.WatchIDs) == 0 {
		return nil, watermark, nil // not provisioned yet
	}

	var wm struct {
		LastSeen map[string]int64 `json:"last_seen"`
	}
	json.Unmarshal(watermark, &wm)
	if wm.LastSeen == nil {
		wm.LastSeen = make(map[string]int64)
	}

	var items []MonitoredItem
	newLastSeen := make(map[string]int64)
	for k, v := range wm.LastSeen {
		newLastSeen[k] = v
	}

	for i, watchID := range cfg.WatchIDs {
		history, err := m.client.GetWatchHistory(ctx, watchID)
		if err != nil {
			log.Printf("changedetect monitor: error fetching history for %s: %v", watchID, err)
			continue
		}

		// Collect timestamps newer than watermark.
		threshold := wm.LastSeen[watchID]
		var newTimestamps []string
		var maxTS int64

		for ts := range history {
			tsInt, err := strconv.ParseInt(ts, 10, 64)
			if err != nil {
				continue
			}
			if tsInt > threshold {
				newTimestamps = append(newTimestamps, ts)
			}
			if tsInt > maxTS {
				maxTS = tsInt
			}
		}

		if maxTS > 0 {
			newLastSeen[watchID] = maxTS
		}

		// Sort newest first and cap.
		sort.Sort(sort.Reverse(sort.StringSlice(newTimestamps)))
		if len(newTimestamps) > 10 {
			newTimestamps = newTimestamps[:10]
		}

		// Determine URL label.
		urlLabel := watchID
		if i < len(cfg.URLs) {
			urlLabel = cfg.URLs[i]
		}

		for _, ts := range newTimestamps {
			snapshot, err := m.client.GetSnapshot(ctx, watchID, ts)
			if err != nil {
				log.Printf("changedetect monitor: error fetching snapshot %s/%s: %v", watchID, ts, err)
				continue
			}
			if len(snapshot) > 2000 {
				snapshot = snapshot[:2000] + "..."
			}

			tsInt, _ := strconv.ParseInt(ts, 10, 64)
			items = append(items, MonitoredItem{
				SourceID:  fmt.Sprintf("cd_%s_%s", watchID, ts),
				Title:     fmt.Sprintf("Change detected: %s", urlLabel),
				Content:   snapshot,
				URL:       urlLabel,
				CreatedAt: time.Unix(tsInt, 0),
			})
		}
	}

	newWM, _ := json.Marshal(map[string]any{"last_seen": newLastSeen})
	return items, newWM, nil
}
