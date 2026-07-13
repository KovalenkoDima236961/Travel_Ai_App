package config

import (
	"fmt"
	"net/url"
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
}

type Config struct {
	Runtime            Runtime
	RabbitMQManagement RabbitMQManagement
	Reminders          ReminderWorker
	Trip               *tripconfig.Config
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

	return &Config{
		Runtime:            runtime,
		RabbitMQManagement: management,
		Reminders:          reminders,
		Trip:               tripCfg,
	}, nil
}

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
