package aiobservability

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// StartCleanupLoop removes expired trace records once per interval. Foreign-key
// cascades remove their events and optional redacted snapshots in the same
// transaction, so the worker never needs to handle those tables separately.
func StartCleanupLoop(parent context.Context, service *Service, interval time.Duration, log *zap.Logger) func(context.Context) error {
	if service == nil || !service.Enabled() {
		return func(context.Context) error { return nil }
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	if log == nil {
		log = zap.NewNop()
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		cleanup := func() {
			count, err := service.Cleanup(ctx)
			if err != nil {
				log.Warn("AI generation trace cleanup failed", zap.Error(err))
				return
			}
			if count > 0 {
				log.Info("AI generation traces expired", zap.Int64("count", count))
			}
		}
		cleanup()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanup()
			}
		}
	}()
	return func(stopCtx context.Context) error {
		cancel()
		select {
		case <-done:
			return nil
		case <-stopCtx.Done():
			return stopCtx.Err()
		}
	}
}
