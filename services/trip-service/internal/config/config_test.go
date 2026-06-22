package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadAppliesAIGenerationTimeoutDefaults(t *testing.T) {
	unsetEnv(t,
		"APP_ENV",
		"HTTP_ADDRESS",
		"HTTP_READ_TIMEOUT",
		"HTTP_WRITE_TIMEOUT",
		"HTTP_IDLE_TIMEOUT",
		"HTTP_SHUTDOWN_TIMEOUT",
		"AUTH_REQUIRED",
		"JWT_ACCESS_SECRET",
		"AUTH_HEADER_NAME",
		"DEV_USER_ID",
		"CORS_ALLOWED_ORIGINS",
		"CORS_ALLOWED_METHODS",
		"CORS_ALLOWED_HEADERS",
		"ITINERARY_GENERATOR_MODE",
		"AI_PLANNING_SERVICE_URL",
		"AI_PLANNING_TIMEOUT_SECONDS",
	)
	t.Setenv("POSTGRES_DB", "trip_service")
	t.Setenv("POSTGRES_USER", "postgres")
	t.Setenv("POSTGRES_PASSWORD", "postgres")
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_MIN_CONNS", "2")
	t.Setenv("POSTGRES_MAX_CONNS", "10")
	t.Setenv("POSTGRES_MIG_PATH", "./migrations")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.HTTPServer.WriteTimeout != 150*time.Second {
		t.Fatalf("expected HTTP write timeout 150s, got %s", cfg.HTTPServer.WriteTimeout)
	}
	if cfg.ItineraryGenerator.AIPlanningTimeoutSeconds != 120 {
		t.Fatalf(
			"expected AI planning timeout 120s, got %d",
			cfg.ItineraryGenerator.AIPlanningTimeoutSeconds,
		)
	}
	if cfg.CORS.AllowedOrigins != "http://localhost:3000" {
		t.Fatalf("expected default CORS origin, got %q", cfg.CORS.AllowedOrigins)
	}
	if cfg.CORS.AllowedMethods != "GET,POST,PATCH,DELETE,OPTIONS" {
		t.Fatalf("expected default CORS methods, got %q", cfg.CORS.AllowedMethods)
	}
	if cfg.CORS.AllowedHeaders != "Content-Type,Authorization" {
		t.Fatalf("expected default CORS headers, got %q", cfg.CORS.AllowedHeaders)
	}
	if !cfg.Auth.Required {
		t.Fatal("expected auth to be required by default")
	}
	if cfg.Auth.JWTAccessSecret != "change-me-in-development" {
		t.Fatalf("expected default JWT secret, got %q", cfg.Auth.JWTAccessSecret)
	}
	if cfg.Auth.HeaderName != "Authorization" {
		t.Fatalf("expected default auth header, got %q", cfg.Auth.HeaderName)
	}
	if cfg.Auth.DevUserID != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("expected default dev user id, got %q", cfg.Auth.DevUserID)
	}
}

func TestLoadReadsCORSOverrides(t *testing.T) {
	unsetEnv(t,
		"APP_ENV",
		"HTTP_ADDRESS",
		"HTTP_READ_TIMEOUT",
		"HTTP_WRITE_TIMEOUT",
		"HTTP_IDLE_TIMEOUT",
		"HTTP_SHUTDOWN_TIMEOUT",
		"AUTH_REQUIRED",
		"JWT_ACCESS_SECRET",
		"AUTH_HEADER_NAME",
		"DEV_USER_ID",
		"ITINERARY_GENERATOR_MODE",
		"AI_PLANNING_SERVICE_URL",
		"AI_PLANNING_TIMEOUT_SECONDS",
	)
	t.Setenv("POSTGRES_DB", "trip_service")
	t.Setenv("POSTGRES_USER", "postgres")
	t.Setenv("POSTGRES_PASSWORD", "postgres")
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_MIN_CONNS", "2")
	t.Setenv("POSTGRES_MAX_CONNS", "10")
	t.Setenv("POSTGRES_MIG_PATH", "./migrations")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000")
	t.Setenv("CORS_ALLOWED_METHODS", "GET,POST,OPTIONS")
	t.Setenv("CORS_ALLOWED_HEADERS", "Content-Type")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.CORS.AllowedOrigins != "http://localhost:3000,http://127.0.0.1:3000" {
		t.Fatalf("expected CORS origin override, got %q", cfg.CORS.AllowedOrigins)
	}
	if cfg.CORS.AllowedMethods != "GET,POST,OPTIONS" {
		t.Fatalf("expected CORS methods override, got %q", cfg.CORS.AllowedMethods)
	}
	if cfg.CORS.AllowedHeaders != "Content-Type" {
		t.Fatalf("expected CORS headers override, got %q", cfg.CORS.AllowedHeaders)
	}
}

func unsetEnv(t *testing.T, names ...string) {
	t.Helper()

	for _, name := range names {
		name := name
		previous, existed := os.LookupEnv(name)
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("unset %s: %v", name, err)
		}

		t.Cleanup(func() {
			if existed {
				_ = os.Setenv(name, previous)
				return
			}
			_ = os.Unsetenv(name)
		})
	}
}
