package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/werd-platform/werd/src/go/api/internal/integration"
	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

var (
	ErrConnectionNotFound  = errors.New("platform connection not found")
	ErrUnsupportedPlatform = errors.New("unsupported platform")
	ErrConnectionDisabled  = errors.New("platform connection is disabled")
)

// ConnectionInfo is the service-layer representation. Credentials are always redacted.
type ConnectionInfo struct {
	ID        string
	ProjectID string
	Platform  string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Platform struct {
	q        *storage.Queries
	registry *integration.Registry
}

func NewPlatform(q *storage.Queries, registry *integration.Registry) *Platform {
	return &Platform{q: q, registry: registry}
}

// CreateConnection validates credentials against the platform adapter, then persists.
func (s *Platform) CreateConnection(ctx context.Context, projectID, platform string, credentials json.RawMessage, enabled bool) (*ConnectionInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	adapter, err := s.registry.Get(platform)
	if err != nil {
		return nil, ErrUnsupportedPlatform
	}

	// TODO: encrypt credentials at rest
	if err := adapter.ValidateCredentials(ctx, credentials); err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}

	conn, err := s.q.CreatePlatformConnection(ctx, storage.CreatePlatformConnectionParams{
		ProjectID: pid, Platform: platform, Credentials: credentials, Enabled: enabled,
	})
	if err != nil {
		return nil, fmt.Errorf("creating connection: %w", err)
	}

	return storageConnToInfo(conn), nil
}

func (s *Platform) ListConnections(ctx context.Context, projectID string) ([]ConnectionInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	conns, err := s.q.ListPlatformConnections(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("listing connections: %w", err)
	}

	result := make([]ConnectionInfo, len(conns))
	for i, c := range conns {
		result[i] = *storageConnToInfo(c)
	}
	return result, nil
}

func (s *Platform) GetConnection(ctx context.Context, projectID, connID string) (*ConnectionInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrConnectionNotFound
	}
	cid, err := uuid.Parse(connID)
	if err != nil {
		return nil, ErrConnectionNotFound
	}

	conn, err := s.q.GetPlatformConnectionByID(ctx, storage.GetPlatformConnectionByIDParams{
		ID: cid, ProjectID: pid,
	})
	if err != nil {
		return nil, ErrConnectionNotFound
	}

	return storageConnToInfo(conn), nil
}

func (s *Platform) UpdateConnection(ctx context.Context, projectID, connID, platform string, credentials json.RawMessage, enabled bool) (*ConnectionInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrConnectionNotFound
	}
	cid, err := uuid.Parse(connID)
	if err != nil {
		return nil, ErrConnectionNotFound
	}

	adapter, err := s.registry.Get(platform)
	if err != nil {
		return nil, ErrUnsupportedPlatform
	}

	if err := adapter.ValidateCredentials(ctx, credentials); err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}

	conn, err := s.q.UpdatePlatformConnection(ctx, storage.UpdatePlatformConnectionParams{
		ID: cid, ProjectID: pid, Platform: platform, Credentials: credentials, Enabled: enabled,
	})
	if err != nil {
		return nil, ErrConnectionNotFound
	}

	return storageConnToInfo(conn), nil
}

func (s *Platform) DeleteConnection(ctx context.Context, projectID, connID string) error {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return ErrConnectionNotFound
	}
	cid, err := uuid.Parse(connID)
	if err != nil {
		return ErrConnectionNotFound
	}

	_, err = s.q.GetPlatformConnectionByID(ctx, storage.GetPlatformConnectionByIDParams{
		ID: cid, ProjectID: pid,
	})
	if err != nil {
		return ErrConnectionNotFound
	}

	return s.q.DeletePlatformConnection(ctx, storage.DeletePlatformConnectionParams{
		ID: cid, ProjectID: pid,
	})
}

// GetCredentials fetches the raw credentials for a platform connection.
// Used internally by the post service during publishing — never exposed via API.
func (s *Platform) GetCredentials(ctx context.Context, projectID, platform string) (json.RawMessage, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrConnectionNotFound
	}

	conn, err := s.q.GetPlatformConnectionByPlatform(ctx, storage.GetPlatformConnectionByPlatformParams{
		ProjectID: pid, Platform: platform,
	})
	if err != nil {
		return nil, ErrConnectionNotFound
	}

	if !conn.Enabled {
		return nil, ErrConnectionDisabled
	}

	return conn.Credentials, nil
}

func storageConnToInfo(c storage.PlatformConnection) *ConnectionInfo {
	return &ConnectionInfo{
		ID:        c.ID.String(),
		ProjectID: c.ProjectID.String(),
		Platform:  c.Platform,
		Enabled:   c.Enabled,
		CreatedAt: c.CreatedAt.Time,
		UpdatedAt: c.UpdatedAt.Time,
	}
}
