package editlocks

import (
	"math"
	"time"

	"github.com/google/uuid"
)

type EditLockView struct {
	Locked              bool      `json:"locked"`
	Scope               string    `json:"scope"`
	TripID              uuid.UUID `json:"tripId"`
	LockedByUserID      uuid.UUID `json:"lockedByUserId,omitempty"`
	LockedByDisplayName *string   `json:"lockedByDisplayName,omitempty"`
	LockedByRole        string    `json:"lockedByRole,omitempty"`
	LockedByCurrentUser bool      `json:"lockedByCurrentUser,omitempty"`
	CreatedAt           time.Time `json:"createdAt,omitempty"`
	ExpiresAt           time.Time `json:"expiresAt,omitempty"`
	TTLSeconds          int       `json:"ttlSeconds,omitempty"`
	Disabled            bool      `json:"disabled,omitempty"`
}

type AcquireEditLockResponse struct {
	Error    string        `json:"error,omitempty"`
	Message  string        `json:"message,omitempty"`
	Acquired bool          `json:"acquired"`
	Renewed  bool          `json:"renewed,omitempty"`
	Disabled bool          `json:"disabled,omitempty"`
	Reason   string        `json:"reason,omitempty"`
	Lock     *EditLockView `json:"lock,omitempty"`
}

type ReleaseEditLockResponse struct {
	Released bool `json:"released"`
}

func NewUnlockedView(tripID uuid.UUID, scope string) EditLockView {
	return EditLockView{
		Locked: false,
		Scope:  scope,
		TripID: tripID,
	}
}

func ViewFromLock(lock EditLock, currentUserID uuid.UUID) EditLockView {
	var displayName *string
	if lock.DisplayName != "" {
		name := lock.DisplayName
		displayName = &name
	}

	ttlStart := lock.LastRenewedAt
	if ttlStart.IsZero() {
		ttlStart = lock.CreatedAt
	}
	ttlSeconds := int(math.Ceil(lock.ExpiresAt.Sub(ttlStart).Seconds()))
	if ttlSeconds < 0 {
		ttlSeconds = 0
	}

	return EditLockView{
		Locked:              true,
		Scope:               lock.Scope,
		TripID:              lock.TripID,
		LockedByUserID:      lock.LockedByUserID,
		LockedByDisplayName: displayName,
		LockedByRole:        lock.LockedByRole,
		LockedByCurrentUser: lock.LockedByUserID == currentUserID,
		CreatedAt:           lock.CreatedAt,
		ExpiresAt:           lock.ExpiresAt,
		TTLSeconds:          ttlSeconds,
	}
}
