package stream

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrMaxConnectionsExceeded = errors.New("maximum notification stream connections exceeded")
	ErrNilClient              = errors.New("notification stream client is nil")
)

// Manager owns in-memory SSE clients for this service instance.
type Manager interface {
	Register(userID uuid.UUID, client *Client) error
	Unregister(userID uuid.UUID, clientID string)
	PublishToUser(ctx context.Context, userID uuid.UUID, event StreamEvent)
	CountForUser(userID uuid.UUID) int
}

type manager struct {
	mu                    sync.RWMutex
	clientsByUser         map[uuid.UUID]map[string]*Client
	maxConnectionsPerUser int
	log                   *zap.Logger
}

// NewManager constructs an in-memory stream manager.
func NewManager(cfg Config, log *zap.Logger) Manager {
	if log == nil {
		log = zap.NewNop()
	}
	maxConnections := cfg.MaxConnectionsPerUser
	if maxConnections <= 0 {
		maxConnections = DefaultMaxConnectionsPerUser
	}
	return &manager{
		clientsByUser:         make(map[uuid.UUID]map[string]*Client),
		maxConnectionsPerUser: maxConnections,
		log:                   log,
	}
}

func (m *manager) Register(userID uuid.UUID, client *Client) error {
	if client == nil {
		return ErrNilClient
	}
	if userID == uuid.Nil {
		return fmt.Errorf("user id is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	clients := m.clientsByUser[userID]
	if clients == nil {
		clients = make(map[string]*Client)
		m.clientsByUser[userID] = clients
	}
	if len(clients) >= m.maxConnectionsPerUser {
		return ErrMaxConnectionsExceeded
	}

	if client.ID == "" {
		client.ID = uuid.NewString()
	}
	if client.UserID == uuid.Nil {
		client.UserID = userID
	}
	if client.ConnectedAt.IsZero() {
		client.ConnectedAt = time.Now().UTC()
	}
	if client.Send == nil {
		client.Send = make(chan StreamEvent, DefaultClientBufferSize)
	}

	clients[client.ID] = client
	return nil
}

func (m *manager) Unregister(userID uuid.UUID, clientID string) {
	var client *Client

	m.mu.Lock()
	if clients := m.clientsByUser[userID]; clients != nil {
		client = clients[clientID]
		delete(clients, clientID)
		if len(clients) == 0 {
			delete(m.clientsByUser, userID)
		}
	}
	m.mu.Unlock()

	if client != nil {
		client.Close()
	}
}

func (m *manager) PublishToUser(ctx context.Context, userID uuid.UUID, event StreamEvent) {
	if ctx == nil {
		ctx = context.Background()
	}

	m.mu.RLock()
	clients := make([]*Client, 0, len(m.clientsByUser[userID]))
	for _, client := range m.clientsByUser[userID] {
		clients = append(clients, client)
	}
	m.mu.RUnlock()

	for _, client := range clients {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if ok := client.trySend(event); !ok {
			m.log.Warn("notification stream client queue full; dropping event",
				zap.String("user_id", userID.String()),
				zap.String("client_id", client.ID),
				zap.String("event", event.Name),
			)
		}
	}
}

func (m *manager) CountForUser(userID uuid.UUID) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clientsByUser[userID])
}
