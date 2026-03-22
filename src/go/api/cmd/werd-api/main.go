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
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"github.com/werd-platform/werd/src/go/api/internal/config"
	"github.com/werd-platform/werd/src/go/api/internal/handler"
	"github.com/werd-platform/werd/src/go/api/internal/integration"
	"github.com/werd-platform/werd/src/go/api/internal/router"
	"github.com/werd-platform/werd/src/go/api/internal/service"
	"github.com/werd-platform/werd/src/go/api/internal/storage"
	"github.com/werd-platform/werd/src/go/api/internal/worker"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Database connection pool with retry.
	ctx := context.Background()
	var pool *pgxpool.Pool
	for attempt := 1; attempt <= 30; attempt++ {
		var err error
		pool, err = pgxpool.New(ctx, cfg.DatabaseURL)
		if err != nil {
			log.Printf("database connect attempt %d/30: %v", attempt, err)
			time.Sleep(2 * time.Second)
			continue
		}
		if err = pool.Ping(ctx); err != nil {
			pool.Close()
			log.Printf("database ping attempt %d/30: %v", attempt, err)
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}
	if pool == nil {
		log.Fatalf("database: failed to connect after 30 attempts")
	}
	defer pool.Close()
	log.Println("database connected")

	// River job queue: run migrations (idempotent).
	riverDriver := riverpgxv5.New(pool)
	migrator, err := rivermigrate.New(riverDriver, nil)
	if err != nil {
		log.Fatalf("river migrator: %v", err)
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		log.Fatalf("river migrate: %v", err)
	}
	log.Println("river migrations complete")

	// Storage and services.
	queries := storage.New(pool)
	authService := service.NewAuth(queries, cfg.JWTSecret)
	projectService := service.NewProject(pool, queries)
	alertService := service.NewAlert(queries)
	keywordService := service.NewKeyword(queries)
	notificationService := service.NewNotification(queries, cfg.NtfyURL)

	// Platform integration: API adapters.
	adapterRegistry := integration.NewRegistry()
	adapterRegistry.Register("bluesky:api", integration.NewBluesky(""))
	adapterRegistry.Register("reddit:api", integration.NewReddit())
	adapterRegistry.Register("hn:api", integration.NewHN())
	adapterRegistry.Register("gmail:api", integration.NewGmail())
	adapterRegistry.Register("google_groups:api", integration.NewGoogleGroups())

	// Browser adapters (only if browser service is configured).
	if cfg.BrowserServiceURL != "" {
		log.Printf("browser service enabled: %s", cfg.BrowserServiceURL)
		adapterRegistry.Register("bluesky:browser", integration.NewBrowserAdapter(cfg.BrowserServiceURL, "bluesky", cfg.InternalAPIKey))
		adapterRegistry.Register("reddit:browser", integration.NewBrowserAdapter(cfg.BrowserServiceURL, "reddit", cfg.InternalAPIKey))
		adapterRegistry.Register("hn:browser", integration.NewBrowserAdapter(cfg.BrowserServiceURL, "hn", cfg.InternalAPIKey))
	}

	monitorSourceService := service.NewMonitorSource(queries)
	platformService := service.NewPlatform(queries, adapterRegistry)
	postService := service.NewPost(queries, platformService, adapterRegistry)

	// River client + workers.
	workers := river.NewWorkers()
	river.AddWorker(workers, &worker.PublishPostWorker{
		Publish: func(ctx context.Context, projectID, postID string) error {
			_, err := postService.ExecutePublish(ctx, projectID, postID)
			return err
		},
	})

	riverClient, err := river.NewClient(riverDriver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: workers,
	})
	if err != nil {
		log.Fatalf("river client: %v", err)
	}
	postService.SetRiverClient(riverClient)

	processingRuleService := service.NewProcessingRuleService(queries)

	// LLM client (optional).
	var llmClient *service.LLMClient
	if cfg.LLMApiURL != "" {
		llmClient = service.NewLLMClient(cfg.LLMApiURL, cfg.LLMApiKey, cfg.LLMModel)
		log.Printf("LLM classification enabled: %s", cfg.LLMApiURL)
	}

	// Processing pipeline.
	processingPipeline := service.NewProcessingPipeline(queries, llmClient)

	// Reply monitoring: platform readers.
	readerRegistry := integration.NewReaderRegistry()
	readerRegistry.Register("reddit", integration.NewRedditReader())
	readerRegistry.Register("hn", integration.NewHNReader())
	readerRegistry.Register("bluesky", integration.NewBlueskyReader())

	replyMonitor := service.NewReplyMonitor(queries, platformService, alertService, readerRegistry, 5*time.Minute)

	// Source monitoring: in-process pollers for thread/subreddit/keyword sources.
	sourceMonitorRegistry := integration.NewSourceMonitorRegistry()
	sourceMonitorRegistry.Register("reddit:thread", integration.NewRedditThreadMonitor())
	sourceMonitorRegistry.Register("reddit:subreddit", integration.NewRedditSubredditMonitor())
	sourceMonitorRegistry.Register("reddit:account", integration.NewRedditAccountMonitor())
	sourceMonitorRegistry.Register("hn:thread", integration.NewHNThreadMonitor())
	sourceMonitorRegistry.Register("hn:keywords", integration.NewHNKeywordMonitor())
	sourceMonitorRegistry.Register("hn:new", integration.NewHNNewMonitor())
	sourceMonitorRegistry.Register("hn:account", integration.NewHNAccountMonitor())
	sourceMonitorRegistry.Register("bluesky:account", integration.NewBlueskyAccountMonitor())
	sourceMonitorRegistry.Register("bluesky:user", integration.NewBlueskyUserMonitor())

	sourcePoller := service.NewSourcePoller(queries, platformService, alertService, sourceMonitorRegistry, processingPipeline, 60*time.Second)

	// Seed admin user from env vars (idempotent).
	if cfg.AdminEmail != "" && cfg.AdminPassword != "" {
		if err := authService.SeedAdmin(ctx, cfg.AdminEmail, cfg.AdminPassword); err != nil {
			log.Fatalf("seed admin: %v", err)
		}
	}

	// Start river job processing.
	if err := riverClient.Start(ctx); err != nil {
		log.Fatalf("river start: %v", err)
	}

	// Start background goroutines.
	go replyMonitor.Run(ctx)
	go sourcePoller.Run(ctx)

	// Handlers and router.
	authHandler := handler.NewAuth(authService)
	projectHandler := handler.NewProject(projectService)
	alertHandler := handler.NewAlert(alertService, keywordService, notificationService)
	notificationHandler := handler.NewNotification(notificationService)
	monitorSourceHandler := handler.NewMonitorSource(monitorSourceService)
	platformHandler := handler.NewPlatform(platformService, postService, replyMonitor)
	processingHandler := handler.NewProcessing(processingRuleService)
	r := router.New(authService, authHandler, projectHandler, alertHandler, notificationHandler, platformHandler, monitorSourceHandler, processingHandler, queries, cfg.InternalAPIKey)

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
	if err := riverClient.Stop(shutdownCtx); err != nil {
		log.Printf("river stop: %v", err)
	}
	log.Println("stopped")
}
