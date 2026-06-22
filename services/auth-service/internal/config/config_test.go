package config

import "testing"

func TestNewLoadsDevelopmentDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("HTTP_ADDRESS", ":8082")
	t.Setenv("POSTGRES_DB", "auth_service")
	t.Setenv("POSTGRES_USER", "postgres")
	t.Setenv("POSTGRES_PASSWORD", "postgres")
	t.Setenv("POSTGRES_HOST", "postgres")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_MIN_CONNS", "2")
	t.Setenv("POSTGRES_MAX_CONNS", "10")
	t.Setenv("POSTGRES_MIG_PATH", "./migrations")
	t.Setenv("JWT_ACCESS_SECRET", DefaultDevelopmentJWTSecret)
	t.Setenv("ACCESS_TOKEN_TTL_MINUTES", "15")
	t.Setenv("REFRESH_TOKEN_TTL_DAYS", "30")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	t.Setenv("CORS_ALLOWED_METHODS", "GET,POST,OPTIONS")
	t.Setenv("CORS_ALLOWED_HEADERS", "Content-Type,Authorization")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	if cfg.HTTPServer.Address != ":8082" {
		t.Fatalf("unexpected http address %q", cfg.HTTPServer.Address)
	}
	if cfg.AccessTokenTTL().Minutes() != 15 {
		t.Fatalf("unexpected access token ttl %s", cfg.AccessTokenTTL())
	}
}

func TestProductionRejectsDefaultJWTSecret(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("HTTP_ADDRESS", ":8082")
	t.Setenv("POSTGRES_DB", "auth_service")
	t.Setenv("POSTGRES_USER", "postgres")
	t.Setenv("POSTGRES_PASSWORD", "postgres")
	t.Setenv("POSTGRES_HOST", "postgres")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_MIN_CONNS", "2")
	t.Setenv("POSTGRES_MAX_CONNS", "10")
	t.Setenv("POSTGRES_MIG_PATH", "./migrations")
	t.Setenv("JWT_ACCESS_SECRET", DefaultDevelopmentJWTSecret)
	t.Setenv("ACCESS_TOKEN_TTL_MINUTES", "15")
	t.Setenv("REFRESH_TOKEN_TTL_DAYS", "30")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	t.Setenv("CORS_ALLOWED_METHODS", "GET,POST,OPTIONS")
	t.Setenv("CORS_ALLOWED_HEADERS", "Content-Type,Authorization")

	if _, err := Load(""); err == nil {
		t.Fatal("expected production config to reject default JWT secret")
	}
}
