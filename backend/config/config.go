// Package config centralizes environment-based configuration for the SchoolHub
// backend. Keeping this in one place avoids scattering os.Getenv calls across
// the codebase and gives every service a single, typed source of truth.
package config

import (
	"fmt"
	"os"
)

// Config holds all runtime configuration for the API server.
type Config struct {
	// Port is the TCP port the HTTP server listens on.
	Port string

	// DatabaseURL is the full PostgreSQL connection string, e.g.
	// postgres://user:password@localhost:5432/schoolhub?sslmode=disable
	DatabaseURL string

	// Environment is one of "development", "test", or "production".
	// It is used to toggle things like verbose error responses.
	Environment string
}

// Load reads configuration from environment variables and returns a Config.
// It returns an error if a required variable is missing, so the server fails
// fast at startup rather than behaving unpredictably later.
func Load() (*Config, error) {
	cfg := &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Environment: getEnv("APP_ENV", "development"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required but was not set")
	}

	return cfg, nil
}

// IsProduction reports whether the server is running in production mode.
// Handlers use this to decide how much detail to include in error responses.
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
