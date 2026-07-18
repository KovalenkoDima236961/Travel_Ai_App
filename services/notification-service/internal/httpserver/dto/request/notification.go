// Package request holds the JSON shapes accepted by the Notification Service
// HTTP API and their mapping into application inputs.
package request

import (
	"strings"
	"time"

	"github.com/google/uuid"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/preferences"
)

// CreateNotificationsBatch is the body of POST /internal/notifications/batch.
// It is sent by trusted internal callers (Trip Service) — never by browsers.
type CreateNotificationsBatch struct {
	Notifications []CreateNotification `json:"notifications"`
}

// CreateNotification is one item in a batch create request.
type CreateNotification struct {
	UserID      string         `json:"userId"`
	TripID      *string        `json:"tripId"`
	ActorUserID *string        `json:"actorUserId"`
	Type        string         `json:"type"`
	Title       string         `json:"title"`
	Message     string         `json:"message"`
	EntityType  *string        `json:"entityType"`
	EntityID    *string        `json:"entityId"`
	Metadata    map[string]any `json:"metadata"`
	Priority    string         `json:"priority"`
	Category    string         `json:"category"`
	DigestKey   *string        `json:"digestKey"`
	DedupeKey   *string        `json:"dedupeKey"`
}

// UpdateNotificationPreferences is the body of PUT /notifications/preferences.
type UpdateNotificationPreferences struct {
	Items    []NotificationPreferenceItem `json:"items"`
	Settings *NotificationSettings        `json:"settings,omitempty"`
}

type NotificationSettings struct {
	QuietHoursEnabled        bool   `json:"quietHoursEnabled"`
	QuietHoursStart          string `json:"quietHoursStart"`
	QuietHoursEnd            string `json:"quietHoursEnd"`
	QuietHoursTimezone       string `json:"quietHoursTimezone"`
	UrgentBypassesQuietHours bool   `json:"urgentBypassesQuietHours"`
	DailyDigestTime          string `json:"dailyDigestTime"`
	WeeklyDigestDay          int    `json:"weeklyDigestDay"`
	WeeklyDigestTime         string `json:"weeklyDigestTime"`
}

type UpsertTripMute struct {
	TripID     string     `json:"tripId"`
	Category   *string    `json:"category"`
	MutedUntil *time.Time `json:"mutedUntil"`
}

type ProcessDigests struct {
	Now   *time.Time `json:"now,omitempty"`
	Limit int        `json:"limit"`
}

// SubscribePush is the body of POST /notifications/push/subscribe.
type SubscribePush struct {
	Subscription PushSubscription `json:"subscription"`
	UserAgent    *string          `json:"userAgent"`
	Browser      *string          `json:"browser"`
	DeviceLabel  *string          `json:"deviceLabel"`
}

// PushSubscription mirrors the browser PushSubscription JSON shape.
type PushSubscription struct {
	Endpoint string               `json:"endpoint"`
	Keys     PushSubscriptionKeys `json:"keys"`
}

// PushSubscriptionKeys holds the browser-generated key material.
type PushSubscriptionKeys struct {
	P256DH string `json:"p256dh"`
	Auth   string `json:"auth"`
}

// UnsubscribePush is the body of DELETE /notifications/push/unsubscribe.
type UnsubscribePush struct {
	Endpoint string `json:"endpoint"`
}

// CleanupNotifications is explicit because the current storage implementation
// permanently removes selected notification rows. Unread items are preserved by
// default in the handler when onlyRead is omitted.
type CleanupNotifications struct {
	OlderThanDays int      `json:"olderThanDays"`
	OnlyRead      *bool    `json:"onlyRead,omitempty"`
	Categories    []string `json:"categories,omitempty"`
}

// NotificationPreferenceItem is one requested channel/category setting. Enabled
// is a pointer so omission is distinguishable from false and can be rejected.
type NotificationPreferenceItem struct {
	Channel      string  `json:"channel"`
	Category     string  `json:"category"`
	Enabled      *bool   `json:"enabled"`
	DeliveryMode *string `json:"deliveryMode"`
}

// ToInputs validates id formats and maps the batch into application
// CreateInputs. Business validation (type/title/message) happens in the service
// so it stays unit-testable independently of transport.
func (b CreateNotificationsBatch) ToInputs() ([]notifications.CreateInput, error) {
	inputs := make([]notifications.CreateInput, 0, len(b.Notifications))
	for i := range b.Notifications {
		input, err := b.Notifications[i].toInput()
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input)
	}
	return inputs, nil
}

func (n CreateNotification) toInput() (notifications.CreateInput, error) {
	userID, err := uuid.Parse(strings.TrimSpace(n.UserID))
	if err != nil {
		return notifications.CreateInput{}, apperrs.NewInvalidInput("userId must be a valid uuid")
	}

	tripID, err := optionalUUID(n.TripID, "tripId")
	if err != nil {
		return notifications.CreateInput{}, err
	}
	actorUserID, err := optionalUUID(n.ActorUserID, "actorUserId")
	if err != nil {
		return notifications.CreateInput{}, err
	}
	entityID, err := optionalUUID(n.EntityID, "entityId")
	if err != nil {
		return notifications.CreateInput{}, err
	}

	return notifications.CreateInput{
		UserID:      userID,
		TripID:      tripID,
		ActorUserID: actorUserID,
		Type:        strings.TrimSpace(n.Type),
		Title:       n.Title,
		Message:     n.Message,
		EntityType:  normalizeEntityType(n.EntityType),
		EntityID:    entityID,
		Metadata:    n.Metadata,
		Priority:    strings.ToLower(strings.TrimSpace(n.Priority)),
		Category:    strings.ToLower(strings.TrimSpace(n.Category)),
		DigestKey:   NormalizeOptionalString(n.DigestKey),
		DedupeKey:   NormalizeOptionalString(n.DedupeKey),
	}, nil
}

func optionalUUID(raw *string, field string) (*uuid.UUID, error) {
	if raw == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil, nil
	}
	id, err := uuid.Parse(trimmed)
	if err != nil {
		return nil, apperrs.NewInvalidInput("%s must be a valid uuid", field)
	}
	return &id, nil
}

func normalizeEntityType(raw *string) *string {
	if raw == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

// ToInputs validates transport-only preference requirements and maps the
// request into application inputs. Vocabulary and duplicate checks stay in the
// preferences service so they are unit-testable outside HTTP.
func (u UpdateNotificationPreferences) ToInputs() ([]preferences.PreferenceInput, error) {
	if u.Items == nil {
		return nil, apperrs.NewInvalidInput("items array is required")
	}

	inputs := make([]preferences.PreferenceInput, 0, len(u.Items))
	for i := range u.Items {
		item := u.Items[i]
		if item.Enabled == nil && item.DeliveryMode == nil {
			return nil, apperrs.NewInvalidInput("deliveryMode or enabled is required")
		}
		mode := ""
		if item.DeliveryMode != nil {
			mode = strings.ToLower(strings.TrimSpace(*item.DeliveryMode))
		}
		enabled := mode != preferences.ModeMuted
		if item.Enabled != nil {
			enabled = *item.Enabled
		}
		inputs = append(inputs, preferences.PreferenceInput{
			Channel:      strings.TrimSpace(item.Channel),
			Category:     strings.TrimSpace(item.Category),
			Enabled:      enabled,
			DeliveryMode: mode,
		})
	}
	return inputs, nil
}

// NormalizeOptionalString trims optional metadata and returns nil for empty
// values so persistence stores NULL rather than blank strings.
func NormalizeOptionalString(raw *string) *string {
	if raw == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
