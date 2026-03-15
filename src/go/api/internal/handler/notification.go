package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/werd-platform/werd/src/go/api/internal/middleware"
	"github.com/werd-platform/werd/src/go/api/internal/service"
	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

type NotificationHandler struct {
	notifSvc *service.Notification
}

func NewNotification(notifSvc *service.Notification) *NotificationHandler {
	return &NotificationHandler{notifSvc: notifSvc}
}

// --- Request/Response types ---

type createRuleRequest struct {
	SourceType  string         `json:"source_type"`
	MinSeverity string         `json:"min_severity"`
	Destination string         `json:"destination"`
	Config      map[string]any `json:"config"`
	Enabled     *bool          `json:"enabled"`
}

type updateRuleRequest struct {
	SourceType  string         `json:"source_type"`
	MinSeverity string         `json:"min_severity"`
	Destination string         `json:"destination"`
	Config      map[string]any `json:"config"`
	Enabled     *bool          `json:"enabled"`
}

type ruleResponse struct {
	ID          string         `json:"id"`
	ProjectID   string         `json:"project_id"`
	SourceType  string         `json:"source_type"`
	MinSeverity string         `json:"min_severity"`
	Destination string         `json:"destination"`
	Config      map[string]any `json:"config"`
	Enabled     bool           `json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
}

// --- Handlers ---

// ListRules handles GET /projects/{id}/rules.
func (h *NotificationHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())

	rules, err := h.notifSvc.ListRules(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list notification rules", err)
		return
	}

	resp := make([]ruleResponse, len(rules))
	for i, r := range rules {
		resp[i] = *ruleInfoToResponse(&r)
	}
	writeJSON(w, http.StatusOK, resp)
}

// CreateRule handles POST /projects/{id}/rules.
func (h *NotificationHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin, storage.ProjectRoleMember) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "member role or higher required"})
		return
	}

	var req createRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.Destination == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "destination is required"})
		return
	}
	if req.SourceType == "" {
		req.SourceType = "all"
	}
	if req.MinSeverity == "" {
		req.MinSeverity = "low"
	}
	if req.Config == nil {
		req.Config = map[string]any{}
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule, err := h.notifSvc.CreateRule(r.Context(), projectID, req.SourceType, req.MinSeverity, req.Destination, req.Config, enabled)
	if err != nil {
		switch err {
		case service.ErrInvalidNotifSourceType:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid source_type"})
		case service.ErrInvalidSeverity:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid min_severity"})
		case service.ErrInvalidDestination:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid destination; must be ntfy, webhook, or email"})
		case service.ErrMissingNtfyTopic:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "ntfy destination requires 'topic' in config"})
		case service.ErrMissingWebhookURL:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "webhook destination requires 'url' in config"})
		default:
			writeError(w, http.StatusInternalServerError, "failed to create notification rule", err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, ruleInfoToResponse(rule))
}

// GetRule handles GET /projects/{id}/rules/{ruleID}.
func (h *NotificationHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	ruleID := chi.URLParam(r, "ruleID")

	rule, err := h.notifSvc.GetRule(r.Context(), projectID, ruleID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, messageResponse{Message: "notification rule not found"})
		return
	}

	writeJSON(w, http.StatusOK, ruleInfoToResponse(rule))
}

// UpdateRule handles PUT /projects/{id}/rules/{ruleID}.
func (h *NotificationHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())
	ruleID := chi.URLParam(r, "ruleID")

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin, storage.ProjectRoleMember) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "member role or higher required"})
		return
	}

	var req updateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.Destination == "" || req.SourceType == "" || req.MinSeverity == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "source_type, min_severity, and destination are required"})
		return
	}
	if req.Config == nil {
		req.Config = map[string]any{}
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule, err := h.notifSvc.UpdateRule(r.Context(), projectID, ruleID, req.SourceType, req.MinSeverity, req.Destination, req.Config, enabled)
	if err != nil {
		switch err {
		case service.ErrRuleNotFound:
			writeJSON(w, http.StatusNotFound, messageResponse{Message: "notification rule not found"})
		case service.ErrInvalidNotifSourceType:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid source_type"})
		case service.ErrInvalidSeverity:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid min_severity"})
		case service.ErrInvalidDestination:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid destination"})
		case service.ErrMissingNtfyTopic:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "ntfy destination requires 'topic' in config"})
		case service.ErrMissingWebhookURL:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "webhook destination requires 'url' in config"})
		default:
			writeError(w, http.StatusInternalServerError, "failed to update notification rule", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, ruleInfoToResponse(rule))
}

// DeleteRule handles DELETE /projects/{id}/rules/{ruleID}.
func (h *NotificationHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())
	ruleID := chi.URLParam(r, "ruleID")

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin, storage.ProjectRoleMember) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "member role or higher required"})
		return
	}

	err := h.notifSvc.DeleteRule(r.Context(), projectID, ruleID)
	if err != nil {
		switch err {
		case service.ErrRuleNotFound:
			writeJSON(w, http.StatusNotFound, messageResponse{Message: "notification rule not found"})
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete notification rule", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, messageResponse{Message: "notification rule deleted"})
}

// --- Helpers ---

func ruleInfoToResponse(r *service.RuleInfo) *ruleResponse {
	return &ruleResponse{
		ID:          r.ID,
		ProjectID:   r.ProjectID,
		SourceType:  r.SourceType,
		MinSeverity: r.MinSeverity,
		Destination: r.Destination,
		Config:      r.Config,
		Enabled:     r.Enabled,
		CreatedAt:   r.CreatedAt,
	}
}
