package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/calendar"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/storage/postgres"
)

type Repository struct {
	db *storage.DB
}

func New(db *storage.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpsertCalendarConnection(ctx context.Context, conn calendar.CalendarConnection) (*calendar.CalendarConnection, error) {
	query := `
INSERT INTO calendar_connections (
    id, user_id, provider, provider_account_email, access_token_encrypted,
    refresh_token_encrypted, token_expires_at, scopes, connected_at,
    updated_at, disconnected_at, status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW(), NULL, 'active')
ON CONFLICT (user_id, provider) DO UPDATE SET
    provider_account_email = EXCLUDED.provider_account_email,
    access_token_encrypted = EXCLUDED.access_token_encrypted,
    refresh_token_encrypted = COALESCE(EXCLUDED.refresh_token_encrypted, calendar_connections.refresh_token_encrypted),
    token_expires_at = EXCLUDED.token_expires_at,
    scopes = EXCLUDED.scopes,
    updated_at = NOW(),
    disconnected_at = NULL,
    status = 'active'
RETURNING id, user_id, provider, provider_account_email, access_token_encrypted,
          refresh_token_encrypted, token_expires_at, scopes, connected_at,
          updated_at, disconnected_at, status`
	return scanConnection(r.db.QueryRow(
		ctx,
		query,
		conn.ID,
		conn.UserID,
		conn.Provider,
		conn.ProviderAccountEmail,
		conn.AccessTokenEncrypted,
		conn.RefreshTokenEncrypted,
		conn.TokenExpiresAt,
		conn.Scopes,
	))
}

func (r *Repository) GetCalendarConnectionByUserAndProvider(ctx context.Context, userID uuid.UUID, provider string) (*calendar.CalendarConnection, error) {
	query := `
SELECT id, user_id, provider, provider_account_email, access_token_encrypted,
       refresh_token_encrypted, token_expires_at, scopes, connected_at,
       updated_at, disconnected_at, status
FROM calendar_connections
WHERE user_id = $1 AND provider = $2`
	conn, err := scanConnection(r.db.QueryRow(ctx, query, userID, provider))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, calendar.ErrCalendarNotConnected
		}
		return nil, err
	}
	return conn, nil
}

func (r *Repository) GetActiveCalendarConnection(ctx context.Context, userID uuid.UUID, provider string) (*calendar.CalendarConnection, error) {
	query := `
SELECT id, user_id, provider, provider_account_email, access_token_encrypted,
       refresh_token_encrypted, token_expires_at, scopes, connected_at,
       updated_at, disconnected_at, status
FROM calendar_connections
WHERE user_id = $1 AND provider = $2 AND status = 'active' AND disconnected_at IS NULL`
	conn, err := scanConnection(r.db.QueryRow(ctx, query, userID, provider))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, calendar.ErrCalendarNotConnected
		}
		return nil, err
	}
	return conn, nil
}

func (r *Repository) DisconnectCalendarConnection(ctx context.Context, userID uuid.UUID, provider string) error {
	_, err := r.db.Exec(
		ctx,
		`UPDATE calendar_connections
		 SET status = 'disconnected', disconnected_at = NOW(), updated_at = NOW()
		 WHERE user_id = $1 AND provider = $2`,
		userID,
		provider,
	)
	if err != nil {
		return fmt.Errorf("disconnect calendar connection: %w", err)
	}
	return nil
}

func (r *Repository) UpdateCalendarTokens(
	ctx context.Context,
	userID uuid.UUID,
	provider, accessTokenEncrypted string,
	refreshTokenEncrypted *string,
	expiresAt *time.Time,
	scopes string,
) error {
	_, err := r.db.Exec(
		ctx,
		`UPDATE calendar_connections
		 SET access_token_encrypted = $3,
		     refresh_token_encrypted = COALESCE($4, refresh_token_encrypted),
		     token_expires_at = $5,
		     scopes = COALESCE(NULLIF($6, ''), scopes),
		     updated_at = NOW()
		 WHERE user_id = $1 AND provider = $2 AND status = 'active'`,
		userID,
		provider,
		accessTokenEncrypted,
		refreshTokenEncrypted,
		expiresAt,
		scopes,
	)
	if err != nil {
		return fmt.Errorf("update calendar tokens: %w", err)
	}
	return nil
}

func (r *Repository) CreateOAuthState(ctx context.Context, state calendar.OAuthState) error {
	_, err := r.db.Exec(
		ctx,
		`INSERT INTO calendar_oauth_states (state, user_id, provider, return_url, expires_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		state.State,
		state.UserID,
		state.Provider,
		state.ReturnURL,
		state.ExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("create oauth state: %w", err)
	}
	return nil
}

func (r *Repository) GetOAuthState(ctx context.Context, state string) (*calendar.OAuthState, error) {
	query := `
SELECT state, user_id, provider, return_url, created_at, expires_at, used_at
FROM calendar_oauth_states
WHERE state = $1`
	row := r.db.QueryRow(ctx, query, state)
	var out calendar.OAuthState
	var returnURL sql.NullString
	var usedAt sql.NullTime
	if err := row.Scan(&out.State, &out.UserID, &out.Provider, &returnURL, &out.CreatedAt, &out.ExpiresAt, &usedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, calendar.ErrInvalidOAuthState
		}
		return nil, fmt.Errorf("scan oauth state: %w", err)
	}
	out.ReturnURL = stringPtr(returnURL)
	out.UsedAt = timePtr(usedAt)
	return &out, nil
}

func (r *Repository) MarkOAuthStateUsed(ctx context.Context, state string) (bool, error) {
	tag, err := r.db.Exec(
		ctx,
		`UPDATE calendar_oauth_states
		 SET used_at = NOW()
		 WHERE state = $1 AND used_at IS NULL AND expires_at > NOW()`,
		state,
	)
	if err != nil {
		return false, fmt.Errorf("mark oauth state used: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repository) DeleteExpiredOAuthStates(ctx context.Context, now time.Time) error {
	_, err := r.db.Exec(ctx, `DELETE FROM calendar_oauth_states WHERE expires_at < $1`, now)
	if err != nil {
		return fmt.Errorf("delete expired oauth states: %w", err)
	}
	return nil
}

func scanConnection(row pgx.Row) (*calendar.CalendarConnection, error) {
	var out calendar.CalendarConnection
	var email sql.NullString
	var refresh sql.NullString
	var expires sql.NullTime
	var scopes sql.NullString
	var disconnected sql.NullTime
	if err := row.Scan(
		&out.ID,
		&out.UserID,
		&out.Provider,
		&email,
		&out.AccessTokenEncrypted,
		&refresh,
		&expires,
		&scopes,
		&out.ConnectedAt,
		&out.UpdatedAt,
		&disconnected,
		&out.Status,
	); err != nil {
		return nil, fmt.Errorf("scan calendar connection: %w", err)
	}
	out.ProviderAccountEmail = stringPtr(email)
	out.RefreshTokenEncrypted = stringPtr(refresh)
	out.TokenExpiresAt = timePtr(expires)
	out.Scopes = stringPtr(scopes)
	out.DisconnectedAt = timePtr(disconnected)
	return &out, nil
}

func stringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	v := value.String
	return &v
}

func timePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	v := value.Time
	return &v
}
