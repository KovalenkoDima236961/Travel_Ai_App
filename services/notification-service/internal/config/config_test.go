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
	if !cfg.SSE.Enabled {
		t.Fatal("expected SSE enabled by default")
	}
	if cfg.SSE.HeartbeatSeconds != 25 || cfg.SSE.WriteTimeoutSeconds != 10 || cfg.SSE.MaxConnectionsPerUser != 5 {
		t.Fatalf("unexpected SSE defaults: %+v", cfg.SSE)
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

func setDevSecrets(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_ACCESS_SECRET", DefaultDevelopmentJWTSecret)
	t.Setenv("INTERNAL_SERVICE_TOKEN", DefaultDevelopmentInternalToken)
}

func TestEmailDefaults(t *testing.T) {
	setBaseEnv(t)
	setDevSecrets(t)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Email.Provider != "mock" {
		t.Fatalf("expected default provider mock, got %q", cfg.Email.Provider)
	}
	if !cfg.Email.Enabled || !cfg.Email.FailOpen {
		t.Fatalf("expected email enabled+fail-open by default, got enabled=%v failOpen=%v", cfg.Email.Enabled, cfg.Email.FailOpen)
	}
	types := cfg.EmailNotificationTypes()
	if len(types) != 8 ||
		types[0] != "collaboration_invited" ||
		types[4] != "workspace_invited" ||
		types[5] != "workspace_member_removed" ||
		types[6] != "workspace_role_changed" ||
		types[7] != "pre_trip_reminder_due" {
		t.Fatalf("unexpected default email types: %v", types)
	}
}

func TestUnsupportedEmailProviderRejected(t *testing.T) {
	setBaseEnv(t)
	setDevSecrets(t)
	t.Setenv("EMAIL_PROVIDER", "carrier-pigeon")

	if _, err := Load(""); err == nil {
		t.Fatal("expected unsupported EMAIL_PROVIDER to be rejected")
	}
}

func TestSMTPProviderRequiresHostAndFrom(t *testing.T) {
	setBaseEnv(t)
	setDevSecrets(t)
	t.Setenv("EMAIL_PROVIDER", "smtp")
	t.Setenv("SMTP_HOST", "")
	t.Setenv("SMTP_FROM_EMAIL", "no-reply@example.com")

	if _, err := Load(""); err == nil {
		t.Fatal("expected smtp provider with empty SMTP_HOST to be rejected")
	}

	t.Setenv("SMTP_HOST", "smtp.example.com")
	t.Setenv("SMTP_FROM_EMAIL", "")
	if _, err := Load(""); err == nil {
		t.Fatal("expected smtp provider with empty SMTP_FROM_EMAIL to be rejected")
	}

	t.Setenv("SMTP_FROM_EMAIL", "no-reply@example.com")
	if _, err := Load(""); err != nil {
		t.Fatalf("expected valid smtp config to load, got %v", err)
	}
}

func TestWebPushDefaultsAndVAPIDKeyNormalization(t *testing.T) {
	setBaseEnv(t)
	setDevSecrets(t)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.WebPush.Enabled {
		t.Fatal("expected web push disabled without VAPID keys")
	}
	if cfg.WebPush.TimeoutSeconds != 8 || cfg.WebPush.TTLSeconds != 3600 || cfg.WebPush.Urgency != "normal" || !cfg.WebPush.FailOpen {
		t.Fatalf("unexpected web push defaults: %+v", cfg.WebPush)
	}

	t.Setenv("WEB_PUSH_VAPID_PUBLIC_KEY", " public ")
	t.Setenv("WEB_PUSH_VAPID_PRIVATE_KEY", " private ")
	cfg, err = Load("")
	if err != nil {
		t.Fatalf("Load with VAPID keys returned error: %v", err)
	}
	if !cfg.WebPush.Enabled || cfg.WebPush.VAPIDPublicKey != "public" || cfg.WebPush.VAPIDPrivateKey != "private" {
		t.Fatalf("expected VAPID keys to enable and trim web push, got %+v", cfg.WebPush)
	}

	t.Setenv("WEB_PUSH_ENABLED", "false")
	cfg, err = Load("")
	if err != nil {
		t.Fatalf("Load with explicitly disabled Web Push returned error: %v", err)
	}
	if cfg.WebPush.Enabled {
		t.Fatal("expected explicit WEB_PUSH_ENABLED=false to keep web push disabled")
	}
}

func TestWebPushRejectsPartialVAPIDKeyPair(t *testing.T) {
	setBaseEnv(t)
	setDevSecrets(t)
	t.Setenv("WEB_PUSH_VAPID_PUBLIC_KEY", "public")

	if _, err := Load(""); err == nil {
		t.Fatal("expected partial VAPID key pair to be rejected")
	}
}
