// Package cleanup contains the bounded, observable data-lifecycle runner used
// by Worker Service. It owns run bookkeeping and orchestration only; each
// service remains the owner of its own rows and files.
package cleanup

import (
	"context"
	"time"
)

const (
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
)

// Params are deliberately small and safe to propagate to an owning service.
// A caller cannot override retention or send arbitrary SQL/file paths.
type Params struct {
	DryRun     bool      `json:"dryRun"`
	BatchSize  int       `json:"batchSize"`
	MaxBatches int       `json:"maxBatches"`
	StartedBy  string    `json:"startedBy,omitempty"`
	RequestID  string    `json:"requestId,omitempty"`
	Now        time.Time `json:"-"`
}

type Result struct {
	TaskName         string   `json:"taskName"`
	DryRun           bool     `json:"dryRun"`
	ScannedCount     int64    `json:"scannedCount"`
	DeletedCount     int64    `json:"deletedCount"`
	ArchivedCount    int64    `json:"archivedCount"`
	SkippedCount     int64    `json:"skippedCount"`
	ErrorCount       int64    `json:"errorCount"`
	FileDeletedCount int64    `json:"fileDeletedCount"`
	BytesFreed       int64    `json:"bytesFreed"`
	DurationMS       int64    `json:"durationMs"`
	Warnings         []string `json:"warnings,omitempty"`
}

// Task is implemented by the small internal HTTP task adapter. Keeping the
// interface local also makes scheduler/runner tests independent of services.
type Task interface {
	Name() string
	Run(context.Context, Params) (Result, error)
}

type DescribedTask interface {
	Task
	Descriptor() Descriptor
}

type Descriptor struct {
	Name             string `json:"name"`
	Description      string `json:"description"`
	OwningService    string `json:"owningService"`
	DefaultRetention string `json:"defaultRetention"`
	DryRunSupported  bool   `json:"dryRunSupported"`
}

type Run struct {
	ID           string     `json:"id"`
	Result       Result     `json:"result"`
	Status       string     `json:"status"`
	StartedBy    string     `json:"startedBy,omitempty"`
	StartedAt    time.Time  `json:"startedAt"`
	CompletedAt  *time.Time `json:"completedAt,omitempty"`
	ErrorMessage string     `json:"errorMessage,omitempty"`
	RequestID    string     `json:"requestId,omitempty"`
}
