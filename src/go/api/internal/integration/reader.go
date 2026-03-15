package integration

import (
	"context"
	"encoding/json"
	"time"
)

// PlatformReply represents a single reply/comment from a platform.
type PlatformReply struct {
	ID        string    `json:"id"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	ParentID  string    `json:"parent_id"`
}

// PlatformReader defines the contract for reading replies from platforms.
type PlatformReader interface {
	// GetReplies fetches replies to a specific post since the given watermark ID.
	// Returns replies newer than sinceID (empty string = get all).
	GetReplies(ctx context.Context, platformPostID string, sinceID string, credentials json.RawMessage) ([]PlatformReply, error)
}

// ReaderRegistry maps platform names to their reader implementations.
type ReaderRegistry struct {
	readers map[string]PlatformReader
}

func NewReaderRegistry() *ReaderRegistry {
	return &ReaderRegistry{readers: make(map[string]PlatformReader)}
}

func (r *ReaderRegistry) Register(platform string, reader PlatformReader) {
	r.readers[platform] = reader
}

func (r *ReaderRegistry) Get(platform string) (PlatformReader, bool) {
	reader, ok := r.readers[platform]
	return reader, ok
}
