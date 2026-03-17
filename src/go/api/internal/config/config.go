package config

import (
	"fmt"
	"os"
)

type Config struct {
	APIPort     string
	DatabaseURL string
	RedisURL    string
	JWTSecret      string
	InternalAPIKey string
	NtfyURL           string
	BrowserServiceURL string // optional: enables browser adapters

	// LLM integration (OpenAI-compatible endpoint).
	LLMApiURL string // optional: enables LLM classification
	LLMApiKey string
	LLMModel  string

	// Optional: seed an admin user on first startup.
	AdminEmail    string
	AdminPassword string
}

func Load() (*Config, error) {
	cfg := &Config{
		APIPort:       envOr("WERD_API_PORT", "8090"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
		RedisURL:      os.Getenv("REDIS_URL"),
		JWTSecret:      os.Getenv("WERD_JWT_SECRET"),
		InternalAPIKey: os.Getenv("WERD_INTERNAL_API_KEY"),
		NtfyURL:           envOr("WERD_NTFY_URL", "http://ntfy:80"),
		BrowserServiceURL: os.Getenv("BROWSER_SERVICE_URL"),
		LLMApiURL:         os.Getenv("WERD_LLM_API_URL"),
		LLMApiKey:         os.Getenv("WERD_LLM_API_KEY"),
		LLMModel:          os.Getenv("WERD_LLM_MODEL"),
		AdminEmail:        os.Getenv("WERD_ADMIN_EMAIL"),
		AdminPassword: os.Getenv("WERD_ADMIN_PASSWORD"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("WERD_JWT_SECRET is required")
	}
	if cfg.InternalAPIKey == "" {
		return nil, fmt.Errorf("WERD_INTERNAL_API_KEY is required")
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
