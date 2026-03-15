package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/werd-platform/werd/src/go/api/internal/config"
	"github.com/werd-platform/werd/src/go/api/internal/handler"
	"github.com/werd-platform/werd/src/go/api/internal/integration"
	"github.com/werd-platform/werd/src/go/api/internal/router"
	"github.com/werd-platform/werd/src/go/api/internal/service"
	"github.com/werd-platform/werd/src/go/api/internal/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Database connection pool.
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("database ping: %v", err)
	}
	log.Println("database connected")

	// Storage and services.
	queries := storage.New(pool)
	authService := service.NewAuth(queries, cfg.JWTSecret)
	projectService := service.NewProject(pool, queries)
	alertService := service.NewAlert(queries)
	keywordService := service.NewKeyword(queries)
	notificationService := service.NewNotification(queries, cfg.NtfyURL)

	// Platform integration.
	adapterRegistry := integration.NewRegistry()
	adapterRegistry.Register("bluesky", integration.NewBluesky(""))

	monitorSourceService := service.NewMonitorSource(queries)
	platformService := service.NewPlatform(queries, adapterRegistry)
	postService := service.NewPost(queries, platformService, adapterRegistry)

	// Seed admin user from env vars (idempotent).
	if cfg.AdminEmail != "" && cfg.AdminPassword != "" {
		if err := authService.SeedAdmin(ctx, cfg.AdminEmail, cfg.AdminPassword); err != nil {
			log.Fatalf("seed admin: %v", err)
		}
	}

	// Handlers and router.
	authHandler := handler.NewAuth(authService)
	projectHandler := handler.NewProject(projectService)
	alertHandler := handler.NewAlert(alertService, keywordService, notificationService)
	notificationHandler := handler.NewNotification(notificationService)
	monitorSourceHandler := handler.NewMonitorSource(monitorSourceService)
	platformHandler := handler.NewPlatform(platformService, postService)
	r := router.New(authService, authHandler, projectHandler, alertHandler, notificationHandler, platformHandler, monitorSourceHandler, queries, cfg.InternalAPIKey)

	// HTTP server with graceful shutdown.
	addr := fmt.Sprintf(":%s", cfg.APIPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("werd-api listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
	log.Println("stopped")
}
