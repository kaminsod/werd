package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"

	"github.com/werd-platform/werd/src/go/monitor-hn/internal/config"
	"github.com/werd-platform/werd/src/go/monitor-hn/internal/hn"
	"github.com/werd-platform/werd/src/go/monitor-hn/internal/poller"
	"github.com/werd-platform/werd/src/go/monitor-hn/internal/webhook"
)

func main() {
	log.Println("werd-monitor-hn starting")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	client := hn.NewClient()
	sender := webhook.NewSender(cfg.WerdAPIURL, cfg.WerdAPIKey)
	p := poller.New(client, sender, cfg.ProjectID, cfg.Keywords, cfg.PollInterval)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := p.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("poller: %v", err)
	}

	log.Println("werd-monitor-hn shutting down")
}
