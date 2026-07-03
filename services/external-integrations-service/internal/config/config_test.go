package config

import "testing"

func setBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HTTP_ADDR", ":8084")
	t.Setenv("POSTGRES_DB", "external_integrations_service")
	t.Setenv("POSTGRES_USER", "travel_ai")
	t.Setenv("POSTGRES_PASSWORD", "a-strong-db-password-value")
	t.Setenv("POSTGRES_HOST", "postgres")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_MIN_CONNS", "2")
	t.Setenv("POSTGRES_MAX_CONNS", "10")
	t.Setenv("POSTGRES_MIG_PATH", "./migrations")
	t.Setenv("JWT_ACCESS_SECRET", "a-strong-production-jwt-access-secret-value")
	t.Setenv("INTERNAL_SERVICE_TOKEN", "a-strong-production-internal-token-value")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")
	t.Setenv("GOOGLE_CALENDAR_ENABLED", "false")
	t.Setenv("CALENDAR_PROVIDER", "mock")
}

func TestProductionRejectsWildcardCORS(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")

	if _, err := Load(""); err == nil {
		t.Fatal("expected wildcard production CORS to be rejected")
	}
}

func TestFoursquareProviderRequiresAPIKey(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("PLACE_PROVIDER", "foursquare")

	if _, err := Load(""); err == nil {
		t.Fatal("expected missing FOURSQUARE_API_KEY to be rejected")
	}
}

func TestProductionMockProvidersPass(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")

	if _, err := Load(""); err != nil {
		t.Fatalf("expected valid production mock-provider config, got %v", err)
	}
}

func TestProviderLimitsNegativeRateLimitRejected(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("ORS_RATE_LIMIT_PER_MINUTE", "-1")

	if _, err := Load(""); err == nil {
		t.Fatal("expected negative rate limit to be rejected")
	}
}

func TestProviderLimitsNegativeDailyQuotaRejected(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("FOURSQUARE_DAILY_QUOTA", "-5")

	if _, err := Load(""); err == nil {
		t.Fatal("expected negative daily quota to be rejected")
	}
}

func TestProviderLimitsZeroQuotaMeansUnlimited(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("ORS_DAILY_QUOTA", "0")
	t.Setenv("ORS_RATE_LIMIT_PER_MINUTE", "0")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("expected zero (unlimited) limits to be valid, got %v", err)
	}
	if cfg.ProviderLimits.ORSDailyQuota != 0 || cfg.ProviderLimits.ORSRatePerMinute != 0 {
		t.Fatalf("expected zero limits, got quota=%d rate=%d", cfg.ProviderLimits.ORSDailyQuota, cfg.ProviderLimits.ORSRatePerMinute)
	}
}

func TestProviderLimitsInvalidTimezoneRejected(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("PROVIDER_LIMITS_TIMEZONE", "Not/AZone")

	if _, err := Load(""); err == nil {
		t.Fatal("expected invalid timezone to be rejected")
	}
}
