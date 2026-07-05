package dto

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
)

const TripTravelerColumns = "id, trip_id, name, email, linked_user_id, role, status, created_by_user_id, created_at, updated_at, removed_at"

func TripTravelerInsertColumns() []string {
	return []string{
		"id", "trip_id", "name", "email", "linked_user_id", "role", "status", "created_by_user_id",
	}
}

func TripTravelerInsertValues(t *entity.TripTraveler) []any {
	return []any{
		IDArg(t.ID),
		IDArg(t.TripID),
		t.Name,
		TextPtrArg(t.Email),
		toPgUUIDPtr(t.LinkedUserID),
		string(t.Role),
		string(t.Status),
		IDArg(t.CreatedByUserID),
	}
}

func TextPtrArg(s *string) pgtype.Text {
	if s == nil || *s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func ScanTripTraveler(row pgx.Row) (*entity.TripTraveler, error) {
	var (
		id, tripID, linkedUserID, createdByUserID pgtype.UUID
		name                                      string
		email                                     pgtype.Text
		role, status                              string
		createdAt, updatedAt, removedAt           pgtype.Timestamp
	)

	err := row.Scan(
		&id,
		&tripID,
		&name,
		&email,
		&linkedUserID,
		&role,
		&status,
		&createdByUserID,
		&createdAt,
		&updatedAt,
		&removedAt,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, domainerrs.ErrNotFound
		}
		return nil, fmt.Errorf("scan trip traveler: %w", err)
	}

	return &entity.TripTraveler{
		ID:              uuid.UUID(id.Bytes),
		TripID:          uuid.UUID(tripID.Bytes),
		Name:            name,
		Email:           fromPgTextPtr(email),
		LinkedUserID:    fromPgUUID(linkedUserID),
		Role:            entity.TripTravelerRole(role),
		Status:          entity.TripTravelerStatus(status),
		CreatedByUserID: uuid.UUID(createdByUserID.Bytes),
		CreatedAt:       createdAt.Time,
		UpdatedAt:       updatedAt.Time,
		RemovedAt:       fromPgTimestampPtr(removedAt),
	}, nil
}

func ScanTripTravelerRows(rows pgx.Rows) ([]entity.TripTraveler, error) {
	travelers := make([]entity.TripTraveler, 0)
	for rows.Next() {
		traveler, err := ScanTripTraveler(rows)
		if err != nil {
			return nil, err
		}
		travelers = append(travelers, *traveler)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trip travelers: %w", err)
	}
	return travelers, nil
}

func fromPgTextPtr(text pgtype.Text) *string {
	if !text.Valid {
		return nil
	}
	value := text.String
	return &value
}
