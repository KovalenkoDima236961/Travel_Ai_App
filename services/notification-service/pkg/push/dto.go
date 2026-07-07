package push

import (
	"time"

	"github.com/google/uuid"
)

// PushSubscription is the sender-facing browser Push API subscription shape.
type PushSubscription struct {
	ID       uuid.UUID
	UserID   uuid.UUID
	Endpoint string
	P256DH   string
	Auth     string
}

// PushPayload is the small JSON payload delivered to a browser service worker.
type PushPayload struct {
	Title          string `json:"title"`
	Body           string `json:"body"`
	URL            string `json:"url"`
	NotificationID string `json:"notificationId,omitempty"`
	Type           string `json:"type"`
	Category       string `json:"category"`
	Icon           string `json:"icon,omitempty"`
	Badge          string `json:"badge,omitempty"`
}

// PushSendResult reports one attempted send to one push subscription.
type PushSendResult struct {
	StatusCode       int
	SubscriptionGone bool
}

// Config configures browser push delivery.
type Config struct {
	Enabled         bool
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	Subject         string
	Timeout         time.Duration
	TTLSeconds      int
	Urgency         string
}
