package presence

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Config controls instance-local trip presence.
type Config struct {
	Enabled                      bool
	HeartbeatInterval            time.Duration
	StaleAfter                   time.Duration
	MaxConnectionsPerUserPerTrip int
	SendFullSnapshot             bool
}

// Normalize applies v1 defaults to unset values.
func Normalize(cfg Config) Config {
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = DefaultHeartbeatInterval
	}
	if cfg.StaleAfter <= 0 {
		cfg.StaleAfter = DefaultStaleAfter
	}
	if cfg.MaxConnectionsPerUserPerTrip <= 0 {
		cfg.MaxConnectionsPerUserPerTrip = DefaultMaxConnectionsPerUserPerTrip
	}
	return cfg
}

// StartCleanupLoop removes stale sessions periodically until the returned close
// function is called.
func StartCleanupLoop(parent context.Context, manager Manager, cfg Config, log *zap.Logger) func(context.Context) error {
	if parent == nil {
		parent = context.Background()
	}
	if log == nil {
		log = zap.NewNop()
	}
	cfg = Normalize(cfg)
	interval := cfg.StaleAfter / defaultCleanupIntervalDivisor
	if interval < defaultCleanupMinimumInterval {
		interval = defaultCleanupMinimumInterval
	}

	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				manager.CleanupStale(now.UTC())
			}
		}
	}()

	return func(shutdownCtx context.Context) error {
		cancel()
		select {
		case <-done:
			return nil
		case <-shutdownCtx.Done():
			log.Warn("trip presence cleanup loop did not stop before shutdown deadline")
			return shutdownCtx.Err()
		}
	}
}
