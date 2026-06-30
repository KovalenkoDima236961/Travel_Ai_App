package activitystream

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
)

const (
	EventActivityCreated   = "activity.created"
	EventActivityHeartbeat = "activity.heartbeat"

	DefaultHeartbeatInterval            = 25 * time.Second
	DefaultWriteTimeout                 = 10 * time.Second
	DefaultMaxConnectionsPerUserPerTrip = 5
	DefaultClientBufferSize             = 20
)

type RegisterClientInput struct {
	ConnectionID string
	TripID       uuid.UUID
	UserID       uuid.UUID
	Role         string
	ConnectedAt  time.Time
	LastSeenAt   time.Time
}

type ActivityStreamEvent struct {
	Name string
	Data any
}

type Manager interface {
	Register(ctx context.Context, input RegisterClientInput) (<-chan ActivityStreamEvent, error)
	Unregister(tripID uuid.UUID, connectionID string)
	Publish(ctx context.Context, tripID uuid.UUID, event activity.EventDTO)
	ClientCount(tripID uuid.UUID) int
}
