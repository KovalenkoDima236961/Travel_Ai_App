package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/pkg/storage/postgres"
)

const (
	DefaultDevelopmentJWTSecret  = "change-me-in-development"
	MinProductionJWTSecretLength = 32
)

type Config struct {
	Env        string          `yaml:"env" env:"APP_ENV" env-default:"development" validate:"required,oneof=development production"`
	HTTPServer HTTPServer      `yaml:"http_server" validate:"required"`
	Postgres   postgres.Config `yaml:"postgres" validate:"required"`
	JWT        JWTConfig       `yaml:"jwt" validate:"required"`
	CORS       CORSConfig      `yaml:"cors" validate:"required"`
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

	return &cfg, nil
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}

func (c *Config) AccessTokenTTL() time.Duration {
	return time.Duration(c.JWT.AccessTokenTTLMinutes) * time.Minute
}

func (c *Config) RefreshTokenTTL() time.Duration {
	return time.Duration(c.JWT.RefreshTokenTTLDays) * 24 * time.Hour
}

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.CORS.AllowedOrigins) == "" && c.Env == "development" {
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
