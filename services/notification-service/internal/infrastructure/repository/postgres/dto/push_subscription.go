package dto

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/storage/postgres"
)

// PushSubscriptionColumns is the canonical projection for push subscription rows.
const PushSubscriptionColumns = "id, user_id, endpoint, p256dh, auth, user_agent, browser, " +
	"device_label, status, created_at, updated_at, last_used_at, disabled_at, disable_reason"

// PushSubscriptionInsertColumns returns the columns set on INSERT. Timestamps
// and status use the upsert statement/defaults.
func PushSubscriptionInsertColumns() []string {
	return []string{
		"id", "user_id", "endpoint", "p256dh", "auth",
		"user_agent", "browser", "device_label",
	}
}

// PushSubscriptionInsertValues returns the values for PushSubscriptionInsertColumns.
func PushSubscriptionInsertValues(subscription *entity.PushSubscription) []any {
	return []any{
		toPgUUID(subscription.ID),
		toPgUUID(subscription.UserID),
		subscription.Endpoint,
		subscription.P256DH,
		subscription.Auth,
		toPgTextPtr(subscription.UserAgent),
		toPgTextPtr(subscription.Browser),
		toPgTextPtr(subscription.DeviceLabel),
	}
}

// ScanPushSubscription reads one row in PushSubscriptionColumns order.
func ScanPushSubscription(row pgx.Row) (*entity.PushSubscription, error) {
	var (
		id, userID    pgtype.UUID
		endpoint      string
		p256dh        string
		auth          string
		userAgent     pgtype.Text
		browser       pgtype.Text
		deviceLabel   pgtype.Text
		status        string
		createdAt     pgtype.Timestamp
		updatedAt     pgtype.Timestamp
		lastUsedAt    pgtype.Timestamp
		disabledAt    pgtype.Timestamp
		disableReason pgtype.Text
	)

	err := row.Scan(
		&id,
		&userID,
		&endpoint,
		&p256dh,
		&auth,
		&userAgent,
		&browser,
		&deviceLabel,
		&status,
		&createdAt,
		&updatedAt,
		&lastUsedAt,
		&disabledAt,
		&disableReason,
	)
	if err != nil {
		if postgres.NoRowsFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan push subscription: %w", err)
	}

	return &entity.PushSubscription{
		ID:            uuid.UUID(id.Bytes),
		UserID:        uuid.UUID(userID.Bytes),
		Endpoint:      endpoint,
		P256DH:        p256dh,
		Auth:          auth,
		UserAgent:     fromPgText(userAgent),
		Browser:       fromPgText(browser),
		DeviceLabel:   fromPgText(deviceLabel),
		Status:        status,
		CreatedAt:     createdAt.Time,
		UpdatedAt:     updatedAt.Time,
		LastUsedAt:    timestampPtr(lastUsedAt),
		DisabledAt:    timestampPtr(disabledAt),
		DisableReason: fromPgText(disableReason),
	}, nil
}

// ScanPushSubscriptionRows maps push subscription rows to domain entities.
func ScanPushSubscriptionRows(rows pgx.Rows) ([]entity.PushSubscription, error) {
	subscriptions := make([]entity.PushSubscription, 0)
	for rows.Next() {
		subscription, err := ScanPushSubscription(rows)
		if err != nil {
			return nil, err
		}
		if subscription != nil {
			subscriptions = append(subscriptions, *subscription)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate push subscriptions: %w", err)
	}
	return subscriptions, nil
}
