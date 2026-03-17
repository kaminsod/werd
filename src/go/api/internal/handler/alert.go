package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/werd-platform/werd/src/go/api/internal/middleware"
	"github.com/werd-platform/werd/src/go/api/internal/service"
	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

type AlertHandler struct {
	alertSvc   *service.Alert
	keywordSvc *service.Keyword
	notifSvc   *service.Notification
}

func NewAlert(alertSvc *service.Alert, keywordSvc *service.Keyword, notifSvc *service.Notification) *AlertHandler {
	return &AlertHandler{alertSvc: alertSvc, keywordSvc: keywordSvc, notifSvc: notifSvc}
}

// --- Request/Response types ---

type ingestRequest struct {
	ProjectID            string   `json:"project_id"`
	SourceType           string   `json:"source_type"`
	SourceID             string   `json:"source_id"`
	Title                string   `json:"title"`
	Content              string   `json:"content"`
	URL                  string   `json:"url"`
	Severity             string   `json:"severity"`
	Tags                 []string `json:"tags"`
	ClassificationReason string   `json:"classification_reason"`
	MonitorSourceID      string   `json:"monitor_source_id"`
}

type alertResponse struct {
	ID                   string    `json:"id"`
	ProjectID            string    `json:"project_id"`
	SourceType           string    `json:"source_type"`
	SourceID             string    `json:"source_id"`
	Title                string    `json:"title"`
	Content              string    `json:"content"`
	URL                  string    `json:"url"`
	MatchedKeywords      []string  `json:"matched_keywords"`
	Severity             string    `json:"severity"`
	Status               string    `json:"status"`
	Tags                 []string  `json:"tags"`
	ClassificationReason string    `json:"classification_reason"`
	MonitorSourceID      string    `json:"monitor_source_id,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type alertListResponse struct {
	Alerts []alertResponse `json:"alerts"`
	Total  int64           `json:"total"`
}

type updateAlertStatusRequest struct {
	Status string `json:"status"`
}

type createKeywordRequest struct {
	Keyword   string `json:"keyword"`
	MatchType string `json:"match_type"`
}

type keywordResponse struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Keyword   string    `json:"keyword"`
	MatchType string    `json:"match_type"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Webhook Ingestion ---

// IngestWebhook handles POST /webhooks/ingest.
func (h *AlertHandler) IngestWebhook(w http.ResponseWriter, r *http.Request) {
	var req ingestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.ProjectID == "" || req.SourceType == "" || req.SourceID == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "project_id, source_type, and source_id are required"})
		return
	}

	alert, isNew, err := h.alertSvc.Ingest(r.Context(), service.IngestRequest{
		ProjectID:            req.ProjectID,
		SourceType:           req.SourceType,
		SourceID:             req.SourceID,
		Title:                req.Title,
		Content:              req.Content,
		URL:                  req.URL,
		Severity:             req.Severity,
		Tags:                 req.Tags,
		ClassificationReason: req.ClassificationReason,
		MonitorSourceID:      req.MonitorSourceID,
	})
	if err != nil {
		switch err {
		case service.ErrInvalidSourceType:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid source_type"})
		case service.ErrInvalidSeverity:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid severity"})
		default:
			writeError(w, http.StatusInternalServerError, "failed to ingest alert", err)
		}
		return
	}

	status := http.StatusOK
	if isNew {
		status = http.StatusCreated
		if h.notifSvc != nil {
			alertCopy := *alert
			go h.notifSvc.RouteAlert(context.Background(), &alertCopy)
		}
	}
	writeJSON(w, status, alertInfoToResponse(alert))
}

// --- Alert Management ---

// ListAlerts handles GET /projects/{id}/alerts.
func (h *AlertHandler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())

	statusFilter := r.URL.Query().Get("status")
	sourceTypeFilter := r.URL.Query().Get("source_type")
	limit := parseIntQuery(r, "limit", 50)
	offset := parseIntQuery(r, "offset", 0)

	result, err := h.alertSvc.List(r.Context(), projectID, statusFilter, sourceTypeFilter, int32(limit), int32(offset))
	if err != nil {
		switch err {
		case service.ErrInvalidStatus:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid status filter"})
		case service.ErrInvalidSourceType:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid source_type filter"})
		default:
			writeError(w, http.StatusInternalServerError, "failed to list alerts", err)
		}
		return
	}

	resp := alertListResponse{
		Total:  result.Total,
		Alerts: make([]alertResponse, len(result.Alerts)),
	}
	for i, a := range result.Alerts {
		resp.Alerts[i] = *alertInfoToResponse(&a)
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetAlert handles GET /projects/{id}/alerts/{alertID}.
func (h *AlertHandler) GetAlert(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	alertID := chi.URLParam(r, "alertID")

	alert, err := h.alertSvc.Get(r.Context(), projectID, alertID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, messageResponse{Message: "alert not found"})
		return
	}

	writeJSON(w, http.StatusOK, alertInfoToResponse(alert))
}

