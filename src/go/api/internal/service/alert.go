package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

var (
	ErrAlertNotFound     = errors.New("alert not found")
	ErrInvalidStatus     = errors.New("invalid alert status")
	ErrInvalidSeverity   = errors.New("invalid alert severity")
	ErrInvalidSourceType = errors.New("invalid source type")
)

type AlertInfo struct {
	ID                   string
	ProjectID            string
	SourceType           string
	SourceID             string
	Title                string
	Content              string
	URL                  string
	MatchedKeywords      []string
	Severity             string
	Status               string
	Tags                 []string
	ClassificationReason string
	MonitorSourceID      string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type IngestRequest struct {
	ProjectID            string
	SourceType           string
	SourceID             string
	Title                string
	Content              string
	URL                  string
	Severity             string
	Tags                 []string
	ClassificationReason string
	MonitorSourceID      string
}

type AlertListResult struct {
	Alerts []AlertInfo
	Total  int64
}

type Alert struct {
	q *storage.Queries
}

func NewAlert(q *storage.Queries) *Alert {
	return &Alert{q: q}
}

// Ingest processes an incoming webhook alert: matches keywords, deduplicates
// via upsert, and persists. Returns the alert and whether it was newly created.
func (s *Alert) Ingest(ctx context.Context, req IngestRequest) (*AlertInfo, bool, error) {
	projectID, err := uuid.Parse(req.ProjectID)
	if err != nil {
		return nil, false, fmt.Errorf("invalid project_id: %w", err)
	}

	sourceType, err := parseMonitorType(req.SourceType)
	if err != nil {
		return nil, false, ErrInvalidSourceType
	}

	if req.SourceID == "" {
		return nil, false, fmt.Errorf("source_id is required")
	}

	severity := storage.AlertSeverityLow
	if req.Severity != "" {
		severity, err = parseAlertSeverity(req.Severity)
		if err != nil {
			return nil, false, ErrInvalidSeverity
		}
	}

	matchedKeywords, err := s.matchKeywords(ctx, projectID, req.Title, req.Content)
	if err != nil {
		return nil, false, fmt.Errorf("matching keywords: %w", err)
	}

	tags := req.Tags
	if tags == nil {
		tags = []string{}
	}

	var monitorSourceID pgtype.UUID
	if req.MonitorSourceID != "" {
		parsed, parseErr := uuid.Parse(req.MonitorSourceID)
		if parseErr == nil {
			monitorSourceID = pgtype.UUID{Bytes: parsed, Valid: true}
		}
	}

	alert, err := s.q.UpsertAlert(ctx, storage.UpsertAlertParams{
		ProjectID:            projectID,
		SourceType:           sourceType,
		SourceID:             req.SourceID,
		Title:                req.Title,
		Content:              req.Content,
		Url:                  req.URL,
		MatchedKeywords:      matchedKeywords,
		Severity:             severity,
		Tags:                 tags,
		ClassificationReason: req.ClassificationReason,
		MonitorSourceID:      monitorSourceID,
	})
	if err != nil {
		return nil, false, fmt.Errorf("upserting alert: %w", err)
	}

	// created_at == updated_at means it was a new insert (trigger only fires on UPDATE).
	isNew := alert.CreatedAt.Time.Equal(alert.UpdatedAt.Time)

	return storageAlertToInfo(alert), isNew, nil
}

// List returns a paginated list of alerts, optionally filtered by status or source type.
func (s *Alert) List(ctx context.Context, projectID, status, sourceType string, limit, offset int32) (*AlertListResult, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	var alerts []storage.Alert
	var total int64

	switch {
	case status != "":
		as, err := parseAlertStatus(status)
		if err != nil {
			return nil, ErrInvalidStatus
		}
		alerts, err = s.q.ListAlertsByStatus(ctx, storage.ListAlertsByStatusParams{
			ProjectID: pid, Status: as, Limit: limit, Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("listing alerts: %w", err)
		}
		total, err = s.q.CountAlertsByStatus(ctx, storage.CountAlertsByStatusParams{
			ProjectID: pid, Status: as,
		})
		if err != nil {
			return nil, fmt.Errorf("counting alerts: %w", err)
		}

	case sourceType != "":
		mt, err := parseMonitorType(sourceType)
		if err != nil {
			return nil, ErrInvalidSourceType
		}
		alerts, err = s.q.ListAlertsBySourceType(ctx, storage.ListAlertsBySourceTypeParams{
			ProjectID: pid, SourceType: mt, Limit: limit, Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("listing alerts: %w", err)
		}
		total, err = s.q.CountAlertsBySourceType(ctx, storage.CountAlertsBySourceTypeParams{
			ProjectID: pid, SourceType: mt,
		})
		if err != nil {
			return nil, fmt.Errorf("counting alerts: %w", err)
		}

	default:
		alerts, err = s.q.ListAlerts(ctx, storage.ListAlertsParams{
			ProjectID: pid, Limit: limit, Offset: offset,
		})
		if err != nil {
			return nil, fmt.Errorf("listing alerts: %w", err)
		}
		total, err = s.q.CountAlerts(ctx, pid)
		if err != nil {
			return nil, fmt.Errorf("counting alerts: %w", err)
		}
	}

	result := make([]AlertInfo, len(alerts))
	for i, a := range alerts {
		result[i] = *storageAlertToInfo(a)
	}
	return &AlertListResult{Alerts: result, Total: total}, nil
}

// Get returns a single alert by ID within a project.
func (s *Alert) Get(ctx context.Context, projectID, alertID string) (*AlertInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrAlertNotFound
	}
	aid, err := uuid.Parse(alertID)
	if err != nil {
		return nil, ErrAlertNotFound
	}

	alert, err := s.q.GetAlertByID(ctx, storage.GetAlertByIDParams{ID: aid, ProjectID: pid})
	if err != nil {
		return nil, ErrAlertNotFound
	}

	return storageAlertToInfo(alert), nil
}

// UpdateStatus changes an alert's triage status.
func (s *Alert) UpdateStatus(ctx context.Context, projectID, alertID, newStatus string) (*AlertInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrAlertNotFound
	}
	aid, err := uuid.Parse(alertID)
	if err != nil {
		return nil, ErrAlertNotFound
	}

	st, err := parseAlertStatus(newStatus)
	if err != nil {
		return nil, ErrInvalidStatus
	}

	alert, err := s.q.UpdateAlertStatus(ctx, storage.UpdateAlertStatusParams{
		ID: aid, ProjectID: pid, Status: st,
	})
	if err != nil {
		return nil, ErrAlertNotFound
	}

	return storageAlertToInfo(alert), nil
}

// matchKeywords fetches all keywords for the project and checks them against
// the provided title and content.
func (s *Alert) matchKeywords(ctx context.Context, projectID uuid.UUID, title, content string) ([]string, error) {
	keywords, err := s.q.ListKeywords(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("fetching keywords: %w", err)
	}

	if len(keywords) == 0 {
		return []string{}, nil
	}

	text := strings.ToLower(title + " " + content)
	var matched []string

	for _, kw := range keywords {
		switch kw.MatchType {
		case storage.KeywordMatchTypeExact:
			if strings.EqualFold(kw.Keyword, title) || strings.EqualFold(kw.Keyword, content) {
				matched = append(matched, kw.Keyword)
			}
		case storage.KeywordMatchTypeSubstring:
			if strings.Contains(text, strings.ToLower(kw.Keyword)) {
				matched = append(matched, kw.Keyword)
			}
		case storage.KeywordMatchTypeRegex:
			re, err := regexp.Compile("(?i)" + kw.Keyword)
			if err != nil {
				continue
			}
			if re.MatchString(title) || re.MatchString(content) {
				matched = append(matched, kw.Keyword)
			}
		}
	}

	if matched == nil {
		matched = []string{}
	}
	return matched, nil
}

// --- Helpers ---

func parseMonitorType(s string) (storage.MonitorType, error) {
	switch storage.MonitorType(s) {
	case storage.MonitorTypeReddit, storage.MonitorTypeHn, storage.MonitorTypeWeb,
		storage.MonitorTypeRss, storage.MonitorTypeGithub, storage.MonitorTypeBluesky:
		return storage.MonitorType(s), nil
	default:
		return "", fmt.Errorf("invalid monitor type: %s", s)
	}
}

func parseAlertSeverity(s string) (storage.AlertSeverity, error) {
	switch storage.AlertSeverity(s) {
	case storage.AlertSeverityLow, storage.AlertSeverityMedium,
		storage.AlertSeverityHigh, storage.AlertSeverityCritical:
		return storage.AlertSeverity(s), nil
	default:
		return "", fmt.Errorf("invalid alert severity: %s", s)
	}
}

func parseAlertStatus(s string) (storage.AlertStatus, error) {
	switch storage.AlertStatus(s) {
	case storage.AlertStatusNew, storage.AlertStatusSeen, storage.AlertStatusTriaged,
		storage.AlertStatusDismissed, storage.AlertStatusResponded:
		return storage.AlertStatus(s), nil
	default:
		return "", fmt.Errorf("invalid alert status: %s", s)
	}
}

func storageAlertToInfo(a storage.Alert) *AlertInfo {
	monitorSourceID := ""
	if a.MonitorSourceID.Valid {
		monitorSourceID = uuid.UUID(a.MonitorSourceID.Bytes).String()
	}
	tags := a.Tags
	if tags == nil {
		tags = []string{}
	}
	return &AlertInfo{
		ID:                   a.ID.String(),
		ProjectID:            a.ProjectID.String(),
		SourceType:           string(a.SourceType),
		SourceID:             a.SourceID,
		Title:                a.Title,
		Content:              a.Content,
		URL:                  a.Url,
		MatchedKeywords:      a.MatchedKeywords,
		Severity:             string(a.Severity),
		Status:               string(a.Status),
		Tags:                 tags,
		ClassificationReason: a.ClassificationReason,
		MonitorSourceID:      monitorSourceID,
		CreatedAt:            a.CreatedAt.Time,
		UpdatedAt:            a.UpdatedAt.Time,
	}
}
