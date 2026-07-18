package cleanup

import (
	"context"
	"errors"
	"testing"
	"time"
)

type testTask struct {
	name   string
	result Result
	err    error
}

func (t testTask) Name() string           { return t.name }
func (t testTask) Descriptor() Descriptor { return Descriptor{Name: t.name, DryRunSupported: true} }
func (t testTask) Run(_ context.Context, p Params) (Result, error) {
	out := t.result
	out.TaskName = t.name
	out.DryRun = p.DryRun
	return out, t.err
}

type testStore struct {
	started        Params
	complete       Run
	completeResult Result
	completeStatus string
	startErr       error
}

func (s *testStore) Start(_ context.Context, task string, p Params, _ time.Duration) (*Run, error) {
	s.started = p
	if s.startErr != nil {
		return nil, s.startErr
	}
	return &Run{ID: "run-1", Result: Result{TaskName: task, DryRun: p.DryRun}, Status: StatusRunning}, nil
}
func (s *testStore) Complete(_ context.Context, run *Run, result Result, status, message string) error {
	s.complete = *run
	s.completeResult = result
	s.completeStatus = status
	return nil
}
func (s *testStore) List(context.Context, int) ([]Run, error)  { return nil, nil }
func (s *testStore) Get(context.Context, string) (*Run, error) { return nil, nil }

func TestRunnerUsesBoundedDefaultsAndRecordsDryRun(t *testing.T) {
	store := &testStore{}
	runner, err := NewRunner(RunnerConfig{DefaultDryRun: true, BatchSize: 50, MaxBatches: 2, LockTTL: time.Hour}, store, []Task{testTask{name: "tokens", result: Result{ScannedCount: 4}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	run, err := runner.Run(context.Background(), "tokens", Params{DryRun: true, StartedBy: "ops"})
	if err != nil {
		t.Fatal(err)
	}
	if store.started.BatchSize != 50 || store.started.MaxBatches != 2 {
		t.Fatalf("defaults = %d/%d", store.started.BatchSize, store.started.MaxBatches)
	}
	if run.ID != "run-1" || store.completeStatus != StatusSucceeded || store.completeResult.DeletedCount != 0 {
		t.Fatalf("unexpected dry-run result: %#v", store.completeResult)
	}
}

func TestRunnerRejectsOversizedManualScope(t *testing.T) {
	runner, err := NewRunner(RunnerConfig{BatchSize: 50, MaxBatches: 2, LockTTL: time.Hour}, &testStore{}, []Task{testTask{name: "tokens"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "tokens", Params{BatchSize: 51, MaxBatches: 1}); err == nil || err.Error() != "cleanup_invalid_scope" {
		t.Fatalf("err = %v", err)
	}
}

func TestRunnerRecordsTaskFailure(t *testing.T) {
	store := &testStore{}
	runner, err := NewRunner(RunnerConfig{BatchSize: 50, MaxBatches: 2, LockTTL: time.Hour}, store, []Task{testTask{name: "tokens", err: errors.New("down")}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Run(context.Background(), "tokens", Params{DryRun: true}); err == nil {
		t.Fatal("expected task failure")
	}
	if store.completeStatus != StatusFailed || store.completeResult.ErrorCount != 1 {
		t.Fatalf("failed run = %#v", store.completeResult)
	}
}

func TestNextDailyAcceptsOnlySafeDailyCron(t *testing.T) {
	next, err := nextDaily("0 3 * * *", time.Date(2026, 7, 18, 4, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if next.Hour() != 3 || next.Day() != 19 {
		t.Fatalf("next = %s", next)
	}
	if _, err := nextDaily("*/5 * * * *", time.Now()); err == nil {
		t.Fatal("expected unsupported schedule to fail")
	}
}
