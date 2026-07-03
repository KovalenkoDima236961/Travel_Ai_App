package config

import "testing"

func setBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HTTP_ADDRESS", ":8083")
	t.Setenv("POSTGRES_DB", "user_service")
	t.Setenv("POSTGRES_USER", "travel_ai")
	t.Setenv("POSTGRES_PASSWORD", "a-strong-db-password-value")
	t.Setenv("POSTGRES_HOST", "postgres")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_MIN_CONNS", "2")
	t.Setenv("POSTGRES_MAX_CONNS", "10")
	t.Setenv("POSTGRES_MIG_PATH", "./migrations")
	t.Setenv("JWT_ACCESS_SECRET", "a-strong-production-jwt-access-secret-value")
	t.Setenv("INTERNAL_SERVICE_TOKEN", "a-strong-production-internal-service-token")
	t.Setenv("NOTIFICATION_SERVICE_TOKEN", "a-strong-production-notification-token")
	t.Setenv("AUTH_SERVICE_URL", "https://auth.example.com")
	t.Setenv("NOTIFICATION_SERVICE_URL", "https://notifications.example.com")
	t.Setenv("PUBLIC_WEB_BASE_URL", "https://app.example.com")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")
}

func TestProductionRejectsShortJWTSecret(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_ACCESS_SECRET", "short")

	if _, err := Load(""); err == nil {
		t.Fatal("expected short production JWT secret to be rejected")
	}
}

func TestProductionRejectsWildcardCORS(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")

	if _, err := Load(""); err == nil {
		t.Fatal("expected wildcard production CORS to be rejected")
	}
}

func TestProductionValidConfigPasses(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")

	if _, err := Load(""); err != nil {
		t.Fatalf("expected valid production config, got %v", err)
	}
}
