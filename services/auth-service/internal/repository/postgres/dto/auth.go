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

// ScanUsers reads a set of user rows (used by the internal batch lookup). It
// returns the users in result order; callers match them back to the requested
// ids and treat any absent id as "not found".
func ScanUsers(rows pgx.Rows) ([]*entity.User, error) {
	users := make([]*entity.User, 0)
	for rows.Next() {
		var user entity.User
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.PasswordHash,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}
		users = append(users, &user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user rows: %w", err)
	}
	return users, nil
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
