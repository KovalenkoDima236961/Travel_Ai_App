package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"

	tripconfig "github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
)

type Runtime struct {
	ServiceName            string `env:"SERVICE_NAME" env-default:"worker-service"`
	Enabled                bool   `env:"WORKER_ENABLED" env-default:"true"`
	HTTPAddress            string `env:"WORKER_HTTP_ADDR" env-default:":8090"`
	ShutdownTimeoutSeconds int    `env:"WORKER_SHUTDOWN_TIMEOUT_SECONDS" env-default:"30"`
	Concurrency            int    `env:"WORKER_CONCURRENCY" env-default:"1"`
	ScheduledJobsEnabled   bool   `env:"WORKER_SCHEDULED_JOBS_ENABLED" env-default:"true"`
}

type Config struct {
	Runtime            Runtime
	RabbitMQManagement RabbitMQManagement
	Reminders          ReminderWorker
	Digests            DigestWorker
	Cleanup            Cleanup
	Trip               *tripconfig.Config
}

// Cleanup keeps the worker as an orchestrator. Retention choices remain in
// configuration and are enforced by the service that owns the data.
type Cleanup struct {
	Enabled                bool   `env:"CLEANUP_JOBS_ENABLED" env-default:"true"`
	DryRunDefault          bool   `env:"CLEANUP_DRY_RUN_DEFAULT" env-default:"true"`
	BatchSize              int    `env:"CLEANUP_BATCH_SIZE" env-default:"500"`
	MaxBatchesPerRun       int    `env:"CLEANUP_MAX_BATCHES_PER_RUN" env-default:"20"`
	LockTTLSeconds         int    `env:"CLEANUP_LOCK_TTL_SECONDS" env-default:"3600"`
	ScheduleCron           string `env:"CLEANUP_SCHEDULE_CRON" env-default:"0 3 * * *"`
	FailOpen               bool   `env:"CLEANUP_FAIL_OPEN" env-default:"false"`
	TimeoutSeconds         int    `env:"CLEANUP_TIMEOUT_SECONDS" env-default:"30"`
	AuthServiceURL         string `env:"AUTH_SERVICE_INTERNAL_URL" env-default:"http://auth-service:8082"`
	NotificationServiceURL string `env:"NOTIFICATION_SERVICE_INTERNAL_URL" env-default:"http://notification-service:8086"`
	TripServiceURL         string `env:"TRIP_SERVICE_INTERNAL_URL" env-default:"http://trip-service:8080"`
	ExternalServiceURL     string `env:"EXTERNAL_INTEGRATIONS_SERVICE_INTERNAL_URL" env-default:"http://external-integrations-service:8084"`
}

type DigestWorker struct {
	Enabled                bool   `env:"NOTIFICATION_DIGEST_WORKER_ENABLED" env-default:"true"`
	NotificationServiceURL string `env:"NOTIFICATION_SERVICE_URL" env-default:"http://notification-service:8086"`
	PollIntervalSeconds    int    `env:"NOTIFICATION_DIGEST_WORKER_POLL_INTERVAL_SECONDS" env-default:"60"`
	BatchSize              int    `env:"NOTIFICATION_DIGEST_WORKER_BATCH_SIZE" env-default:"100"`
	TimeoutSeconds         int    `env:"NOTIFICATION_DIGEST_WORKER_TIMEOUT_SECONDS" env-default:"15"`
}

type ReminderWorker struct {
	Enabled             bool   `env:"REMINDER_WORKER_ENABLED" env-default:"true"`
	TripServiceURL      string `env:"TRIP_SERVICE_URL" env-default:"http://trip-service:8080"`
	PollIntervalSeconds int    `env:"REMINDER_WORKER_POLL_INTERVAL_SECONDS" env-default:"300"`
	BatchSize           int    `env:"REMINDER_WORKER_BATCH_SIZE" env-default:"100"`
	LookaheadMinutes    int    `env:"REMINDER_WORKER_LOOKAHEAD_MINUTES" env-default:"0"`
	TimeoutSeconds      int    `env:"REMINDER_WORKER_TIMEOUT_SECONDS" env-default:"10"`
}

