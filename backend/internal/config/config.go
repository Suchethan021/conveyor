package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration, sourced exclusively from the
// environment. Nothing is hardcoded; secrets never live in the codebase.
type Config struct {
	DatabaseURL       string
	Port              string
	WorkerConcurrency int

	// Auth / OAuth. Validated by the auth layer when it is constructed, not
	// here, so the rest of the service can boot without them during early dev.
	SessionSecret      string
	GitHubClientID     string
	GitHubClientSecret string
	GitHubCallbackURL  string
	FrontendURL        string

	// AllowDevLogin enables a local-only login shortcut for testing without
	// GitHub. Must be false in any real deployment.
	AllowDevLogin bool

	// CORSAllowedOrigins are the browser origins allowed to call the API with
	// credentials. Only needed when the frontend is on a different origin than
	// the backend (the split-domain deploy); empty falls back to FrontendURL.
	CORSAllowedOrigins []string
}

// Load reads and validates configuration from environment variables.
func Load() (*Config, error) {
	c := &Config{
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		Port:              getenv("PORT", "8080"),
		WorkerConcurrency: getenvInt("WORKER_CONCURRENCY", 2),

		SessionSecret:      os.Getenv("SESSION_SECRET"),
		GitHubClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		GitHubClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		GitHubCallbackURL:  getenv("GITHUB_CALLBACK_URL", "http://localhost:8080/api/auth/github/callback"),
		FrontendURL:        getenv("FRONTEND_URL", "http://localhost:5173"),
		AllowDevLogin:      os.Getenv("ALLOW_DEV_LOGIN") == "true",
		CORSAllowedOrigins: splitList(os.Getenv("CORS_ALLOWED_ORIGINS")),
	}

	// Default the CORS allow-list to the frontend origin.
	if len(c.CORSAllowedOrigins) == 0 && c.FrontendURL != "" {
		c.CORSAllowedOrigins = []string{c.FrontendURL}
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if c.SessionSecret == "" {
		return nil, fmt.Errorf("SESSION_SECRET is required")
	}
	if c.WorkerConcurrency < 1 {
		return nil, fmt.Errorf("WORKER_CONCURRENCY must be >= 1")
	}
	return c, nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// splitList parses a comma-separated env value into a trimmed, non-empty slice.
func splitList(v string) []string {
	if v == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
