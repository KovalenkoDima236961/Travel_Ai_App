package config

import "testing"

func setBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", "staging")
	t.Setenv("HTTP_ADDRESS", ":8080")
	t.Setenv("POSTGRES_DB", "trip_service")
	t.Setenv("POSTGRES_USER", "travel_ai")
	t.Setenv("POSTGRES_PASSWORD", "a-strong-db-password-value")
	t.Setenv("POSTGRES_HOST", "postgres")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_MIN_CONNS", "2")
	t.Setenv("POSTGRES_MAX_CONNS", "10")
	t.Setenv("POSTGRES_MIG_PATH", "../trip-service/migrations")
	t.Setenv("JWT_ACCESS_SECRET", "a-strong-production-jwt-access-secret-value")
	t.Setenv("INTERNAL_SERVICE_TOKEN", "a-strong-production-internal-token-value")
	t.Setenv("EXTERNAL_INTEGRATIONS_SERVICE_TOKEN", "a-strong-production-internal-token-value")
	t.Setenv("NOTIFICATION_SERVICE_TOKEN", "a-strong-production-internal-token-value")
	t.Setenv("PUBLIC_SHARE_ACCESS_SECRET", "a-strong-public-share-token-value")
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://app.example.com")
	t.Setenv("PUBLIC_WEB_BASE_URL", "https://app.example.com")
	t.Setenv("GENERATION_JOB_DISPATCH_MODE", "queue")
	t.Setenv("RABBITMQ_URL", "amqp://travel_ai:a-strong-rabbitmq-password@rabbitmq:5672/")
	t.Setenv("RABBITMQ_MANAGEMENT_URL", "http://rabbitmq:15672")
	t.Setenv("RABBITMQ_MANAGEMENT_USER", "travel_ai")
	t.Setenv("RABBITMQ_MANAGEMENT_PASSWORD", "a-strong-rabbitmq-password")
	t.Setenv("OPS_DASHBOARD_ENABLED", "false")
}

func TestStrictEnvRejectsGuestRabbitMQManagementCredentials(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("RABBITMQ_MANAGEMENT_USER", "guest")
	t.Setenv("RABBITMQ_MANAGEMENT_PASSWORD", "guest")

	if _, err := Load(""); err == nil {
		t.Fatal("expected guest RabbitMQ management credentials to be rejected")
	}
}

func TestStrictEnvValidConfigPasses(t *testing.T) {
	setBaseEnv(t)

	if _, err := Load(""); err != nil {
		t.Fatalf("expected valid worker config, got %v", err)
	}
}

func TestCleanupConfigRejectsUnsafeBatchAndCron(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("CLEANUP_BATCH_SIZE", "1001")
	if _, err := Load(""); err == nil {
		t.Fatal("expected oversized cleanup batch to be rejected")
	}

	setBaseEnv(t)
	t.Setenv("CLEANUP_SCHEDULE_CRON", "* * * * *")
	if _, err := Load(""); err == nil {
		t.Fatal("expected non-daily cleanup cron to be rejected")
	}
}
