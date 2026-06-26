package postgres

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
)

func (r *Repository) CreateGenerationJob(
	ctx context.Context,
	job *entity.GenerationJob,
) (*entity.GenerationJob, error) {
	query, args, err := r.db.Builder.
		Insert("trip_generation_jobs").
		Columns(dto.GenerationJobInsertColumns()...).
		Values(dto.GenerationJobInsertValues(job)...).
		Suffix("RETURNING " + dto.GenerationJobColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create generation job: %w", err)
	}

	return dto.ScanGenerationJob(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetGenerationJobByID(
	ctx context.Context,
	id uuid.UUID,
) (*entity.GenerationJob, error) {
	query, args, err := r.db.Builder.
		Select(dto.GenerationJobColumns).
		From("trip_generation_jobs").
		Where(sq.Eq{"id": dto.IDArg(id)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get generation job: %w", err)
	}

	return dto.ScanGenerationJob(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetGenerationJobByIDAndTrip(
	ctx context.Context,
	id, tripID uuid.UUID,
) (*entity.GenerationJob, error) {
	query, args, err := r.db.Builder.
		Select(dto.GenerationJobColumns).
		From("trip_generation_jobs").
		Where(sq.Eq{
			"id":      dto.IDArg(id),
			"trip_id": dto.IDArg(tripID),
		}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get generation job by trip: %w", err)
	}

	return dto.ScanGenerationJob(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListGenerationJobsByTrip(
	ctx context.Context,
	tripID uuid.UUID,
	limit int,
) ([]entity.GenerationJob, error) {
	query, args, err := r.db.Builder.
		Select(dto.GenerationJobColumns).
		From("trip_generation_jobs").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list generation jobs: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query generation jobs: %w", err)
	}
	defer rows.Close()

	return dto.ScanGenerationJobRows(rows)
}

func (r *Repository) ClaimNextGenerationJob(ctx context.Context) (*entity.GenerationJob, error) {
	query := "UPDATE trip_generation_jobs " +
		"SET status = 'running', started_at = NOW(), updated_at = NOW() " +
		"WHERE id = (" +
		"SELECT id FROM trip_generation_jobs " +
		"WHERE status = 'queued' " +
		"ORDER BY created_at ASC " +
		"FOR UPDATE SKIP LOCKED " +
		"LIMIT 1" +
		") " +
		"RETURNING " + dto.GenerationJobColumns

	return dto.ScanGenerationJob(r.db.QueryRow(ctx, query))
}

func (r *Repository) CompleteGenerationJob(
	ctx context.Context,
	id uuid.UUID,
	resultItineraryRevision int,
) (*entity.GenerationJob, error) {
	query, args, err := r.db.Builder.
		Update("trip_generation_jobs").
		Set("status", string(entity.GenerationJobStatusCompleted)).
		Set("result_itinerary_revision", resultItineraryRevision).
		Set("completed_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(id)}).
		Suffix("RETURNING " + dto.GenerationJobColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build complete generation job: %w", err)
	}

	return dto.ScanGenerationJob(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) FailGenerationJob(
	ctx context.Context,
	id uuid.UUID,
	errorCode string,
	errorMessage string,
) (*entity.GenerationJob, error) {
	query, args, err := r.db.Builder.
		Update("trip_generation_jobs").
		Set("status", string(entity.GenerationJobStatusFailed)).
		Set("error_code", errorCode).
		Set("error_message", errorMessage).
		Set("completed_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(id)}).
		Suffix("RETURNING " + dto.GenerationJobColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build fail generation job: %w", err)
	}

	return dto.ScanGenerationJob(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) CancelQueuedGenerationJob(
	ctx context.Context,
	id uuid.UUID,
) (*entity.GenerationJob, error) {
	query, args, err := r.db.Builder.
		Update("trip_generation_jobs").
		Set("status", string(entity.GenerationJobStatusCancelled)).
		Set("cancelled_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":     dto.IDArg(id),
			"status": string(entity.GenerationJobStatusQueued),
		}).
		Suffix("RETURNING " + dto.GenerationJobColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build cancel generation job: %w", err)
	}

	return dto.ScanGenerationJob(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) MarkStaleRunningGenerationJobsFailed(
	ctx context.Context,
	startedBefore time.Time,
	errorCode string,
	errorMessage string,
) (int64, error) {
	query, args, err := r.db.Builder.
		Update("trip_generation_jobs").
		Set("status", string(entity.GenerationJobStatusFailed)).
		Set("error_code", errorCode).
		Set("error_message", errorMessage).
		Set("completed_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"status": string(entity.GenerationJobStatusRunning)}).
		Where(sq.Lt{"started_at": startedBefore}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build fail stale generation jobs: %w", err)
	}

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("fail stale generation jobs: %w", err)
	}
	return tag.RowsAffected(), nil
}
