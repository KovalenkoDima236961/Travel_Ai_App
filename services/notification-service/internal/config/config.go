package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/storage/postgres"
)

const (
	// DefaultDevelopmentJWTSecret is the access-token secret used for local
	// development. It must never be used in production.
	DefaultDevelopmentJWTSecret = "change-me-in-development"
	// MinProductionJWTSecretLength is the minimum acceptable secret length in
	// production.
	MinProductionJWTSecretLength = 32

	// DefaultDevelopmentInternalToken is the shared service-to-service token used
	// for local development. Trip Service presents it on internal endpoints.
	DefaultDevelopmentInternalToken = "dev-internal-service-token"
)

// Config is the root application configuration. It is loaded from a YAML file
// (path passed via the -config flag) with environment-variable overrides, then
// validated.
type Config struct {
	Env        string          `yaml:"env" env:"APP_ENV" env-default:"development" validate:"required,oneof=development production"`
	HTTPServer HTTPServer      `yaml:"http_server" validate:"required"`
	Postgres   postgres.Config `yaml:"postgres" validate:"required"`
	JWT        JWTConfig       `yaml:"jwt" validate:"required"`
	Internal   InternalConfig  `yaml:"internal" validate:"required"`
	CORS       CORSConfig      `yaml:"cors" validate:"required"`
	Email      EmailConfig     `yaml:"email" validate:"required"`
	Users      UsersConfig     `yaml:"users" validate:"required"`
	SSE        SSEConfig       `yaml:"sse" validate:"required"`
}

// EmailConfig controls optional email delivery for selected notification types.
// Email is sent synchronously after in-app notification rows are created. The
// mock provider never sends external mail and is the local-dev default.
type EmailConfig struct {
	// Enabled turns email sending on/off globally.
	Enabled bool `yaml:"enabled" env:"EMAIL_NOTIFICATIONS_ENABLED" env-default:"true"`
	// FailOpen, when true, logs email errors but never fails in-app notification
	// creation. When false, a send failure surfaces as a 502 from the batch
	// endpoint (in-app rows are already committed and are never rolled back).
	FailOpen bool `yaml:"fail_open" env:"EMAIL_NOTIFICATIONS_FAIL_OPEN" env-default:"true"`
	// Provider selects the email sender implementation: "mock" or "smtp".
	Provider string `yaml:"provider" env:"EMAIL_PROVIDER" env-default:"mock" validate:"required,oneof=mock smtp"`
	// Types is the comma-separated allowlist of notification types that may
	// trigger an email. Anything outside this list is skipped.
	Types string `yaml:"types" env:"EMAIL_NOTIFICATION_TYPES" env-default:"collaboration_invited,comment_created,collaborator_role_changed,collaborator_removed"`
	// PublicWebBaseURL is used to build safe links back to the Web App.
	PublicWebBaseURL string     `yaml:"public_web_base_url" env:"PUBLIC_WEB_BASE_URL" env-default:"http://localhost:3000"`
	SMTP             SMTPConfig `yaml:"smtp"`
}

// SMTPConfig holds SMTP provider settings. SMTP_PASSWORD must never be logged.
type SMTPConfig struct {
	Host      string `yaml:"host" env:"SMTP_HOST" env-default:""`
	Port      int    `yaml:"port" env:"SMTP_PORT" env-default:"587"`
	Username  string `yaml:"username" env:"SMTP_USERNAME" env-default:""`
	Password  string `yaml:"password" env:"SMTP_PASSWORD" env-default:""`
	FromEmail string `yaml:"from_email" env:"SMTP_FROM_EMAIL" env-default:"no-reply@localhost"`
	FromName  string `yaml:"from_name" env:"SMTP_FROM_NAME" env-default:"AI Travel Planner"`
	UseTLS    bool   `yaml:"use_tls" env:"SMTP_USE_TLS" env-default:"true"`
}

