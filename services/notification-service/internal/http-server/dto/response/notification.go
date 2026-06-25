package response

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/preferences"
)

// Notification is the user-facing JSON representation of a notification.
type Notification struct {
	ID          string         `json:"id"`
	UserID      string         `json:"userId"`
	TripID      *string        `json:"tripId"`
	ActorUserID *string        `json:"actorUserId"`
	Type        string         `json:"type"`
	Title       string         `json:"title"`
	Message     string         `json:"message"`
	EntityType  *string        `json:"entityType"`
	EntityID    *string        `json:"entityId"`
	Metadata    map[string]any `json:"metadata"`
	ReadAt      *string        `json:"readAt"`
	CreatedAt   string         `json:"createdAt"`
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
	Channel  string `json:"channel"`
	Category string `json:"category"`
	Enabled  bool   `json:"enabled"`
}

// NotificationPreferences is the full effective preference matrix.
type NotificationPreferences struct {
	Items []NotificationPreference `json:"items"`
}

// NewNotification maps a domain notification to its API representation.
func NewNotification(n entity.Notification) Notification {
	metadata := n.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	return Notification{
		ID:          n.ID.String(),
		UserID:      n.UserID.String(),
		TripID:      uuidPtrString(n.TripID),
		ActorUserID: uuidPtrString(n.ActorUserID),
		Type:        n.Type,
		Title:       n.Title,
		Message:     n.Message,
		EntityType:  n.EntityType,
		EntityID:    uuidPtrString(n.EntityID),
		Metadata:    metadata,
		ReadAt:      timePtrString(n.ReadAt),
		CreatedAt:   n.CreatedAt.UTC().Format(time.RFC3339Nano),
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
			Channel:  item.Channel,
			Category: item.Category,
			Enabled:  item.Enabled,
		})
	}
	return NotificationPreferences{Items: items}
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
