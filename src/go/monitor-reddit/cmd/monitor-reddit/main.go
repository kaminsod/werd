package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"

	"github.com/werd-platform/werd/src/go/monitor-reddit/internal/config"
	"github.com/werd-platform/werd/src/go/monitor-reddit/internal/poller"
	"github.com/werd-platform/werd/src/go/monitor-reddit/internal/reddit"
	"github.com/werd-platform/werd/src/go/monitor-reddit/internal/webhook"
)

func main() {
	log.Println("werd-monitor-reddit starting")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	client := reddit.NewClient(
		cfg.RedditClientID, cfg.RedditClientSecret,
		cfg.RedditUsername, cfg.RedditPassword,
		cfg.RedditUserAgent,
	)
	sender := webhook.NewSender(cfg.WerdAPIURL, cfg.WerdAPIKey)
	p := poller.New(client, sender, cfg.ProjectID, cfg.Subreddits, cfg.PollInterval)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := p.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("poller: %v", err)
	}

	log.Println("werd-monitor-reddit shutting down")
}
