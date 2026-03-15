package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/werd-platform/werd/src/go/api/internal/middleware"
	"github.com/werd-platform/werd/src/go/api/internal/service"
)

type Auth struct {
	svc *service.Auth
}

func NewAuth(svc *service.Auth) *Auth {
	return &Auth{svc: svc}
}

// --- Request/Response types ---

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type messageResponse struct {
	Message string `json:"message"`
}

// --- Handlers ---

// Login handles POST /auth/login.
func (h *Auth) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "email and password are required"})
		return
	}

	token, user, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, messageResponse{Message: "invalid email or password"})
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		Token: token,
		User: userResponse{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
		},
	})
}

// Me handles GET /auth/me.
func (h *Auth) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	user, err := h.svc.GetUser(r.Context(), userID)
	if err != nil {
		if err == service.ErrAuthUserNotFound {
			// JWT is valid but user no longer exists (e.g., DB was reset).
			// Return 401 so the frontend clears the stale token.
			writeJSON(w, http.StatusUnauthorized, messageResponse{Message: "user no longer exists — please log in again"})
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to fetch user", err)
		return
	}

	writeJSON(w, http.StatusOK, userResponse{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	})
}

// ChangePassword handles PUT /auth/me/password.
func (h *Auth) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "invalid request body"})
		return
	}

	if req.CurrentPassword == "" || req.NewPassword == "" {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "current_password and new_password are required"})
		return
	}

	if len(req.NewPassword) < 8 {
		writeJSON(w, http.StatusBadRequest, messageResponse{Message: "new password must be at least 8 characters"})
		return
	}

	if err := h.svc.ChangePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		if err == service.ErrInvalidCredentials {
			writeJSON(w, http.StatusUnauthorized, messageResponse{Message: "current password is incorrect"})
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to change password", err)
		return
	}

	writeJSON(w, http.StatusOK, messageResponse{Message: "password changed"})
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError logs the error server-side and returns a JSON error response
// with both a human-readable message and the underlying error detail.
func writeError(w http.ResponseWriter, status int, msg string, err error) {
	detail := ""
	if err != nil {
		detail = err.Error()
		log.Printf("ERROR [%d] %s: %v", status, msg, err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{Message: msg, Detail: detail})
}

type errorResponse struct {
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}
