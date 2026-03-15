package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

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
	q *storage.Queries
}

func NewMonitorSource(q *storage.Queries) *MonitorSource {
	return &MonitorSource{q: q}
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

	_, err = s.q.GetMonitorSourceByID(ctx, storage.GetMonitorSourceByIDParams{
		ID: sid, ProjectID: pid,
	})
	if err != nil {
		return ErrSourceNotFound
	}

	return s.q.DeleteMonitorSource(ctx, storage.DeleteMonitorSourceParams{
		ID: sid, ProjectID: pid,
	})
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
