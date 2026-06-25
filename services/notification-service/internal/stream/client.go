package stream

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const DefaultClientBufferSize = 20

// Client represents one active SSE connection for a user.
type Client struct {
	ID          string
	UserID      uuid.UUID
	Send        chan StreamEvent
	ConnectedAt time.Time

	mu     sync.Mutex
	closed bool
}

// NewClient constructs a client with the default buffered event queue.
func NewClient(userID uuid.UUID) *Client {
	return NewClientWithBuffer(userID, DefaultClientBufferSize)
}

// NewClientWithBuffer constructs a client with a custom queue size. It is used
// by tests to exercise full-channel drop behavior.
func NewClientWithBuffer(userID uuid.UUID, bufferSize int) *Client {
	if bufferSize < 0 {
		bufferSize = 0
	}
	return &Client{
		ID:          uuid.NewString(),
		UserID:      userID,
		Send:        make(chan StreamEvent, bufferSize),
		ConnectedAt: time.Now().UTC(),
	}
}

func (c *Client) trySend(event StreamEvent) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return false
	}
	select {
	case c.Send <- event:
		return true
	default:
		return false
	}
}

// Close closes the client's send channel exactly once.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}
	close(c.Send)
	c.closed = true
}
