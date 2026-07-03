package config

import (
	"fmt"
	"net/url"
	"os"
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
	MinProductionTokenLength     = 32
	MinProductionDBPassword      = 16

	// DefaultDevelopmentInternalToken is the shared service-to-service token used
	// for local development. Trip Service presents it on internal endpoints.
	DefaultDevelopmentInternalToken = "dev-internal-service-token"
)

// Config is the root application configuration. It is loaded from a YAML file
// (path passed via the -config flag) with environment-variable overrides, then
// validated.
type Config struct {
	Env        string          `yaml:"env" env:"APP_ENV" env-default:"local" validate:"required,oneof=local staging production development test"`
	HTTPServer HTTPServer      `yaml:"http_server" validate:"required"`
	Postgres   postgres.Config `yaml:"postgres" validate:"required"`
	JWT        JWTConfig       `yaml:"jwt" validate:"required"`
	Internal   InternalConfig  `yaml:"internal" validate:"required"`
	CORS       CORSConfig      `yaml:"cors" validate:"required"`
	Email      EmailConfig     `yaml:"email" validate:"required"`
	WebPush    WebPushConfig   `yaml:"web_push" validate:"required"`
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

// WebPushConfig controls browser Web Push delivery using VAPID. The private
// key must never be exposed to the Web App or logs.
type WebPushConfig struct {
	// Enabled turns browser push delivery on/off globally. It is normalized to
	// false when the VAPID key pair is incomplete.
	Enabled bool `yaml:"enabled" env:"WEB_PUSH_ENABLED" env-default:"false"`
	// VAPIDPublicKey is safe to expose to browsers for PushManager.subscribe.
	VAPIDPublicKey string `yaml:"vapid_public_key" env:"WEB_PUSH_VAPID_PUBLIC_KEY" env-default:""`
	// VAPIDPrivateKey signs VAPID JWTs and must remain secret.
	VAPIDPrivateKey string `yaml:"vapid_private_key" env:"WEB_PUSH_VAPID_PRIVATE_KEY" env-default:""`
	// Subject identifies the application server to push services.
	Subject string `yaml:"subject" env:"WEB_PUSH_SUBJECT" env-default:"mailto:support@example.com"`
	// TimeoutSeconds bounds each push-service request.
	TimeoutSeconds int `yaml:"timeout_seconds" env:"WEB_PUSH_TIMEOUT_SECONDS" env-default:"8" validate:"min=1"`
	// TTLSeconds is the push-service message TTL.
	TTLSeconds int `yaml:"ttl_seconds" env:"WEB_PUSH_TTL_SECONDS" env-default:"3600" validate:"min=0"`
	// Urgency is forwarded as the Web Push Urgency header.
	Urgency string `yaml:"urgency" env:"WEB_PUSH_URGENCY" env-default:"normal"`
	// FailOpen controls whether push failures fail the internal batch endpoint.
	FailOpen bool `yaml:"fail_open" env:"WEB_PUSH_FAIL_OPEN" env-default:"true"`
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
	AllowedMethods string `yaml:"allowed_methods" env:"CORS_ALLOWED_METHODS" env-default:"GET,POST,PUT,PATCH,DELETE,OPTIONS"`
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
	if err := cfg.validateWebPush(); err != nil {
		return nil, err
	}
	if err := cfg.validatePostgres(); err != nil {
		return nil, err
	}
	if err := cfg.validateCORS(); err != nil {
		return nil, err
	}
	if err := cfg.validatePublicURLs(); err != nil {
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

func (c *Config) IsStrictEnv() bool {
	return c.Env == "staging" || c.Env == "production"
}

// SSEHeartbeatInterval returns the configured heartbeat period.
func (c *Config) SSEHeartbeatInterval() time.Duration {
	return time.Duration(c.SSE.HeartbeatSeconds) * time.Second
}

// SSEWriteTimeout returns the per-event SSE write timeout.
func (c *Config) SSEWriteTimeout() time.Duration {
	return time.Duration(c.SSE.WriteTimeoutSeconds) * time.Second
}

// WebPushTimeout returns the per-push request timeout.
func (c *Config) WebPushTimeout() time.Duration {
	return time.Duration(c.WebPush.TimeoutSeconds) * time.Second
}

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.CORS.AllowedOrigins) == "" && isLocalEnv(c.Env) {
		c.CORS.AllowedOrigins = "http://localhost:3000"
	}
	if strings.TrimSpace(c.CORS.AllowedMethods) == "" {
		c.CORS.AllowedMethods = "GET,POST,PUT,PATCH,DELETE,OPTIONS"
	}
	if strings.TrimSpace(c.CORS.AllowedHeaders) == "" {
		c.CORS.AllowedHeaders = "Content-Type,Authorization"
	}
	c.WebPush.VAPIDPublicKey = strings.TrimSpace(c.WebPush.VAPIDPublicKey)
	c.WebPush.VAPIDPrivateKey = strings.TrimSpace(c.WebPush.VAPIDPrivateKey)
	c.WebPush.Subject = strings.TrimSpace(c.WebPush.Subject)
	c.WebPush.Urgency = strings.ToLower(strings.TrimSpace(c.WebPush.Urgency))
	if c.WebPush.Urgency == "" {
		c.WebPush.Urgency = "normal"
	}
	_, webPushEnabledExplicit := os.LookupEnv("WEB_PUSH_ENABLED")
	if !webPushEnabledExplicit && c.WebPush.VAPIDPublicKey != "" && c.WebPush.VAPIDPrivateKey != "" {
		c.WebPush.Enabled = true
	}
}

func (c *Config) validateJWTSecret() error {
	secret := strings.TrimSpace(c.JWT.AccessSecret)
	if secret == "" {
		return fmt.Errorf("JWT_ACCESS_SECRET is required")
	}
	if !c.IsStrictEnv() {
		c.JWT.AccessSecret = secret
		return nil
	}
	if isUnsafeSecret(secret, DefaultDevelopmentJWTSecret) {
		return fmt.Errorf("JWT_ACCESS_SECRET must not use a development default in %s", c.Env)
	}
	if len(secret) < MinProductionJWTSecretLength {
		return fmt.Errorf("JWT_ACCESS_SECRET must be at least %d characters in %s", MinProductionJWTSecretLength, c.Env)
	}
	c.JWT.AccessSecret = secret
	return nil
}

func (c *Config) validateInternalToken() error {
	token := strings.TrimSpace(c.Internal.ServiceToken)
	if token == "" {
		return fmt.Errorf("INTERNAL_SERVICE_TOKEN is required")
	}
	if c.IsStrictEnv() && isUnsafeSecret(token, DefaultDevelopmentInternalToken) {
		return fmt.Errorf("INTERNAL_SERVICE_TOKEN must not use a development default in %s", c.Env)
	}
	if c.IsStrictEnv() && len(token) < MinProductionTokenLength {
		return fmt.Errorf("INTERNAL_SERVICE_TOKEN must be at least %d characters in %s", MinProductionTokenLength, c.Env)
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
		if c.IsStrictEnv() && strings.TrimSpace(c.Email.SMTP.Password) == "" {
			return fmt.Errorf("SMTP_PASSWORD is required when EMAIL_PROVIDER=smtp in %s", c.Env)
		}
	}
	return nil
}

// validateWebPush enforces VAPID-key consistency. A missing full key pair keeps
// push disabled; a partial key pair is considered a configuration error because
// it almost always indicates a typo or incomplete secret rollout.
func (c *Config) validateWebPush() error {
	hasPublic := c.WebPush.VAPIDPublicKey != ""
	hasPrivate := c.WebPush.VAPIDPrivateKey != ""
	if hasPublic != hasPrivate {
		return fmt.Errorf("WEB_PUSH_VAPID_PUBLIC_KEY and WEB_PUSH_VAPID_PRIVATE_KEY must be configured together")
	}
	if !hasPublic || !hasPrivate {
		c.WebPush.Enabled = false
		return nil
	}
	switch c.WebPush.Urgency {
	case "very-low", "low", "normal", "high":
	default:
		return fmt.Errorf("WEB_PUSH_URGENCY must be one of very-low, low, normal, high")
	}
	if c.WebPush.Enabled && strings.TrimSpace(c.WebPush.Subject) == "" {
		return fmt.Errorf("WEB_PUSH_SUBJECT is required when web push is enabled")
	}
	if c.IsStrictEnv() && c.WebPush.Enabled && isUnsafeSecret(c.WebPush.VAPIDPrivateKey) {
		return fmt.Errorf("WEB_PUSH_VAPID_PRIVATE_KEY must not use a development default in %s", c.Env)
	}
	return nil
}

func (c *Config) validatePostgres() error {
	password := strings.TrimSpace(c.Postgres.Password)
	if password == "" {
		return fmt.Errorf("POSTGRES_PASSWORD is required")
	}
	if c.IsStrictEnv() {
		if isUnsafeSecret(password, "postgres") {
			return fmt.Errorf("POSTGRES_PASSWORD must not use a development default in %s", c.Env)
		}
		if len(password) < MinProductionDBPassword {
			return fmt.Errorf("POSTGRES_PASSWORD must be at least %d characters in %s", MinProductionDBPassword, c.Env)
		}
	}
	c.Postgres.Password = password
	return nil
}

func (c *Config) validateCORS() error {
	origins := strings.TrimSpace(c.CORS.AllowedOrigins)
	if origins == "" {
		if c.IsStrictEnv() {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS is required in %s", c.Env)
		}
		return nil
	}
	if c.IsStrictEnv() && origins == "*" {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS must not be wildcard in %s", c.Env)
	}
	if c.IsStrictEnv() {
		for _, origin := range strings.Split(origins, ",") {
			if err := validateHTTPURL("CORS_ALLOWED_ORIGINS", origin, false); err != nil {
				return err
			}
			if c.IsProduction() && isLocalhostURL(origin) {
				return fmt.Errorf("CORS_ALLOWED_ORIGINS must not use localhost in production")
			}
		}
	}
	c.CORS.AllowedOrigins = origins
	return nil
}

func (c *Config) validatePublicURLs() error {
	publicWeb := strings.TrimRight(strings.TrimSpace(c.Email.PublicWebBaseURL), "/")
	if publicWeb == "" {
		return fmt.Errorf("PUBLIC_WEB_BASE_URL is required")
	}
	if err := validateHTTPURL("PUBLIC_WEB_BASE_URL", publicWeb, c.IsProduction()); err != nil {
		if c.IsStrictEnv() || c.Email.Enabled {
			return err
		}
	}
	if c.IsProduction() && isLocalhostURL(publicWeb) {
		return fmt.Errorf("PUBLIC_WEB_BASE_URL must not use localhost in production")
	}
	c.Email.PublicWebBaseURL = publicWeb
	return nil
}

func validateHTTPURL(name, value string, requireHTTPS bool) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Errorf("%s contains an empty URL", name)
	}
	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must contain valid http/https URLs", name)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use http or https URLs", name)
	}
	if requireHTTPS && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use https in production", name)
	}
	return nil
}

func isLocalhostURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func isLocalEnv(env string) bool {
	return env == "local" || env == "development" || env == "test"
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
