package service

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/werd-platform/werd/src/go/api/internal/integration"
	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

// SourcePoller polls monitor sources for new items and ingests them as alerts.
type SourcePoller struct {
	q           *storage.Queries
	platformSvc *Platform
	alertSvc    *Alert
	monitors    *integration.SourceMonitorRegistry
	pipeline    *ProcessingPipeline
	interval    time.Duration
}

func NewSourcePoller(
	q *storage.Queries,
	platformSvc *Platform,
	alertSvc *Alert,
	monitors *integration.SourceMonitorRegistry,
	pipeline *ProcessingPipeline,
	interval time.Duration,
) *SourcePoller {
	return &SourcePoller{
		q:           q,
		platformSvc: platformSvc,
		alertSvc:    alertSvc,
		monitors:    monitors,
		pipeline:    pipeline,
		interval:    interval,
	}
}

// Run starts the source polling loop. Call in a goroutine.
func (p *SourcePoller) Run(ctx context.Context) {
	log.Printf("source poller started (interval: %s)", p.interval)

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("source poller stopped")
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *SourcePoller) poll(ctx context.Context) {
	sources, err := p.q.ListAllEnabledMonitorSources(ctx, 20)
	if err != nil {
		log.Printf("source poller: failed to list sources: %v", err)
		return
	}

	for _, source := range sources {
		// Parse config to get mode and poll_interval.
		var cfg struct {
			Mode             string `json:"mode"`
			PollIntervalSecs int    `json:"poll_interval_secs"`
		}
		json.Unmarshal(source.Config, &cfg)

		// Check if enough time has passed since last poll.
		if source.LastPollAt.Valid && cfg.PollIntervalSecs > 0 {
			nextPoll := source.LastPollAt.Time.Add(time.Duration(cfg.PollIntervalSecs) * time.Second)
			if time.Now().Before(nextPoll) {
				continue // not due yet
			}
		}

		// Determine the monitor key: "{type}:{mode}".
		mode := cfg.Mode
		if mode == "" {
			mode = "default"
		}
		monitorKey := string(source.Type) + ":" + mode

		monitor, err := p.monitors.Get(monitorKey)
		if err != nil {
			// No monitor for this type:mode — skip silently.
			continue
		}

		// Get credentials if needed.
		var creds json.RawMessage
		switch source.Type {
		case storage.MonitorTypeReddit:
			conn, err := p.platformSvc.GetConnectionForPublish(ctx, source.ProjectID.String(), "reddit")
			if err != nil {
				log.Printf("source poller: no reddit credentials for project %s: %v", source.ProjectID, err)
				continue
			}
			creds = conn.Credentials
		case storage.MonitorTypeBluesky:
			conn, err := p.platformSvc.GetConnectionForPublish(ctx, source.ProjectID.String(), "bluesky")
			if err != nil {
				log.Printf("source poller: no bluesky credentials for project %s: %v", source.ProjectID, err)
				continue
			}
			creds = conn.Credentials
		// hn, web, rss, github: no credentials needed
		}

		// Check if this is a first poll (no watermark = initialization).
		isFirstPoll := len(source.Watermark) == 0 || string(source.Watermark) == "{}"

		// Poll.
		items, newWatermark, err := monitor.Poll(ctx, source.Config, source.Watermark, creds)
		if err != nil {
			log.Printf("source poller: error polling %s source %s: %v", monitorKey, source.ID, err)
			// Update last_poll_at even on error to prevent tight loop.
			p.q.UpdateSourceWatermark(ctx, storage.UpdateSourceWatermarkParams{
				ID: source.ID, Watermark: source.Watermark,
			})
			continue
		}

		// On first poll, set watermark but don't ingest (prevents startup flood).
		if isFirstPoll {
			log.Printf("source poller: initialized %s source %s (watermark set, %d items skipped)", monitorKey, source.ID, len(items))
			p.q.UpdateSourceWatermark(ctx, storage.UpdateSourceWatermarkParams{
				ID: source.ID, Watermark: newWatermark,
			})
			continue
		}

		// Run processing pipeline (filter + classify).
		sourceID := source.ID
		projectID := source.ProjectID
		filteredItems := items
		var classifyResults []ProcessingResult

		if p.pipeline != nil && len(items) > 0 {
			filteredItems, classifyResults, err = p.pipeline.Process(ctx, projectID, sourceID, string(source.Type), items)
			if err != nil {
				log.Printf("source poller: pipeline error for %s source %s: %v", monitorKey, source.ID, err)
				// Fall through with unfiltered items.
				filteredItems = items
				classifyResults = nil
			}
		}

		// Ingest filtered items as alerts.
		ingested := 0
		for i, item := range filteredItems {
			req := IngestRequest{
				ProjectID:       projectID.String(),
				SourceType:      string(source.Type),
				SourceID:        item.SourceID,
				Title:           item.Title,
				Content:         item.Content,
				URL:             item.URL,
				Severity:        "low",
				Tags:            []string{},
				MonitorSourceID: sourceID.String(),
			}

			// Apply classification results if available.
			if classifyResults != nil && i < len(classifyResults) {
				cr := classifyResults[i]
				if cr.Severity != "" {
					req.Severity = cr.Severity
				}
				if len(cr.Tags) > 0 {
					req.Tags = cr.Tags
				}
				req.ClassificationReason = cr.ClassificationReason
			}

			_, _, err := p.alertSvc.Ingest(ctx, req)
			if err != nil {
				log.Printf("source poller: error ingesting %s: %v", item.SourceID, err)
				continue
			}
			ingested++
		}

		if ingested > 0 {
			log.Printf("source poller: %s source %s: ingested %d/%d items (filtered from %d)", monitorKey, source.ID, ingested, len(filteredItems), len(items))
		}

		// Update watermark.
		p.q.UpdateSourceWatermark(ctx, storage.UpdateSourceWatermarkParams{
			ID: source.ID, Watermark: newWatermark,
		})
	}
}

// ensure uuid is used (referenced in pipeline Process call).
var _ = uuid.UUID{}
