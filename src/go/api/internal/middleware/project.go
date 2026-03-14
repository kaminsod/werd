package middleware

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

const (
	ProjectIDKey   contextKey = "project_id"
	ProjectRoleKey contextKey = "project_role"
)

// RequireProjectMember returns middleware that extracts the project ID from
// the URL ({id}), verifies the authenticated user is a member, and stores
// both the project ID and the user's role in the request context.
// Returns 404 if the project doesn't exist or the user is not a member.
func RequireProjectMember(q *storage.Queries) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			idParam := chi.URLParam(r, "id")
			projectID, err := uuid.Parse(idParam)
			if err != nil {
				http.Error(w, `{"message":"project not found"}`, http.StatusNotFound)
				return
			}

			userID := UserIDFromContext(r.Context())
			if userID == "" {
				http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			uid, err := uuid.Parse(userID)
			if err != nil {
				http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			member, err := q.GetProjectMember(r.Context(), storage.GetProjectMemberParams{
				ProjectID: projectID,
				UserID:    uid,
			})
			if err != nil {
				http.Error(w, `{"message":"project not found"}`, http.StatusNotFound)
				return
			}

			ctx := context.WithValue(r.Context(), ProjectIDKey, projectID.String())
			ctx = context.WithValue(ctx, ProjectRoleKey, string(member.Role))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ProjectIDFromContext extracts the project ID from the context.
func ProjectIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(ProjectIDKey).(string)
	return id
}

// ProjectRoleFromContext extracts the user's project role from the context.
func ProjectRoleFromContext(ctx context.Context) string {
	role, _ := ctx.Value(ProjectRoleKey).(string)
	return role
}
