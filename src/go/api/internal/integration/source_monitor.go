package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// MonitoredItem represents a single item found by a source monitor.
type MonitoredItem struct {
	SourceID  string    `json:"source_id"`
	Title     string    `json:"title"`
	Content   string    `json:"content"`
	URL       string    `json:"url"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

// SourceMonitor fetches new items from a platform based on monitor source config.
type SourceMonitor interface {
	// Poll checks for new items. Returns items and a new watermark string.
	// The watermark is opaque — stored as-is and passed back on the next poll.
	Poll(ctx context.Context, config json.RawMessage, watermark json.RawMessage, credentials json.RawMessage) ([]MonitoredItem, json.RawMessage, error)
}

// SourceMonitorRegistry maps "{type}:{mode}" keys to monitor implementations.
type SourceMonitorRegistry struct {
	monitors map[string]SourceMonitor
}

func NewSourceMonitorRegistry() *SourceMonitorRegistry {
	return &SourceMonitorRegistry{monitors: make(map[string]SourceMonitor)}
}

func (r *SourceMonitorRegistry) Register(key string, monitor SourceMonitor) {
	r.monitors[key] = monitor
}

func (r *SourceMonitorRegistry) Get(key string) (SourceMonitor, error) {
	m, ok := r.monitors[key]
	if !ok {
		return nil, fmt.Errorf("no source monitor for: %s", key)
	}
	return m, nil
}
