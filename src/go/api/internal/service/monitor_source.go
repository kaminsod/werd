package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/werd-platform/werd/src/go/api/internal/integration"
	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

var (
	ErrSourceNotFound = errors.New("monitor source not found")
)

type SourceInfo struct {
	ID        string
	ProjectID string
	Type      string
	Config    map[string]any
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type MonitorSource struct {
	q            *storage.Queries
	changedetect *integration.ChangedetectClient // nil if not configured
}

func NewMonitorSource(q *storage.Queries, changedetect *integration.ChangedetectClient) *MonitorSource {
	return &MonitorSource{q: q, changedetect: changedetect}
}

func (s *MonitorSource) Create(ctx context.Context, projectID, sourceType string, config map[string]any, enabled bool) (*SourceInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	mt, err := parseMonitorType(sourceType)
	if err != nil {
		return nil, ErrInvalidSourceType
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}

	src, err := s.q.CreateMonitorSource(ctx, storage.CreateMonitorSourceParams{
		ProjectID: pid, Type: mt, Config: configJSON, Enabled: enabled,
	})
	if err != nil {
		return nil, fmt.Errorf("creating monitor source: %w", err)
	}

	// Provision changedetection.io watches for web sources.
	if mt == storage.MonitorTypeWeb && s.changedetect != nil {
		s.provisionWebWatches(ctx, pid, src.ID, config)
	}

	return storageSourceToInfo(src), nil
}

func (s *MonitorSource) List(ctx context.Context, projectID string) ([]SourceInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	sources, err := s.q.ListMonitorSources(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("listing sources: %w", err)
	}

	result := make([]SourceInfo, len(sources))
	for i, src := range sources {
		result[i] = *storageSourceToInfo(src)
	}
	return result, nil
}

func (s *MonitorSource) Get(ctx context.Context, projectID, sourceID string) (*SourceInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrSourceNotFound
	}
	sid, err := uuid.Parse(sourceID)
	if err != nil {
		return nil, ErrSourceNotFound
	}

	src, err := s.q.GetMonitorSourceByID(ctx, storage.GetMonitorSourceByIDParams{
		ID: sid, ProjectID: pid,
	})
	if err != nil {
		return nil, ErrSourceNotFound
	}

	return storageSourceToInfo(src), nil
}

func (s *MonitorSource) Update(ctx context.Context, projectID, sourceID, sourceType string, config map[string]any, enabled bool) (*SourceInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrSourceNotFound
	}
	sid, err := uuid.Parse(sourceID)
	if err != nil {
		return nil, ErrSourceNotFound
	}

	mt, err := parseMonitorType(sourceType)
	if err != nil {
		return nil, ErrInvalidSourceType
	}

	// For web sources, diff old vs new URLs and reconcile watches.
	if mt == storage.MonitorTypeWeb && s.changedetect != nil {
		oldSrc, err := s.q.GetMonitorSourceByID(ctx, storage.GetMonitorSourceByIDParams{
			ID: sid, ProjectID: pid,
		})
		if err == nil {
			var oldCfg map[string]any
			json.Unmarshal(oldSrc.Config, &oldCfg)
			s.reconcileWebWatches(ctx, pid, sid, oldCfg, config)
		}
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}

	src, err := s.q.UpdateMonitorSource(ctx, storage.UpdateMonitorSourceParams{
		ID: sid, ProjectID: pid, Type: mt, Config: configJSON, Enabled: enabled,
	})
	if err != nil {
		return nil, ErrSourceNotFound
	}

	return storageSourceToInfo(src), nil
}

func (s *MonitorSource) Delete(ctx context.Context, projectID, sourceID string) error {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return ErrSourceNotFound
	}
	sid, err := uuid.Parse(sourceID)
	if err != nil {
		return ErrSourceNotFound
	}

	src, err := s.q.GetMonitorSourceByID(ctx, storage.GetMonitorSourceByIDParams{
		ID: sid, ProjectID: pid,
	})
	if err != nil {
		return ErrSourceNotFound
	}

	// Clean up changedetection.io watches for web sources.
	if src.Type == storage.MonitorTypeWeb && s.changedetect != nil {
		s.deprovisionWebWatches(ctx, pid, src)
	}

	return s.q.DeleteMonitorSource(ctx, storage.DeleteMonitorSourceParams{
		ID: sid, ProjectID: pid,
	})
}

