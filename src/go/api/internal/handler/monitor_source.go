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

type MonitorSourceHandler struct {
	svc *service.MonitorSource
}

func NewMonitorSource(svc *service.MonitorSource) *MonitorSourceHandler {
	return &MonitorSourceHandler{svc: svc}
}

// --- Request/Response types ---

type createSourceRequest struct {
	Type    string         `json:"type"`
	Config  map[string]any `json:"config"`
	Enabled *bool          `json:"enabled"`
}

type updateSourceRequest struct {
	Type    string         `json:"type"`
	Config  map[string]any `json:"config"`
	Enabled *bool          `json:"enabled"`
}

type sourceResponse struct {
	ID        string         `json:"id"`
	ProjectID string         `json:"project_id"`
	Type      string         `json:"type"`
	Config    map[string]any `json:"config"`
	Enabled   bool           `json:"enabled"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

// --- Handlers ---

// ListSources handles GET /projects/{id}/sources.
func (h *MonitorSourceHandler) ListSources(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())

	sources, err := h.svc.List(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list monitor sources", err)
		return
	}

	resp := make([]sourceResponse, len(sources))
	for i, s := range sources {
		resp[i] = *sourceInfoToResponse(&s)
	}
	writeJSON(w, http.StatusOK, resp)
}

// CreateSource handles POST /projects/{id}/sources.
func (h *MonitorSourceHandler) CreateSource(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin, storage.ProjectRoleMember) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "member role or higher required"})
		return
	}

	var req createSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.Type == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "type is required"})
		return
	}
	if req.Config == nil {
		req.Config = map[string]any{}
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	src, err := h.svc.Create(r.Context(), projectID, req.Type, req.Config, enabled)
	if err != nil {
		switch err {
		case service.ErrInvalidSourceType:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid type; must be reddit, hn, web, rss, or github"})
		default:
			writeError(w, http.StatusInternalServerError, "failed to create monitor source", err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, sourceInfoToResponse(src))
}

// GetSource handles GET /projects/{id}/sources/{sourceID}.
func (h *MonitorSourceHandler) GetSource(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	sourceID := chi.URLParam(r, "sourceID")

	src, err := h.svc.Get(r.Context(), projectID, sourceID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, messageResponse{Message: "monitor source not found"})
		return
	}

	writeJSON(w, http.StatusOK, sourceInfoToResponse(src))
}

// UpdateSource handles PUT /projects/{id}/sources/{sourceID}.
func (h *MonitorSourceHandler) UpdateSource(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())
	sourceID := chi.URLParam(r, "sourceID")

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin, storage.ProjectRoleMember) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "member role or higher required"})
		return
	}

	var req updateSourceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.Type == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "type is required"})
		return
	}
	if req.Config == nil {
		req.Config = map[string]any{}
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	src, err := h.svc.Update(r.Context(), projectID, sourceID, req.Type, req.Config, enabled)
	if err != nil {
		switch err {
		case service.ErrSourceNotFound:
			writeJSON(w, http.StatusNotFound, messageResponse{Message: "monitor source not found"})
		case service.ErrInvalidSourceType:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid type"})
		default:
			writeError(w, http.StatusInternalServerError, "failed to update monitor source", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, sourceInfoToResponse(src))
}

// DeleteSource handles DELETE /projects/{id}/sources/{sourceID}.
func (h *MonitorSourceHandler) DeleteSource(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())
	sourceID := chi.URLParam(r, "sourceID")

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin, storage.ProjectRoleMember) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "member role or higher required"})
		return
	}

	err := h.svc.Delete(r.Context(), projectID, sourceID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, messageResponse{Message: "monitor source not found"})
		return
	}

	writeJSON(w, http.StatusOK, messageResponse{Message: "monitor source deleted"})
}

// --- Helpers ---

func sourceInfoToResponse(s *service.SourceInfo) *sourceResponse {
	return &sourceResponse{
		ID: s.ID, ProjectID: s.ProjectID, Type: s.Type,
		Config: s.Config, Enabled: s.Enabled,
		CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt,
	}
}
