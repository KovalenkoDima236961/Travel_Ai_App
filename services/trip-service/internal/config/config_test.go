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
		"PUBLIC_SHARING_ENABLED",
		"PUBLIC_WEB_BASE_URL",
		"SHARE_TOKEN_BYTES",
		"TRIP_PRESENCE_ENABLED",
		"TRIP_PRESENCE_HEARTBEAT_SECONDS",
		"TRIP_PRESENCE_STALE_AFTER_SECONDS",
		"TRIP_PRESENCE_MAX_CONNECTIONS_PER_USER_PER_TRIP",
		"TRIP_PRESENCE_SEND_FULL_SNAPSHOT",
		"TRIP_EDIT_LOCKS_ENABLED",
		"TRIP_EDIT_LOCK_TTL_SECONDS",
		"TRIP_EDIT_LOCK_RENEW_SECONDS",
		"TRIP_EDIT_LOCK_STALE_CLEANUP_SECONDS",
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
	if !cfg.PublicSharing.Enabled {
		t.Fatal("expected public sharing to be enabled by default")
	}
	if cfg.PublicSharing.PublicWebBaseURL != "http://localhost:3000" {
		t.Fatalf("expected public web base URL default, got %q", cfg.PublicSharing.PublicWebBaseURL)
	}
	if cfg.PublicSharing.ShareTokenBytes != 32 {
		t.Fatalf("expected share token bytes default 32, got %d", cfg.PublicSharing.ShareTokenBytes)
	}
	if !cfg.Presence.Enabled {
		t.Fatal("expected trip presence to be enabled by default")
	}
	if cfg.Presence.HeartbeatSeconds != 25 {
		t.Fatalf("expected trip presence heartbeat 25s, got %d", cfg.Presence.HeartbeatSeconds)
	}
	if cfg.Presence.StaleAfterSeconds != 60 {
		t.Fatalf("expected trip presence stale after 60s, got %d", cfg.Presence.StaleAfterSeconds)
	}
	if cfg.Presence.MaxConnectionsPerUserPerTrip != 5 {
		t.Fatalf("expected trip presence max connections 5, got %d", cfg.Presence.MaxConnectionsPerUserPerTrip)
	}
	if !cfg.Presence.SendFullSnapshot {
		t.Fatal("expected trip presence full snapshots by default")
	}
	if !cfg.EditLocks.Enabled {
		t.Fatal("expected trip edit locks to be enabled by default")
	}
	if cfg.EditLocks.TTLSeconds != 180 {
		t.Fatalf("expected trip edit lock TTL 180s, got %d", cfg.EditLocks.TTLSeconds)
	}
	if cfg.EditLocks.RenewSeconds != 45 {
		t.Fatalf("expected trip edit lock renew interval 45s, got %d", cfg.EditLocks.RenewSeconds)
	}
	if cfg.EditLocks.StaleCleanupSeconds != 30 {
		t.Fatalf("expected trip edit lock cleanup interval 30s, got %d", cfg.EditLocks.StaleCleanupSeconds)
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
		"PUBLIC_SHARING_ENABLED",
		"PUBLIC_WEB_BASE_URL",
		"SHARE_TOKEN_BYTES",
		"TRIP_PRESENCE_ENABLED",
		"TRIP_PRESENCE_HEARTBEAT_SECONDS",
		"TRIP_PRESENCE_STALE_AFTER_SECONDS",
		"TRIP_PRESENCE_MAX_CONNECTIONS_PER_USER_PER_TRIP",
		"TRIP_PRESENCE_SEND_FULL_SNAPSHOT",
		"TRIP_EDIT_LOCKS_ENABLED",
		"TRIP_EDIT_LOCK_TTL_SECONDS",
		"TRIP_EDIT_LOCK_RENEW_SECONDS",
		"TRIP_EDIT_LOCK_STALE_CLEANUP_SECONDS",
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
