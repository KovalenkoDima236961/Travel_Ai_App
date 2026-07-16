package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/pkg/storage/postgres"
)

const (
	DefaultDevelopmentJWTSecret  = "change-me-in-development"
	MinProductionJWTSecretLength = 32
	MinProductionTokenLength     = 32
	MinProductionDBPassword      = 16

	// DefaultDevelopmentInternalToken is the shared service-to-service token used
	// for local development. Internal callers (e.g. Notification Service resolving
	// recipient emails) present it on internal endpoints.
	DefaultDevelopmentInternalToken = "dev-internal-service-token"
)

type Config struct {
	Env        string               `yaml:"env" env:"APP_ENV" env-default:"local" validate:"required,oneof=local staging production development test"`
	HTTPServer HTTPServer           `yaml:"http_server" validate:"required"`
	Postgres   postgres.Config      `yaml:"postgres" validate:"required"`
	JWT        JWTConfig            `yaml:"jwt" validate:"required"`
	Internal   InternalConfig       `yaml:"internal" validate:"required"`
	RateLimits AuthRateLimitsConfig `yaml:"rate_limits" validate:"required"`
	CORS       CORSConfig           `yaml:"cors" validate:"required"`
}

type AuthRateLimitsConfig struct {
	LoginPerMinute    int `yaml:"login_per_minute" env:"AUTH_LOGIN_RATE_LIMIT_PER_MINUTE" env-default:"10" validate:"min=1,max=10000"`
	RegisterPerMinute int `yaml:"register_per_minute" env:"AUTH_REGISTER_RATE_LIMIT_PER_MINUTE" env-default:"10" validate:"min=1,max=10000"`
	RefreshPerMinute  int `yaml:"refresh_per_minute" env:"AUTH_REFRESH_RATE_LIMIT_PER_MINUTE" env-default:"30" validate:"min=1,max=10000"`
}

// InternalConfig controls service-to-service authentication for internal
// endpoints (currently POST /internal/users/batch). Callers present the shared
// token in the X-Internal-Service-Token header. This mirrors the v1 scheme used
// by Notification Service and can be replaced later by mTLS or signed tokens.
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

type HTTPServer struct {
	Address         string        `yaml:"address" env:"HTTP_ADDRESS" env-default:":8082" validate:"required"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env:"HTTP_READ_TIMEOUT" env-default:"15s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env:"HTTP_WRITE_TIMEOUT" env-default:"15s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"15s"`
}

type JWTConfig struct {
	AccessSecret          string `yaml:"access_secret" env:"JWT_ACCESS_SECRET" env-default:"change-me-in-development"`
	AccessTokenTTLMinutes int    `yaml:"access_token_ttl_minutes" env:"ACCESS_TOKEN_TTL_MINUTES" env-default:"15" validate:"min=1"`
	RefreshTokenTTLDays   int    `yaml:"refresh_token_ttl_days" env:"REFRESH_TOKEN_TTL_DAYS" env-default:"30" validate:"min=1"`
}

type CORSConfig struct {
	AllowedOrigins string `yaml:"allowed_origins" env:"CORS_ALLOWED_ORIGINS" env-default:"http://localhost:3000"`
	AllowedMethods string `yaml:"allowed_methods" env:"CORS_ALLOWED_METHODS" env-default:"GET,POST,OPTIONS"`
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
	if err := cfg.validatePostgres(); err != nil {
		return nil, err
	}
	if err := cfg.validateCORS(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func (c *Config) IsStrictEnv() bool {
	return c.Env == "staging" || c.Env == "production"
}

func (c *Config) AccessTokenTTL() time.Duration {
	return time.Duration(c.JWT.AccessTokenTTLMinutes) * time.Minute
}

func (c *Config) RefreshTokenTTL() time.Duration {
	return time.Duration(c.JWT.RefreshTokenTTLDays) * 24 * time.Hour
}

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.CORS.AllowedOrigins) == "" && isLocalEnv(c.Env) {
		c.CORS.AllowedOrigins = "http://localhost:3000"
	}
	if strings.TrimSpace(c.CORS.AllowedMethods) == "" {
		c.CORS.AllowedMethods = "GET,POST,OPTIONS"
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
	if err := validateRotatingTokens(c.Internal.ServiceTokens, c.Env, c.IsStrictEnv()); err != nil {
		return err
	}
	return nil
}

func validateRotatingTokens(value, env string, strict bool) error {
	for _, raw := range strings.Split(value, ",") {
		token := strings.TrimSpace(raw)
		if token == "" {
			continue
		}
		if strict && (isUnsafeSecret(token, DefaultDevelopmentInternalToken) || len(token) < MinProductionTokenLength) {
			return fmt.Errorf("INTERNAL_SERVICE_TOKENS contains an unsafe token in %s", env)
		}
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
