package editlocks

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Config controls instance-local advisory edit locks.
type Config struct {
	Enabled         bool
	TTL             time.Duration
	RenewalInterval time.Duration
	CleanupInterval time.Duration
}

func Normalize(cfg Config) Config {
	if cfg.TTL <= 0 {
		cfg.TTL = DefaultTTL
	}
	if cfg.RenewalInterval <= 0 {
		cfg.RenewalInterval = DefaultRenewalInterval
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = DefaultCleanupInterval
	}
	return cfg
}

// StartCleanupLoop removes expired locks periodically until the returned close
// function is called.
func StartCleanupLoop(parent context.Context, manager Manager, cfg Config, log *zap.Logger) func(context.Context) error {
	if parent == nil {
		parent = context.Background()
	}
	if log == nil {
		log = zap.NewNop()
	}
	cfg = Normalize(cfg)

	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(cfg.CleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				manager.CleanupExpired(now.UTC())
			}
		}
	}()

	return func(shutdownCtx context.Context) error {
		cancel()
		select {
		case <-done:
			return nil
		case <-shutdownCtx.Done():
			log.Warn("trip edit-lock cleanup loop did not stop before shutdown deadline")
			return shutdownCtx.Err()
		}
	}
}
