package reminders

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type Config struct {
	Enabled          bool
	PollInterval     time.Duration
	BatchSize        int
	LookaheadMinutes int
}

type Worker struct {
	client *Client
	cfg    Config
	log    *zap.Logger
}

func NewWorker(client *Client, cfg Config, log *zap.Logger) *Worker {
	if log == nil {
		log = zap.NewNop()
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 5 * time.Minute
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	return &Worker{client: client, cfg: cfg, log: log}
}

func (w *Worker) Start(ctx context.Context) func(context.Context) error {
	if !w.cfg.Enabled || w.client == nil {
		return func(context.Context) error { return nil }
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.run(runCtx)
	}()
	return func(context.Context) error {
		cancel()
		<-done
		return nil
	}
}

func (w *Worker) run(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()
	w.process(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.process(ctx)
		}
	}
}

func (w *Worker) process(ctx context.Context) {
	now := time.Now().UTC()
	if w.cfg.LookaheadMinutes > 0 {
		now = now.Add(time.Duration(w.cfg.LookaheadMinutes) * time.Minute)
	}
	result, err := w.client.ProcessDue(ctx, ProcessInput{
		Now:   now,
		Limit: w.cfg.BatchSize,
	})
	if err != nil {
		w.log.Warn("reminder worker process due failed", zap.Error(err))
		return
	}
	if result.Processed > 0 || result.Failed > 0 {
		w.log.Info("reminder worker processed due reminders",
			zap.Int("processed", result.Processed),
			zap.Int("sent", result.Sent),
			zap.Int("failed", result.Failed),
		)
	}
}
