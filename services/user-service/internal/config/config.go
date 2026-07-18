package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/pkg/storage/postgres"
)

const (
	DefaultDevelopmentJWTSecret     = "change-me-in-development"
	DefaultDevelopmentInternalToken = "dev-internal-service-token"
	MinProductionJWTSecretLength    = 32
	MinProductionTokenLength        = 32
	MinProductionDBPassword         = 16
)

// Config is the root application configuration. It is loaded from a YAML file
// with environment-variable overrides, then validated during bootstrap.
type Config struct {
	Env           string              `yaml:"env" env:"APP_ENV" env-default:"local" validate:"required,oneof=local staging production development test"`
	HTTPServer    HTTPServer          `yaml:"http_server" validate:"required"`
	Auth          AuthConfig          `yaml:"auth" validate:"required"`
	Internal      InternalConfig      `yaml:"internal" validate:"required"`
	CORS          CORSConfig          `yaml:"cors" validate:"required"`
	Postgres      postgres.Config     `yaml:"postgres" validate:"required"`
	AuthUsers     AuthUsersConfig     `yaml:"auth_users" validate:"required"`
	Notifications NotificationsConfig `yaml:"notifications" validate:"required"`
	DataExports   DataExportsConfig   `yaml:"data_exports"`
	TripExports   TripExportsConfig   `yaml:"trip_exports"`
}

// DataExportsConfig controls private, authenticated account export packages.
// It has no public URL, cloud-storage, or deletion controls.
type DataExportsConfig struct {
	Enabled                bool   `yaml:"enabled" env:"DATA_EXPORT_ENABLED" env-default:"true"`
	StorageDir             string `yaml:"storage_dir" env:"DATA_EXPORT_STORAGE_DIR" env-default:"./data/exports"`
	TTLHours               int    `yaml:"ttl_hours" env:"DATA_EXPORT_TTL_HOURS" env-default:"24" validate:"min=1,max=168"`
	MaxAccountExportMB     int    `yaml:"max_account_export_mb" env:"DATA_EXPORT_MAX_ACCOUNT_EXPORT_MB" env-default:"250" validate:"min=1,max=500"`
	CleanupEnabled         bool   `yaml:"cleanup_enabled" env:"DATA_EXPORT_CLEANUP_ENABLED" env-default:"true"`
	CleanupIntervalMinutes int    `yaml:"cleanup_interval_minutes" env:"DATA_EXPORT_CLEANUP_INTERVAL_MINUTES" env-default:"60" validate:"min=1,max=1440"`
}

// TripExportsConfig controls the private service-to-service handoff that adds
// authorized Trip Service archives to an account package.
type TripExportsConfig struct {
	Enabled        bool   `yaml:"enabled" env:"ACCOUNT_EXPORT_TRIP_DATA_ENABLED" env-default:"true"`
	TripServiceURL string `yaml:"trip_service_url" env:"TRIP_SERVICE_URL" env-default:"http://trip-service:8080"`
	ServiceToken   string `yaml:"service_token" env:"TRIP_SERVICE_TOKEN" env-default:"dev-internal-service-token"`
	TimeoutSeconds int    `yaml:"timeout_seconds" env:"TRIP_EXPORT_TIMEOUT_SECONDS" env-default:"60" validate:"min=1,max=300"`
}

// HTTPServer holds the HTTP listener configuration.
type HTTPServer struct {
	Address         string        `yaml:"address" env:"HTTP_ADDRESS" env-default:":8083" validate:"required"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env:"HTTP_READ_TIMEOUT" env-default:"15s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env:"HTTP_WRITE_TIMEOUT" env-default:"15s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"15s"`
}

// AuthConfig controls local JWT validation for protected user endpoints.
type AuthConfig struct {
	Required        bool   `yaml:"required" env:"AUTH_REQUIRED" env-default:"true"`
	JWTAccessSecret string `yaml:"jwt_access_secret" env:"JWT_ACCESS_SECRET" env-default:"change-me-in-development"`
	HeaderName      string `yaml:"header_name" env:"AUTH_HEADER_NAME" env-default:"Authorization" validate:"required"`
	DevUserID       string `yaml:"dev_user_id" env:"DEV_USER_ID" env-default:"00000000-0000-0000-0000-000000000001" validate:"required,uuid"`
}

