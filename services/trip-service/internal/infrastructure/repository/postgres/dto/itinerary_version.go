package dto

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

// ItineraryVersionColumns is the canonical full snapshot column order.
const ItineraryVersionColumns = "id, trip_id, user_id, created_by_user_id, version_number, source, itinerary, metadata, created_at"

// ItineraryVersionSummaryColumns omits the large itinerary JSON for list views.
const ItineraryVersionSummaryColumns = "id, trip_id, user_id, created_by_user_id, version_number, source, metadata, created_at"

// ItineraryVersionInsertColumns returns the columns set on INSERT.
func ItineraryVersionInsertColumns() []string {
	return []string{
		"id", "trip_id", "user_id", "created_by_user_id", "version_number", "source", "itinerary", "metadata",
	}
}

// ItineraryVersionInsertValues returns values for ItineraryVersionInsertColumns.
func ItineraryVersionInsertValues(v *entity.ItineraryVersion) ([]any, error) {
	metadata, err := marshalMetadata(v.Metadata)
	if err != nil {
		return nil, err
	}
	return []any{
		toPgUUID(v.ID),
		toPgUUID(v.TripID),
		toPgUUID(v.UserID),
		toPgUUIDPtr(v.CreatedByUserID),
		v.VersionNumber,
		string(v.Source),
		[]byte(v.Itinerary),
		metadata,
	}, nil
}

// ScanItineraryVersion reads a full snapshot row.
func ScanItineraryVersion(row pgx.Row) (*entity.ItineraryVersion, error) {
	var (
		id, tripID, userID, createdByUserID pgtype.UUID
		versionNumber                       int
		source                              string
		itineraryRaw                        []byte
		metadataRaw                         []byte
		createdAt                           pgtype.Timestamp
	)

	err := row.Scan(
		&id,
		&tripID,
		&userID,
		&createdByUserID,
		&versionNumber,
		&source,
		&itineraryRaw,
		&metadataRaw,
		&createdAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan itinerary version: %w", err)
	}

	return itineraryVersionFromScannedValues(
		id,
		tripID,
		userID,
		createdByUserID,
		versionNumber,
		source,
		itineraryRaw,
		metadataRaw,
		createdAt.Time,
	)
}

// ScanItineraryVersionSummary reads a row without the itinerary payload.
func ScanItineraryVersionSummary(row pgx.Row) (*entity.ItineraryVersion, error) {
	var (
		id, tripID, userID, createdByUserID pgtype.UUID
		versionNumber                       int
		source                              string
		metadataRaw                         []byte
		createdAt                           pgtype.Timestamp
	)

	err := row.Scan(
		&id,
		&tripID,
		&userID,
		&createdByUserID,
		&versionNumber,
		&source,
		&metadataRaw,
		&createdAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan itinerary version summary: %w", err)
	}

	return itineraryVersionFromScannedValues(
		id,
		tripID,
		userID,
		createdByUserID,
		versionNumber,
		source,
		nil,
		metadataRaw,
		createdAt.Time,
	)
}

func itineraryVersionFromScannedValues(
	id, tripID, userID, createdByUserID pgtype.UUID,
	versionNumber int,
	source string,
	itineraryRaw []byte,
	metadataRaw []byte,
	createdAt time.Time,
) (*entity.ItineraryVersion, error) {
	metadata, err := unmarshalMetadata(metadataRaw)
	if err != nil {
		return nil, err
	}

	return &entity.ItineraryVersion{
		ID:              uuid.UUID(id.Bytes),
		TripID:          uuid.UUID(tripID.Bytes),
		UserID:          uuid.UUID(userID.Bytes),
		CreatedByUserID: fromPgUUID(createdByUserID),
		VersionNumber:   versionNumber,
		Source:          entity.ItineraryVersionSource(source),
		Itinerary:       json.RawMessage(itineraryRaw),
		Metadata:        metadata,
		CreatedAt:       createdAt,
	}, nil
}

func marshalMetadata(metadata map[string]any) ([]byte, error) {
	if metadata == nil {
		return nil, nil
	}
	b, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal itinerary version metadata: %w", err)
	}
	return b, nil
}

func unmarshalMetadata(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}

	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil, fmt.Errorf("unmarshal itinerary version metadata: %w", err)
	}
	if metadata == nil {
		return map[string]any{}, nil
	}
	return metadata, nil
}
