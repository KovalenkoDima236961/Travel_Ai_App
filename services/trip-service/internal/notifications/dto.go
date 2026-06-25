// Package notifications is the Trip Service client for the Notification Service.
// Trip Service calls it synchronously after a successful collaboration / comment
// / itinerary action to fan out in-app notifications to affected users.
//
// This package owns only the outbound integration: building recipient payloads
// and POSTing them to the Notification Service internal batch endpoint. The
// fail-open / enabled orchestration lives in the trip use case (service layer),
// mirroring how activity recording is wired.
package notifications

import "github.com/google/uuid"

// NotificationCreateInput is one notification to create. The caller (trip use
// case) supplies fully-resolved recipient and actor ids; the actor is never the
// recipient (self-notifications are filtered before sending, and the
// Notification Service skips them defensively too).
type NotificationCreateInput struct {
	UserID      uuid.UUID
	TripID      *uuid.UUID
	ActorUserID *uuid.UUID
	Type        string
	Title       string
	Message     string
	EntityType  *string
	EntityID    *uuid.UUID
	Metadata    map[string]any
}

// --- wire shapes (JSON sent to / received from Notification Service) ---

type batchRequest struct {
	Notifications []notificationPayload `json:"notifications"`
}

type notificationPayload struct {
	UserID      string         `json:"userId"`
	TripID      *string        `json:"tripId,omitempty"`
	ActorUserID *string        `json:"actorUserId,omitempty"`
	Type        string         `json:"type"`
	Title       string         `json:"title"`
	Message     string         `json:"message"`
	EntityType  *string        `json:"entityType,omitempty"`
	EntityID    *string        `json:"entityId,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type batchResponse struct {
	Created int `json:"created"`
}

func toPayload(in NotificationCreateInput) notificationPayload {
	return notificationPayload{
		UserID:      in.UserID.String(),
		TripID:      uuidPtrString(in.TripID),
		ActorUserID: uuidPtrString(in.ActorUserID),
		Type:        in.Type,
		Title:       in.Title,
		Message:     in.Message,
		EntityType:  in.EntityType,
		EntityID:    uuidPtrString(in.EntityID),
		Metadata:    in.Metadata,
	}
}

func uuidPtrString(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}
