package activitystream

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
)

type manager struct {
	mu     sync.RWMutex
	trips  map[uuid.UUID]map[string]*client
	cfg    Config
	log    *zap.Logger
	nowUTC func() time.Time
}

type client struct {
	RegisterClientInput
	send chan ActivityStreamEvent

	mu     sync.Mutex
	closed bool
}

func NewManager(cfg Config, log *zap.Logger) Manager {
	if log == nil {
		log = zap.NewNop()
	}
	return &manager{
		trips:  make(map[uuid.UUID]map[string]*client),
		cfg:    Normalize(cfg),
		log:    log,
		nowUTC: func() time.Time { return time.Now().UTC() },
	}
}

func (m *manager) Register(_ context.Context, input RegisterClientInput) (<-chan ActivityStreamEvent, error) {
	if input.TripID == uuid.Nil {
		return nil, fmt.Errorf("trip id is required")
	}
	if input.UserID == uuid.Nil {
		return nil, fmt.Errorf("user id is required")
	}

	now := m.nowUTC()
	if input.ConnectionID == "" {
		input.ConnectionID = uuid.NewString()
	}
	if input.ConnectedAt.IsZero() {
		input.ConnectedAt = now
	}
	if input.LastSeenAt.IsZero() {
		input.LastSeenAt = now
	}
	input.Role = strings.TrimSpace(input.Role)

	c := &client{
		RegisterClientInput: input,
		send:                make(chan ActivityStreamEvent, m.cfg.ClientBufferSize),
	}

	m.mu.Lock()
	clients := m.trips[input.TripID]
	if clients == nil {
		clients = make(map[string]*client)
		m.trips[input.TripID] = clients
	}
	activeForUser := 0
	for _, existing := range clients {
		if existing.UserID == input.UserID {
			activeForUser++
		}
	}
	if activeForUser >= m.cfg.MaxConnectionsPerUserPerTrip {
		m.mu.Unlock()
		return nil, ErrMaxConnectionsExceeded
	}
	clients[input.ConnectionID] = c
	m.mu.Unlock()

	return c.send, nil
}

func (m *manager) Unregister(tripID uuid.UUID, connectionID string) {
	var removed *client

	m.mu.Lock()
	if clients := m.trips[tripID]; clients != nil {
		removed = clients[connectionID]
		if removed != nil {
			delete(clients, connectionID)
		}
		if len(clients) == 0 {
			delete(m.trips, tripID)
		}
	}
	m.mu.Unlock()

	if removed != nil {
		removed.close()
	}
}

func (m *manager) Publish(_ context.Context, tripID uuid.UUID, event activity.EventDTO) {
	m.mu.RLock()
	clients := make([]*client, 0, len(m.trips[tripID]))
	for _, c := range m.trips[tripID] {
		clients = append(clients, c)
	}
	m.mu.RUnlock()

	streamEvent := ActivityStreamEvent{
		Name: EventActivityCreated,
		Data: ActivityCreatedPayload{Event: event},
	}
	for _, c := range clients {
		if ok := c.trySend(streamEvent); !ok {
			m.log.Warn("trip activity stream client queue full; dropping event",
				zap.String("trip_id", tripID.String()),
				zap.String("user_id", c.UserID.String()),
				zap.String("connection_id", c.ConnectionID),
				zap.String("event", event.EventType),
			)
		}
	}
}

func (m *manager) ClientCount(tripID uuid.UUID) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.trips[tripID])
}

func (c *client) trySend(event ActivityStreamEvent) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return false
	}
	select {
	case c.send <- event:
		return true
	default:
		return false
	}
}

func (c *client) close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}
	close(c.send)
	c.closed = true
}
