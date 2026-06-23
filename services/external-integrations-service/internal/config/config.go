package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/ilyakaznacheev/cleanenv"
)

const (
	PlaceProviderMock = "mock"
)

// Config is the root application configuration. It is loaded from a YAML file
// with environment-variable overrides, then validated during bootstrap.
type Config struct {
	Env           string              `yaml:"env" env:"APP_ENV" env-default:"development" validate:"required,oneof=development production test"`
	HTTPServer    HTTPServer          `yaml:"http_server" validate:"required"`
	CORS          CORSConfig          `yaml:"cors" validate:"required"`
	PlaceProvider PlaceProviderConfig `yaml:"place_provider" validate:"required"`
}

// HTTPServer holds the HTTP listener configuration.
type HTTPServer struct {
	Address         string        `yaml:"address" env:"HTTP_ADDR" env-default:":8084" validate:"required"`
	ReadTimeout     time.Duration `yaml:"read_timeout" env:"HTTP_READ_TIMEOUT" env-default:"15s"`
	WriteTimeout    time.Duration `yaml:"write_timeout" env:"HTTP_WRITE_TIMEOUT" env-default:"30s"`
	IdleTimeout     time.Duration `yaml:"idle_timeout" env:"HTTP_IDLE_TIMEOUT" env-default:"60s"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env:"HTTP_SHUTDOWN_TIMEOUT" env-default:"15s"`
}

// CORSConfig controls browser access to External Integrations Service.
type CORSConfig struct {
	AllowedOrigins string `yaml:"allowed_origins" env:"CORS_ALLOWED_ORIGINS" env-default:"http://localhost:3000"`
	AllowedMethods string `yaml:"allowed_methods" env:"CORS_ALLOWED_METHODS" env-default:"GET,OPTIONS"`
	AllowedHeaders string `yaml:"allowed_headers" env:"CORS_ALLOWED_HEADERS" env-default:"Content-Type,Authorization"`
}

// PlaceProviderConfig selects the place provider adapter.
type PlaceProviderConfig struct {
	Provider           string `yaml:"provider" env:"PLACE_PROVIDER" env-default:"mock"`
	GooglePlacesAPIKey string `yaml:"google_places_api_key" env:"GOOGLE_PLACES_API_KEY"`
	MapboxAPIKey       string `yaml:"mapbox_api_key" env:"MAPBOX_API_KEY"`
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

	cfg.PlaceProvider.Provider = strings.ToLower(strings.TrimSpace(cfg.PlaceProvider.Provider))
	if cfg.PlaceProvider.Provider == "" {
		return nil, fmt.Errorf("PLACE_PROVIDER is required")
	}

	cfg.PlaceProvider.GooglePlacesAPIKey = strings.TrimSpace(cfg.PlaceProvider.GooglePlacesAPIKey)
	cfg.PlaceProvider.MapboxAPIKey = strings.TrimSpace(cfg.PlaceProvider.MapboxAPIKey)

	return &cfg, nil
}

// IsProduction reports whether the service runs in a production profile.
func (c *Config) IsProduction() bool { return c.Env == "production" }

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.CORS.AllowedOrigins) == "" && c.Env == "development" {
		c.CORS.AllowedOrigins = "http://localhost:3000"
	}
	if strings.TrimSpace(c.CORS.AllowedMethods) == "" {
		c.CORS.AllowedMethods = "GET,OPTIONS"
	}
	if strings.TrimSpace(c.CORS.AllowedHeaders) == "" {
		c.CORS.AllowedHeaders = "Content-Type,Authorization"
	}
}
