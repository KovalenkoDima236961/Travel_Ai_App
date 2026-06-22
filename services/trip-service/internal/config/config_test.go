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