// InternalConfig controls service-to-service endpoints such as workspace access
// checks consumed by Trip Service.
type InternalConfig struct {
	ServiceToken  string `yaml:"service_token" env:"INTERNAL_SERVICE_TOKEN" env-default:"dev-internal-service-token" validate:"required"`
	ServiceTokens string `yaml:"service_tokens" env:"INTERNAL_SERVICE_TOKENS"`
}

func (c InternalConfig) ActiveServiceTokens() string {
	if tokens := strings.TrimSpace(c.ServiceTokens); tokens != "" {
		return tokens
	}
	return c.ServiceToken
}

// AuthUsersConfig points at Auth Service for email/user lookup enrichment.
type AuthUsersConfig struct {
	AuthServiceURL string `yaml:"auth_service_url" env:"AUTH_SERVICE_URL" env-default:"http://auth-service:8082"`
	TimeoutSeconds int    `yaml:"timeout_seconds" env:"AUTH_USER_LOOKUP_TIMEOUT_SECONDS" env-default:"5" validate:"min=1"`
}

// NotificationsConfig controls workspace notification fan-out.
type NotificationsConfig struct {
	Enabled                  bool   `yaml:"enabled" env:"USER_WORKSPACE_NOTIFICATIONS_ENABLED" env-default:"true"`
	FailOpen                 bool   `yaml:"fail_open" env:"USER_WORKSPACE_NOTIFICATIONS_FAIL_OPEN" env-default:"true"`
	NotificationServiceURL   string `yaml:"notification_service_url" env:"NOTIFICATION_SERVICE_URL" env-default:"http://notification-service:8086"`
	NotificationServiceToken string `yaml:"notification_service_token" env:"NOTIFICATION_SERVICE_TOKEN" env-default:"dev-internal-service-token"`
	PublicWebBaseURL         string `yaml:"public_web_base_url" env:"PUBLIC_WEB_BASE_URL" env-default:"http://localhost:3000"`
	TimeoutSeconds           int    `yaml:"timeout_seconds" env:"NOTIFICATION_SERVICE_TIMEOUT_SECONDS" env-default:"3" validate:"min=1"`
}

// CORSConfig controls browser access to the User Service API.
type CORSConfig struct {
	AllowedOrigins string `yaml:"allowed_origins" env:"CORS_ALLOWED_ORIGINS" env-default:"http://localhost:3000"`
	AllowedMethods string `yaml:"allowed_methods" env:"CORS_ALLOWED_METHODS" env-default:"GET,POST,PUT,PATCH,DELETE,OPTIONS"`
	AllowedHeaders string `yaml:"allowed_headers" env:"CORS_ALLOWED_HEADERS" env-default:"Content-Type,Authorization"`
}

