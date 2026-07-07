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

// TripActivityEventColumns is the canonical column projection for activity rows.
const TripActivityEventColumns = "id, trip_id, actor_user_id, event_type, entity_type, entity_id, metadata, created_at"

// TripActivityEventInsertColumns returns the columns set on INSERT. created_at
// is intentionally omitted so the table default (NOW()) applies.
func TripActivityEventInsertColumns() []string {
	return []string{"id", "trip_id", "actor_user_id", "event_type", "entity_type", "entity_id", "metadata"}
}

// TripActivityEventInsertValues returns values matching TripActivityEventInsertColumns.
// Metadata is always marshalled to a non-nil JSON object so the NOT NULL column
// constraint holds even for events with no metadata.
func TripActivityEventInsertValues(e *entity.TripActivityEvent) ([]any, error) {
	metadata, err := marshalActivityMetadata(e.Metadata)
	if err != nil {
		return nil, err
	}
	return []any{
		toPgUUID(e.ID),
		toPgUUID(e.TripID),
		toPgUUIDPtr(e.ActorUserID),
		e.EventType,
		toPgTextPtr(e.EntityType),
		toPgUUIDPtr(e.EntityID),
		metadata,
	}, nil
}

// ScanTripActivityEvent maps a single activity row (in TripActivityEventColumns
// order) to its domain entity.
func ScanTripActivityEvent(row pgx.Row) (*entity.TripActivityEvent, error) {
	var (
		id, tripID  pgtype.UUID
		actorUserID pgtype.UUID
		eventType   string
		entityType  pgtype.Text
		entityID    pgtype.UUID
		metadataRaw []byte
		createdAt   pgtype.Timestamp
	)

	err := row.Scan(
		&id,
		&tripID,
		&actorUserID,
		&eventType,
		&entityType,
		&entityID,
		&metadataRaw,
		&createdAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip activity event: %w", err)
	}

	metadata, err := unmarshalMetadata(metadataRaw)
	if err != nil {
		return nil, err
	}

	return &entity.TripActivityEvent{
		ID:          uuid.UUID(id.Bytes),
		TripID:      uuid.UUID(tripID.Bytes),
		ActorUserID: fromPgUUID(actorUserID),
		EventType:   eventType,
		EntityType:  fromPgText(entityType),
		EntityID:    fromPgUUID(entityID),
		Metadata:    metadata,
		CreatedAt:   createdAt.Time,
	}, nil
}

// ScanTripActivityEventRows maps a set of activity rows to domain entities.
func ScanTripActivityEventRows(rows pgx.Rows) ([]entity.TripActivityEvent, error) {
	events := make([]entity.TripActivityEvent, 0)
	for rows.Next() {
		event, err := ScanTripActivityEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, *event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip activity events: %w", err)
	}
	return events, nil
}

// marshalActivityMetadata always returns a non-nil JSON object so the NOT NULL
// metadata column is satisfied.
func marshalActivityMetadata(metadata map[string]any) ([]byte, error) {
	if len(metadata) == 0 {
		return []byte("{}"), nil
	}
	b, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal trip activity metadata: %w", err)
	}
	return b, nil
}

func toPgTextPtr(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func fromPgText(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}
