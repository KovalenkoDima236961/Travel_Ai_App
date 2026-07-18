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

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/dataexport"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/repository/postgres/dto"
)

const exportJobColumns = `id, user_id, export_type, status, scope_json, file_path, file_name, mime_type,
size_bytes, checksum_sha256, error_code, error_message_safe, expires_at, created_at, updated_at`

func (r *Repository) CreateAccountExportJob(ctx context.Context, job dataexport.Job) (*dataexport.Job, error) {
	if len(job.Scope) == 0 {
		job.Scope = json.RawMessage(`{}`)
	}
	query, args, err := r.db.Builder.Insert("data_export_jobs").Columns("id", "user_id", "export_type", "status", "scope_json").
		Values(dto.UUIDArg(job.ID), dto.UUIDArg(job.UserID), job.ExportType, job.Status, []byte(job.Scope)).
		Suffix("RETURNING " + exportJobColumns).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create account export: %w", err)
	}
	return scanExportJob(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetAccountExportJob(ctx context.Context, id, userID uuid.UUID) (*dataexport.Job, error) {
	query, args, err := r.db.Builder.Select(exportJobColumns).From("data_export_jobs").Where(sq.Eq{"id": dto.UUIDArg(id), "user_id": dto.UUIDArg(userID)}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get account export: %w", err)
	}
	return scanExportJob(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) CompleteAccountExportJob(ctx context.Context, id uuid.UUID, path, name, mimeType string, size int64, checksum string, expires time.Time) (*dataexport.Job, error) {
	query, args, err := r.db.Builder.Update("data_export_jobs").Set("status", dataexport.Completed).Set("file_path", path).Set("file_name", name).Set("mime_type", mimeType).Set("size_bytes", size).Set("checksum_sha256", checksum).Set("expires_at", expires.UTC()).Set("completed_at", sq.Expr("NOW()")).Set("updated_at", sq.Expr("NOW()")).Where(sq.Eq{"id": dto.UUIDArg(id)}).Suffix("RETURNING " + exportJobColumns).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build complete account export: %w", err)
	}
	return scanExportJob(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) FailAccountExportJob(ctx context.Context, id uuid.UUID, code, message string) error {
	query, args, err := r.db.Builder.Update("data_export_jobs").Set("status", dataexport.Failed).Set("error_code", code).Set("error_message_safe", message).Set("updated_at", sq.Expr("NOW()")).Where(sq.Eq{"id": dto.UUIDArg(id)}).ToSql()
	if err != nil {
		return fmt.Errorf("build fail account export: %w", err)
	}
	if _, err := r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("fail account export: %w", err)
	}
	return nil
}

func (r *Repository) ExpireAccountExportJob(ctx context.Context, id uuid.UUID) error {
	query, args, err := r.db.Builder.Update("data_export_jobs").Set("status", dataexport.Expired).Set("file_path", nil).Set("updated_at", sq.Expr("NOW()")).Where(sq.Eq{"id": dto.UUIDArg(id), "status": dataexport.Completed}).ToSql()
	if err != nil {
		return fmt.Errorf("build expire account export: %w", err)
	}
	if _, err := r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("expire account export: %w", err)
	}
	return nil
}

func (r *Repository) ListExpiredAccountExportJobs(ctx context.Context, now time.Time) ([]dataexport.Job, error) {
	query, args, err := r.db.Builder.Select(exportJobColumns).From("data_export_jobs").Where(sq.Eq{"status": dataexport.Completed}).Where(sq.Lt{"expires_at": now.UTC()}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list expired account exports: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list expired account exports: %w", err)
	}
	defer rows.Close()
	jobs := []dataexport.Job{}
	for rows.Next() {
		job, err := scanExportJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expired account exports: %w", err)
	}
	return jobs, nil
}

func (r *Repository) CreateAccountCleanupRequest(ctx context.Context, id, userID uuid.UUID, reason *string, exportRequestedFirst bool) error {
	query, args, err := r.db.Builder.Insert("account_cleanup_requests").Columns("id", "user_id", "reason", "export_requested_first").Values(dto.UUIDArg(id), dto.UUIDArg(userID), reason, exportRequestedFirst).ToSql()
	if err != nil {
		return fmt.Errorf("build account cleanup request: %w", err)
	}
	if _, err := r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("create account cleanup request: %w", err)
	}
	return nil
}

type exportJobScanner interface{ Scan(...any) error }

func scanExportJob(row exportJobScanner) (*dataexport.Job, error) {
	var job dataexport.Job
	var scope []byte
	err := row.Scan(&job.ID, &job.UserID, &job.ExportType, &job.Status, &scope, &job.FilePath, &job.FileName, &job.MIMEType, &job.SizeBytes, &job.ChecksumSHA256, &job.ErrorCode, &job.ErrorMessageSafe, &job.ExpiresAt, &job.CreatedAt, &job.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domainerrs.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan account export: %w", err)
	}
	job.Scope = append(json.RawMessage(nil), scope...)
	job.CreatedAt = job.CreatedAt.UTC()
	job.UpdatedAt = job.UpdatedAt.UTC()
	if job.ExpiresAt != nil {
		value := job.ExpiresAt.UTC()
		job.ExpiresAt = &value
	}
	return &job, nil
}