// Load reads configuration from the given YAML path, or environment only when
// path is empty, and validates it.
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
	if err := cfg.validateServiceURLs(); err != nil {
		return nil, err
	}
	if err := cfg.validatePostgres(); err != nil {
		return nil, err
	}
	if err := cfg.validateCORS(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// IsProduction reports whether the service runs in a production profile.
func (c *Config) IsProduction() bool { return c.Env == "production" }

func (c *Config) IsStrictEnv() bool { return c.Env == "staging" || c.Env == "production" }

// UsesDefaultDevelopmentJWTSecret reports whether a warning should be logged.
func (c *Config) UsesDefaultDevelopmentJWTSecret() bool {
	return !c.IsProduction() && c.Auth.JWTAccessSecret == DefaultDevelopmentJWTSecret
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
}

func (c *Config) validateInternalToken() error {
	token := strings.TrimSpace(c.Internal.ServiceToken)
	if token == "" {
		return fmt.Errorf("INTERNAL_SERVICE_TOKEN is required")
	}
	if c.IsStrictEnv() {
		if isUnsafeSecret(token, DefaultDevelopmentInternalToken) {
			return fmt.Errorf("INTERNAL_SERVICE_TOKEN must not use a development default in %s", c.Env)
		}
		if len(token) < MinProductionTokenLength {
			return fmt.Errorf("INTERNAL_SERVICE_TOKEN must be at least %d characters in %s", MinProductionTokenLength, c.Env)
		}
	}
	c.Internal.ServiceToken = token
	for _, raw := range strings.Split(c.Internal.ServiceTokens, ",") {
		rotatingToken := strings.TrimSpace(raw)
		if rotatingToken == "" {
			continue
		}
		if c.IsStrictEnv() && (isUnsafeSecret(rotatingToken, DefaultDevelopmentInternalToken) || len(rotatingToken) < MinProductionTokenLength) {
			return fmt.Errorf("INTERNAL_SERVICE_TOKENS contains an unsafe token in %s", c.Env)
		}
	}
	return nil
}

func (c *Config) validateServiceURLs() error {
	checks := []struct {
		name         string
		value        string
		enabled      bool
		requireHTTPS bool
	}{
		{"AUTH_SERVICE_URL", c.AuthUsers.AuthServiceURL, true, false},
		{"NOTIFICATION_SERVICE_URL", c.Notifications.NotificationServiceURL, c.Notifications.Enabled, false},
		{"PUBLIC_WEB_BASE_URL", c.Notifications.PublicWebBaseURL, c.Notifications.Enabled, c.IsProduction()},
		{"TRIP_SERVICE_URL", c.TripExports.TripServiceURL, c.TripExports.Enabled, false},
	}
	for _, check := range checks {
		if !check.enabled {
			continue
		}
		if err := validateHTTPURL(check.name, check.value, check.requireHTTPS); err != nil {
			if c.IsStrictEnv() || check.name == "PUBLIC_WEB_BASE_URL" {
				return err
			}
		}
	}
	if c.Notifications.Enabled {
		token := strings.TrimSpace(c.Notifications.NotificationServiceToken)
		if token == "" {
			return fmt.Errorf("NOTIFICATION_SERVICE_TOKEN is required when workspace notifications are enabled")
		}
		if c.IsStrictEnv() {
			if isUnsafeSecret(token, DefaultDevelopmentInternalToken) {
				return fmt.Errorf("NOTIFICATION_SERVICE_TOKEN must not use a development default in %s", c.Env)
			}
			if len(token) < MinProductionTokenLength {
				return fmt.Errorf("NOTIFICATION_SERVICE_TOKEN must be at least %d characters in %s", MinProductionTokenLength, c.Env)
			}
		}
		c.Notifications.NotificationServiceToken = token
	}
	if c.TripExports.Enabled {
		token := strings.TrimSpace(c.TripExports.ServiceToken)
		if c.IsStrictEnv() && isUnsafeSecret(token, DefaultDevelopmentInternalToken) {
			// The Trip handoff uses the same internal trust boundary as existing
			// workspace calls. This keeps established production deployments
			// compatible while still rejecting a weak effective token below.
			token = c.Internal.ServiceToken
		}
		if token == "" {
			return fmt.Errorf("TRIP_SERVICE_TOKEN is required when account trip exports are enabled")
		}
		if c.IsStrictEnv() && (isUnsafeSecret(token, DefaultDevelopmentInternalToken) || len(token) < MinProductionTokenLength) {
			return fmt.Errorf("TRIP_SERVICE_TOKEN must be a non-development token of at least %d characters in %s", MinProductionTokenLength, c.Env)
		}
		c.TripExports.ServiceToken = token
	}
	c.AuthUsers.AuthServiceURL = strings.TrimRight(strings.TrimSpace(c.AuthUsers.AuthServiceURL), "/")
	c.Notifications.NotificationServiceURL = strings.TrimRight(strings.TrimSpace(c.Notifications.NotificationServiceURL), "/")
	c.Notifications.PublicWebBaseURL = strings.TrimRight(strings.TrimSpace(c.Notifications.PublicWebBaseURL), "/")
	c.TripExports.TripServiceURL = strings.TrimRight(strings.TrimSpace(c.TripExports.TripServiceURL), "/")
	return nil
}

func (c *Config) validateJWTSecret() error {
	secret := strings.TrimSpace(c.Auth.JWTAccessSecret)
	if secret == "" {
		return fmt.Errorf("JWT_ACCESS_SECRET is required")
	}
	if !c.IsStrictEnv() {
		c.Auth.JWTAccessSecret = secret
		return nil
	}
	if isUnsafeSecret(secret, DefaultDevelopmentJWTSecret) {
		return fmt.Errorf("JWT_ACCESS_SECRET must not use a development default in %s", c.Env)
	}
	if len(secret) < MinProductionJWTSecretLength {
		return fmt.Errorf("JWT_ACCESS_SECRET must be at least %d characters in %s", MinProductionJWTSecretLength, c.Env)
	}
	c.Auth.JWTAccessSecret = secret
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

func isLocalEnv(env string) bool {
	return env == "local" || env == "development" || env == "test"
}

func isLocalhostURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
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
