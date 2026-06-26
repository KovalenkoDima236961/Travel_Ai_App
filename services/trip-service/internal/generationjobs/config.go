package generationjobs

import "time"

type Config struct {
	Enabled               bool
	WorkerEnabled         bool
	PollInterval          time.Duration
	MaxConcurrent         int
	MaxRunning            time.Duration
	FailOpenNotifications bool
}

func NormalizeConfig(cfg Config) Config {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.MaxConcurrent < 1 {
		cfg.MaxConcurrent = 1
	}
	if cfg.MaxRunning <= 0 {
		cfg.MaxRunning = 10 * time.Minute
	}
	return cfg
}
