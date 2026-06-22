package dto

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/auth-service/pkg/storage/postgres"
)

const (
	UserColumns = "id, email, password_hash, created_at, updated_at"

	RefreshTokenColumns = "id, user_id, token_hash, expires_at, revoked_at, created_at"
)

func UserInsertColumns() []string {
	return []string{"id", "email", "password_hash"}
}

func UserInsertValues(user *entity.User) []any {
	return []any{user.ID, user.Email, user.PasswordHash}
}

func RefreshTokenInsertColumns() []string {
	return []string{"id", "user_id", "token_hash", "expires_at"}
}

func RefreshTokenInsertValues(token *entity.RefreshToken) []any {
	return []any{token.ID, token.UserID, token.TokenHash, token.ExpiresAt}
}

func IDArg(id uuid.UUID) uuid.UUID {
	return id
}

func ScanUser(row pgx.Row) (*entity.User, error) {
	var user entity.User
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &user, nil
}

func ScanRefreshToken(row pgx.Row) (*entity.RefreshToken, error) {
	var (
		token     entity.RefreshToken
		revokedAt pgtype.Timestamptz
	)
	err := row.Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&revokedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan refresh token: %w", err)
	}
	if revokedAt.Valid {
		t := revokedAt.Time
		token.RevokedAt = &t
	}
	return &token, nil
}