type RabbitMQManagement struct {
	URL      string `env:"RABBITMQ_MANAGEMENT_URL" env-default:"http://rabbitmq:15672"`
	User     string `env:"RABBITMQ_MANAGEMENT_USER" env-default:"guest"`
	Password string `env:"RABBITMQ_MANAGEMENT_PASSWORD" env-default:"guest"`
}

func Load(tripConfigPath string) (*Config, error) {
	tripCfg, err := tripconfig.Load(tripConfigPath)
	if err != nil {
		return nil, err
	}

	var runtime Runtime
	if err := cleanenv.ReadEnv(&runtime); err != nil {
		return nil, fmt.Errorf("read worker env config: %w", err)
	}
	if runtime.ShutdownTimeoutSeconds < 1 {
		runtime.ShutdownTimeoutSeconds = 30
	}
	if runtime.Concurrency < 1 {
		runtime.Concurrency = 1
	}
	var management RabbitMQManagement
	if err := cleanenv.ReadEnv(&management); err != nil {
		return nil, fmt.Errorf("read rabbitmq management env config: %w", err)
	}
	if err := validateRabbitMQManagement(management, tripCfg.IsStrictEnv()); err != nil {
		return nil, err
	}
	var cleanup Cleanup
	if err := cleanenv.ReadEnv(&cleanup); err != nil {
		return nil, fmt.Errorf("read cleanup config: %w", err)
	}
	if err := validateCleanup(&cleanup, tripCfg.Env); err != nil {
		return nil, err
	}
	var reminders ReminderWorker
	if err := cleanenv.ReadEnv(&reminders); err != nil {
		return nil, fmt.Errorf("read reminder worker env config: %w", err)
	}
	if reminders.PollIntervalSeconds < 1 {
		reminders.PollIntervalSeconds = 300
	}
	if reminders.BatchSize < 1 {
		reminders.BatchSize = 100
	}
	if reminders.BatchSize > 500 {
		reminders.BatchSize = 500
	}
	if reminders.TimeoutSeconds < 1 {
		reminders.TimeoutSeconds = 10
	}
	if err := validateHTTPURL("TRIP_SERVICE_URL", reminders.TripServiceURL); err != nil {
		if tripCfg.IsStrictEnv() {
			return nil, err
		}
	}
	var digests DigestWorker
	if err := cleanenv.ReadEnv(&digests); err != nil {
		return nil, fmt.Errorf("read digest worker env config: %w", err)
	}
	if digests.PollIntervalSeconds < 1 {
		digests.PollIntervalSeconds = 60
	}
	if digests.BatchSize < 1 {
		digests.BatchSize = 100
	}
	if digests.BatchSize > 500 {
		digests.BatchSize = 500
	}
	if digests.TimeoutSeconds < 1 {
		digests.TimeoutSeconds = 15
	}
	if err := validateHTTPURL("NOTIFICATION_SERVICE_URL", digests.NotificationServiceURL); err != nil && tripCfg.IsStrictEnv() {
		return nil, err
	}

	return &Config{
		Runtime:            runtime,
		RabbitMQManagement: management,
		Reminders:          reminders,
		Digests:            digests,
		Cleanup:            cleanup,
		Trip:               tripCfg,
	}, nil
}

func (c Cleanup) LockTTL() time.Duration { return time.Duration(c.LockTTLSeconds) * time.Second }
func (c Cleanup) Timeout() time.Duration { return time.Duration(c.TimeoutSeconds) * time.Second }

