package digests

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type Config struct {
	Enabled      bool
	PollInterval time.Duration
	BatchSize    int
}
type Worker struct {
	client *Client
	cfg    Config
	log    *zap.Logger
}

func NewWorker(client *Client, cfg Config, log *zap.Logger) *Worker {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Minute
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &Worker{client: client, cfg: cfg, log: log}
}
func (w *Worker) Start(ctx context.Context) func(context.Context) error {
	if !w.cfg.Enabled || w.client == nil {
		return func(context.Context) error { return nil }
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() { defer close(done); w.run(runCtx) }()
	return func(context.Context) error { cancel(); <-done; return nil }
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
	result, err := w.client.ProcessDue(ctx, ProcessInput{Now: time.Now().UTC(), Limit: w.cfg.BatchSize})
	if err != nil {
		w.log.Warn("notification digest worker failed", zap.Error(err))
		return
	}
	if result.Processed > 0 {
		w.log.Info("notification digest worker processed batches", zap.Int("processed", result.Processed), zap.Int("sent", result.Sent), zap.Int("failed", result.Failed), zap.Int("retrying", result.Retrying))
	}
}
