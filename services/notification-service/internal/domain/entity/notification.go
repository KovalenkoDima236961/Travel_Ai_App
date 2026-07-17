// Package entity holds the Notification Service domain models. They are plain Go
// types with no persistence or transport concerns.
package entity

import (
	"time"

	"github.com/google/uuid"
)

// Notification is a single in-app notification owned by exactly one recipient
// user. It is created by Trip Service (via the internal batch endpoint) after a
// collaboration/comment/itinerary action and read back by the recipient.
//
// Metadata is a small, free-form JSON object with rendering hints (day number,
// item name, role, etc.). It must never contain secrets: no JWTs, refresh
// tokens, passwords, share passwords, public-share access tokens, or API keys.
type Notification struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	TripID         *uuid.UUID
	ActorUserID    *uuid.UUID
	Type           string
	Title          string
	Message        string
	EntityType     *string
	EntityID       *uuid.UUID
	Metadata       map[string]any
	Priority       string
	Category       string
	DigestKey      *string
	DedupeKey      *string
	GroupedCount   int
	DigestBatchID  *uuid.UUID
	DeliveryMode   *string
	DeliveryStatus *string
	ExpiresAt      *time.Time
	LatestEventAt  time.Time
	ReadAt         *time.Time
	CreatedAt      time.Time
}

// IsRead reports whether the notification has been marked read.
func (n *Notification) IsRead() bool {
	return n.ReadAt != nil
}
