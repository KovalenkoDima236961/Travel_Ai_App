package cleanup

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"
)

type RunnerConfig struct {
	DefaultDryRun         bool
	BatchSize, MaxBatches int
	LockTTL               time.Duration
}
type Runner struct {
	cfg         RunnerConfig
	store       RunStore
	tasks       map[string]Task
	descriptors map[string]Descriptor
	log         *zap.Logger
}

func NewRunner(cfg RunnerConfig, store RunStore, tasks []Task, log *zap.Logger) (*Runner, error) {
	if store == nil {
		return nil, fmt.Errorf("cleanup run store is required")
	}
	if cfg.BatchSize < 1 || cfg.BatchSize > 1000 {
		return nil, fmt.Errorf("cleanup batch size must be between 1 and 1000")
	}
	if cfg.MaxBatches < 1 || cfg.MaxBatches > 100 {
		return nil, fmt.Errorf("cleanup max batches must be between 1 and 100")
	}
	if cfg.LockTTL <= 0 {
		cfg.LockTTL = time.Hour
	}
	if log == nil {
		log = zap.NewNop()
	}
	r := &Runner{cfg: cfg, store: store, tasks: map[string]Task{}, descriptors: map[string]Descriptor{}, log: log}
	for _, task := range tasks {
		if task == nil {
			continue
		}
		name := task.Name()
		if name == "" {
			return nil, fmt.Errorf("cleanup task name is required")
		}
		if _, exists := r.tasks[name]; exists {
			return nil, fmt.Errorf("duplicate cleanup task %q", name)
		}
		r.tasks[name] = task
		describedTask, ok := task.(DescribedTask)
		if !ok {
			return nil, fmt.Errorf("cleanup task %q must provide a descriptor", name)
		}
		r.descriptors[name] = describedTask.Descriptor()
	}
	return r, nil
}
func (r *Runner) Tasks() []Descriptor {
	out := make([]Descriptor, 0, len(r.descriptors))
	for _, d := range r.descriptors {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
func (r *Runner) Runs(ctx context.Context, limit int) ([]Run, error)   { return r.store.List(ctx, limit) }
func (r *Runner) RunByID(ctx context.Context, id string) (*Run, error) { return r.store.Get(ctx, id) }
func (r *Runner) Run(ctx context.Context, taskName string, params Params) (*Run, error) {
	task, ok := r.tasks[strings.TrimSpace(taskName)]
	if !ok {
		return nil, fmt.Errorf("cleanup_task_not_found")
	}
	if params.BatchSize == 0 {
		params.BatchSize = r.cfg.BatchSize
	}
	if params.MaxBatches == 0 {
		params.MaxBatches = r.cfg.MaxBatches
	}
	if params.StartedBy == "" {
		params.StartedBy = "system"
	}
	if params.Now.IsZero() {
		params.Now = time.Now().UTC()
	}
	if params.BatchSize < 1 || params.BatchSize > r.cfg.BatchSize || params.MaxBatches < 1 || params.MaxBatches > r.cfg.MaxBatches {
		return nil, fmt.Errorf("cleanup_invalid_scope")
	}
	run, err := r.store.Start(ctx, task.Name(), params, r.cfg.LockTTL)
	if err != nil {
		if err == ErrAlreadyRunning {
			return nil, fmt.Errorf("cleanup_already_running")
		}
		return nil, err
	}
	started := time.Now()
	result, taskErr := task.Run(ctx, params)
	result.TaskName, result.DryRun, result.DurationMS = task.Name(), params.DryRun, time.Since(started).Milliseconds()
	status, message := StatusSucceeded, ""
	if taskErr != nil {
		status, message, result.ErrorCount = StatusFailed, "cleanup_internal_error", result.ErrorCount+1
	}
	if err := r.store.Complete(ctx, run, result, status, message); err != nil {
		return nil, err
	}
	recordResult(result, status, float64(time.Now().Unix()))
	fields := []zap.Field{zap.String("task", result.TaskName), zap.Bool("dryRun", result.DryRun), zap.Int64("scannedCount", result.ScannedCount), zap.Int64("deletedCount", result.DeletedCount), zap.Int64("fileDeletedCount", result.FileDeletedCount), zap.Int64("bytesFreed", result.BytesFreed), zap.Int64("durationMs", result.DurationMS), zap.String("status", status), zap.String("requestId", params.RequestID)}
	if taskErr != nil {
		r.log.Warn("cleanup_run", append(fields, zap.String("errorCode", message), zap.Error(taskErr))...)
		return run, taskErr
	}
	r.log.Info("cleanup_run", fields...)
	return run, nil
}
