package integration

import (
	"context"
	"encoding/json"
	"fmt"
)

// HN implements PlatformAdapter for Hacker News.
// HN is monitoring-only — connections are accepted but publishing is not supported.
type HN struct{}

func NewHN() *HN {
	return &HN{}
}

// ValidateCredentials always succeeds for HN (no credentials needed).
func (h *HN) ValidateCredentials(_ context.Context, _ json.RawMessage) error {
	return nil
}

// Publish always returns an error — HN has no posting API.
func (h *HN) Publish(_ context.Context, _ PublishContent, _ json.RawMessage) (*PublishResult, error) {
	return nil, fmt.Errorf("hacker news does not support publishing — monitoring only")
}