func validateCleanup(c *Cleanup, appEnv string) error {
	if c == nil {
		return fmt.Errorf("cleanup config is required")
	}
	if strings.TrimSpace(appEnv) == "" {
		return fmt.Errorf("cleanup cannot run when APP_ENV is unknown")
	}
	if c.BatchSize < 1 || c.BatchSize > 1000 {
		return fmt.Errorf("CLEANUP_BATCH_SIZE must be between 1 and 1000")
	}
	if c.MaxBatchesPerRun < 1 || c.MaxBatchesPerRun > 100 {
		return fmt.Errorf("CLEANUP_MAX_BATCHES_PER_RUN must be between 1 and 100")
	}
	if c.LockTTLSeconds < 60 || c.LockTTLSeconds > 86400 {
		return fmt.Errorf("CLEANUP_LOCK_TTL_SECONDS must be between 60 and 86400")
	}
	if c.TimeoutSeconds < 1 || c.TimeoutSeconds > 300 {
		return fmt.Errorf("CLEANUP_TIMEOUT_SECONDS must be between 1 and 300")
	}
	parts := strings.Fields(c.ScheduleCron)
	if len(parts) != 5 || parts[2] != "*" || parts[3] != "*" || parts[4] != "*" {
		return fmt.Errorf("CLEANUP_SCHEDULE_CRON must use daily form 'minute hour * * *'")
	}
	minute, minuteErr := strconv.Atoi(parts[0])
	hour, hourErr := strconv.Atoi(parts[1])
	if minuteErr != nil || minute < 0 || minute > 59 || hourErr != nil || hour < 0 || hour > 23 {
		return fmt.Errorf("CLEANUP_SCHEDULE_CRON is invalid")
	}
	if !c.DryRunDefault && !envExplicitlySet("CLEANUP_DRY_RUN_DEFAULT") {
		return fmt.Errorf("destructive cleanup requires an explicit CLEANUP_DRY_RUN_DEFAULT=false")
	}
	if !c.Enabled {
		return nil
	}
	for name, value := range map[string]string{"AUTH_SERVICE_INTERNAL_URL": c.AuthServiceURL, "NOTIFICATION_SERVICE_INTERNAL_URL": c.NotificationServiceURL, "TRIP_SERVICE_INTERNAL_URL": c.TripServiceURL, "EXTERNAL_INTEGRATIONS_SERVICE_INTERNAL_URL": c.ExternalServiceURL} {
		if err := validateHTTPURL(name, value); err != nil {
			return err
		}
	}
	return nil
}

func envExplicitlySet(name string) bool { _, ok := os.LookupEnv(name); return ok }

func (c *Config) ShutdownTimeout() time.Duration {
	return time.Duration(c.Runtime.ShutdownTimeoutSeconds) * time.Second
}

func validateRabbitMQManagement(cfg RabbitMQManagement, strict bool) error {
	if strings.TrimSpace(cfg.URL) == "" {
		return fmt.Errorf("RABBITMQ_MANAGEMENT_URL is required")
	}
	parsed, err := url.Parse(strings.TrimSpace(cfg.URL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("RABBITMQ_MANAGEMENT_URL must be a valid http/https URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("RABBITMQ_MANAGEMENT_URL must use http or https")
	}
	if !strict {
		return nil
	}
	user := strings.TrimSpace(cfg.User)
	password := strings.TrimSpace(cfg.Password)
	if user == "" {
		return fmt.Errorf("RABBITMQ_MANAGEMENT_USER is required in staging or production")
	}
	if password == "" {
		return fmt.Errorf("RABBITMQ_MANAGEMENT_PASSWORD is required in staging or production")
	}
	if strings.EqualFold(user, "guest") || isUnsafeSecret(password, "guest") {
		return fmt.Errorf("RabbitMQ management credentials must not use guest defaults in staging or production")
	}
	if len(password) < 16 {
		return fmt.Errorf("RABBITMQ_MANAGEMENT_PASSWORD must be at least 16 characters in staging or production")
	}
	return nil
}

func validateHTTPURL(name, value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("%s is required", name)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must be a valid http/https URL", name)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use http or https", name)
	}
	return nil
}

func isUnsafeSecret(value string, additional ...string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	disallowed := []string{"secret", "password", "dev", "changeme", "change-me", "guest", "admin"}
	disallowed = append(disallowed, additional...)
	for _, item := range disallowed {
		if normalized == strings.ToLower(strings.TrimSpace(item)) {
			return true
		}
	}
	return false
}