// UsersConfig points at the service that owns user identity. In v1 Auth Service
// owns recipient email, so the user-lookup client targets AuthServiceURL. The
// shared internal service token (Internal.ServiceToken) authenticates the call.
// UserServiceURL is reserved for future display-name/profile enrichment.
type UsersConfig struct {
	AuthServiceURL string `yaml:"auth_service_url" env:"AUTH_SERVICE_URL" env-default:"http://auth-service:8082"`
	UserServiceURL string `yaml:"user_service_url" env:"USER_SERVICE_URL" env-default:"http://user-service:8083"`
	TimeoutSeconds int    `yaml:"timeout_seconds" env:"USER_LOOKUP_TIMEOUT_SECONDS" env-default:"5" validate:"min=1"`
}

// SSEConfig controls the authenticated Server-Sent Events stream for real-time
// in-app notification delivery. Delivery is in-memory and instance-local.
type SSEConfig struct {
	Enabled               bool `yaml:"enabled" env:"NOTIFICATION_SSE_ENABLED" env-default:"true"`
	HeartbeatSeconds      int  `yaml:"heartbeat_seconds" env:"NOTIFICATION_SSE_HEARTBEAT_SECONDS" env-default:"25" validate:"min=1"`
	WriteTimeoutSeconds   int  `yaml:"write_timeout_seconds" env:"NOTIFICATION_SSE_WRITE_TIMEOUT_SECONDS" env-default:"10" validate:"min=1"`
	MaxConnectionsPerUser int  `yaml:"max_connections_per_user" env:"NOTIFICATION_SSE_MAX_CONNECTIONS_PER_USER" env-default:"5" validate:"min=1"`
}

// HTTPServer holds the HTTP listener configuration.
type HTTPServer struct {
	Address         string        `yaml:"address" env:"HTTP_ADDRESS" env-default:":8086" validate:"required"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env:"HTTP_READ_TIMEOUT" env-default:"15s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env:"HTTP_WRITE_TIMEOUT" env-default:"15s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"15s"`
}

// JWTConfig controls local validation of Auth Service access tokens on
// user-facing endpoints. Issuer/Audience are optional and reserved for future
// stricter validation; v1 validates signature, expiry, and the subject claim.
type JWTConfig struct {
	AccessSecret string `yaml:"access_secret" env:"JWT_ACCESS_SECRET" env-default:"change-me-in-development"`
	Issuer       string `yaml:"issuer" env:"JWT_ISSUER" env-default:""`
	Audience     string `yaml:"audience" env:"JWT_AUDIENCE" env-default:""`
	HeaderName   string `yaml:"header_name" env:"AUTH_HEADER_NAME" env-default:"Authorization" validate:"required"`
}

// InternalConfig controls service-to-service authentication for internal
// endpoints (currently POST /internal/notifications/batch). Trip Service
// presents the shared token in the X-Internal-Service-Token header.
//
// This is a deliberately simple v1 scheme. It can be replaced later by mTLS,
// signed service tokens, or an event bus without touching callers.
type InternalConfig struct {
	ServiceToken string `yaml:"service_token" env:"INTERNAL_SERVICE_TOKEN" env-default:"dev-internal-service-token" validate:"required"`
}

// CORSConfig controls browser access to the Notification Service API.
type CORSConfig struct {
	AllowedOrigins string `yaml:"allowed_origins" env:"CORS_ALLOWED_ORIGINS" env-default:"http://localhost:3000"`
	AllowedMethods string `yaml:"allowed_methods" env:"CORS_ALLOWED_METHODS" env-default:"GET,PUT,PATCH,OPTIONS"`
	AllowedHeaders string `yaml:"allowed_headers" env:"CORS_ALLOWED_HEADERS" env-default:"Content-Type,Authorization"`
}

// MustLoad loads and validates the configuration, panicking on any error.
// It is intended for use during application bootstrap.
func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(fmt.Errorf("config: %w", err))
	}
	return cfg
}

