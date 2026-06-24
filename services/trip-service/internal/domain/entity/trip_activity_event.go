package entity

import (
	"time"

	"github.com/google/uuid"
)

// TripActivityEvent is a single persisted entry in a trip's activity feed /
// audit log. Events are recorded only after a user action has succeeded and are
// visible to the trip owner and accepted collaborators (never to public share
// viewers). Metadata is a small, sanitized JSON object describing the event; it
// never contains secrets, passwords, tokens, comment bodies, or full itinerary
// payloads.
type TripActivityEvent struct {
	ID          uuid.UUID
	TripID      uuid.UUID
	ActorUserID *uuid.UUID
	EventType   string
	EntityType  *string
	EntityID    *uuid.UUID
	Metadata    map[string]any
	CreatedAt   time.Time
}
