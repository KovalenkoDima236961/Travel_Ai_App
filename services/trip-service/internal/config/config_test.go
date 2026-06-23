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
		"USER_SERVICE_URL",
		"USER_CONTEXT_ENABLED",
		"USER_CONTEXT_TIMEOUT_SECONDS",
		"USER_CONTEXT_FAIL_OPEN",
		"EXTERNAL_INTEGRATIONS_SERVICE_URL",
		"WEATHER_CONTEXT_ENABLED",
		"WEATHER_CONTEXT_TIMEOUT_SECONDS",
		"WEATHER_CONTEXT_FAIL_OPEN",
		"PLACE_ENRICHMENT_ENABLED",
		"PLACE_ENRICHMENT_FAIL_OPEN",
		"PLACE_ENRICHMENT_TIMEOUT_SECONDS",
		"PLACE_ENRICHMENT_MIN_CONFIDENCE",
		"PLACE_ENRICHMENT_MAX_ITEMS",
		"PLACE_ENRICHMENT_OVERWRITE_EXISTING",
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
	if cfg.CORS.AllowedMethods != "GET,POST,PUT,PATCH,DELETE,OPTIONS" {
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
	if !cfg.UserContext.Enabled {
		t.Fatal("expected user context to be enabled by default")
	}
	if cfg.UserContext.UserServiceURL != "http://user-service:8083" {
		t.Fatalf("expected default user service URL, got %q", cfg.UserContext.UserServiceURL)
	}
	if cfg.UserContext.TimeoutSeconds != 5 {
		t.Fatalf("expected default user context timeout 5s, got %d", cfg.UserContext.TimeoutSeconds)
	}
	if !cfg.UserContext.FailOpen {
		t.Fatal("expected user context to fail open by default")
	}
	if !cfg.WeatherContext.Enabled {
		t.Fatal("expected weather context to be enabled by default")
	}
	if cfg.WeatherContext.ExternalIntegrationsServiceURL != "http://external-integrations-service:8084" {
		t.Fatalf("expected default external integrations URL, got %q", cfg.WeatherContext.ExternalIntegrationsServiceURL)
	}
	if cfg.WeatherContext.TimeoutSeconds != 5 {
		t.Fatalf("expected default weather context timeout 5s, got %d", cfg.WeatherContext.TimeoutSeconds)
	}
	if !cfg.WeatherContext.FailOpen {
		t.Fatal("expected weather context to fail open by default")
	}
	if !cfg.PlaceEnrichment.Enabled {
		t.Fatal("expected place enrichment to be enabled by default")
	}
	if cfg.PlaceEnrichment.ExternalIntegrationsServiceURL != "http://external-integrations-service:8084" {
		t.Fatalf("expected default place enrichment URL, got %q", cfg.PlaceEnrichment.ExternalIntegrationsServiceURL)
	}
	if !cfg.PlaceEnrichment.FailOpen {
		t.Fatal("expected place enrichment to fail open by default")
	}
	if cfg.PlaceEnrichment.TimeoutSeconds != 5 {
		t.Fatalf("expected place enrichment timeout 5s, got %d", cfg.PlaceEnrichment.TimeoutSeconds)
	}
	if cfg.PlaceEnrichment.MinConfidence != 0.75 {
		t.Fatalf("expected place enrichment min confidence 0.75, got %f", cfg.PlaceEnrichment.MinConfidence)
	}
	if cfg.PlaceEnrichment.MaxItems != 20 {
		t.Fatalf("expected place enrichment max items 20, got %d", cfg.PlaceEnrichment.MaxItems)
	}
	if cfg.PlaceEnrichment.OverwriteExisting {
		t.Fatal("expected place enrichment not to overwrite existing places by default")
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
		"USER_SERVICE_URL",
		"USER_CONTEXT_ENABLED",
		"USER_CONTEXT_TIMEOUT_SECONDS",
		"USER_CONTEXT_FAIL_OPEN",
		"EXTERNAL_INTEGRATIONS_SERVICE_URL",
		"WEATHER_CONTEXT_ENABLED",
		"WEATHER_CONTEXT_TIMEOUT_SECONDS",
		"WEATHER_CONTEXT_FAIL_OPEN",
		"PLACE_ENRICHMENT_ENABLED",
		"PLACE_ENRICHMENT_FAIL_OPEN",
		"PLACE_ENRICHMENT_TIMEOUT_SECONDS",
		"PLACE_ENRICHMENT_MIN_CONFIDENCE",
		"PLACE_ENRICHMENT_MAX_ITEMS",
		"PLACE_ENRICHMENT_OVERWRITE_EXISTING",
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
