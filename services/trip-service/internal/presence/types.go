package presence

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	PresenceStateViewing = "viewing"
	PresenceStateEditing = "editing"

	EventPresenceSnapshot  = "presence.snapshot"
	EventPresenceHeartbeat = "presence.heartbeat"

	DefaultHeartbeatInterval            = 25 * time.Second
	DefaultStaleAfter                   = 60 * time.Second
	DefaultMaxConnectionsPerUserPerTrip = 5
	DefaultClientBufferSize             = 20
	defaultCleanupMinimumInterval       = time.Second
	defaultCleanupIntervalDivisor       = 2
)

var (
	ErrInvalidState           = errors.New("invalid presence state")
	ErrMaxConnectionsExceeded = errors.New("maximum trip presence connections exceeded")
)

// PresenceSession represents one active tab/device viewing a private trip.
type PresenceSession struct {
	SessionID   string
	TripID      uuid.UUID
	UserID      uuid.UUID
	Role        string
	DisplayName string
	State       string
	ConnectedAt time.Time
	LastSeenAt  time.Time
}

// PresenceEvent is one Server-Sent Event queued for a connected client.
type PresenceEvent struct {
	Name string
	Data any
}

// Manager owns instance-local in-memory trip presence.
type Manager interface {
	Register(ctx context.Context, session PresenceSession) (<-chan PresenceEvent, error)
	Unregister(tripID uuid.UUID, sessionID string)
	UpdateState(tripID uuid.UUID, userID uuid.UUID, state string) error
	Snapshot(tripID uuid.UUID) PresenceSnapshot
	PublishSnapshot(tripID uuid.UUID)
	CleanupStale(now time.Time)
}

// IsValidState reports whether state is one of the public v1 states.
func IsValidState(state string) bool {
	return state == PresenceStateViewing || state == PresenceStateEditing
}
