package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/werd-platform/werd/src/go/api/internal/handler"
	"github.com/werd-platform/werd/src/go/api/internal/middleware"
	"github.com/werd-platform/werd/src/go/api/internal/service"
	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

func New(
	authSvc *service.Auth,
	authH *handler.Auth,
	projectH *handler.ProjectHandler,
	queries *storage.Queries,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware.
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	// Public routes.
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	r.Post("/auth/login", authH.Login)

	// Protected routes (require valid JWT).
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(authSvc))

		// Auth.
		r.Get("/auth/me", authH.Me)
		r.Put("/auth/me/password", authH.ChangePassword)

		// Projects (list + create don't need project membership).
		r.Get("/projects", projectH.ListProjects)
		r.Post("/projects", projectH.CreateProject)

		// Project-scoped routes (require membership).
		r.Route("/projects/{id}", func(r chi.Router) {
			r.Use(middleware.RequireProjectMember(queries))

			r.Get("/", projectH.GetProject)
			r.Put("/", projectH.UpdateProject)
			r.Delete("/", projectH.DeleteProject)

			r.Get("/members", projectH.ListMembers)
			r.Post("/members", projectH.AddMember)
			r.Put("/members/{userID}", projectH.UpdateMemberRole)
			r.Delete("/members/{userID}", projectH.RemoveMember)
		})
	})

	return r
}
