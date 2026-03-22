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
	alertH *handler.AlertHandler,
	notifH *handler.NotificationHandler,
	platformH *handler.PlatformHandler,
	sourceH *handler.MonitorSourceHandler,
	processingH *handler.ProcessingHandler,
	queries *storage.Queries,
	internalAPIKey string,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware.
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)

	// Health check (root level, no /api prefix — used by Docker healthcheck).
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// All API routes under /api prefix.
	r.Route("/api", func(r chi.Router) {
		// Also expose healthz under /api for convenience.
		r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"status":"ok"}`))
		})

		// Public.
		r.Post("/auth/login", authH.Login)

		// Webhook ingestion (internal services, API key auth).
		r.Group(func(r chi.Router) {
			r.Use(middleware.RequireInternalAPIKey(internalAPIKey))
			r.Post("/webhooks/ingest", alertH.IngestWebhook)
		})

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

				// Alerts.
				r.Get("/alerts", alertH.ListAlerts)
				r.Get("/alerts/{alertID}", alertH.GetAlert)
				r.Put("/alerts/{alertID}", alertH.UpdateAlertStatus)

				// Keywords.
				r.Get("/keywords", alertH.ListKeywords)
				r.Post("/keywords", alertH.CreateKeyword)
				r.Delete("/keywords/{kwID}", alertH.DeleteKeyword)

				// Notification rules.
				r.Get("/rules", notifH.ListRules)
				r.Post("/rules", notifH.CreateRule)
				r.Get("/rules/{ruleID}", notifH.GetRule)
				r.Put("/rules/{ruleID}", notifH.UpdateRule)
				r.Delete("/rules/{ruleID}", notifH.DeleteRule)

				// Monitor sources.
				r.Get("/sources", sourceH.ListSources)
				r.Post("/sources", sourceH.CreateSource)
				r.Get("/sources/{sourceID}", sourceH.GetSource)
				r.Put("/sources/{sourceID}", sourceH.UpdateSource)
				r.Delete("/sources/{sourceID}", sourceH.DeleteSource)

				// Processing rules.
			r.Get("/processing-rules", processingH.ListRules)
			r.Post("/processing-rules", processingH.CreateRule)
			r.Get("/processing-rules/{ruleID}", processingH.GetRule)
			r.Put("/processing-rules/{ruleID}", processingH.UpdateRule)
			r.Delete("/processing-rules/{ruleID}", processingH.DeleteRule)

			// Platform connections.
				r.Get("/connections", platformH.ListConnections)
				r.Post("/connections", platformH.CreateConnection)
				r.Post("/connections/create-account", platformH.CreateAccount)
				r.Get("/connections/{connID}", platformH.GetConnection)
				r.Put("/connections/{connID}", platformH.UpdateConnection)
				r.Delete("/connections/{connID}", platformH.DeleteConnection)

				// Published posts.
				r.Get("/posts", platformH.ListPosts)
				r.Post("/posts", platformH.CreatePost)
				r.Get("/posts/{postID}", platformH.GetPost)
				r.Put("/posts/{postID}", platformH.UpdatePost)
				r.Delete("/posts/{postID}", platformH.DeletePost)
				r.Post("/posts/{postID}/publish", platformH.PublishPost)
				r.Post("/posts/{postID}/unschedule", platformH.CancelSchedule)
				r.Put("/posts/{postID}/monitor", platformH.SetPostMonitor)
			})
		})
	})

	return r
}
