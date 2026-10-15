package config

import (
	"net/http"
	"testing"
)

func TestLoadCookieSameSite(t *testing.T) {
	url := "postgres://u:p@localhost:5432/schoolhub?sslmode=disable"

	t.Run("defaults to lax", func(t *testing.T) {
		t.Setenv("DATABASE_URL", url)
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got := cfg.CookieSameSiteMode(); got != http.SameSiteLaxMode {
			t.Fatalf("CookieSameSiteMode = %v, want Lax", got)
		}
	})

	t.Run("lax allowed in production", func(t *testing.T) {
		t.Setenv("DATABASE_URL", url)
		t.Setenv("APP_ENV", "production")
		t.Setenv("COOKIE_SAMESITE", "lax")
		if _, err := Load(); err != nil {
			t.Fatalf("Load: %v", err)
		}
	})

	t.Run("none requires production", func(t *testing.T) {
		t.Setenv("DATABASE_URL", url)
		t.Setenv("APP_ENV", "development")
		t.Setenv("COOKIE_SAMESITE", "none")
		if _, err := Load(); err == nil {
			t.Fatal("Load with COOKIE_SAMESITE=none outside production: expected error, got nil")
		}
	})

	t.Run("none allowed in production", func(t *testing.T) {
		t.Setenv("DATABASE_URL", url)
		t.Setenv("APP_ENV", "production")
		t.Setenv("COOKIE_SAMESITE", "none")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got := cfg.CookieSameSiteMode(); got != http.SameSiteNoneMode {
			t.Fatalf("CookieSameSiteMode = %v, want None", got)
		}
	})

	t.Run("invalid value fails fast", func(t *testing.T) {
		t.Setenv("DATABASE_URL", url)
		t.Setenv("COOKIE_SAMESITE", "bogus")
		if _, err := Load(); err == nil {
			t.Fatal("Load with invalid COOKIE_SAMESITE: expected error, got nil")
		}
	})

	t.Run("strict maps through", func(t *testing.T) {
		t.Setenv("DATABASE_URL", url)
		t.Setenv("COOKIE_SAMESITE", "strict")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if got := cfg.CookieSameSiteMode(); got != http.SameSiteStrictMode {
			t.Fatalf("CookieSameSiteMode = %v, want Strict", got)
		}
	})
}
