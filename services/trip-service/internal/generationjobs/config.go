package generationjobs

import "time"

const (
	DispatchModeInProcess = "in_process"
	DispatchModeQueue     = "queue"
)

type Config struct {
	Enabled               bool
	WorkerEnabled         bool
	DispatchMode          string
	PollInterval          time.Duration
	MaxConcurrent         int
	MaxRunning            time.Duration
	PublishTimeout        time.Duration
	PublishFailOpen       bool
	FailOpenNotifications bool
}

func NormalizeConfig(cfg Config) Config {
	if cfg.DispatchMode == "" {
		cfg.DispatchMode = DispatchModeInProcess
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.MaxConcurrent < 1 {
		cfg.MaxConcurrent = 1
	}
	if cfg.MaxRunning <= 0 {
		cfg.MaxRunning = 10 * time.Minute
	}
	if cfg.PublishTimeout <= 0 {
		cfg.PublishTimeout = 5 * time.Second
	}
	return cfg
}

func (c Config) QueueMode() bool {
	return c.DispatchMode == DispatchModeQueue
}