// UpdateAlertStatus handles PUT /projects/{id}/alerts/{alertID}.
func (h *AlertHandler) UpdateAlertStatus(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	alertID := chi.URLParam(r, "alertID")

	var req updateAlertStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.Status == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "status is required"})
		return
	}

	alert, err := h.alertSvc.UpdateStatus(r.Context(), projectID, alertID, req.Status)
	if err != nil {
		switch err {
		case service.ErrAlertNotFound:
			writeJSON(w, http.StatusNotFound, messageResponse{Message: "alert not found"})
		case service.ErrInvalidStatus:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid status"})
		default:
			writeError(w, http.StatusInternalServerError, "failed to update alert", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, alertInfoToResponse(alert))
}

// --- Keyword Management ---

// ListKeywords handles GET /projects/{id}/keywords.
func (h *AlertHandler) ListKeywords(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())

	keywords, err := h.keywordSvc.List(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list keywords", err)
		return
	}

	resp := make([]keywordResponse, len(keywords))
	for i, kw := range keywords {
		resp[i] = *keywordInfoToResponse(&kw)
	}
	writeJSON(w, http.StatusOK, resp)
}

// CreateKeyword handles POST /projects/{id}/keywords.
func (h *AlertHandler) CreateKeyword(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin, storage.ProjectRoleMember) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "member role or higher required"})
		return
	}

	var req createKeywordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.Keyword == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "keyword is required"})
		return
	}
	if req.MatchType == "" {
		req.MatchType = "substring"
	}

	kw, err := h.keywordSvc.Create(r.Context(), projectID, req.Keyword, req.MatchType)
	if err != nil {
		switch err {
		case service.ErrInvalidMatchType:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid match_type; must be exact, substring, or regex"})
		case service.ErrInvalidRegex:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid regex pattern"})
		default:
			writeError(w, http.StatusInternalServerError, "failed to create keyword", err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, keywordInfoToResponse(kw))
}

// DeleteKeyword handles DELETE /projects/{id}/keywords/{kwID}.
func (h *AlertHandler) DeleteKeyword(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())
	kwID := chi.URLParam(r, "kwID")

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin, storage.ProjectRoleMember) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "member role or higher required"})
		return
	}

	err := h.keywordSvc.Delete(r.Context(), projectID, kwID)
	if err != nil {
		switch err {
		case service.ErrKeywordNotFound:
			writeJSON(w, http.StatusNotFound, messageResponse{Message: "keyword not found"})
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete keyword", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, messageResponse{Message: "keyword deleted"})
}

// --- Helpers ---

func alertInfoToResponse(a *service.AlertInfo) *alertResponse {
	return &alertResponse{
		ID:                   a.ID,
		ProjectID:            a.ProjectID,
		SourceType:           a.SourceType,
		SourceID:             a.SourceID,
		Title:                a.Title,
		Content:              a.Content,
		URL:                  a.URL,
		MatchedKeywords:      a.MatchedKeywords,
		Severity:             a.Severity,
		Status:               a.Status,
		Tags:                 a.Tags,
		ClassificationReason: a.ClassificationReason,
		MonitorSourceID:      a.MonitorSourceID,
		CreatedAt:            a.CreatedAt,
		UpdatedAt:            a.UpdatedAt,
	}
}

func keywordInfoToResponse(kw *service.KeywordInfo) *keywordResponse {
	return &keywordResponse{
		ID:        kw.ID,
		ProjectID: kw.ProjectID,
		Keyword:   kw.Keyword,
		MatchType: kw.MatchType,
		CreatedAt: kw.CreatedAt,
	}
}

func parseIntQuery(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return defaultVal
	}
	return v
}
