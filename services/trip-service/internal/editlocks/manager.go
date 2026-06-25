package editlocks

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type manager struct {
	mu     sync.RWMutex
	locks  map[string]EditLock
	nowUTC func() time.Time
}

func NewManager() Manager {
	return &manager{
		locks:  make(map[string]EditLock),
		nowUTC: func() time.Time { return time.Now().UTC() },
	}
}

func (m *manager) AcquireOrRenew(_ context.Context, input AcquireLockInput) (AcquireLockResult, error) {
	scope, err := NormalizeScope(strings.TrimSpace(input.Scope))
	if err != nil {
		return AcquireLockResult{}, err
	}
	if input.TripID == uuid.Nil {
		return AcquireLockResult{}, fmt.Errorf("trip id is required")
	}
	if input.UserID == uuid.Nil {
		return AcquireLockResult{}, fmt.Errorf("user id is required")
	}
	ttl := input.TTL
	if ttl <= 0 {
		ttl = DefaultTTL
	}

	now := m.nowUTC()
	key := lockKey(input.TripID, scope)

	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.locks[key]; ok && existing.ExpiresAt.After(now) {
		if existing.LockedByUserID != input.UserID {
			view := ViewFromLock(existing, input.UserID)
			return AcquireLockResult{
				Acquired:       false,
				BlockedByOther: true,
				Lock:           &view,
			}, nil
		}

		existing.LockedByRole = strings.TrimSpace(input.Role)
		existing.DisplayName = strings.TrimSpace(input.DisplayName)
		existing.LastRenewedAt = now
		existing.ExpiresAt = now.Add(ttl)
		m.locks[key] = existing
		view := ViewFromLock(existing, input.UserID)
		return AcquireLockResult{
			Acquired: true,
			Renewed:  true,
			Lock:     &view,
		}, nil
	}

	lock := EditLock{
		TripID:         input.TripID,
		Scope:          scope,
		LockedByUserID: input.UserID,
		LockedByRole:   strings.TrimSpace(input.Role),
		DisplayName:    strings.TrimSpace(input.DisplayName),
		CreatedAt:      now,
		ExpiresAt:      now.Add(ttl),
		LastRenewedAt:  now,
	}
	m.locks[key] = lock
	view := ViewFromLock(lock, input.UserID)
	return AcquireLockResult{
		Acquired: true,
		Lock:     &view,
	}, nil
}

func (m *manager) Get(_ context.Context, tripID uuid.UUID, scope string, currentUserID uuid.UUID) (*EditLockView, error) {
	scope, err := NormalizeScope(strings.TrimSpace(scope))
	if err != nil {
		return nil, err
	}
	if tripID == uuid.Nil {
		return nil, fmt.Errorf("trip id is required")
	}

	now := m.nowUTC()
	key := lockKey(tripID, scope)

	m.mu.Lock()
	defer m.mu.Unlock()

	lock, ok := m.locks[key]
	if !ok {
		return nil, nil
	}
	if !lock.ExpiresAt.After(now) {
		delete(m.locks, key)
		return nil, nil
	}
	view := ViewFromLock(lock, currentUserID)
	return &view, nil
}

func (m *manager) Release(_ context.Context, tripID uuid.UUID, scope string, userID uuid.UUID) (bool, error) {
	scope, err := NormalizeScope(strings.TrimSpace(scope))
	if err != nil {
		return false, err
	}
	if tripID == uuid.Nil {
		return false, fmt.Errorf("trip id is required")
	}
	if userID == uuid.Nil {
		return false, fmt.Errorf("user id is required")
	}

	now := m.nowUTC()
	key := lockKey(tripID, scope)

	m.mu.Lock()
	defer m.mu.Unlock()

	lock, ok := m.locks[key]
	if !ok {
		return false, nil
	}
	if !lock.ExpiresAt.After(now) {
		delete(m.locks, key)
		return false, nil
	}
	if lock.LockedByUserID != userID {
		return false, nil
	}
	delete(m.locks, key)
	return true, nil
}

func (m *manager) CleanupExpired(now time.Time) {
	if now.IsZero() {
		now = m.nowUTC()
	}
	now = now.UTC()

	m.mu.Lock()
	defer m.mu.Unlock()

	for key, lock := range m.locks {
		if !lock.ExpiresAt.After(now) {
			delete(m.locks, key)
		}
	}
}

func lockKey(tripID uuid.UUID, scope string) string {
	return tripID.String() + ":" + scope
}
