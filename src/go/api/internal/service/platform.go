package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/werd-platform/werd/src/go/api/internal/integration"
	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

var (
	ErrConnectionNotFound    = errors.New("platform connection not found")
	ErrUnsupportedPlatform   = errors.New("unsupported platform")
	ErrBrowserNotConfigured  = errors.New("browser method is not available — the browser automation service is not configured")
	ErrConnectionDisabled    = errors.New("platform connection is disabled")
	ErrInvalidMethod         = errors.New("method must be 'api' or 'browser'")
)

// ConnectionInfo is the service-layer representation. Credentials are always redacted.
type ConnectionInfo struct {
	ID        string
	ProjectID string
	Platform  string
	Method    string
	Target    string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ConnectionWithCreds is used internally by the publish flow — never exposed via API.
type ConnectionWithCreds struct {
	Platform    string
	Method      string
	Credentials json.RawMessage
}

type Platform struct {
	q        *storage.Queries
	registry *integration.Registry
}

func NewPlatform(q *storage.Queries, registry *integration.Registry) *Platform {
	return &Platform{q: q, registry: registry}
}

// registryKey constructs the adapter registry key: "platform:method".
func registryKey(platform, method string) string {
	return platform + ":" + method
}

func validateMethod(method string) error {
	if method != "api" && method != "browser" {
		return ErrInvalidMethod
	}
	return nil
}

// CreateConnection validates credentials against the platform adapter, then persists.
func (s *Platform) CreateConnection(ctx context.Context, projectID, platform, method string, credentials json.RawMessage, enabled bool) (*ConnectionInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	if err := validateMethod(method); err != nil {
		return nil, err
	}

	adapter, err := s.registry.Get(registryKey(platform, method))
	if err != nil {
		// Distinguish "unknown platform" from "browser not configured".
		if method == "browser" {
			if _, apiErr := s.registry.Get(registryKey(platform, "api")); apiErr == nil {
				return nil, ErrBrowserNotConfigured
			}
		}
		return nil, ErrUnsupportedPlatform
	}

	// TODO: encrypt credentials at rest
	if err := adapter.ValidateCredentials(ctx, credentials); err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}

	conn, err := s.q.CreatePlatformConnection(ctx, storage.CreatePlatformConnectionParams{
		ProjectID: pid, Platform: platform, Method: method,
		Credentials: credentials, Enabled: enabled,
	})
	if err != nil {
		return nil, fmt.Errorf("creating connection: %w", err)
	}

	return connInfoFromCreate(conn), nil
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
		result[i] = *connInfoFromList(c)
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

	return connInfoFromGet(conn), nil
}

func (s *Platform) UpdateConnection(ctx context.Context, projectID, connID, platform, method string, credentials json.RawMessage, enabled bool) (*ConnectionInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrConnectionNotFound
	}
	cid, err := uuid.Parse(connID)
	if err != nil {
		return nil, ErrConnectionNotFound
	}

	if err := validateMethod(method); err != nil {
		return nil, err
	}

	adapter, err := s.registry.Get(registryKey(platform, method))
	if err != nil {
		if method == "browser" {
			if _, apiErr := s.registry.Get(registryKey(platform, "api")); apiErr == nil {
				return nil, ErrBrowserNotConfigured
			}
		}
		return nil, ErrUnsupportedPlatform
	}

	if err := adapter.ValidateCredentials(ctx, credentials); err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}

	conn, err := s.q.UpdatePlatformConnection(ctx, storage.UpdatePlatformConnectionParams{
		ID: cid, ProjectID: pid, Platform: platform, Method: method,
		Credentials: credentials, Enabled: enabled,
	})
	if err != nil {
		return nil, ErrConnectionNotFound
	}

	return connInfoFromUpdate(conn), nil
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

