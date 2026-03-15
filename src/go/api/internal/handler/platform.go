package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/werd-platform/werd/src/go/api/internal/middleware"
	"github.com/werd-platform/werd/src/go/api/internal/service"
	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

type PlatformHandler struct {
	platformSvc *service.Platform
	postSvc     *service.Post
}

func NewPlatform(platformSvc *service.Platform, postSvc *service.Post) *PlatformHandler {
	return &PlatformHandler{platformSvc: platformSvc, postSvc: postSvc}
}

// --- Request/Response types ---

type createConnectionRequest struct {
	Platform    string          `json:"platform"`
	Method      string          `json:"method"`
	Credentials json.RawMessage `json:"credentials"`
	Enabled     *bool           `json:"enabled"`
}

type updateConnectionRequest struct {
	Platform    string          `json:"platform"`
	Method      string          `json:"method"`
	Credentials json.RawMessage `json:"credentials"`
	Enabled     *bool           `json:"enabled"`
}

type connectionResponse struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Platform  string    `json:"platform"`
	Method    string    `json:"method"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type createPostRequest struct {
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	URL       string   `json:"url"`
	PostType  string   `json:"post_type"`
	Platforms []string `json:"platforms"`
}

type updatePostRequest struct {
	Title     string   `json:"title"`
	Content   string   `json:"content"`
	URL       string   `json:"url"`
	PostType  string   `json:"post_type"`
	Platforms []string `json:"platforms"`
}

type postResponse struct {
	ID          string     `json:"id"`
	ProjectID   string     `json:"project_id"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	URL         string     `json:"url,omitempty"`
	PostType    string     `json:"post_type"`
	Platforms   []string   `json:"platforms"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type postListResponse struct {
	Posts []postResponse `json:"posts"`
	Total int64          `json:"total"`
}

type publishResponse struct {
	Post    postResponse                    `json:"post"`
	Results []service.PlatformPublishResult `json:"results"`
}

// ============================================================================
// Connection handlers
// ============================================================================

// ListConnections handles GET /projects/{id}/connections.
func (h *PlatformHandler) ListConnections(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())

	conns, err := h.platformSvc.ListConnections(r.Context(), projectID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list connections", err)
		return
	}

	resp := make([]connectionResponse, len(conns))
	for i, c := range conns {
		resp[i] = *connInfoToResponse(&c)
	}
	writeJSON(w, http.StatusOK, resp)
}

// CreateConnection handles POST /projects/{id}/connections.
func (h *PlatformHandler) CreateConnection(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "admin or owner role required"})
		return
	}

	var req createConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.Platform == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "platform is required"})
		return
	}
	if req.Method == "" {
		req.Method = "api"
	}
	if req.Method != "api" && req.Method != "browser" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "method must be 'api' or 'browser'"})
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	conn, err := h.platformSvc.CreateConnection(r.Context(), projectID, req.Platform, req.Method, req.Credentials, enabled)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrBrowserNotConfigured):
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: err.Error()})
		case errors.Is(err, service.ErrUnsupportedPlatform):
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "unsupported platform"})
		default:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusCreated, connInfoToResponse(conn))
}

// GetConnection handles GET /projects/{id}/connections/{connID}.
func (h *PlatformHandler) GetConnection(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	connID := chi.URLParam(r, "connID")

	conn, err := h.platformSvc.GetConnection(r.Context(), projectID, connID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, messageResponse{Message: "connection not found"})
		return
	}

	writeJSON(w, http.StatusOK, connInfoToResponse(conn))
}

// UpdateConnection handles PUT /projects/{id}/connections/{connID}.
func (h *PlatformHandler) UpdateConnection(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())
	connID := chi.URLParam(r, "connID")

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "admin or owner role required"})
		return
	}

	var req updateConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.Platform == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "platform is required"})
		return
	}
	if req.Method == "" {
		req.Method = "api"
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	conn, err := h.platformSvc.UpdateConnection(r.Context(), projectID, connID, req.Platform, req.Method, req.Credentials, enabled)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrConnectionNotFound):
			writeJSON(w, http.StatusNotFound, messageResponse{Message: "connection not found"})
		case errors.Is(err, service.ErrUnsupportedPlatform):
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "unsupported platform"})
		default:
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: err.Error()})
		}
		return
	}

	writeJSON(w, http.StatusOK, connInfoToResponse(conn))
}

// DeleteConnection handles DELETE /projects/{id}/connections/{connID}.
func (h *PlatformHandler) DeleteConnection(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())
	connID := chi.URLParam(r, "connID")

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "admin or owner role required"})
		return
	}

	if err := h.platformSvc.DeleteConnection(r.Context(), projectID, connID); err != nil {
		writeJSON(w, http.StatusNotFound, messageResponse{Message: "connection not found"})
		return
	}

	writeJSON(w, http.StatusOK, messageResponse{Message: "connection deleted"})
}

// ============================================================================
// Post handlers
// ============================================================================

// ListPosts handles GET /projects/{id}/posts.
func (h *PlatformHandler) ListPosts(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	statusFilter := r.URL.Query().Get("status")
	limit := parseIntQuery(r, "limit", 50)
	offset := parseIntQuery(r, "offset", 0)

	result, err := h.postSvc.List(r.Context(), projectID, statusFilter, int32(limit), int32(offset))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list posts", err)
		return
	}

	resp := postListResponse{
		Total: result.Total,
		Posts: make([]postResponse, len(result.Posts)),
	}
	for i, p := range result.Posts {
		resp.Posts[i] = *postInfoToResponse(&p)
	}
	writeJSON(w, http.StatusOK, resp)
}

// CreatePost handles POST /projects/{id}/posts.
func (h *PlatformHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin, storage.ProjectRoleMember) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "member role or higher required"})
		return
	}

	var req createPostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.Content == "" && req.Title == "" && req.URL == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "content, title, or url is required"})
		return
	}
	if len(req.Platforms) == 0 {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "at least one platform is required"})
		return
	}

	post, err := h.postSvc.Create(r.Context(), projectID, req.Title, req.Content, req.URL, req.PostType, req.Platforms)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrUnsupportedPlatform):
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: err.Error()})
		default:
			writeError(w, http.StatusInternalServerError, "failed to create post", err)
		}
		return
	}

	writeJSON(w, http.StatusCreated, postInfoToResponse(post))
}

// GetPost handles GET /projects/{id}/posts/{postID}.
func (h *PlatformHandler) GetPost(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	postID := chi.URLParam(r, "postID")

	post, err := h.postSvc.Get(r.Context(), projectID, postID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, messageResponse{Message: "post not found"})
		return
	}

	writeJSON(w, http.StatusOK, postInfoToResponse(post))
}

// UpdatePost handles PUT /projects/{id}/posts/{postID}.
func (h *PlatformHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())
	postID := chi.URLParam(r, "postID")

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin, storage.ProjectRoleMember) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "member role or higher required"})
		return
	}

	var req updatePostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.Content == "" && req.Title == "" && req.URL == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "content, title, or url is required"})
		return
	}
	if len(req.Platforms) == 0 {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "at least one platform is required"})
		return
	}

	post, err := h.postSvc.Update(r.Context(), projectID, postID, req.Title, req.Content, req.URL, req.PostType, req.Platforms)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPostNotFound):
			writeJSON(w, http.StatusNotFound, messageResponse{Message: "post not found"})
		case errors.Is(err, service.ErrPostNotDraft):
			writeJSON(w, http.StatusConflict, messageResponse{Message: "only draft posts can be edited"})
		case errors.Is(err, service.ErrUnsupportedPlatform):
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: err.Error()})
		default:
			writeError(w, http.StatusInternalServerError, "failed to update post", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, postInfoToResponse(post))
}

// DeletePost handles DELETE /projects/{id}/posts/{postID}.
func (h *PlatformHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())
	postID := chi.URLParam(r, "postID")

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin, storage.ProjectRoleMember) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "member role or higher required"})
		return
	}

	err := h.postSvc.Delete(r.Context(), projectID, postID)
	if err != nil {
		switch err {
		case service.ErrPostNotFound:
			writeJSON(w, http.StatusNotFound, messageResponse{Message: "post not found"})
		case service.ErrPostNotDraft:
			writeJSON(w, http.StatusConflict, messageResponse{Message: "only draft posts can be deleted"})
		default:
			writeError(w, http.StatusInternalServerError, "failed to delete post", err)
		}
		return
	}

	writeJSON(w, http.StatusOK, messageResponse{Message: "post deleted"})
}

// PublishPost handles POST /projects/{id}/posts/{postID}/publish.
func (h *PlatformHandler) PublishPost(w http.ResponseWriter, r *http.Request) {
	projectID := middleware.ProjectIDFromContext(r.Context())
	role := middleware.ProjectRoleFromContext(r.Context())
	postID := chi.URLParam(r, "postID")

	if !requireRole(role, storage.ProjectRoleOwner, storage.ProjectRoleAdmin, storage.ProjectRoleMember) {
		writeJSON(w, http.StatusForbidden, messageResponse{Message: "member role or higher required"})
		return
	}

	results, err := h.postSvc.Publish(r.Context(), projectID, postID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPostNotFound):
			writeJSON(w, http.StatusNotFound, messageResponse{Message: "post not found"})
		case errors.Is(err, service.ErrPostNotDraft):
			writeJSON(w, http.StatusConflict, messageResponse{Message: "only draft posts can be published"})
		case errors.Is(err, service.ErrNoPlatforms):
			writeJSON(w, http.StatusBadRequest, messageResponse{Message: "no platforms specified"})
		case errors.Is(err, service.ErrPublishFailed):
			post, getErr := h.postSvc.Get(r.Context(), projectID, postID)
			if getErr != nil {
				writeError(w, http.StatusInternalServerError, "publish failed", err)
				return
			}
			writeJSON(w, http.StatusMultiStatus, publishResponse{
				Post: *postInfoToResponse(post), Results: results,
			})
		default:
			writeError(w, http.StatusInternalServerError, "publish failed", err)
		}
		return
	}

	post, err := h.postSvc.Get(r.Context(), projectID, postID)
	if err != nil {
		writeJSON(w, http.StatusOK, messageResponse{Message: "post published"})
		return
	}

	writeJSON(w, http.StatusOK, publishResponse{
		Post: *postInfoToResponse(post), Results: results,
	})
}

// --- Helpers ---

func connInfoToResponse(c *service.ConnectionInfo) *connectionResponse {
	return &connectionResponse{
		ID: c.ID, ProjectID: c.ProjectID, Platform: c.Platform, Method: c.Method,
		Enabled: c.Enabled, CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
}

func postInfoToResponse(p *service.PostInfo) *postResponse {
	return &postResponse{
		ID: p.ID, ProjectID: p.ProjectID, Title: p.Title, Content: p.Content,
		URL: p.URL, PostType: p.PostType, Platforms: p.Platforms,
		ScheduledAt: p.ScheduledAt, PublishedAt: p.PublishedAt,
		Status: p.Status, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}
