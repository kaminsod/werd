package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

var (
	ErrRuleNotFound           = errors.New("notification rule not found")
	ErrInvalidDestination     = errors.New("invalid notification destination")
	ErrInvalidNotifSourceType = errors.New("invalid notification source type")
	ErrMissingNtfyTopic       = errors.New("ntfy destination requires 'topic' in config")
	ErrMissingWebhookURL      = errors.New("webhook destination requires 'url' in config")
)

type RuleInfo struct {
	ID          string
	ProjectID   string
	SourceType  string
	MinSeverity string
	Destination string
	Config      map[string]any
	Enabled     bool
	CreatedAt   time.Time
}

type Notification struct {
	q       *storage.Queries
	ntfyURL string
	httpCli *http.Client
}

func NewNotification(q *storage.Queries, ntfyURL string) *Notification {
	return &Notification{
		q:       q,
		ntfyURL: ntfyURL,
		httpCli: &http.Client{Timeout: 10 * time.Second},
	}
}

// --- Rule CRUD ---

func (s *Notification) CreateRule(ctx context.Context, projectID, sourceType, minSeverity, destination string, config map[string]any, enabled bool) (*RuleInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	st, err := parseNotifSourceType(sourceType)
	if err != nil {
		return nil, ErrInvalidNotifSourceType
	}

	sev, err := parseAlertSeverity(minSeverity)
	if err != nil {
		return nil, ErrInvalidSeverity
	}

	dest, err := parseNotifDestination(destination)
	if err != nil {
		return nil, ErrInvalidDestination
	}

	if err := validateDestinationConfig(dest, config); err != nil {
		return nil, err
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}

	rule, err := s.q.CreateNotificationRule(ctx, storage.CreateNotificationRuleParams{
		ProjectID:   pid,
		SourceType:  st,
		MinSeverity: sev,
		Destination: dest,
		Config:      configJSON,
		Enabled:     enabled,
	})
	if err != nil {
		return nil, fmt.Errorf("creating rule: %w", err)
	}

	return storageRuleToInfo(rule), nil
}

func (s *Notification) ListRules(ctx context.Context, projectID string) ([]RuleInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrProjectNotFound
	}

	rules, err := s.q.ListNotificationRules(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("listing rules: %w", err)
	}

	result := make([]RuleInfo, len(rules))
	for i, r := range rules {
		result[i] = *storageRuleToInfo(r)
	}
	return result, nil
}

func (s *Notification) GetRule(ctx context.Context, projectID, ruleID string) (*RuleInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrRuleNotFound
	}
	rid, err := uuid.Parse(ruleID)
	if err != nil {
		return nil, ErrRuleNotFound
	}

	rule, err := s.q.GetNotificationRuleByID(ctx, storage.GetNotificationRuleByIDParams{
		ID: rid, ProjectID: pid,
	})
	if err != nil {
		return nil, ErrRuleNotFound
	}

	return storageRuleToInfo(rule), nil
}

func (s *Notification) UpdateRule(ctx context.Context, projectID, ruleID, sourceType, minSeverity, destination string, config map[string]any, enabled bool) (*RuleInfo, error) {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return nil, ErrRuleNotFound
	}
	rid, err := uuid.Parse(ruleID)
	if err != nil {
		return nil, ErrRuleNotFound
	}

	st, err := parseNotifSourceType(sourceType)
	if err != nil {
		return nil, ErrInvalidNotifSourceType
	}

	sev, err := parseAlertSeverity(minSeverity)
	if err != nil {
		return nil, ErrInvalidSeverity
	}

	dest, err := parseNotifDestination(destination)
	if err != nil {
		return nil, ErrInvalidDestination
	}

	if err := validateDestinationConfig(dest, config); err != nil {
		return nil, err
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}

	rule, err := s.q.UpdateNotificationRule(ctx, storage.UpdateNotificationRuleParams{
		ID: rid, ProjectID: pid,
		SourceType: st, MinSeverity: sev, Destination: dest,
		Config: configJSON, Enabled: enabled,
	})
	if err != nil {
		return nil, ErrRuleNotFound
	}

	return storageRuleToInfo(rule), nil
}

