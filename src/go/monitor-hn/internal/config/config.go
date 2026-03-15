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
	PollInterval time.Duration
	Keywords     []string // optional pre-filter
}

func Load() (*Config, error) {
	cfg := &Config{
		WerdAPIURL: os.Getenv("WERD_API_URL"),
		WerdAPIKey: os.Getenv("WERD_INTERNAL_API_KEY"),
		ProjectID:  os.Getenv("WERD_PROJECT_ID"),
	}

	interval := envOr("WERD_POLL_INTERVAL", "60s")
	d, err := time.ParseDuration(interval)
	if err != nil {
		return nil, fmt.Errorf("invalid WERD_POLL_INTERVAL: %w", err)
	}
	cfg.PollInterval = d

	kw := os.Getenv("WERD_HN_KEYWORDS")
	if kw != "" {
		for _, k := range strings.Split(kw, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				cfg.Keywords = append(cfg.Keywords, strings.ToLower(k))
			}
		}
	}

	if cfg.WerdAPIURL == "" {
		return nil, fmt.Errorf("WERD_API_URL is required")
	}
	if cfg.WerdAPIKey == "" {
		return nil, fmt.Errorf("WERD_INTERNAL_API_KEY is required")
	}
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("WERD_PROJECT_ID is required")
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
