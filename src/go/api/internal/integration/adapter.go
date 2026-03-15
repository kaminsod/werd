package integration

import (
	"context"
	"encoding/json"
	"fmt"
)

// PlatformAdapter defines the contract for social platform integrations.
type PlatformAdapter interface {
	// Publish posts content to the platform using the provided credentials.
	Publish(ctx context.Context, content string, credentials json.RawMessage) (*PublishResult, error)

	// ValidateCredentials checks whether credentials are well-formed and can
	// authenticate with the platform.
	ValidateCredentials(ctx context.Context, credentials json.RawMessage) error
}

// PublishResult holds the outcome of a successful publish operation.
type PublishResult struct {
	PlatformPostID string `json:"platform_post_id"`
	URL            string `json:"url"`
}

// Registry maps platform names to their adapter implementations.
type Registry struct {
	adapters map[string]PlatformAdapter
}

func NewRegistry() *Registry {
	return &Registry{adapters: make(map[string]PlatformAdapter)}
}

func (r *Registry) Register(platform string, adapter PlatformAdapter) {
	r.adapters[platform] = adapter
}

func (r *Registry) Get(platform string) (PlatformAdapter, error) {
	a, ok := r.adapters[platform]
	if !ok {
		return nil, fmt.Errorf("no adapter registered for platform: %s", platform)
	}
	return a, nil
}

func (r *Registry) Platforms() []string {
	platforms := make([]string, 0, len(r.adapters))
	for p := range r.adapters {
		platforms = append(platforms, p)
	}
	return platforms
}