// Load reads configuration from the given YAML path (or environment only when
// path is empty) and validates it.
func Load(path string) (*Config, error) {
	var cfg Config

	if path != "" {
		if err := cleanenv.ReadConfig(path, &cfg); err != nil {
			return nil, fmt.Errorf("read config file %q: %w", path, err)
		}
	} else if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("read env config: %w", err)
	}

	cfg.applyDefaults()

	v := validator.New()
	if err := v.Struct(cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	if err := cfg.validateJWTSecret(); err != nil {
		return nil, err
	}
	if err := cfg.validateInternalToken(); err != nil {
		return nil, err
	}
	if err := cfg.validateEmail(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// EmailNotificationTypes parses the comma-separated allowlist into a trimmed,
// lower-cased slice. Empty entries are dropped.
func (c *Config) EmailNotificationTypes() []string {
	parts := strings.Split(c.Email.Types, ",")
	types := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			types = append(types, trimmed)
		}
	}
	return types
}

// IsProduction reports whether the service runs in a production profile.
func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

// SSEHeartbeatInterval returns the configured heartbeat period.
func (c *Config) SSEHeartbeatInterval() time.Duration {
	return time.Duration(c.SSE.HeartbeatSeconds) * time.Second
}

// SSEWriteTimeout returns the per-event SSE write timeout.
func (c *Config) SSEWriteTimeout() time.Duration {
	return time.Duration(c.SSE.WriteTimeoutSeconds) * time.Second
}

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.CORS.AllowedOrigins) == "" && c.Env == "development" {
		c.CORS.AllowedOrigins = "http://localhost:3000"
	}
	if strings.TrimSpace(c.CORS.AllowedMethods) == "" {
		c.CORS.AllowedMethods = "GET,PUT,PATCH,OPTIONS"
	}
	if strings.TrimSpace(c.CORS.AllowedHeaders) == "" {
		c.CORS.AllowedHeaders = "Content-Type,Authorization"
	}
}

func (c *Config) validateJWTSecret() error {
	secret := strings.TrimSpace(c.JWT.AccessSecret)
	if secret == "" {
		return fmt.Errorf("JWT_ACCESS_SECRET is required")
	}
	if !c.IsProduction() {
		c.JWT.AccessSecret = secret
		return nil
	}
	if secret == DefaultDevelopmentJWTSecret {
		return fmt.Errorf("JWT_ACCESS_SECRET must not use the development default in production")
	}
	if len(secret) < MinProductionJWTSecretLength {
		return fmt.Errorf("JWT_ACCESS_SECRET must be at least %d characters in production", MinProductionJWTSecretLength)
	}
	c.JWT.AccessSecret = secret
	return nil
}

func (c *Config) validateInternalToken() error {
	token := strings.TrimSpace(c.Internal.ServiceToken)
	if token == "" {
		return fmt.Errorf("INTERNAL_SERVICE_TOKEN is required")
	}
	if c.IsProduction() && token == DefaultDevelopmentInternalToken {
		return fmt.Errorf("INTERNAL_SERVICE_TOKEN must not use the development default in production")
	}
	c.Internal.ServiceToken = token
	return nil
}

// validateEmail enforces provider-specific requirements. The SMTP provider needs
// at least a host and a from-address; the mock provider has no requirements.
func (c *Config) validateEmail() error {
	provider := strings.ToLower(strings.TrimSpace(c.Email.Provider))
	c.Email.Provider = provider
	if provider == "smtp" {
		if strings.TrimSpace(c.Email.SMTP.Host) == "" {
			return fmt.Errorf("SMTP_HOST is required when EMAIL_PROVIDER=smtp")
		}
		if strings.TrimSpace(c.Email.SMTP.FromEmail) == "" {
			return fmt.Errorf("SMTP_FROM_EMAIL is required when EMAIL_PROVIDER=smtp")
		}
	}
	return nil
}
