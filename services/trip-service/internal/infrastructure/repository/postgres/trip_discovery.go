package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/tripdiscovery"
)

const discoverySessionColumns = "id, user_id, workspace_id, parent_session_id, mode, prompt, " +
	"output_language, status, request_json, response_json, created_trip_id, created_at, updated_at"

func (r *Repository) CreateTripDiscoverySession(
	ctx context.Context,
	session *tripdiscovery.Session,
) (*tripdiscovery.Session, error) {
	requestJSON, err := json.Marshal(session.Request)
	if err != nil {
		return nil, fmt.Errorf("marshal discovery request: %w", err)
	}
	responseJSON, err := json.Marshal(session.Response)
	if err != nil {
		return nil, fmt.Errorf("marshal discovery response: %w", err)
	}
	query, args, err := r.db.Builder.
		Insert("trip_discovery_sessions").
		Columns(
			"id", "user_id", "workspace_id", "parent_session_id", "mode", "prompt",
			"output_language", "status", "request_json", "response_json",
		).
		Values(
			session.ID,
			session.UserID,
			session.WorkspaceID,
			session.ParentSessionID,
			session.Mode,
			nullString(session.Prompt),
			session.OutputLanguage,
			session.Status,
			requestJSON,
			responseJSON,
		).
		Suffix("RETURNING " + discoverySessionColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create discovery session: %w", err)
	}
	return scanTripDiscoverySession(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetTripDiscoverySessionByIDAndUser(
	ctx context.Context,
	id, userID uuid.UUID,
) (*tripdiscovery.Session, error) {
	query, args, err := r.db.Builder.
		Select(discoverySessionColumns).
		From("trip_discovery_sessions").
		Where(sq.Eq{"id": id, "user_id": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get discovery session: %w", err)
	}
	return scanTripDiscoverySession(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListTripDiscoverySessionsByUser(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]tripdiscovery.Session, error) {
	query, args, err := r.db.Builder.
		Select(discoverySessionColumns).
		From("trip_discovery_sessions").
		Where(sq.Eq{"user_id": userID}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list discovery sessions: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query discovery sessions: %w", err)
	}
	defer rows.Close()
	items := make([]tripdiscovery.Session, 0)
	for rows.Next() {
		item, err := scanTripDiscoverySession(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate discovery sessions: %w", err)
	}
	return items, nil
}

func (r *Repository) MarkTripDiscoverySessionCreatedTrip(
	ctx context.Context,
	id, userID, tripID uuid.UUID,
) (*tripdiscovery.Session, error) {
	query, args, err := r.db.Builder.
		Update("trip_discovery_sessions").
		Set("status", "created_trip").
		Set("created_trip_id", tripID).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": id, "user_id": userID}).
		Suffix("RETURNING " + discoverySessionColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mark discovery session: %w", err)
	}
	return scanTripDiscoverySession(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) UpdateTripCreationMetadata(
	ctx context.Context,
	id, userID uuid.UUID,
	metadata map[string]any,
) (*entity.Trip, error) {
	raw, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal trip creation metadata: %w", err)
	}
	query, args, err := r.db.Builder.
		Update("trips").
		Set("creation_metadata", raw).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": id, "user_id": userID}).
		Suffix("RETURNING " + dto.Columns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update trip creation metadata: %w", err)
	}
	return dto.Scan(r.db.QueryRow(ctx, query, args...))
}

func scanTripDiscoverySession(row pgx.Row) (*tripdiscovery.Session, error) {
	var (
		id, userID, workspaceID, parentSessionID, createdTripID pgtype.UUID
		mode, outputLanguage, status                            string
		prompt                                                  pgtype.Text
		requestJSON, responseJSON                               []byte
		createdAt, updatedAt                                    pgtype.Timestamp
	)
	err := row.Scan(
		&id,
		&userID,
		&workspaceID,
		&parentSessionID,
		&mode,
		&prompt,
		&outputLanguage,
		&status,
		&requestJSON,
		&responseJSON,
		&createdTripID,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan discovery session: %w", err)
	}
	var request tripdiscovery.AIRequest
	if err := json.Unmarshal(requestJSON, &request); err != nil {
		return nil, fmt.Errorf("decode discovery request: %w", err)
	}
	var response tripdiscovery.SuggestionResponse
	if err := json.Unmarshal(responseJSON, &response); err != nil {
		return nil, fmt.Errorf("decode discovery response: %w", err)
	}
	return &tripdiscovery.Session{
		ID:              uuid.UUID(id.Bytes),
		UserID:          uuid.UUID(userID.Bytes),
		WorkspaceID:     nullableUUID(workspaceID),
		ParentSessionID: nullableUUID(parentSessionID),
		Mode:            tripdiscovery.Mode(mode),
		Prompt:          prompt.String,
		OutputLanguage:  outputLanguage,
		Status:          status,
		Request:         request,
		Response:        response,
		CreatedTripID:   nullableUUID(createdTripID),
		CreatedAt:       createdAt.Time,
		UpdatedAt:       updatedAt.Time,
	}, nil
}

func nullableUUID(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	result := uuid.UUID(value.Bytes)
	return &result
}

func nullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
