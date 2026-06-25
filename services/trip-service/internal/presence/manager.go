package presence

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type manager struct {
	mu     sync.RWMutex
	trips  map[uuid.UUID]map[string]*client
	cfg    Config
	log    *zap.Logger
	nowUTC func() time.Time
}

type client struct {
	PresenceSession
	send chan PresenceEvent

	mu     sync.Mutex
	closed bool
}

// NewManager constructs an in-memory presence manager.
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

func (m *manager) Register(_ context.Context, session PresenceSession) (<-chan PresenceEvent, error) {
	if session.TripID == uuid.Nil {
		return nil, fmt.Errorf("trip id is required")
	}
	if session.UserID == uuid.Nil {
		return nil, fmt.Errorf("user id is required")
	}
	if !IsValidState(session.State) {
		return nil, ErrInvalidState
	}

	now := m.nowUTC()
	if session.SessionID == "" {
		session.SessionID = uuid.NewString()
	}
	if session.ConnectedAt.IsZero() {
		session.ConnectedAt = now
	}
	if session.LastSeenAt.IsZero() {
		session.LastSeenAt = now
	}
	session.Role = strings.TrimSpace(session.Role)
	session.DisplayName = strings.TrimSpace(session.DisplayName)

	c := &client{
		PresenceSession: session,
		send:            make(chan PresenceEvent, DefaultClientBufferSize),
	}

	m.mu.Lock()
	sessions := m.trips[session.TripID]
	if sessions == nil {
		sessions = make(map[string]*client)
		m.trips[session.TripID] = sessions
	}
	activeForUser := 0
	for _, existing := range sessions {
		if existing.UserID == session.UserID {
			activeForUser++
		}
	}
	if activeForUser >= m.cfg.MaxConnectionsPerUserPerTrip {
		m.mu.Unlock()
		return nil, ErrMaxConnectionsExceeded
	}

	sessions[session.SessionID] = c
	snapshot, clients := snapshotAndClientsLocked(session.TripID, sessions)
	m.mu.Unlock()

	event := PresenceEvent{Name: EventPresenceSnapshot, Data: snapshot}
	if !c.trySend(event) {
		m.log.Warn("trip presence queue full; dropping initial snapshot",
			zap.String("trip_id", session.TripID.String()),
			zap.String("user_id", session.UserID.String()),
			zap.String("session_id", session.SessionID),
		)
	}
	m.publishToClients(session.TripID, clients, event)
	return c.send, nil
}

func (m *manager) Unregister(tripID uuid.UUID, sessionID string) {
	var removed *client
	var shouldPublish bool

	m.mu.Lock()
	if sessions := m.trips[tripID]; sessions != nil {
		removed = sessions[sessionID]
		if removed != nil {
			delete(sessions, sessionID)
			shouldPublish = true
		}
		if len(sessions) == 0 {
			delete(m.trips, tripID)
		}
	}
	m.mu.Unlock()

	if removed != nil {
		removed.close()
	}
	if shouldPublish {
		m.PublishSnapshot(tripID)
	}
}

func (m *manager) UpdateState(tripID uuid.UUID, userID uuid.UUID, state string) error {
	if !IsValidState(state) {
		return ErrInvalidState
	}
	if tripID == uuid.Nil || userID == uuid.Nil {
		return nil
	}

	now := m.nowUTC()
	updated := false
	m.mu.Lock()
	if sessions := m.trips[tripID]; sessions != nil {
		for _, session := range sessions {
			if session.UserID == userID {
				session.State = state
				session.LastSeenAt = now
				updated = true
			}
		}
	}
	m.mu.Unlock()

	if updated {
		m.PublishSnapshot(tripID)
	}
	return nil
}

func (m *manager) Snapshot(tripID uuid.UUID) PresenceSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return snapshotLocked(tripID, m.trips[tripID])
}

func (m *manager) PublishSnapshot(tripID uuid.UUID) {
	m.mu.RLock()
	snapshot, clients := snapshotAndClientsLocked(tripID, m.trips[tripID])
	m.mu.RUnlock()

	m.publishToClients(tripID, clients, PresenceEvent{Name: EventPresenceSnapshot, Data: snapshot})
}

