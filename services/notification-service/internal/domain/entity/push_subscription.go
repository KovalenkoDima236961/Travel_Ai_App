package entity

import (
	"time"

	"github.com/google/uuid"
)

const (
	PushSubscriptionStatusActive   = "active"
	PushSubscriptionStatusDisabled = "disabled"
)

// PushSubscription is a browser Push API subscription registered by one user.
// Endpoint, P-256 DH, and auth values are not passwords, but should still be
// treated as sensitive-ish transport credentials and never logged in full.
type PushSubscription struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Endpoint      string
	P256DH        string
	Auth          string
	UserAgent     *string
	Browser       *string
	DeviceLabel   *string
	Status        string
	CreatedAt     time.Time
	UpdatedAt     time.Time
	LastUsedAt    *time.Time
	DisabledAt    *time.Time
	DisableReason *string
}
