package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres/dto"
)

func (r *Repository) UpsertTripAvailabilityResponse(
	ctx context.Context,
	response *entity.TripAvailabilityResponse,
) (*entity.TripAvailabilityResponse, error) {
	available, err := dto.AvailabilityRangesJSON(response.AvailableRanges)
	if err != nil {
		return nil, err
	}
	unavailable, err := dto.AvailabilityRangesJSON(response.UnavailableRanges)
	if err != nil {
		return nil, err
	}
	preferred, err := dto.AvailabilityRangesJSON(response.PreferredRanges)
	if err != nil {
		return nil, err
	}

	query, args, err := r.db.Builder.
		Insert("trip_availability_responses").
		Columns(
			"id",
			"trip_id",
			"user_id",
			"available_ranges_json",
			"unavailable_ranges_json",
			"preferred_ranges_json",
			"min_trip_days",
			"max_trip_days",
			"timezone",
			"notes",
		).
		Values(
			dto.IDArg(response.ID),
			dto.IDArg(response.TripID),
			dto.IDArg(response.UserID),
			available,
			unavailable,
			preferred,
			dto.IntNullableArg(response.MinTripDays),
			dto.IntNullableArg(response.MaxTripDays),
			dto.TextNullableArg(response.Timezone),
			dto.TextNullableArg(response.Notes),
		).
		Suffix(`
ON CONFLICT (trip_id, user_id) DO UPDATE SET
    available_ranges_json = EXCLUDED.available_ranges_json,
    unavailable_ranges_json = EXCLUDED.unavailable_ranges_json,
    preferred_ranges_json = EXCLUDED.preferred_ranges_json,
    min_trip_days = EXCLUDED.min_trip_days,
    max_trip_days = EXCLUDED.max_trip_days,
    timezone = EXCLUDED.timezone,
    notes = EXCLUDED.notes,
    updated_at = NOW()
RETURNING ` + dto.TripAvailabilityResponseColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build upsert trip availability response: %w", err)
	}
	return dto.ScanTripAvailabilityResponse(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) GetTripAvailabilityResponseByTripAndUser(
	ctx context.Context,
	tripID, userID uuid.UUID,
) (*entity.TripAvailabilityResponse, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripAvailabilityResponseColumns).
		From("trip_availability_responses").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "user_id": dto.IDArg(userID)}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build get trip availability response: %w", err)
	}
	return dto.ScanTripAvailabilityResponse(r.db.QueryRow(ctx, query, args...))
}

func (r *Repository) ListTripAvailabilityResponsesByTrip(
	ctx context.Context,
	tripID uuid.UUID,
) ([]entity.TripAvailabilityResponse, error) {
	query, args, err := r.db.Builder.
		Select(dto.TripAvailabilityResponseColumns).
		From("trip_availability_responses").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		OrderBy("updated_at DESC", "created_at DESC", "id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build list trip availability responses: %w", err)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query trip availability responses: %w", err)
	}
	defer rows.Close()
	return dto.ScanTripAvailabilityResponseRows(rows)
}

func (r *Repository) DeleteTripAvailabilityResponse(ctx context.Context, tripID, userID uuid.UUID) error {
	query, args, err := r.db.Builder.
		Delete("trip_availability_responses").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID), "user_id": dto.IDArg(userID)}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build delete trip availability response: %w", err)
	}
	if _, err := r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("delete trip availability response: %w", err)
	}
	return nil
}

func (r *Repository) CountTripAvailabilityResponsesByTrip(ctx context.Context, tripID uuid.UUID) (int, error) {
	query, args, err := r.db.Builder.
		Select("COUNT(*)").
		From("trip_availability_responses").
		Where(sq.Eq{"trip_id": dto.IDArg(tripID)}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("build count trip availability responses: %w", err)
	}
	var count int
	if err := r.db.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("count trip availability responses: %w", err)
	}
	return count, nil
}

func (r *Repository) UpdateTripDatesAndMetadata(
	ctx context.Context,
	id, userID uuid.UUID,
	startDate *time.Time,
	days int32,
	route *aggregate.TripRoute,
	metadata map[string]any,
) (*entity.Trip, error) {
	routeRaw, err := json.Marshal(route)
	if err != nil {
		return nil, fmt.Errorf("marshal route: %w", err)
	}
	if route == nil {
		routeRaw = nil
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataRaw, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal trip creation metadata: %w", err)
	}

	query, args, err := r.db.Builder.
		Update("trips").
		Set("start_date", dto.DateArg(startDate)).
		Set("days", days).
		Set("route_json", routeRaw).
		Set("creation_metadata", metadataRaw).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": dto.IDArg(id), "user_id": dto.IDArg(userID)}).
		Suffix("RETURNING " + dto.Columns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build update trip dates and metadata: %w", err)
	}
	return dto.Scan(r.db.QueryRow(ctx, query, args...))
}
