// Package request holds the JSON shapes accepted by the Notification Service
// HTTP API and their mapping into application inputs.
package request

import (
	"strings"

	"github.com/google/uuid"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
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
