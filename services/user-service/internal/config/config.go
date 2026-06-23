package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/pkg/storage/postgres"
)

const (
	DefaultDevelopmentJWTSecret  = "change-me-in-development"
	MinProductionJWTSecretLength = 32
)

// Config is the root application configuration. It is loaded from a YAML file
// with environment-variable overrides, then validated during bootstrap.
type Config struct {
	Env        string          `yaml:"env" env:"APP_ENV" env-default:"development" validate:"required,oneof=development production"`
	HTTPServer HTTPServer      `yaml:"http_server" validate:"required"`
	Auth       AuthConfig      `yaml:"auth" validate:"required"`
	CORS       CORSConfig      `yaml:"cors" validate:"required"`
	Postgres   postgres.Config `yaml:"postgres" validate:"required"`
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

// CORSConfig controls browser access to the User Service API.
type CORSConfig struct {
	AllowedOrigins string `yaml:"allowed_origins" env:"CORS_ALLOWED_ORIGINS" env-default:"http://localhost:3000"`
	AllowedMethods string `yaml:"allowed_methods" env:"CORS_ALLOWED_METHODS" env-default:"GET,PUT,PATCH,OPTIONS"`
	AllowedHeaders string `yaml:"allowed_headers" env:"CORS_ALLOWED_HEADERS" env-default:"Content-Type,Authorization"`
}

// MustLoad loads and validates the configuration, panicking on any error.
func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(fmt.Errorf("config: %w", err))
	}
	return cfg
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

	return &cfg, nil
}

// IsProduction reports whether the service runs in a production profile.
func (c *Config) IsProduction() bool { return c.Env == "production" }

// UsesDefaultDevelopmentJWTSecret reports whether a warning should be logged.
func (c *Config) UsesDefaultDevelopmentJWTSecret() bool {
	return !c.IsProduction() && c.Auth.JWTAccessSecret == DefaultDevelopmentJWTSecret
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
	secret := strings.TrimSpace(c.Auth.JWTAccessSecret)
	if secret == "" {
		return fmt.Errorf("JWT_ACCESS_SECRET is required")
	}
	if !c.IsProduction() {
		c.Auth.JWTAccessSecret = secret
		return nil
	}
	if secret == DefaultDevelopmentJWTSecret {
		return fmt.Errorf("JWT_ACCESS_SECRET must not use the development default in production")
	}
	if len(secret) < MinProductionJWTSecretLength {
		return fmt.Errorf("JWT_ACCESS_SECRET must be at least %d characters in production", MinProductionJWTSecretLength)
	}
	c.Auth.JWTAccessSecret = secret
	return nil
}