func (s *Notification) DeleteRule(ctx context.Context, projectID, ruleID string) error {
	pid, err := uuid.Parse(projectID)
	if err != nil {
		return ErrRuleNotFound
	}
	rid, err := uuid.Parse(ruleID)
	if err != nil {
		return ErrRuleNotFound
	}

	_, err = s.q.GetNotificationRuleByID(ctx, storage.GetNotificationRuleByIDParams{
		ID: rid, ProjectID: pid,
	})
	if err != nil {
		return ErrRuleNotFound
	}

	return s.q.DeleteNotificationRule(ctx, storage.DeleteNotificationRuleParams{
		ID: rid, ProjectID: pid,
	})
}

// --- Routing Engine ---

// RouteAlert evaluates all enabled rules for the alert's project and dispatches
// notifications to matching destinations. Designed to run in a goroutine —
// logs errors instead of returning them.
func (s *Notification) RouteAlert(ctx context.Context, alert *AlertInfo) {
	pid, err := uuid.Parse(alert.ProjectID)
	if err != nil {
		log.Printf("notification: invalid project_id %s: %v", alert.ProjectID, err)
		return
	}

	rules, err := s.q.ListEnabledRulesForProject(ctx, pid)
	if err != nil {
		log.Printf("notification: failed to fetch rules for project %s: %v", alert.ProjectID, err)
		return
	}

	if len(rules) == 0 {
		return
	}

	alertSourceType := storage.NotificationSourceType(alert.SourceType)
	alertSeverity := storage.AlertSeverity(alert.Severity)

	for _, rule := range rules {
		if rule.SourceType != storage.NotificationSourceTypeAll && rule.SourceType != alertSourceType {
			continue
		}

		if !severityGTE(alertSeverity, rule.MinSeverity) {
			continue
		}

		var cfg map[string]any
		if len(rule.Config) > 0 {
			json.Unmarshal(rule.Config, &cfg)
		}

		switch rule.Destination {
		case storage.NotificationDestinationNtfy:
			s.dispatchNtfy(ctx, alert, cfg)
		case storage.NotificationDestinationWebhook:
			s.dispatchWebhook(ctx, alert, cfg)
		case storage.NotificationDestinationEmail:
			log.Printf("notification: email not implemented, skipping rule %s", rule.ID)
		}
	}
}

// --- Dispatchers ---

