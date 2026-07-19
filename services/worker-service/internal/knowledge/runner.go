// Package knowledge implements the worker-side curated knowledge ingestion
// command. It reuses Trip Service's owner-approved normalized store rather
// than creating a second knowledge database.
package knowledge

import (
	"context"
	"fmt"

	tripknowledge "github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge"
)

type Request struct {
	DataDir     string
	Destination string
	DryRun      bool
	Reindex     bool
}

type Runner struct{ store *tripknowledge.Store }

func NewRunner(store *tripknowledge.Store) *Runner { return &Runner{store: store} }

// Run loads, normalizes, and upserts the curated corpus. Embedding work is
// deliberately separate: callers can enqueue or run reindex only after a
// successful committed ingestion. This makes a failed vector dependency
// fail-open and prevents duplicate relational knowledge rows.
func (r *Runner) Run(ctx context.Context, request Request) (tripknowledge.IngestionResult, error) {
	if r == nil || r.store == nil {
		return tripknowledge.IngestionResult{}, fmt.Errorf("knowledge runner store is required")
	}
	dataset, err := tripknowledge.LoadCurated(request.DataDir, request.Destination)
	if err != nil {
		return tripknowledge.IngestionResult{}, err
	}
	return r.store.UpsertCurated(ctx, dataset, request.DryRun)
}
