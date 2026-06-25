package dto

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/storage/postgres"
)

// Columns is the canonical column projection for notification rows.
const Columns = "id, user_id, trip_id, actor_user_id, type, title, message, " +
	"entity_type, entity_id, metadata, read_at, created_at"

// InsertColumns returns the columns set on INSERT. created_at and read_at are
// intentionally omitted so the table defaults (NOW() / NULL) apply.
func InsertColumns() []string {
	return []string{
		"id", "user_id", "trip_id", "actor_user_id", "type", "title",
		"message", "entity_type", "entity_id", "metadata",
	}
}

// InsertValues returns the values for InsertColumns, in matching order. Metadata
// is always marshalled to a non-nil JSON object so the NOT NULL column holds.
func InsertValues(n *entity.Notification) ([]any, error) {
	metadata, err := marshalMetadata(n.Metadata)
	if err != nil {
		return nil, err
	}
	return []any{
		toPgUUID(n.ID),
		toPgUUID(n.UserID),
		toPgUUIDPtr(n.TripID),
		toPgUUIDPtr(n.ActorUserID),
		n.Type,
		n.Title,
		n.Message,
		toPgTextPtr(n.EntityType),
		toPgUUIDPtr(n.EntityID),
		metadata,
	}, nil
}

// IDArg encodes a notification id for use in a WHERE clause.
func IDArg(id uuid.UUID) pgtype.UUID {
	return toPgUUID(id)
}

// Scan reads a single row (in Columns order) into a domain Notification. It
// returns domain errs.ErrNotFound when the row is absent.
func Scan(row pgx.Row) (*entity.Notification, error) {
	var (
		id, userID  pgtype.UUID
		tripID      pgtype.UUID
		actorUserID pgtype.UUID
		ntype       string
		title       string
		message     string
		entityType  pgtype.Text
		entityID    pgtype.UUID
		metadataRaw []byte
		readAt      pgtype.Timestamp
		createdAt   pgtype.Timestamp
	)

	err := row.Scan(
		&id,
		&userID,
		&tripID,
		&actorUserID,
		&ntype,
		&title,
		&message,
		&entityType,
		&entityID,
		&metadataRaw,
		&readAt,
		&createdAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan notification: %w", err)
	}

	metadata, err := unmarshalMetadata(metadataRaw)
	if err != nil {
		return nil, err
	}

	return &entity.Notification{
		ID:          uuid.UUID(id.Bytes),
		UserID:      uuid.UUID(userID.Bytes),
		TripID:      fromPgUUID(tripID),
		ActorUserID: fromPgUUID(actorUserID),
		Type:        ntype,
		Title:       title,
		Message:     message,
		EntityType:  fromPgText(entityType),
		EntityID:    fromPgUUID(entityID),
		Metadata:    metadata,
		ReadAt:      timestampPtr(readAt),
		CreatedAt:   createdAt.Time,
	}, nil
}

// ScanRows maps a set of notification rows to domain entities.
func ScanRows(rows pgx.Rows) ([]entity.Notification, error) {
	notifications := make([]entity.Notification, 0)
	for rows.Next() {
		notification, err := Scan(rows)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, *notification)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notifications: %w", err)
	}
	return notifications, nil
}

// --- mapping helpers: domain (plain Go) <-> pgtype ---

func marshalMetadata(metadata map[string]any) ([]byte, error) {
	if len(metadata) == 0 {
		return []byte("{}"), nil
	}
	b, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("marshal notification metadata: %w", err)
	}
	return b, nil
}

func unmarshalMetadata(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var metadata map[string]any
	if err := json.Unmarshal(raw, &metadata); err != nil {
		return nil, fmt.Errorf("unmarshal notification metadata: %w", err)
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	return metadata, nil
}

func toPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

func toPgUUIDPtr(id *uuid.UUID) pgtype.UUID {
	if id == nil {
		return pgtype.UUID{Valid: false}
	}
	return pgtype.UUID{Bytes: *id, Valid: true}
}

func fromPgUUID(p pgtype.UUID) *uuid.UUID {
	if !p.Valid {
		return nil
	}
	id := uuid.UUID(p.Bytes)
	return &id
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

func timestampPtr(ts pgtype.Timestamp) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}
