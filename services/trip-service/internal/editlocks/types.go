package editlocks

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	ScopeItinerary = "itinerary"

	DefaultTTL             = 180 * time.Second
	DefaultRenewalInterval = 45 * time.Second
	DefaultCleanupInterval = 30 * time.Second
)

var ErrInvalidScope = errors.New("invalid edit lock scope")

// EditLock is the internal in-memory lock model.
type EditLock struct {
	TripID         uuid.UUID
	Scope          string
	LockedByUserID uuid.UUID
	LockedByRole   string
	DisplayName    string
	CreatedAt      time.Time
	ExpiresAt      time.Time
	LastRenewedAt  time.Time
}

type AcquireLockInput struct {
	TripID      uuid.UUID
	Scope       string
	UserID      uuid.UUID
	Role        string
	DisplayName string
	TTL         time.Duration
}

type AcquireLockResult struct {
	Acquired       bool
	Renewed        bool
	BlockedByOther bool
	Lock           *EditLockView
}

// Manager owns instance-local advisory edit locks.
type Manager interface {
	AcquireOrRenew(ctx context.Context, input AcquireLockInput) (AcquireLockResult, error)
	Get(ctx context.Context, tripID uuid.UUID, scope string, currentUserID uuid.UUID) (*EditLockView, error)
	Release(ctx context.Context, tripID uuid.UUID, scope string, userID uuid.UUID) (bool, error)
	CleanupExpired(now time.Time)
}

func NormalizeScope(scope string) (string, error) {
	if scope == "" {
		return ScopeItinerary, nil
	}
	if scope != ScopeItinerary {
		return "", ErrInvalidScope
	}
	return scope, nil
}
