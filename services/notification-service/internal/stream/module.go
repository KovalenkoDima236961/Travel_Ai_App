package stream

import "time"

const (
	DefaultHeartbeatInterval     = 25 * time.Second
	DefaultWriteTimeout          = 10 * time.Second
	DefaultMaxConnectionsPerUser = 5
)

// Config controls the in-memory SSE stream. It is intentionally instance-local;
// v1 does not provide cross-instance fanout or replay.
type Config struct {
	Enabled               bool
	HeartbeatInterval     time.Duration
	WriteTimeout          time.Duration
	MaxConnectionsPerUser int
}

func (c Config) normalized() Config {
	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = DefaultHeartbeatInterval
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = DefaultWriteTimeout
	}
	if c.MaxConnectionsPerUser <= 0 {
		c.MaxConnectionsPerUser = DefaultMaxConnectionsPerUser
	}
	return c
}

// Normalize applies v1 defaults to unset values.
func Normalize(cfg Config) Config {
	return cfg.normalized()
}
