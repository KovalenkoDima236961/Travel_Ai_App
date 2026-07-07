package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
)

// PreferenceColumns is the canonical projection for notification preference rows.
const PreferenceColumns = "id, user_id, channel, category, enabled, created_at, updated_at"

// PreferenceInsertColumns returns the columns set on preference INSERT/UPSERT.
// id, created_at, and updated_at are left to database defaults.
func PreferenceInsertColumns() []string {
	return []string{"user_id", "channel", "category", "enabled"}
}

// PreferenceInsertValues returns values in PreferenceInsertColumns order.
func PreferenceInsertValues(userID pgtype.UUID, channel, category string, enabled bool) []any {
	return []any{userID, channel, category, enabled}
}

// ScanPreference reads a single preference row in PreferenceColumns order.
func ScanPreference(row pgx.Row) (*entity.NotificationPreference, error) {
	var (
		id        pgtype.UUID
		userID    pgtype.UUID
		channel   string
		category  string
		enabled   bool
		createdAt pgtype.Timestamp
		updatedAt pgtype.Timestamp
	)

	if err := row.Scan(&id, &userID, &channel, &category, &enabled, &createdAt, &updatedAt); err != nil {
		return nil, fmt.Errorf("scan notification preference: %w", err)
	}

	return &entity.NotificationPreference{
		ID:        uuid.UUID(id.Bytes),
		UserID:    uuid.UUID(userID.Bytes),
		Channel:   channel,
		Category:  category,
		Enabled:   enabled,
		CreatedAt: timestampValue(createdAt),
		UpdatedAt: timestampValue(updatedAt),
	}, nil
}

// ScanPreferenceRows maps a set of preference rows to domain entities.
func ScanPreferenceRows(rows pgx.Rows) ([]entity.NotificationPreference, error) {
	preferences := make([]entity.NotificationPreference, 0)
	for rows.Next() {
		preference, err := ScanPreference(rows)
		if err != nil {
			return nil, err
		}
		preferences = append(preferences, *preference)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notification preferences: %w", err)
	}
	return preferences, nil
}

func timestampValue(ts pgtype.Timestamp) time.Time {
	if !ts.Valid {
		return time.Time{}
	}
	return ts.Time
}
