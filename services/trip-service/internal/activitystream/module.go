package activitystream

import "time"

type Config struct {
	Enabled                      bool
	HeartbeatInterval            time.Duration
	WriteTimeout                 time.Duration
	MaxConnectionsPerUserPerTrip int
	ClientBufferSize             int
}

func Normalize(cfg Config) Config {
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = DefaultHeartbeatInterval
	}
	if cfg.WriteTimeout <= 0 {
		cfg.WriteTimeout = DefaultWriteTimeout
	}
	if cfg.MaxConnectionsPerUserPerTrip <= 0 {
		cfg.MaxConnectionsPerUserPerTrip = DefaultMaxConnectionsPerUserPerTrip
	}
	if cfg.ClientBufferSize <= 0 {
		cfg.ClientBufferSize = DefaultClientBufferSize
	}
	return cfg
}
