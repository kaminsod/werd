package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/werd-platform/werd/src/go/api/internal/integration"
	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

// ReplyMonitor periodically checks published posts for new replies
// and ingests them as alerts.
type ReplyMonitor struct {
	q           *storage.Queries
	platformSvc *Platform
	alertSvc    *Alert
	readers     *integration.ReaderRegistry
	interval    time.Duration
}

func NewReplyMonitor(
	q *storage.Queries,
	platformSvc *Platform,
	alertSvc *Alert,
	readers *integration.ReaderRegistry,
	interval time.Duration,
) *ReplyMonitor {
	return &ReplyMonitor{
		q:           q,
		platformSvc: platformSvc,
		alertSvc:    alertSvc,
		readers:     readers,
		interval:    interval,
	}
}

// Run starts the reply monitoring loop. Call in a goroutine.
func (m *ReplyMonitor) Run(ctx context.Context) {
	log.Printf("reply monitor started (interval: %s)", m.interval)

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("reply monitor stopped")
			return
		case <-ticker.C:
			m.check(ctx)
		}
	}
}

func (m *ReplyMonitor) check(ctx context.Context) {
	// Fetch up to 20 monitored results, oldest-checked first.
	results, err := m.q.ListMonitoredResults(ctx, 20)
	if err != nil {
		log.Printf("reply monitor: failed to list monitored results: %v", err)
		return
	}

	for _, result := range results {
		reader, ok := m.readers.Get(result.Platform)
		if !ok {
			log.Printf("reply monitor: no reader for platform %s", result.Platform)
			continue
		}

		// Get credentials for this platform connection.
		creds, err := m.platformSvc.GetConnectionForPublish(ctx, result.ProjectID.String(), result.Platform)
		if err != nil {
			log.Printf("reply monitor: no credentials for %s in project %s: %v", result.Platform, result.ProjectID, err)
			continue
		}

		// Fetch new replies.
		replies, err := reader.GetReplies(ctx, result.PlatformPostID, result.LastKnownReplyID, json.RawMessage(creds.Credentials))
		if err != nil {
			log.Printf("reply monitor: error fetching replies for %s post %s: %v", result.Platform, result.PlatformPostID, err)
			continue
		}

		if len(replies) == 0 {
			// Update checkpoint even if no new replies (resets last_reply_check).
			m.q.UpdateReplyCheckpoint(ctx, storage.UpdateReplyCheckpointParams{
				ID:               result.ID,
				LastKnownReplyID: result.LastKnownReplyID,
			})
			continue
		}

		// Ingest each reply as an alert.
		latestReplyID := result.LastKnownReplyID
		for _, reply := range replies {
			_, _, err := m.alertSvc.Ingest(ctx, IngestRequest{
				ProjectID:  result.ProjectID.String(),
				SourceType: result.Platform,
				SourceID:   fmt.Sprintf("reply_%s", reply.ID),
				Title:      fmt.Sprintf("Reply from @%s", reply.Author),
				Content:    reply.Content,
				URL:        reply.URL,
				Severity:   "medium", // replies to own posts are higher priority
			})
			if err != nil {
				log.Printf("reply monitor: error ingesting reply %s: %v", reply.ID, err)
				continue
			}
			latestReplyID = reply.ID
		}

		// Update checkpoint.
		m.q.UpdateReplyCheckpoint(ctx, storage.UpdateReplyCheckpointParams{
			ID:               result.ID,
			LastKnownReplyID: latestReplyID,
		})

		log.Printf("reply monitor: %d new replies for %s post %s", len(replies), result.Platform, result.PlatformPostID)
	}
}

// EnableMonitoring enables reply monitoring for a specific post platform result.
func (m *ReplyMonitor) EnableMonitoring(ctx context.Context, resultID string, enable bool) error {
	rid, err := uuid.Parse(resultID)
	if err != nil {
		return fmt.Errorf("invalid result ID: %w", err)
	}
	return m.q.SetMonitorReplies(ctx, storage.SetMonitorRepliesParams{
		ID:             rid,
		MonitorReplies: enable,
	})
}
