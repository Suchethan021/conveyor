package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration, sourced exclusively from the
// environment. Nothing is hardcoded; secrets never live in the codebase.
type Config struct {
	DatabaseURL       string
	Port              string
	WorkerConcurrency int
}

// Load reads and validates configuration from environment variables.
func Load() (*Config, error) {
	c := &Config{
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		Port:              getenv("PORT", "8080"),
		WorkerConcurrency: getenvInt("WORKER_CONCURRENCY", 2),
	}

	if c.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
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

func getenvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
