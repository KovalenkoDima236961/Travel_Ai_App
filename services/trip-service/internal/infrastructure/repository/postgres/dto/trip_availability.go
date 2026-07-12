package dto

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

const TripAvailabilityResponseColumns = "id, trip_id, user_id, available_ranges_json, unavailable_ranges_json, preferred_ranges_json, min_trip_days, max_trip_days, timezone, notes, created_at, updated_at"

func AvailabilityRangesJSON(ranges []entity.AvailabilityDateRange) ([]byte, error) {
	if ranges == nil {
		ranges = []entity.AvailabilityDateRange{}
	}
	raw, err := json.Marshal(ranges)
	if err != nil {
		return nil, fmt.Errorf("marshal availability ranges: %w", err)
	}
	return raw, nil
}

func ScanTripAvailabilityResponse(row pgx.Row) (*entity.TripAvailabilityResponse, error) {
	var (
		id, tripID, userID           pgtype.UUID
		availableRaw, unavailableRaw []byte
		preferredRaw                 []byte
		minTripDays, maxTripDays     pgtype.Int4
		timezone, notes              pgtype.Text
		createdAt, updatedAt         pgtype.Timestamp
	)
	err := row.Scan(
		&id,
		&tripID,
		&userID,
		&availableRaw,
		&unavailableRaw,
		&preferredRaw,
		&minTripDays,
		&maxTripDays,
		&timezone,
		&notes,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip availability response: %w", err)
	}

	available, err := unmarshalAvailabilityRanges(availableRaw)
	if err != nil {
		return nil, err
	}
	unavailable, err := unmarshalAvailabilityRanges(unavailableRaw)
	if err != nil {
		return nil, err
	}
	preferred, err := unmarshalAvailabilityRanges(preferredRaw)
	if err != nil {
		return nil, err
	}

	return &entity.TripAvailabilityResponse{
		ID:                uuid.UUID(id.Bytes),
		TripID:            uuid.UUID(tripID.Bytes),
		UserID:            uuid.UUID(userID.Bytes),
		AvailableRanges:   available,
		UnavailableRanges: unavailable,
		PreferredRanges:   preferred,
		MinTripDays:       intPtr(minTripDays),
		MaxTripDays:       intPtr(maxTripDays),
		Timezone:          textValue(timezone),
		Notes:             textValue(notes),
		CreatedAt:         createdAt.Time,
		UpdatedAt:         updatedAt.Time,
	}, nil
}

func ScanTripAvailabilityResponseRows(rows pgx.Rows) ([]entity.TripAvailabilityResponse, error) {
	items := make([]entity.TripAvailabilityResponse, 0)
	for rows.Next() {
		item, err := ScanTripAvailabilityResponse(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip availability responses: %w", err)
	}
	return items, nil
}

func IntNullableArg(value *int) pgtype.Int4 {
	if value == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*value), Valid: true}
}

func unmarshalAvailabilityRanges(raw []byte) ([]entity.AvailabilityDateRange, error) {
	if len(raw) == 0 {
		return []entity.AvailabilityDateRange{}, nil
	}
	var ranges []entity.AvailabilityDateRange
	if err := json.Unmarshal(raw, &ranges); err != nil {
		return nil, fmt.Errorf("unmarshal availability ranges: %w", err)
	}
	if ranges == nil {
		return []entity.AvailabilityDateRange{}, nil
	}
	return ranges, nil
}