func (s *Notification) dispatchNtfy(ctx context.Context, alert *AlertInfo, cfg map[string]any) {
	topic, _ := cfg["topic"].(string)
	if topic == "" {
		log.Printf("notification: ntfy rule missing topic for alert %s", alert.ID)
		return
	}

	if s.ntfyURL == "" {
		log.Printf("notification: WERD_NTFY_URL not configured, skipping for alert %s", alert.ID)
		return
	}

	payload := map[string]any{
		"topic":    topic,
		"title":    fmt.Sprintf("[%s] %s", alert.Severity, alert.Title),
		"message":  truncate(alert.Content, 500),
		"priority": severityToNtfyPriority(alert.Severity),
		"tags":     []string{alert.SourceType, alert.Severity},
	}
	if alert.URL != "" {
		payload["click"] = alert.URL
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("notification: failed to marshal ntfy payload for alert %s: %v", alert.ID, err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.ntfyURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("notification: failed to create ntfy request for alert %s: %v", alert.ID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpCli.Do(req)
	if err != nil {
		log.Printf("notification: ntfy request failed for alert %s: %v", alert.ID, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("notification: ntfy returned status %d for alert %s", resp.StatusCode, alert.ID)
	}
}

func (s *Notification) dispatchWebhook(ctx context.Context, alert *AlertInfo, cfg map[string]any) {
	url, _ := cfg["url"].(string)
	if url == "" {
		log.Printf("notification: webhook rule missing url for alert %s", alert.ID)
		return
	}

	payload := map[string]any{
		"event":            "alert.new",
		"alert_id":         alert.ID,
		"project_id":       alert.ProjectID,
		"source_type":      alert.SourceType,
		"source_id":        alert.SourceID,
		"title":            alert.Title,
		"content":          alert.Content,
		"url":              alert.URL,
		"matched_keywords": alert.MatchedKeywords,
		"severity":         alert.Severity,
		"status":           alert.Status,
		"created_at":       alert.CreatedAt,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("notification: failed to marshal webhook payload for alert %s: %v", alert.ID, err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		log.Printf("notification: failed to create webhook request for alert %s: %v", alert.ID, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	if secret, _ := cfg["secret"].(string); secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		req.Header.Set("X-Werd-Signature", hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := s.httpCli.Do(req)
	if err != nil {
		log.Printf("notification: webhook failed for alert %s to %s: %v", alert.ID, url, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		log.Printf("notification: webhook returned status %d for alert %s to %s", resp.StatusCode, alert.ID, url)
	}
}

// --- Helpers ---

func severityRank(s storage.AlertSeverity) int {
	switch s {
	case storage.AlertSeverityLow:
		return 0
	case storage.AlertSeverityMedium:
		return 1
	case storage.AlertSeverityHigh:
		return 2
	case storage.AlertSeverityCritical:
		return 3
	default:
		return 0
	}
}

func severityGTE(alertSev, threshold storage.AlertSeverity) bool {
	return severityRank(alertSev) >= severityRank(threshold)
}

func severityToNtfyPriority(severity string) int {
	switch storage.AlertSeverity(severity) {
	case storage.AlertSeverityCritical:
		return 5
	case storage.AlertSeverityHigh:
		return 4
	case storage.AlertSeverityMedium:
		return 3
	case storage.AlertSeverityLow:
		return 2
	default:
		return 3
	}
}

func parseNotifSourceType(s string) (storage.NotificationSourceType, error) {
	switch storage.NotificationSourceType(s) {
	case storage.NotificationSourceTypeReddit, storage.NotificationSourceTypeHn,
		storage.NotificationSourceTypeWeb, storage.NotificationSourceTypeRss,
		storage.NotificationSourceTypeGithub, storage.NotificationSourceTypeAll:
		return storage.NotificationSourceType(s), nil
	default:
		return "", fmt.Errorf("invalid notification source type: %s", s)
	}
}

func parseNotifDestination(s string) (storage.NotificationDestination, error) {
	switch storage.NotificationDestination(s) {
	case storage.NotificationDestinationNtfy, storage.NotificationDestinationEmail,
		storage.NotificationDestinationWebhook:
		return storage.NotificationDestination(s), nil
	default:
		return "", fmt.Errorf("invalid notification destination: %s", s)
	}
}

func validateDestinationConfig(dest storage.NotificationDestination, config map[string]any) error {
	if config == nil {
		config = map[string]any{}
	}
	switch dest {
	case storage.NotificationDestinationNtfy:
		if topic, _ := config["topic"].(string); topic == "" {
			return ErrMissingNtfyTopic
		}
	case storage.NotificationDestinationWebhook:
		if url, _ := config["url"].(string); url == "" {
			return ErrMissingWebhookURL
		}
	}
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func storageRuleToInfo(r storage.NotificationRule) *RuleInfo {
	var cfg map[string]any
	if len(r.Config) > 0 {
		json.Unmarshal(r.Config, &cfg)
	}
	if cfg == nil {
		cfg = map[string]any{}
	}
	return &RuleInfo{
		ID:          r.ID.String(),
		ProjectID:   r.ProjectID.String(),
		SourceType:  string(r.SourceType),
		MinSeverity: string(r.MinSeverity),
		Destination: string(r.Destination),
		Config:      cfg,
		Enabled:     r.Enabled,
		CreatedAt:   r.CreatedAt.Time,
	}
}
