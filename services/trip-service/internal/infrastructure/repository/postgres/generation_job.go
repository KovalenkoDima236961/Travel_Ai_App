package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
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

func (r *Repository) ListOpsGenerationJobs(
	ctx context.Context,
	filters generationjobs.OpsJobListFilters,
) ([]entity.GenerationJob, error) {
	builder := r.db.Builder.
		Select(dto.GenerationJobColumns).
		From("trip_generation_jobs").
		OrderBy("created_at DESC", "id DESC").
		Limit(uint64(filters.Limit)).
		Offset(uint64(filters.Offset))

	if filters.Status != nil {
		builder = builder.Where(sq.Eq{"status": string(*filters.Status)})
	}
	if filters.JobType != nil {
		builder = builder.Where(sq.Eq{"job_type": string(*filters.JobType)})
	}
	if filters.TripID != nil {
		builder = builder.Where(sq.Eq{"trip_id": dto.IDArg(*filters.TripID)})
	}
	if filters.UserID != nil {
		builder = builder.Where(sq.Eq{"requested_by_user_id": dto.IDArg(*filters.UserID)})
	}
	if filters.ErrorCode != "" {
		builder = builder.Where(sq.Eq{"error_code": filters.ErrorCode})
	}
	if filters.CreatedAfter != nil {
		builder = builder.Where(sq.GtOrEq{"created_at": *filters.CreatedAfter})
	}
	if filters.CreatedBefore != nil {
		builder = builder.Where(sq.LtOrEq{"created_at": *filters.CreatedBefore})
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list ops generation jobs: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query ops generation jobs: %w", err)
	}
	defer rows.Close()
	return dto.ScanGenerationJobRows(rows)
}

func (r *Repository) CountOpsJobsByStatus(ctx context.Context) (map[entity.GenerationJobStatus]int, error) {
	rows, err := r.db.Query(ctx, "SELECT status, COUNT(*) FROM trip_generation_jobs GROUP BY status")
	if err != nil {
		return nil, fmt.Errorf("count generation jobs by status: %w", err)
	}
	defer rows.Close()

	out := map[entity.GenerationJobStatus]int{}
	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan generation job status count: %w", err)
		}
		out[entity.GenerationJobStatus(status)] = int(count)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate generation job status counts: %w", err)
	}
	return out, nil
}

func (r *Repository) CountOpsJobsByType(ctx context.Context) (map[entity.GenerationJobType]int, error) {
	rows, err := r.db.Query(ctx, "SELECT job_type, COUNT(*) FROM trip_generation_jobs GROUP BY job_type")
	if err != nil {
		return nil, fmt.Errorf("count generation jobs by type: %w", err)
	}
	defer rows.Close()

	out := map[entity.GenerationJobType]int{}
	for rows.Next() {
		var jobType string
		var count int64
		if err := rows.Scan(&jobType, &count); err != nil {
			return nil, fmt.Errorf("scan generation job type count: %w", err)
		}
		out[entity.GenerationJobType(jobType)] = int(count)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate generation job type counts: %w", err)
	}
	return out, nil
}

func (r *Repository) ListRecentFailedOpsJobs(ctx context.Context, limit int) ([]entity.GenerationJob, error) {
	if limit < 1 {
		limit = 10
	}
	query, args, err := r.db.Builder.
		Select(dto.GenerationJobColumns).
		From("trip_generation_jobs").
		Where(sq.Eq{"status": string(entity.GenerationJobStatusFailed)}).
		OrderBy("updated_at DESC", "created_at DESC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build recent failed generation jobs: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query recent failed generation jobs: %w", err)
	}
	defer rows.Close()
	return dto.ScanGenerationJobRows(rows)
}

func (r *Repository) CountStaleRunningGenerationJobs(ctx context.Context, startedBefore time.Time) (int, error) {
	query, args, err := r.db.Builder.
		Select("COUNT(*)").
		From("trip_generation_jobs").
		Where(sq.Eq{"status": string(entity.GenerationJobStatusRunning)}).
		Where(sq.Lt{"started_at": startedBefore}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build stale running generation job count: %w", err)
	}
	var count int64
	if err := r.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count stale running generation jobs: %w", err)
	}
	return int(count), nil
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

func (r *Repository) ClaimGenerationJob(ctx context.Context, id uuid.UUID) (*entity.GenerationJob, error) {
	query, args, err := r.db.Builder.
		Update("trip_generation_jobs").
		Set("status", string(entity.GenerationJobStatusRunning)).
		Set("started_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":     dto.IDArg(id),
			"status": string(entity.GenerationJobStatusQueued),
		}).
		Suffix("RETURNING " + dto.GenerationJobColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build claim generation job: %w", err)
	}

	return dto.ScanGenerationJob(r.db.QueryRow(ctx, query, args...))
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

func (r *Repository) ResetRunningGenerationJobToQueued(
	ctx context.Context,
	id uuid.UUID,
	errorCode string,
	errorMessage string,
) (*entity.GenerationJob, error) {
	query, args, err := r.db.Builder.
		Update("trip_generation_jobs").
		Set("status", string(entity.GenerationJobStatusQueued)).
		Set("started_at", nil).
		Set("error_code", errorCode).
		Set("error_message", errorMessage).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":     dto.IDArg(id),
			"status": string(entity.GenerationJobStatusRunning),
		}).
		Suffix("RETURNING " + dto.GenerationJobColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build reset generation job for retry: %w", err)
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

func (r *Repository) CancelOpsGenerationJob(
	ctx context.Context,
	id uuid.UUID,
	errorCode string,
	errorMessage string,
) (*entity.GenerationJob, error) {
	query, args, err := r.db.Builder.
		Update("trip_generation_jobs").
		Set("status", string(entity.GenerationJobStatusCancelled)).
		Set("cancelled_at", sq.Expr("NOW()")).
		Set("error_code", errorCode).
		Set("error_message", errorMessage).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{
			"id":     dto.IDArg(id),
			"status": string(entity.GenerationJobStatusQueued),
		}).
		Suffix("RETURNING " + dto.GenerationJobColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build ops cancel generation job: %w", err)
	}

	return dto.ScanGenerationJob(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) MarkOpsGenerationJobFailed(
	ctx context.Context,
	id uuid.UUID,
	startedBefore time.Time,
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
		Where(sq.Eq{
			"id":     dto.IDArg(id),
			"status": string(entity.GenerationJobStatusRunning),
		}).
		Where(sq.Lt{"started_at": startedBefore}).
		Suffix("RETURNING " + dto.GenerationJobColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build ops mark generation job failed: %w", err)
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

func (r *Repository) CreateOpsAuditEvent(ctx context.Context, event generationjobs.OpsAuditEvent) error {
	var metadata any
	if event.Metadata != nil {
		raw, err := json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("marshal ops audit metadata: %w", err)
		}
		metadata = raw
	}
	query, args, err := r.db.Builder.
		Insert("ops_audit_events").
		Columns(
			"id",
			"actor_user_id",
			"actor_email",
			"action",
			"entity_type",
			"entity_id",
			"reason",
			"metadata",
		).
		Values(
			dto.IDArg(event.ID),
			dto.IDArg(event.ActorUserID),
			event.ActorEmail,
			event.Action,
			event.EntityType,
			dto.IDArg(event.EntityID),
			event.Reason,
			metadata,
		).
		ToSql()
	if err != nil {
		return fmt.Errorf("build create ops audit event: %w", err)
	}
	if _, err := r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("create ops audit event: %w", err)
	}
	return nil
}
