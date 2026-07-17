package response

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/preferences"
)

// Notification is the user-facing JSON representation of a notification.
type Notification struct {
	ID             string         `json:"id"`
	UserID         string         `json:"userId"`
	TripID         *string        `json:"tripId"`
	ActorUserID    *string        `json:"actorUserId"`
	Type           string         `json:"type"`
	Title          string         `json:"title"`
	Message        string         `json:"message"`
	EntityType     *string        `json:"entityType"`
	EntityID       *string        `json:"entityId"`
	Metadata       map[string]any `json:"metadata"`
	ReadAt         *string        `json:"readAt"`
	CreatedAt      string         `json:"createdAt"`
	Priority       string         `json:"priority"`
	Category       string         `json:"category"`
	DigestKey      *string        `json:"digestKey"`
	DedupeKey      *string        `json:"dedupeKey"`
	GroupedCount   int            `json:"groupedCount"`
	DigestBatchID  *string        `json:"digestBatchId"`
	DeliveryMode   *string        `json:"deliveryMode"`
	DeliveryStatus *string        `json:"deliveryStatus"`
	ExpiresAt      *string        `json:"expiresAt"`
	LatestEventAt  string         `json:"latestEventAt"`
}

// NotificationList is one newest-first page of notifications plus an opaque
// cursor for the next page. NextCursor is null when no more rows exist.
type NotificationList struct {
	Items      []Notification `json:"items"`
	NextCursor *string        `json:"nextCursor"`
}

// UnreadCount is the response shape for the unread-count endpoint.
type UnreadCount struct {
	Count int `json:"count"`
}

// Success is the minimal acknowledgement returned by the mark-read endpoints.
type Success struct {
	Success bool `json:"success"`
}

// NotificationPreference is one user-facing preference setting.
type NotificationPreference struct {
	Channel      string `json:"channel"`
	Category     string `json:"category"`
	Enabled      bool   `json:"enabled"`
	DeliveryMode string `json:"deliveryMode"`
}

// NotificationPreferences is the full effective preference matrix.
type NotificationPreferences struct {
	Items    []NotificationPreference `json:"items"`
	Settings NotificationSettings     `json:"settings"`
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

type TripMute struct {
	ID         string  `json:"id"`
	TripID     string  `json:"tripId"`
	Category   *string `json:"category"`
	MutedUntil *string `json:"mutedUntil"`
	CreatedAt  string  `json:"createdAt"`
	UpdatedAt  string  `json:"updatedAt"`
}

type TripMutes struct {
	Items []TripMute `json:"items"`
}

// PushPublicKey exposes the VAPID public key when browser push is enabled.
type PushPublicKey struct {
	Enabled   bool    `json:"enabled"`
	PublicKey *string `json:"publicKey"`
}

// PushSubscribe acknowledges a subscribe attempt.
type PushSubscribe struct {
	Subscribed bool `json:"subscribed"`
	Enabled    bool `json:"enabled"`
}

// PushUnsubscribe acknowledges a device unsubscribe attempt.
type PushUnsubscribe struct {
	Unsubscribed bool `json:"unsubscribed"`
}

// PushStatus reports browser push state for the current user.
type PushStatus struct {
	Enabled             bool `json:"enabled"`
	ActiveSubscriptions int  `json:"activeSubscriptions"`
}

// NewNotification maps a domain notification to its API representation.
func NewNotification(n entity.Notification) Notification {
	metadata := n.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	latestEventAt := n.LatestEventAt
	if latestEventAt.IsZero() {
		latestEventAt = n.CreatedAt
	}
	groupedCount := n.GroupedCount
	if groupedCount < 1 {
		groupedCount = 1
	}
	return Notification{
		ID:             n.ID.String(),
		UserID:         n.UserID.String(),
		TripID:         uuidPtrString(n.TripID),
		ActorUserID:    uuidPtrString(n.ActorUserID),
		Type:           n.Type,
		Title:          n.Title,
		Message:        n.Message,
		EntityType:     n.EntityType,
		EntityID:       uuidPtrString(n.EntityID),
		Metadata:       metadata,
		ReadAt:         timePtrString(n.ReadAt),
		CreatedAt:      n.CreatedAt.UTC().Format(time.RFC3339Nano),
		Priority:       n.Priority,
		Category:       n.Category,
		DigestKey:      n.DigestKey,
		DedupeKey:      n.DedupeKey,
		GroupedCount:   groupedCount,
		DigestBatchID:  uuidPtrString(n.DigestBatchID),
		DeliveryMode:   n.DeliveryMode,
		DeliveryStatus: n.DeliveryStatus,
		ExpiresAt:      timePtrString(n.ExpiresAt),
		LatestEventAt:  latestEventAt.UTC().Format(time.RFC3339Nano),
	}
}

// NewNotificationList maps a page of domain notifications to the list response.
func NewNotificationList(notifications []entity.Notification, nextCursor string) NotificationList {
	items := make([]Notification, 0, len(notifications))
	for i := range notifications {
		items = append(items, NewNotification(notifications[i]))
	}
	var cursor *string
	if nextCursor != "" {
		cursor = &nextCursor
	}
	return NotificationList{Items: items, NextCursor: cursor}
}

// NewNotificationPreferences maps the preferences use-case result to JSON.
func NewNotificationPreferences(result *preferences.PreferencesResult) NotificationPreferences {
	if result == nil {
		return NotificationPreferences{Items: []NotificationPreference{}}
	}
	items := make([]NotificationPreference, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, NotificationPreference{
			Channel:      item.Channel,
			Category:     item.Category,
			Enabled:      item.Enabled,
			DeliveryMode: item.DeliveryMode,
		})
	}
	return NotificationPreferences{Items: items, Settings: NotificationSettings{
		QuietHoursEnabled: result.Settings.QuietHoursEnabled,
		QuietHoursStart:   result.Settings.QuietHoursStart, QuietHoursEnd: result.Settings.QuietHoursEnd,
		QuietHoursTimezone:       result.Settings.QuietHoursTimezone,
		UrgentBypassesQuietHours: result.Settings.UrgentBypassesQuietHours,
		DailyDigestTime:          result.Settings.DailyDigestTime, WeeklyDigestDay: result.Settings.WeeklyDigestDay,
		WeeklyDigestTime: result.Settings.WeeklyDigestTime,
	}}
}

func NewTripMutes(items []entity.NotificationTripMute) TripMutes {
	out := make([]TripMute, 0, len(items))
	for _, item := range items {
		out = append(out, TripMute{ID: item.ID.String(), TripID: item.TripID.String(), Category: item.Category,
			MutedUntil: timePtrString(item.MutedUntil), CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339Nano),
			UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339Nano)})
	}
	return TripMutes{Items: out}
}

func timePtrString(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339Nano)
	return &s
}

// uuidPtrString renders a nil id as JSON null rather than the zero uuid.
func uuidPtrString(p *uuid.UUID) *string {
	if p == nil {
		return nil
	}
	s := p.String()
	return &s
}
