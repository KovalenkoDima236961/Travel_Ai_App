package config

import "testing"

func setBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("HTTP_ADDRESS", ":8086")
	t.Setenv("POSTGRES_DB", "notification_service")
	t.Setenv("POSTGRES_USER", "postgres")
	t.Setenv("POSTGRES_PASSWORD", "postgres")
	t.Setenv("POSTGRES_HOST", "postgres")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_MIN_CONNS", "2")
	t.Setenv("POSTGRES_MAX_CONNS", "10")
	t.Setenv("POSTGRES_MIG_PATH", "./migrations")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
}

func TestLoadDevelopmentDefaults(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_ACCESS_SECRET", DefaultDevelopmentJWTSecret)
	t.Setenv("INTERNAL_SERVICE_TOKEN", DefaultDevelopmentInternalToken)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.HTTPServer.Address != ":8086" {
		t.Fatalf("unexpected http address %q", cfg.HTTPServer.Address)
	}
	if cfg.JWT.HeaderName != "Authorization" {
		t.Fatalf("unexpected jwt header name %q", cfg.JWT.HeaderName)
	}
	if cfg.Internal.ServiceToken != DefaultDevelopmentInternalToken {
		t.Fatalf("unexpected internal token %q", cfg.Internal.ServiceToken)
	}
}

func TestProductionRejectsDefaultJWTSecret(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_ACCESS_SECRET", DefaultDevelopmentJWTSecret)
	t.Setenv("INTERNAL_SERVICE_TOKEN", "a-strong-production-internal-token-value")

	if _, err := Load(""); err == nil {
		t.Fatal("expected production config to reject default JWT secret")
	}
}

func TestProductionRejectsDefaultInternalToken(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("JWT_ACCESS_SECRET", "a-strong-production-jwt-access-secret-value")
	t.Setenv("INTERNAL_SERVICE_TOKEN", DefaultDevelopmentInternalToken)

	if _, err := Load(""); err == nil {
		t.Fatal("expected production config to reject default internal token")
	}
}
