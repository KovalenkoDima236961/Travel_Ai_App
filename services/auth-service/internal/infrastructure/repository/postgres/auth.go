package postgres

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/infrastructure/repository/postgres/dto"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/pkg/storage/postgres"
)

type Repository struct {
	db *storage.DB
}

func New(database *storage.DB) *Repository {
	return &Repository{db: database}
}

func (r *Repository) CreateUser(ctx context.Context, user *entity.User) (*entity.User, error) {
	query, args, err := r.db.Builder.
		Insert("users").
		Columns(dto.UserInsertColumns()...).
		Values(dto.UserInsertValues(user)...).
		Suffix("RETURNING " + dto.UserColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create user: %w", err)
	}

	created, err := dto.ScanUser(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		if storage.UniqueConstraintViolation(err) {
			return nil, domainerrs.ErrAlreadyExists
		}
		return nil, err
	}
	return created, nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*entity.User, error) {
	query, args, err := r.db.Builder.
		Select(dto.UserColumns).
		From("users").
		Where(sq.Eq{"email": email}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get user by email: %w", err)
	}

	return dto.ScanUser(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	query, args, err := r.db.Builder.
		Select(dto.UserColumns).
		From("users").
		Where(sq.Eq{"id": dto.IDArg(id)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get user by id: %w", err)
	}

	return dto.ScanUser(r.db.QueryRow(ctx, query, args...))
}

// GetUsersByIDs loads the registered users matching any of the given ids. It is
// used by the internal batch lookup (service-to-service). Absent ids are simply
// not present in the result; the caller decides how to handle a partial match.
func (r *Repository) GetUsersByIDs(ctx context.Context, ids []uuid.UUID) ([]*entity.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args, err := r.db.Builder.
		Select(dto.UserColumns).
		From("users").
		Where(sq.Eq{"id": ids}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get users by ids: %w", err)
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query users by ids: %w", err)
	}
	defer rows.Close()

	return dto.ScanUsers(rows)
}

func (r *Repository) CreateRefreshToken(ctx context.Context, token *entity.RefreshToken) (*entity.RefreshToken, error) {
	query, args, err := r.db.Builder.
		Insert("refresh_tokens").
		Columns(dto.RefreshTokenInsertColumns()...).
		Values(dto.RefreshTokenInsertValues(token)...).
		Suffix("RETURNING " + dto.RefreshTokenColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create refresh token: %w", err)
	}

	created, err := dto.ScanRefreshToken(r.db.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (r *Repository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*entity.RefreshToken, error) {
	query, args, err := r.db.Builder.
		Select(dto.RefreshTokenColumns).
		From("refresh_tokens").
		Where(sq.Eq{"token_hash": tokenHash}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get refresh token by hash: %w", err)
	}

	return dto.ScanRefreshToken(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) RotateRefreshToken(
	ctx context.Context,
	oldTokenID uuid.UUID,
	revokedAt time.Time,
	newToken *entity.RefreshToken,
) (*entity.RefreshToken, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin refresh token rotation: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	revokeQuery, revokeArgs, err := r.db.Builder.
		Update("refresh_tokens").
		Set("revoked_at", revokedAt).
		Where(sq.Eq{"id": dto.IDArg(oldTokenID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build revoke old refresh token: %w", err)
	}

	tag, err := tx.Exec(ctx, revokeQuery, revokeArgs...)
	if err != nil {
		return nil, fmt.Errorf("revoke old refresh token: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return nil, domainerrs.ErrNotFound
	}

	createQuery, createArgs, err := r.db.Builder.
		Insert("refresh_tokens").
		Columns(dto.RefreshTokenInsertColumns()...).
		Values(dto.RefreshTokenInsertValues(newToken)...).
		Suffix("RETURNING " + dto.RefreshTokenColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create rotated refresh token: %w", err)
	}

	created, err := dto.ScanRefreshToken(tx.QueryRow(ctx, createQuery, createArgs...))
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit refresh token rotation: %w", err)
	}

	return created, nil
}

func (r *Repository) RevokeRefreshTokenByHash(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	query, args, err := r.db.Builder.
		Update("refresh_tokens").
		Set("revoked_at", revokedAt).
		Where(sq.Eq{"token_hash": tokenHash}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build revoke refresh token: %w", err)
	}

	_, err = r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	return nil
}
