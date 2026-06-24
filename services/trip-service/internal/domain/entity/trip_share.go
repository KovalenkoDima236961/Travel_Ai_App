package entity

import (
	"time"

	"github.com/google/uuid"
)

// TripShare is the owner-managed public read-only link for a trip.
// v1 keeps exactly one share row per trip.
type TripShare struct {
	ID               uuid.UUID
	TripID           uuid.UUID
	UserID           uuid.UUID
	ShareToken       string
	Enabled          bool
	ExpiresAt        *time.Time
	PasswordHash     *string
	PasswordRequired bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DisabledAt       *time.Time
}