func (m *manager) CleanupStale(now time.Time) {
	if now.IsZero() {
		now = m.nowUTC()
	}
	cutoff := now.UTC().Add(-m.cfg.StaleAfter)
	affected := make([]uuid.UUID, 0)
	removed := make([]*client, 0)

	m.mu.Lock()
	for tripID, sessions := range m.trips {
		tripAffected := false
		for sessionID, session := range sessions {
			if session.LastSeenAt.Before(cutoff) {
				removed = append(removed, session)
				delete(sessions, sessionID)
				tripAffected = true
			}
		}
		if len(sessions) == 0 {
			delete(m.trips, tripID)
		}
		if tripAffected {
			affected = append(affected, tripID)
		}
	}
	m.mu.Unlock()

	for _, session := range removed {
		session.close()
	}
	for _, tripID := range affected {
		m.PublishSnapshot(tripID)
	}
}

func (m *manager) publishToClients(tripID uuid.UUID, clients []*client, event PresenceEvent) {
	for _, c := range clients {
		if ok := c.trySend(event); !ok {
			m.log.Warn("trip presence client queue full; dropping event",
				zap.String("trip_id", tripID.String()),
				zap.String("user_id", c.UserID.String()),
				zap.String("session_id", c.SessionID),
				zap.String("event", event.Name),
			)
		}
	}
}

func snapshotAndClientsLocked(tripID uuid.UUID, sessions map[string]*client) (PresenceSnapshot, []*client) {
	clients := make([]*client, 0, len(sessions))
	for _, session := range sessions {
		clients = append(clients, session)
	}
	return snapshotFromClients(tripID, clients), clients
}

func snapshotLocked(tripID uuid.UUID, sessions map[string]*client) PresenceSnapshot {
	clients := make([]*client, 0, len(sessions))
	for _, session := range sessions {
		clients = append(clients, session)
	}
	return snapshotFromClients(tripID, clients)
}

func snapshotFromClients(tripID uuid.UUID, clients []*client) PresenceSnapshot {
	type aggregate struct {
		userID      uuid.UUID
		displayName string
		role        string
		state       string
		connectedAt time.Time
		lastSeenAt  time.Time
	}

	byUser := make(map[uuid.UUID]*aggregate)
	for _, c := range clients {
		current := byUser[c.UserID]
		if current == nil {
			byUser[c.UserID] = &aggregate{
				userID:      c.UserID,
				displayName: c.DisplayName,
				role:        c.Role,
				state:       c.State,
				connectedAt: c.ConnectedAt,
				lastSeenAt:  c.LastSeenAt,
			}
			continue
		}
		if current.displayName == "" && c.DisplayName != "" {
			current.displayName = c.DisplayName
		}
		if current.role == "" && c.Role != "" {
			current.role = c.Role
		}
		if c.State == PresenceStateEditing {
			current.state = PresenceStateEditing
		}
		if c.ConnectedAt.Before(current.connectedAt) {
			current.connectedAt = c.ConnectedAt
		}
		if c.LastSeenAt.After(current.lastSeenAt) {
			current.lastSeenAt = c.LastSeenAt
		}
	}

	users := make([]PresenceUser, 0, len(byUser))
	for _, item := range byUser {
		displayName := displayNamePtr(item.displayName)
		users = append(users, PresenceUser{
			UserID:      item.userID.String(),
			DisplayName: displayName,
			Role:        item.role,
			State:       item.state,
			ConnectedAt: item.connectedAt.UTC().Format(time.RFC3339Nano),
			LastSeenAt:  item.lastSeenAt.UTC().Format(time.RFC3339Nano),
		})
	}
	slices.SortFunc(users, func(a, b PresenceUser) int {
		if a.ConnectedAt != b.ConnectedAt {
			if a.ConnectedAt < b.ConnectedAt {
				return -1
			}
			return 1
		}
		if a.UserID < b.UserID {
			return -1
		}
		if a.UserID > b.UserID {
			return 1
		}
		return 0
	})

	return PresenceSnapshot{
		TripID: tripID.String(),
		Users:  users,
	}
}

func displayNamePtr(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func (c *client) trySend(event PresenceEvent) bool {
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
