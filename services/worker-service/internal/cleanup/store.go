package cleanup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

var ErrAlreadyRunning = errors.New("cleanup task is already running")

type RunStore interface {
	Start(context.Context, string, Params, time.Duration) (*Run, error)
	Complete(context.Context, *Run, Result, string, string) error
	List(context.Context, int) ([]Run, error)
	Get(context.Context, string) (*Run, error)
}

type PostgresRunStore struct{ db *postgres.DB }

func NewPostgresRunStore(db *postgres.DB) *PostgresRunStore { return &PostgresRunStore{db: db} }

func (s *PostgresRunStore) Start(ctx context.Context, task string, params Params, lockTTL time.Duration) (*Run, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("cleanup run store is unavailable")
	}
	if lockTTL <= 0 {
		lockTTL = time.Hour
	}
	now := params.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	// A dead worker must not permanently suppress cleanup. The partial unique
	// index below then atomically admits one current runner per task.
	_, _ = s.db.Exec(ctx, "UPDATE cleanup_runs SET status = $1, completed_at = $2, error_message = $3 WHERE task_name = $4 AND status = $5 AND lock_expires_at < $2", StatusFailed, now, "cleanup lock expired", task, StatusRunning)
	run := &Run{ID: uuid.NewString(), Result: Result{TaskName: task, DryRun: params.DryRun}, Status: StatusRunning, StartedBy: params.StartedBy, StartedAt: now, RequestID: params.RequestID}
	_, err := s.db.Exec(ctx, `INSERT INTO cleanup_runs (id, task_name, status, dry_run, started_by, started_at, request_id, lock_expires_at)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, NULLIF($7, ''), $8)`, run.ID, task, run.Status, params.DryRun, params.StartedBy, now, params.RequestID, now.Add(lockTTL))
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrAlreadyRunning
		}
		return nil, fmt.Errorf("create cleanup run: %w", err)
	}
	return run, nil
}

func (s *PostgresRunStore) Complete(ctx context.Context, run *Run, result Result, status, errorMessage string) error {
	if s == nil || s.db == nil || run == nil {
		return fmt.Errorf("cleanup run store is unavailable")
	}
	now := time.Now().UTC()
	warnings, err := json.Marshal(result.Warnings)
	if err != nil {
		return fmt.Errorf("encode cleanup warnings: %w", err)
	}
	_, err = s.db.Exec(ctx, `UPDATE cleanup_runs SET status=$1, completed_at=$2, scanned_count=$3, deleted_count=$4, archived_count=$5, skipped_count=$6, error_count=$7, file_deleted_count=$8, bytes_freed=$9, warnings=$10, error_message=NULLIF($11, ''), lock_expires_at=NULL WHERE id=$12`,
		status, now, result.ScannedCount, result.DeletedCount, result.ArchivedCount, result.SkippedCount, result.ErrorCount, result.FileDeletedCount, result.BytesFreed, warnings, errorMessage, run.ID)
	if err != nil {
		return fmt.Errorf("complete cleanup run: %w", err)
	}
	run.Result, run.Status, run.ErrorMessage, run.CompletedAt = result, status, errorMessage, &now
	return nil
}

func (s *PostgresRunStore) List(ctx context.Context, limit int) ([]Run, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := s.db.Query(ctx, `SELECT id, task_name, status, dry_run, COALESCE(started_by, ''), started_at, completed_at, scanned_count, deleted_count, archived_count, skipped_count, error_count, file_deleted_count, bytes_freed, warnings, COALESCE(error_message, ''), COALESCE(request_id, '') FROM cleanup_runs ORDER BY started_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list cleanup runs: %w", err)
	}
	defer rows.Close()
	runs := make([]Run, 0)
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, *run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cleanup runs: %w", err)
	}
	return runs, nil
}
func (s *PostgresRunStore) Get(ctx context.Context, id string) (*Run, error) {
	row := s.db.QueryRow(ctx, `SELECT id, task_name, status, dry_run, COALESCE(started_by, ''), started_at, completed_at, scanned_count, deleted_count, archived_count, skipped_count, error_count, file_deleted_count, bytes_freed, warnings, COALESCE(error_message, ''), COALESCE(request_id, '') FROM cleanup_runs WHERE id=$1`, id)
	run, err := scanRun(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, pgx.ErrNoRows
	}
	return run, err
}

type rowScanner interface{ Scan(...any) error }

func scanRun(row rowScanner) (*Run, error) {
	var run Run
	var warnings []byte
	var completed *time.Time
	err := row.Scan(&run.ID, &run.Result.TaskName, &run.Status, &run.Result.DryRun, &run.StartedBy, &run.StartedAt, &completed, &run.Result.ScannedCount, &run.Result.DeletedCount, &run.Result.ArchivedCount, &run.Result.SkippedCount, &run.Result.ErrorCount, &run.Result.FileDeletedCount, &run.Result.BytesFreed, &warnings, &run.ErrorMessage, &run.RequestID)
	if err != nil {
		return nil, err
	}
	run.CompletedAt = completed
	if len(warnings) > 0 {
		_ = json.Unmarshal(warnings, &run.Result.Warnings)
	}
	return &run, nil
}
