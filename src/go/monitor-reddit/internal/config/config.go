package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	WerdAPIURL   string
	WerdAPIKey   string
	ProjectID    string
	Subreddits   []string
	PollInterval time.Duration

	RedditClientID     string
	RedditClientSecret string
	RedditUsername      string
	RedditPassword      string
	RedditUserAgent     string
}

func Load() (*Config, error) {
	cfg := &Config{
		WerdAPIURL:         os.Getenv("WERD_API_URL"),
		WerdAPIKey:         os.Getenv("WERD_INTERNAL_API_KEY"),
		ProjectID:          os.Getenv("WERD_PROJECT_ID"),
		RedditClientID:     os.Getenv("REDDIT_CLIENT_ID"),
		RedditClientSecret: os.Getenv("REDDIT_CLIENT_SECRET"),
		RedditUsername:      os.Getenv("REDDIT_USERNAME"),
		RedditPassword:      os.Getenv("REDDIT_PASSWORD"),
		RedditUserAgent:     envOr("REDDIT_USER_AGENT", "werd-monitor-reddit/1.0"),
	}

	subs := os.Getenv("WERD_SUBREDDITS")
	if subs != "" {
		for _, s := range strings.Split(subs, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				cfg.Subreddits = append(cfg.Subreddits, s)
			}
		}
	}

	interval := envOr("WERD_POLL_INTERVAL", "60s")
	d, err := time.ParseDuration(interval)
	if err != nil {
		return nil, fmt.Errorf("invalid WERD_POLL_INTERVAL: %w", err)
	}
	cfg.PollInterval = d

	if cfg.WerdAPIURL == "" {
		return nil, fmt.Errorf("WERD_API_URL is required")
	}
	if cfg.WerdAPIKey == "" {
		return nil, fmt.Errorf("WERD_INTERNAL_API_KEY is required")
	}
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("WERD_PROJECT_ID is required")
	}
	if len(cfg.Subreddits) == 0 {
		return nil, fmt.Errorf("WERD_SUBREDDITS is required")
	}
	if cfg.RedditClientID == "" || cfg.RedditClientSecret == "" {
		return nil, fmt.Errorf("REDDIT_CLIENT_ID and REDDIT_CLIENT_SECRET are required")
	}
	if cfg.RedditUsername == "" || cfg.RedditPassword == "" {
		return nil, fmt.Errorf("REDDIT_USERNAME and REDDIT_PASSWORD are required")
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
