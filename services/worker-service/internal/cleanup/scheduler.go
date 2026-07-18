package cleanup

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Scheduler intentionally supports the conservative daily cron form used by
// v1 (minute hour * * *). It avoids a new cron dependency and fails startup
// for an ambiguous schedule instead of running an unexpected destructive job.
type Scheduler struct {
	runner   *Runner
	enabled  bool
	failOpen bool
	cron     string
	log      *zap.Logger
}

func NewScheduler(runner *Runner, enabled, failOpen bool, cron string, log *zap.Logger) (*Scheduler, error) {
	if log == nil {
		log = zap.NewNop()
	}
	if _, err := nextDaily(cron, time.Now().UTC()); err != nil {
		return nil, err
	}
	return &Scheduler{runner: runner, enabled: enabled, failOpen: failOpen, cron: cron, log: log}, nil
}
func (s *Scheduler) Start(ctx context.Context) func(context.Context) error {
	if !s.enabled || s.runner == nil {
		return func(context.Context) error { return nil }
	}
	runCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			next, err := nextDaily(s.cron, time.Now().UTC())
			if err != nil {
				s.log.Error("cleanup schedule invalid", zap.Error(err))
				return
			}
			timer := time.NewTimer(time.Until(next))
			select {
			case <-runCtx.Done():
				timer.Stop()
				return
			case <-timer.C:
				for _, task := range s.runner.Tasks() {
					if _, err := s.runner.Run(runCtx, task.Name, Params{DryRun: s.runner.cfg.DefaultDryRun, StartedBy: "system"}); err != nil {
						s.log.Warn("scheduled cleanup task failed", zap.String("task", task.Name), zap.Error(err))
						if !s.failOpen {
							break
						}
					}
				}
			}
		}
	}()
	return func(context.Context) error { cancel(); <-done; return nil }
}
func nextDaily(cron string, from time.Time) (time.Time, error) {
	parts := strings.Fields(cron)
	if len(parts) != 5 || parts[2] != "*" || parts[3] != "*" || parts[4] != "*" {
		return time.Time{}, fmt.Errorf("CLEANUP_SCHEDULE_CRON must use daily form 'minute hour * * *'")
	}
	minute, err := strconv.Atoi(parts[0])
	if err != nil || minute < 0 || minute > 59 {
		return time.Time{}, fmt.Errorf("CLEANUP_SCHEDULE_CRON minute is invalid")
	}
	hour, err := strconv.Atoi(parts[1])
	if err != nil || hour < 0 || hour > 23 {
		return time.Time{}, fmt.Errorf("CLEANUP_SCHEDULE_CRON hour is invalid")
	}
	candidate := time.Date(from.Year(), from.Month(), from.Day(), hour, minute, 0, 0, time.UTC)
	if !candidate.After(from) {
		candidate = candidate.AddDate(0, 0, 1)
	}
	return candidate, nil
}
