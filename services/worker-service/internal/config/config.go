package config

import (
	"fmt"
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
	Runtime Runtime
	Trip    *tripconfig.Config
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

	return &Config{
		Runtime: runtime,
		Trip:    tripCfg,
	}, nil
}

func MustLoad(tripConfigPath string) *Config {
	cfg, err := Load(tripConfigPath)
	if err != nil {
		panic(fmt.Errorf("config: %w", err))
	}
	return cfg
}

func (c *Config) ShutdownTimeout() time.Duration {
	return time.Duration(c.Runtime.ShutdownTimeoutSeconds) * time.Second
}
