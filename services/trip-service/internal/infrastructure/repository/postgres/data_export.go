package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/dataexport"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
)

const dataExportJobColumns = `id, user_id, export_type, status, scope_json, file_path, file_name, mime_type,
size_bytes, checksum_sha256, error_code, error_message_safe, expires_at, created_at, started_at, completed_at, updated_at`

func (r *Repository) CreateDataExportJob(ctx context.Context, job dataexport.Job) (*dataexport.Job, error) {
	if len(job.Scope) == 0 {
		job.Scope = json.RawMessage(`{}`)
	}
	query, args, err := r.db.Builder.Insert("data_export_jobs").
		Columns("id", "user_id", "export_type", "status", "scope_json").
		Values(dto.IDArg(job.ID), dto.IDArg(job.UserID), job.ExportType, job.Status, []byte(job.Scope)).
		Suffix("RETURNING " + dataExportJobColumns).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create export job: %w", err)
	}
	return scanDataExportJob(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetDataExportJob(ctx context.Context, id, userID uuid.UUID) (*dataexport.Job, error) {
	query, args, err := r.db.Builder.Select(dataExportJobColumns).From("data_export_jobs").
		Where(sq.Eq{"id": dto.IDArg(id), "user_id": dto.IDArg(userID)}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get export job: %w", err)
	}
	return scanDataExportJob(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) CompleteDataExportJob(ctx context.Context, id uuid.UUID, filePath, fileName, mimeType string, size int64, checksum string, expiresAt time.Time) (*dataexport.Job, error) {
	query, args, err := r.db.Builder.Update("data_export_jobs").
		Set("status", dataexport.StatusCompleted).
		Set("file_path", filePath).
		Set("file_name", fileName).
		Set("mime_type", mimeType).
		Set("size_bytes", size).
		Set("checksum_sha256", checksum).
		Set("expires_at", expiresAt.UTC()).
		Set("completed_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(id)}).
		Suffix("RETURNING " + dataExportJobColumns).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build complete export job: %w", err)
	}
	return scanDataExportJob(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) FailDataExportJob(ctx context.Context, id uuid.UUID, code, safeMessage string) error {
	query, args, err := r.db.Builder.Update("data_export_jobs").
		Set("status", dataexport.StatusFailed).
		Set("error_code", code).
		Set("error_message_safe", safeMessage).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(id)}).ToSql()
	if err != nil {
		return fmt.Errorf("build fail export job: %w", err)
	}
	if _, err := r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("fail export job: %w", err)
	}
	return nil
}

func (r *Repository) ListExpiredDataExportJobs(ctx context.Context, now time.Time) ([]dataexport.Job, error) {
	query, args, err := r.db.Builder.Select(dataExportJobColumns).From("data_export_jobs").
		Where(sq.Eq{"status": dataexport.StatusCompleted}).
		Where(sq.Lt{"expires_at": now.UTC()}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list expired export jobs: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list expired export jobs: %w", err)
	}
	defer rows.Close()
	jobs := []dataexport.Job{}
	for rows.Next() {
		job, err := scanDataExportJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expired export jobs: %w", err)
	}
	return jobs, nil
}

func (r *Repository) ExpireDataExportJob(ctx context.Context, id uuid.UUID) error {
	query, args, err := r.db.Builder.Update("data_export_jobs").
		Set("status", dataexport.StatusExpired).
		Set("file_path", nil).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(id), "status": dataexport.StatusCompleted}).ToSql()
	if err != nil {
		return fmt.Errorf("build expire export job: %w", err)
	}
	if _, err := r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("expire export job: %w", err)
	}
	return nil
}

func scanDataExportJob(row pgx.Row) (*dataexport.Job, error) {
	var job dataexport.Job
	var scope []byte
	err := row.Scan(&job.ID, &job.UserID, &job.ExportType, &job.Status, &scope, &job.FilePath, &job.FileName,
		&job.MIMEType, &job.SizeBytes, &job.ChecksumSHA256, &job.ErrorCode, &job.ErrorMessageSafe,
		&job.ExpiresAt, &job.CreatedAt, &job.StartedAt, &job.CompletedAt, &job.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domainerrs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan export job: %w", err)
	}
	job.Scope = append(json.RawMessage(nil), scope...)
	job.CreatedAt = job.CreatedAt.UTC()
	job.UpdatedAt = job.UpdatedAt.UTC()
	if job.ExpiresAt != nil {
		value := job.ExpiresAt.UTC()
		job.ExpiresAt = &value
	}
	if job.StartedAt != nil {
		value := job.StartedAt.UTC()
		job.StartedAt = &value
	}
	if job.CompletedAt != nil {
		value := job.CompletedAt.UTC()
		job.CompletedAt = &value
	}
	return &job, nil
}
