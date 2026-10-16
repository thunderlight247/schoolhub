// Package config centralizes environment-based configuration for the SchoolHub
// backend. Keeping this in one place avoids scattering os.Getenv calls across
// the codebase and gives every service a single, typed source of truth.
package config

import (
	"fmt"
	"net/http"
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

	// FrontendURL is the single origin allowed to make credentialed
	// (cookie-bearing) cross-origin requests to this API — see the CORS
	// middleware in cmd/server/main.go. Must be an exact origin
	// (scheme + host + port), not a wildcard: browsers refuse
	// "Access-Control-Allow-Origin: *" combined with credentials.
	FrontendURL string

	// CookieDomain scopes the session cookie. Empty means a host-only
	// cookie, which is correct for local dev (frontend and backend both
	// on localhost, different ports) and for a production deployment
	// where frontend and backend share a registrable domain. A
	// split-domain deployment needs this revisited — see M4's report.
	CookieDomain string

	// CookieSameSite is the raw SameSite attribute for the session
	// cookie: "lax" (default, correct when frontend/API share a site),
	// "none" (required for a split-domain deployment — see
	// CookieConfig's comment in internal/auth for why), or "strict".
	// Empty means lax. "none" additionally requires Secure=true (i.e.
	// APP_ENV=production): browsers reject SameSite=None without Secure.
	CookieSameSite string

	// ClerkSecretKey enables the Clerk authentication bridge (POST
	// /api/v1/auth/clerk-session) when set. When empty (the default) the
	// classic email/password session flow is the only auth path and the
	// bridge endpoint is not mounted. Never exposed to the client.
	ClerkSecretKey string

	// ResendAPIKey enables outbound email (account welcome messages and
	// school-wide announcement notifications) via Resend. Empty disables
	// sending entirely — the app is fully functional without it.
	ResendAPIKey string

	// EmailFromAddress is the verified Resend sender address, e.g.
	// "SchoolHub Academy <no-reply@schoolhub.example>". Ignored when no
	// API key is set.
	EmailFromAddress string

	// DemoMode enables the one-click demo sign-in (POST
	// /api/v1/auth/demo-login and GET /api/v1/auth/demo-mode). Off by
	// default: it lets anyone on the internet open a session for the demo
	// admin/teacher/parent personas, so it must only be set on demo
	// deployments — never on one holding real data.
	DemoMode bool
}

// Load reads configuration from environment variables and returns a Config.
// It returns an error if a required variable is missing, so the server fails
// fast at startup rather than behaving unpredictably later.
func Load() (*Config, error) {
	cfg := &Config{
		Port:             getEnv("PORT", "8080"),
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		Environment:      getEnv("APP_ENV", "development"),
		FrontendURL:      getEnv("FRONTEND_URL", "http://localhost:3000"),
		CookieDomain:     getEnv("COOKIE_DOMAIN", ""),
		CookieSameSite:   getEnv("COOKIE_SAMESITE", "lax"),
		ClerkSecretKey:   os.Getenv("CLERK_SECRET_KEY"),		ResendAPIKey:    os.Getenv("RESEND_API_KEY"),
		EmailFromAddress: getEnv("EMAIL_FROM_ADDRESS", "SchoolHub Academy <onboarding@resend.dev>"),
		DemoMode:        getEnv("DEMO_MODE", "") == "1" || getEnv("DEMO_MODE", "") == "true",
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required but was not set")
	}

	switch cfg.CookieSameSite {
	case "", "lax", "strict", "none":
	default:
		return nil, fmt.Errorf("config: invalid COOKIE_SAMESITE %q (want lax, strict, or none)", cfg.CookieSameSite)
	}
	if cfg.CookieSameSite == "none" && !cfg.IsProduction() {
		return nil, fmt.Errorf("config: COOKIE_SAMESITE=none requires APP_ENV=production (browsers reject SameSite=None without the Secure attribute)")
	}

	return cfg, nil
}

// CookieSameSiteMode maps the COOKIE_SAMESITE env value to the
// http.SameSite constants the auth handlers need. "lax" is the default
// (matching what the handlers previously hardcoded); Load validates the
// raw string and the none-needs-production rule before callers run.
func (c *Config) CookieSameSiteMode() http.SameSite {
	switch c.CookieSameSite {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
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