// provisionWebWatches creates watches in changedetection.io for each URL.
func (s *MonitorSource) provisionWebWatches(ctx context.Context, projectID, sourceID uuid.UUID, config map[string]any) {
	urls := extractStringSlice(config, "urls")
	if len(urls) == 0 {
		return
	}

	tag := sourceID.String()
	var watchIDs []string
	for _, u := range urls {
		watchID, err := s.changedetect.CreateWatch(ctx, u, tag, "Werd: "+u)
		if err != nil {
			log.Printf("changedetect: failed to create watch for %s: %v", u, err)
			continue
		}
		watchIDs = append(watchIDs, watchID)
	}

	if len(watchIDs) > 0 {
		// Write watch_ids back into the source config.
		config["watch_ids"] = watchIDs
		configJSON, _ := json.Marshal(config)
		s.q.UpdateMonitorSource(ctx, storage.UpdateMonitorSourceParams{
			ID: sourceID, ProjectID: projectID,
			Type: storage.MonitorTypeWeb, Config: configJSON, Enabled: true,
		})

		// Track in service_instances.
		instanceConfig, _ := json.Marshal(map[string]any{"watch_ids": watchIDs, "urls": urls})
		s.q.CreateServiceInstance(ctx, storage.CreateServiceInstanceParams{
			ProjectID:  projectID,
			Service:    storage.ServiceNameChangedetect,
			ExternalID: sourceID.String(),
			Config:     instanceConfig,
			Status:     storage.ServiceStatusActive,
		})

		log.Printf("changedetect: provisioned %d watches for source %s", len(watchIDs), sourceID)
	}
}

// reconcileWebWatches diffs old vs new URLs and creates/deletes watches.
func (s *MonitorSource) reconcileWebWatches(ctx context.Context, projectID, sourceID uuid.UUID, oldCfg, newCfg map[string]any) {
	oldURLs := extractStringSlice(oldCfg, "urls")
	newURLs := extractStringSlice(newCfg, "urls")
	oldWatchIDs := extractStringSlice(oldCfg, "watch_ids")

	// Build URL→watchID map from old config.
	urlToWatch := make(map[string]string)
	for i, u := range oldURLs {
		if i < len(oldWatchIDs) {
			urlToWatch[u] = oldWatchIDs[i]
		}
	}

	newSet := make(map[string]bool)
	for _, u := range newURLs {
		newSet[u] = true
	}

	// Delete watches for removed URLs.
	for _, u := range oldURLs {
		if !newSet[u] {
			if wid, ok := urlToWatch[u]; ok {
				if err := s.changedetect.DeleteWatch(ctx, wid); err != nil {
					log.Printf("changedetect: failed to delete watch %s: %v", wid, err)
				}
			}
		}
	}

	// Create watches for added URLs.
	tag := sourceID.String()
	for _, u := range newURLs {
		if _, exists := urlToWatch[u]; !exists {
			watchID, err := s.changedetect.CreateWatch(ctx, u, tag, "Werd: "+u)
			if err != nil {
				log.Printf("changedetect: failed to create watch for %s: %v", u, err)
				continue
			}
			urlToWatch[u] = watchID
		}
	}

	// Rebuild watch_ids in order of newURLs.
	var watchIDs []string
	for _, u := range newURLs {
		if wid, ok := urlToWatch[u]; ok {
			watchIDs = append(watchIDs, wid)
		}
	}
	newCfg["watch_ids"] = watchIDs

	// Update service instance.
	instanceConfig, _ := json.Marshal(map[string]any{"watch_ids": watchIDs, "urls": newURLs})
	inst, err := s.q.GetServiceInstanceByExternalID(ctx, storage.GetServiceInstanceByExternalIDParams{
		ProjectID: projectID, Service: storage.ServiceNameChangedetect, ExternalID: sourceID.String(),
	})
	if err == nil {
		s.q.UpdateServiceInstance(ctx, storage.UpdateServiceInstanceParams{
			ID: inst.ID, Config: instanceConfig, Status: storage.ServiceStatusActive,
		})
	}
}

// deprovisionWebWatches deletes all watches from changedetection.io.
func (s *MonitorSource) deprovisionWebWatches(ctx context.Context, projectID uuid.UUID, src storage.MonitorSource) {
	var cfg map[string]any
	json.Unmarshal(src.Config, &cfg)

	watchIDs := extractStringSlice(cfg, "watch_ids")
	for _, wid := range watchIDs {
		if err := s.changedetect.DeleteWatch(ctx, wid); err != nil {
			log.Printf("changedetect: failed to delete watch %s: %v", wid, err)
		}
	}

	// Clean up service instance.
	s.q.DeleteServiceInstanceByExternalID(ctx, storage.DeleteServiceInstanceByExternalIDParams{
		ProjectID: projectID, Service: storage.ServiceNameChangedetect, ExternalID: src.ID.String(),
	})

	if len(watchIDs) > 0 {
		log.Printf("changedetect: deprovisioned %d watches for source %s", len(watchIDs), src.ID)
	}
}

func extractStringSlice(m map[string]any, key string) []string {
	val, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := val.([]any)
	if !ok {
		// Try direct string slice (from typed JSON unmarshal).
		if ss, ok := val.([]string); ok {
			return ss
		}
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, v := range arr {
		if s, ok := v.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func storageSourceToInfo(src storage.MonitorSource) *SourceInfo {
	var cfg map[string]any
	if len(src.Config) > 0 {
		json.Unmarshal(src.Config, &cfg)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	return &SourceInfo{
		ID:        src.ID.String(),
		ProjectID: src.ProjectID.String(),
		Type:      string(src.Type),
		Config:    cfg,
		Enabled:   src.Enabled,
		CreatedAt: src.CreatedAt.Time,
		UpdatedAt: src.UpdatedAt.Time,
	}
}
