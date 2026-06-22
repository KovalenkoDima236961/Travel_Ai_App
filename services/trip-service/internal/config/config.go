package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"

	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/validation"
)

// Config is the root application configuration. It is loaded from a YAML file
// (path passed via the -config flag) with environment-variable overrides, then
// validated using the project's validation package.
type Config struct {
	Env                string                   `yaml:"env" env:"APP_ENV" env-default:"development" validate:"required,oneof=development production"`
	HTTPServer         HTTPServer               `yaml:"http_server"`
	Auth               AuthConfig               `yaml:"auth"`
	CORS               CORSConfig               `yaml:"cors"`
	Postgres           postgres.Config          `yaml:"postgres"`
	ItineraryGenerator ItineraryGeneratorConfig `yaml:"itinerary_generator"`
}

// HTTPServer holds the HTTP listener configuration.
type HTTPServer struct {
	Address         string        `yaml:"address" env:"HTTP_ADDRESS" env-default:":8080" validate:"required"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env:"HTTP_READ_TIMEOUT" env-default:"15s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env:"HTTP_WRITE_TIMEOUT" env-default:"150s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"15s"`
}

// AuthConfig controls local JWT validation for protected trip endpoints.
type AuthConfig struct {
	Required        bool   `yaml:"required" env:"AUTH_REQUIRED" env-default:"true"`
	JWTAccessSecret string `yaml:"jwt_access_secret" env:"JWT_ACCESS_SECRET" env-default:"change-me-in-development" validate:"required"`
	HeaderName      string `yaml:"header_name" env:"AUTH_HEADER_NAME" env-default:"Authorization" validate:"required"`
	DevUserID       string `yaml:"dev_user_id" env:"DEV_USER_ID" env-default:"00000000-0000-0000-0000-000000000001" validate:"required,uuid"`
}

// CORSConfig controls browser access to the Trip Service API.
type CORSConfig struct {
	AllowedOrigins string `yaml:"allowed_origins" env:"CORS_ALLOWED_ORIGINS"`
	AllowedMethods string `yaml:"allowed_methods" env:"CORS_ALLOWED_METHODS" env-default:"GET,POST,PATCH,DELETE,OPTIONS"`
	AllowedHeaders string `yaml:"allowed_headers" env:"CORS_ALLOWED_HEADERS" env-default:"Content-Type,Authorization"`
}

// ItineraryGeneratorConfig selects the itinerary generation adapter.
type ItineraryGeneratorConfig struct {
	Mode                     string `yaml:"mode" env:"ITINERARY_GENERATOR_MODE" env-default:"mock"`
	AIPlanningServiceURL     string `yaml:"ai_planning_service_url" env:"AI_PLANNING_SERVICE_URL" env-default:"http://ai-planning-service:8000"`
	AIPlanningTimeoutSeconds int    `yaml:"ai_planning_timeout_seconds" env:"AI_PLANNING_TIMEOUT_SECONDS" env-default:"120" validate:"min=1"`
}

// IsProduction reports whether the service runs in a production profile.
func (c *Config) IsProduction() bool { return c.Env == "production" }

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

	validator, err := validation.NewValidator()
	if err != nil {
		return nil, fmt.Errorf("init validator: %w", err)
	}
	if err := validator.Validate(&cfg); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.CORS.AllowedOrigins) == "" && c.Env == "development" {
		c.CORS.AllowedOrigins = "http://localhost:3000"
	}
	if strings.TrimSpace(c.CORS.AllowedMethods) == "" {
		c.CORS.AllowedMethods = "GET,POST,PATCH,DELETE,OPTIONS"
	}
	if strings.TrimSpace(c.CORS.AllowedHeaders) == "" {
		c.CORS.AllowedHeaders = "Content-Type,Authorization"
	}
}
