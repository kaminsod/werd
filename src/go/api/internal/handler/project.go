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

type ProjectHandler struct {
	svc *service.Project
}

func NewProject(svc *service.Project) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

// --- Request/Response types ---

type createProjectRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type updateProjectRequest struct {
	Name     string         `json:"name"`
	Slug     string         `json:"slug"`
	Settings map[string]any `json:"settings"`
}

type addMemberRequest struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type updateMemberRoleRequest struct {
	Role string `json:"role"`
}

type projectResponse struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Slug      string         `json:"slug"`
	Settings  map[string]any `json:"settings"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type memberResponse struct {
	UserID    string    `json:"user_id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

// --- Role check helper ---

func requireRole(role string, allowed ...storage.ProjectRole) bool {
	for _, a := range allowed {
		if role == string(a) {
			return true
		}
	}
	return false
}

// --- Handlers ---

// CreateProject handles POST /projects.
func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var req createProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.Name == "" || req.Slug == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "name and slug are required"})
		return
	}

	proj, err := h.svc.Create(r.Context(), userID, req.Name, req.Slug)
	if err != nil {
		switch err {
		case service.ErrSlugTaken:
			writeJSON(w, http.StatusConflict, messageResponse{Message: "slug already taken"})
		case service.ErrInvalidSlug:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, messageResponse{Message: "failed to create project"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, projectToResponse(proj))
}

// ListProjects handles GET /projects.
func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	projects, err := h.svc.List(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, messageResponse{Message: "failed to list projects"})
		return
	}

	resp := make([]projectResponse, len(projects))
	for i, p := range projects {
		resp[i] = *projectToResponse(&p)
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetProject handles GET /projects/{id}.
func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())

	proj, err := h.svc.Get(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, messageResponse{Message: "project not found"})
		return
	}

	writeJSON(w, http.StatusOK, projectToResponse(proj))
}

// UpdateProject handles PUT /projects/{id}.
func (h *ProjectHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "admin or owner role required"})
		return
	}

	var req updateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.Name == "" || req.Slug == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "name and slug are required"})
		return
	}
	if req.Settings == nil {
		req.Settings = map[string]any{}
	}

	proj, err := h.svc.Update(r.Context(), projectID, req.Name, req.Slug, req.Settings)
	if err != nil {
		switch err {
		case service.ErrSlugTaken:
			writeJSON(w, http.StatusConflict, messageResponse{Message: "slug already taken"})
		case service.ErrInvalidSlug:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: err.Error()})
		default:
			writeJSON(w, http.StatusInternalServerError, messageResponse{Message: "failed to update project"})
		}
		return
	}

	writeJSON(w, http.StatusOK, projectToResponse(proj))
}

// DeleteProject handles DELETE /projects/{id}.
func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())

	if !requireRole(role, storage.ProjectRoleOwner) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "owner role required"})
		return
	}

	if err := h.svc.Delete(r.Context(), projectID); err != nil {
		writeJSON(w, http.StatusInternalServerError, messageResponse{Message: "failed to delete project"})
		return
	}

	writeJSON(w, http.StatusOK, messageResponse{Message: "project deleted"})
}

// ListMembers handles GET /projects/{id}/members.
func (h *ProjectHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())

	members, err := h.svc.ListMembers(r.Context(), projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, messageResponse{Message: "failed to list members"})
		return
	}

	resp := make([]memberResponse, len(members))
	for i, m := range members {
		resp[i] = memberResponse{
			UserID:    m.UserID,
			Email:     m.Email,
			Name:      m.Name,
			Role:      m.Role,
			CreatedAt: m.CreatedAt,
		}
	}
	writeJSON(w, http.StatusOK, resp)
}

// AddMember handles POST /projects/{id}/members.
func (h *ProjectHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "admin or owner role required"})
		return
	}

	var req addMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.UserID == "" || req.Role == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "user_id and role are required"})
		return
	}

	member, err := h.svc.AddMember(r.Context(), projectID, req.UserID, req.Role)
	if err != nil {
		switch err {
		case service.ErrAlreadyMember:
			writeJSON(w, http.StatusConflict, messageResponse{Message: "user is already a member"})
		case service.ErrUserNotFound:
			writeJSON(w, http.StatusNotFound, messageResponse{Message: "user not found"})
		case service.ErrInsufficientRole:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "cannot add member with owner role"})
		default:
			writeJSON(w, http.StatusInternalServerError, messageResponse{Message: "failed to add member"})
		}
		return
	}

	writeJSON(w, http.StatusCreated, memberResponse{
		UserID:    member.UserID,
		Email:     member.Email,
		Name:      member.Name,
		Role:      member.Role,
		CreatedAt: member.CreatedAt,
	})
}

// UpdateMemberRole handles PUT /projects/{id}/members/{userID}.
func (h *ProjectHandler) UpdateMemberRole(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	callerRole := middleware.ProjectRoleFromContext(r.Context())
	targetUserID := chi.URLParam(r, "userID")

	if !requireRole(callerRole, storage.ProjectRoleOwner, storage.ProjectRoleAdmin) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "admin or owner role required"})
		return
	}

	var req updateMemberRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.Role == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "role is required"})
		return
	}

	err := h.svc.UpdateMemberRole(r.Context(), projectID, callerRole, targetUserID, req.Role)
	if err != nil {
		switch err {
		case service.ErrNotProjectMember:
			writeJSON(w, http.StatusNotFound, messageResponse{Message: "member not found"})
		case service.ErrCannotDemoteOwner:
			writeJSON(w, http.StatusForbidden, messageResponse{Message: "cannot change the owner's role"})
		case service.ErrInsufficientRole:
			writeJSON(w, http.StatusForbidden, messageResponse{Message: "insufficient permissions"})
		default:
			writeJSON(w, http.StatusInternalServerError, messageResponse{Message: "failed to update member role"})
		}
		return
	}

	writeJSON(w, http.StatusOK, messageResponse{Message: "role updated"})
}

// RemoveMember handles DELETE /projects/{id}/members/{userID}.
func (h *ProjectHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	callerID := middleware.UserIDFromContext(r.Context())
	callerRole := middleware.ProjectRoleFromContext(r.Context())
	targetUserID := chi.URLParam(r, "userID")

	isSelfLeave := callerID == targetUserID
	if !isSelfLeave && !requireRole(callerRole, storage.ProjectRoleOwner, storage.ProjectRoleAdmin) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "admin or owner role required"})
		return
	}

	err := h.svc.RemoveMember(r.Context(), projectID, callerID, callerRole, targetUserID)
	if err != nil {
		switch err {
		case service.ErrNotProjectMember:
			writeJSON(w, http.StatusNotFound, messageResponse{Message: "member not found"})
		case service.ErrCannotDemoteOwner:
			writeJSON(w, http.StatusForbidden, messageResponse{Message: "cannot remove the project owner"})
		case service.ErrCannotLeaveAsOwner:
			writeJSON(w, http.StatusForbidden, messageResponse{Message: "owner cannot leave; transfer ownership first"})
		case service.ErrInsufficientRole:
			writeJSON(w, http.StatusForbidden, messageResponse{Message: "insufficient permissions"})
		default:
			writeJSON(w, http.StatusInternalServerError, messageResponse{Message: "failed to remove member"})
		}
		return
	}

	writeJSON(w, http.StatusOK, messageResponse{Message: "member removed"})
}

// --- Helpers ---

func projectToResponse(p *service.ProjectInfo) *projectResponse {
	return &projectResponse{
		ID:        p.ID,
		Name:      p.Name,
		Slug:      p.Slug,
		Settings:  p.Settings,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}