// CreateAccountAndConnect creates an account on a platform via browser automation,
// then stores the resulting credentials as a new platform connection.
func (s *Platform) CreateAccountAndConnect(ctx context.Context, projectID, platform, email, username, password string) (*ConnectionInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	// Get the browser adapter for this platform.
	adapter, err := s.registry.Get(registryKey(platform, "browser"))
	if err != nil {
		return nil, ErrBrowserNotConfigured
	}

	browserAdapter, ok := adapter.(*integration.BrowserAdapter)
	if !ok {
		return nil, fmt.Errorf("adapter for %s:browser does not support account creation", platform)
	}

	result, err := browserAdapter.CreateAccount(ctx, email, username, password)
	if err != nil {
		return nil, fmt.Errorf("account creation failed: %w", err)
	}

	if !result.Success {
		return nil, fmt.Errorf("account creation failed: %s", result.Error)
	}

	// Marshal credentials from the response.
	creds, err := json.Marshal(result.Credentials)
	if err != nil {
		return nil, fmt.Errorf("marshaling credentials: %w", err)
	}

	// Store as a browser connection (the account was created via browser).
	conn, err := s.q.CreatePlatformConnection(ctx, storage.CreatePlatformConnectionParams{
		ProjectID: pid, Platform: platform, Method: "browser",
		Credentials: creds, Enabled: true,
	})
	if err != nil {
		return nil, fmt.Errorf("storing connection: %w", err)
	}

	return connInfoFromCreate(conn), nil
}

// GetConnectionForPublish fetches the enabled connection for a platform (prefers API over browser).
// Returns credentials, method, and the registry key for adapter lookup.
func (s *Platform) GetConnectionForPublish(ctx context.Context, projectID, platform string) (*ConnectionWithCreds, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrConnectionNotFound
	}

	conn, err := s.q.GetEnabledConnection(ctx, storage.GetEnabledConnectionParams{
		ProjectID: pid, Platform: platform,
	})
	if err != nil {
		return nil, ErrConnectionNotFound
	}

	return &ConnectionWithCreds{
		Platform:    conn.Platform,
		Method:      conn.Method,
		Credentials: conn.Credentials,
	}, nil
}

// connRow is a common interface for all sqlc-generated platform connection row types.
type connRow interface {
	getID() uuid.UUID
	getProjectID() uuid.UUID
	getPlatform() string
	getMethod() string
	getEnabled() bool
	getCreatedAt() time.Time
	getUpdatedAt() time.Time
}

func makeConnInfo(id uuid.UUID, projectID uuid.UUID, platform, method string, enabled bool, createdAt, updatedAt time.Time) *ConnectionInfo {
	return &ConnectionInfo{
		ID:        id.String(),
		ProjectID: projectID.String(),
		Platform:  platform,
		Method:    method,
		Enabled:   enabled,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func connInfoFromCreate(c storage.CreatePlatformConnectionRow) *ConnectionInfo {
	ci := makeConnInfo(c.ID, c.ProjectID, c.Platform, c.Method, c.Enabled, c.CreatedAt.Time, c.UpdatedAt.Time)
	ci.Target = extractTarget(c.Platform, c.Credentials)
	return ci
}

func connInfoFromList(c storage.ListPlatformConnectionsRow) *ConnectionInfo {
	ci := makeConnInfo(c.ID, c.ProjectID, c.Platform, c.Method, c.Enabled, c.CreatedAt.Time, c.UpdatedAt.Time)
	ci.Target = extractTarget(c.Platform, c.Credentials)
	return ci
}

func connInfoFromGet(c storage.GetPlatformConnectionByIDRow) *ConnectionInfo {
	ci := makeConnInfo(c.ID, c.ProjectID, c.Platform, c.Method, c.Enabled, c.CreatedAt.Time, c.UpdatedAt.Time)
	ci.Target = extractTarget(c.Platform, c.Credentials)
	return ci
}

func connInfoFromUpdate(c storage.UpdatePlatformConnectionRow) *ConnectionInfo {
	ci := makeConnInfo(c.ID, c.ProjectID, c.Platform, c.Method, c.Enabled, c.CreatedAt.Time, c.UpdatedAt.Time)
	ci.Target = extractTarget(c.Platform, c.Credentials)
	return ci
}

// extractTarget extracts a safe, non-secret display string from credential JSON.
func extractTarget(platform string, credentials []byte) string {
	var creds map[string]interface{}
	if err := json.Unmarshal(credentials, &creds); err != nil {
		return ""
	}
	switch platform {
	case "reddit":
		if sub, ok := creds["subreddit"].(string); ok && sub != "" {
			return "r/" + sub
		}
	case "bluesky":
		if id, ok := creds["identifier"].(string); ok && id != "" {
			if !strings.HasPrefix(id, "@") {
				return "@" + id
			}
			return id
		}
	case "hn":
		if user, ok := creds["username"].(string); ok && user != "" {
			return user
		}
	case "gmail":
		if email, ok := creds["email"].(string); ok && email != "" {
			return email
		}
	case "google_groups":
		if group, ok := creds["group_email"].(string); ok && group != "" {
			return group
		}
	}
	return ""
}
