package postgres

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/routealternatives"
)

func (r *Repository) CreateRouteAlternativeSession(
	ctx context.Context,
	session *routealternatives.Session,
) (*routealternatives.Session, error) {
	query, args, err := r.db.Builder.
		Insert("route_alternative_sessions").
		Columns(
			"id",
			"user_id",
			"trip_id",
			"workspace_id",
			"source",
			"prompt",
			"output_language",
			"status",
			"request_json",
			"response_json",
			"parent_session_id",
		).
		Values(
			dto.IDArg(session.ID),
			dto.IDArg(session.UserID),
			dto.UUIDNullableArg(session.TripID),
			dto.UUIDNullableArg(session.WorkspaceID),
			session.Source,
			dto.TextNullableArg(session.Prompt),
			session.OutputLanguage,
			session.Status,
			[]byte(session.RequestJSON),
			[]byte(session.ResponseJSON),
			dto.UUIDNullableArg(session.ParentSessionID),
		).
		Suffix("RETURNING " + dto.RouteAlternativeSessionColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build create route alternative session: %w", err)
	}
	return dto.ScanRouteAlternativeSession(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetRouteAlternativeSessionByID(
	ctx context.Context,
	id uuid.UUID,
) (*routealternatives.Session, error) {
	query, args, err := r.db.Builder.
		Select(dto.RouteAlternativeSessionColumns).
		From("route_alternative_sessions").
		Where(sq.Eq{"id": dto.IDArg(id)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get route alternative session: %w", err)
	}
	return dto.ScanRouteAlternativeSession(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListRouteAlternativeSessionsByTrip(
	ctx context.Context,
	tripID uuid.UUID,
	limit int,
) ([]routealternatives.Session, error) {
	query, args, err := r.db.Builder.
		Select(dto.RouteAlternativeSessionColumns).
		From("route_alternative_sessions").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		Where(sq.NotEq{"status": routealternatives.StatusArchived}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip route alternative sessions: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip route alternative sessions: %w", err)
	}
	defer rows.Close()
	return dto.ScanRouteAlternativeSessionRows(rows)
}

func (r *Repository) ListRouteAlternativeSessionsByUser(
	ctx context.Context,
	userID uuid.UUID,
	limit int,
) ([]routealternatives.Session, error) {
	query, args, err := r.db.Builder.
		Select(dto.RouteAlternativeSessionColumns).
		From("route_alternative_sessions").
		Where(sq.Eq{"user_id": dto.IDArg(userID)}).
		Where(sq.NotEq{"status": routealternatives.StatusArchived}).
		OrderBy("created_at DESC").
		Limit(uint64(limit)).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list user route alternative sessions: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query user route alternative sessions: %w", err)
	}
	defer rows.Close()
	return dto.ScanRouteAlternativeSessionRows(rows)
}

func (r *Repository) MarkRouteAlternativeSessionCreatedTrip(
	ctx context.Context,
	id uuid.UUID,
	alternativeID string,
	createdTripID uuid.UUID,
) (*routealternatives.Session, error) {
	query, args, err := r.db.Builder.
		Update("route_alternative_sessions").
		Set("status", routealternatives.StatusCreatedTrip).
		Set("selected_alternative_id", alternativeID).
		Set("created_trip_id", dto.IDArg(createdTripID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(id)}).
		Suffix("RETURNING " + dto.RouteAlternativeSessionColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mark route alternative created trip: %w", err)
	}
	return dto.ScanRouteAlternativeSession(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) MarkRouteAlternativeSessionApplied(
	ctx context.Context,
	id uuid.UUID,
	alternativeID string,
	appliedToTripID uuid.UUID,
) (*routealternatives.Session, error) {
	query, args, err := r.db.Builder.
		Update("route_alternative_sessions").
		Set("status", routealternatives.StatusApplied).
		Set("selected_alternative_id", alternativeID).
		Set("applied_to_trip_id", dto.IDArg(appliedToTripID)).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(id)}).
		Suffix("RETURNING " + dto.RouteAlternativeSessionColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build mark route alternative applied: %w", err)
	}
	return dto.ScanRouteAlternativeSession(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ArchiveRouteAlternativeSession(
	ctx context.Context,
	id uuid.UUID,
) (*routealternatives.Session, error) {
	query, args, err := r.db.Builder.
		Update("route_alternative_sessions").
		Set("status", routealternatives.StatusArchived).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(id)}).
		Suffix("RETURNING " + dto.RouteAlternativeSessionColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build archive route alternative session: %w", err)
	}
	return dto.ScanRouteAlternativeSession(r.db.QueryRow(ctx, query, args...))
}
