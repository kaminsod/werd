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

type ProcessingHandler struct {
	svc *service.ProcessingRuleService
}

func NewProcessing(svc *service.ProcessingRuleService) *ProcessingHandler {
	return &ProcessingHandler{svc: svc}
}

// --- Request/Response types ---

type createProcessingRuleRequest struct {
	SourceID string         `json:"source_id"` // empty = project-wide
	Name     string         `json:"name"`
	Phase    string         `json:"phase"`
	RuleType string         `json:"rule_type"`
	Config   map[string]any `json:"config"`
	Priority *int           `json:"priority"`
	Enabled  *bool          `json:"enabled"`
}

type updateProcessingRuleRequest struct {
	SourceID string         `json:"source_id"`
	Name     string         `json:"name"`
	Phase    string         `json:"phase"`
	RuleType string         `json:"rule_type"`
	Config   map[string]any `json:"config"`
	Priority *int           `json:"priority"`
	Enabled  *bool          `json:"enabled"`
}

type processingRuleResponse struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"project_id"`
	SourceID  string         `json:"source_id,omitempty"`
	Name      string         `json:"name"`
	Phase     string         `json:"phase"`
	RuleType  string         `json:"rule_type"`
	Config    map[string]any `json:"config"`
	Priority  int            `json:"priority"`
	Enabled   bool           `json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// --- Handlers ---

// ListProcessingRules handles GET /projects/{id}/processing-rules.
func (h *ProcessingHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())

	rules, err := h.svc.List(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list processing rules", err)
		return
	}

	resp := make([]processingRuleResponse, len(rules))
	for i, r := range rules {
		resp[i] = *processingRuleInfoToResponse(&r)
	}
	writeJSON(w, http.StatusOK, resp)
}

// CreateProcessingRule handles POST /projects/{id}/processing-rules.
func (h *ProcessingHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin, storage.ProjectRoleMember) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "member role or higher required"})
		return
	}

	var req createProcessingRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.Phase == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "phase is required"})
		return
	}
	if req.RuleType == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "rule_type is required"})
		return
	}
	if req.Config == nil {
		req.Config = map[string]any{}
	}
	priority := 0
	if req.Priority != nil {
		priority = *req.Priority
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule, err := h.svc.Create(r.Context(), projectID, req.SourceID, req.Name, req.Phase, req.RuleType, req.Config, priority, enabled)
	if err != nil {
		switch err {
		case service.ErrInvalidPhase:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid phase; must be filter or classify"})
		case service.ErrInvalidRuleType:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid rule_type; must be keyword, regex, or llm"})
		default:
			writeError(w, http.StatusInternalServerError, "failed to create processing rule", err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, processingRuleInfoToResponse(rule))
}

// GetProcessingRule handles GET /projects/{id}/processing-rules/{ruleID}.
func (h *ProcessingHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	ruleID := chi.URLParam(r, "ruleID")

	rule, err := h.svc.Get(r.Context(), projectID, ruleID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, messageResponse{Message: "processing rule not found"})
		return
	}

	writeJSON(w, http.StatusOK, processingRuleInfoToResponse(rule))
}

// UpdateProcessingRule handles PUT /projects/{id}/processing-rules/{ruleID}.
func (h *ProcessingHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())
	ruleID := chi.URLParam(r, "ruleID")

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin, storage.ProjectRoleMember) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "member role or higher required"})
		return
	}

	var req updateProcessingRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.Phase == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "phase is required"})
		return
	}
	if req.RuleType == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "rule_type is required"})
		return
	}
	if req.Config == nil {
		req.Config = map[string]any{}
	}
	priority := 0
	if req.Priority != nil {
		priority = *req.Priority
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule, err := h.svc.Update(r.Context(), projectID, ruleID, req.SourceID, req.Name, req.Phase, req.RuleType, req.Config, priority, enabled)
	if err != nil {
		switch err {
		case service.ErrProcessingRuleNotFound:
			writeJSON(w, http.StatusNotFound, messageResponse{Message: "processing rule not found"})
		case service.ErrInvalidPhase:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid phase"})
		case service.ErrInvalidRuleType:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid rule_type"})
		default:
			writeError(w, http.StatusInternalServerError, "failed to update processing rule", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, processingRuleInfoToResponse(rule))
}

// DeleteProcessingRule handles DELETE /projects/{id}/processing-rules/{ruleID}.
func (h *ProcessingHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())
	ruleID := chi.URLParam(r, "ruleID")

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin, storage.ProjectRoleMember) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "member role or higher required"})
		return
	}

	err := h.svc.Delete(r.Context(), projectID, ruleID)
	if err != nil {
		switch err {
		case service.ErrProcessingRuleNotFound:
			writeJSON(w, http.StatusNotFound, messageResponse{Message: "processing rule not found"})
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete processing rule", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, messageResponse{Message: "processing rule deleted"})
}

// --- Helpers ---

func processingRuleInfoToResponse(r *service.ProcessingRuleInfo) *processingRuleResponse {
	return &processingRuleResponse{
		ID:        r.ID,
		ProjectID: r.ProjectID,
		SourceID:  r.SourceID,
		Name:      r.Name,
		Phase:     r.Phase,
		RuleType:  r.RuleType,
		Config:    r.Config,
		Priority:  r.Priority,
		Enabled:   r.Enabled,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}
