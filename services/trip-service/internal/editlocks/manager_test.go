package editlocks

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestManagerAcquireRenewConflictAndRelease(t *testing.T) {
	manager := NewManager().(*manager)
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	manager.nowUTC = func() time.Time { return now }

	tripID := uuid.New()
	userID := uuid.New()
	otherID := uuid.New()

	acquired, err := manager.AcquireOrRenew(context.Background(), AcquireLockInput{
		TripID: tripID,
		Scope:  ScopeItinerary,
		UserID: userID,
		Role:   "owner",
		TTL:    time.Minute,
	})
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	if !acquired.Acquired || acquired.Renewed || acquired.BlockedByOther || acquired.Lock == nil {
		t.Fatalf("expected first acquire, got %+v", acquired)
	}
	if !acquired.Lock.LockedByCurrentUser || acquired.Lock.TTLSeconds != 60 {
		t.Fatalf("unexpected first acquire lock view: %+v", acquired.Lock)
	}

	now = now.Add(10 * time.Second)
	renewed, err := manager.AcquireOrRenew(context.Background(), AcquireLockInput{
		TripID: tripID,
		Scope:  ScopeItinerary,
		UserID: userID,
		Role:   "owner",
		TTL:    time.Minute,
	})
	if err != nil {
		t.Fatalf("renew: %v", err)
	}
	if !renewed.Acquired || !renewed.Renewed || renewed.Lock == nil {
		t.Fatalf("expected renewal, got %+v", renewed)
	}
	if got := renewed.Lock.ExpiresAt.Sub(now); got != time.Minute {
		t.Fatalf("expected renewed expiry one minute from now, got %s", got)
	}

	blocked, err := manager.AcquireOrRenew(context.Background(), AcquireLockInput{
		TripID: tripID,
		Scope:  ScopeItinerary,
		UserID: otherID,
		Role:   "editor",
		TTL:    time.Minute,
	})
	if err != nil {
		t.Fatalf("conflicting acquire: %v", err)
	}
	if blocked.Acquired || !blocked.BlockedByOther || blocked.Lock == nil || blocked.Lock.LockedByCurrentUser {
		t.Fatalf("expected blocked by other user, got %+v", blocked)
	}

	released, err := manager.Release(context.Background(), tripID, ScopeItinerary, otherID)
	if err != nil {
		t.Fatalf("release by non-owner: %v", err)
	}
	if released {
		t.Fatal("expected non-owner release to fail")
	}

	released, err = manager.Release(context.Background(), tripID, ScopeItinerary, userID)
	if err != nil {
		t.Fatalf("release by owner: %v", err)
	}
	if !released {
		t.Fatal("expected owner release to remove lock")
	}
	lock, err := manager.Get(context.Background(), tripID, ScopeItinerary, userID)
	if err != nil {
		t.Fatalf("get after release: %v", err)
	}
	if lock != nil {
		t.Fatalf("expected no lock after release, got %+v", lock)
	}
}

func TestManagerExpiredLockCanBeReplacedAndCleanedUp(t *testing.T) {
	manager := NewManager().(*manager)
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	manager.nowUTC = func() time.Time { return now }

	tripID := uuid.New()
	firstUserID := uuid.New()
	secondUserID := uuid.New()

	if _, err := manager.AcquireOrRenew(context.Background(), AcquireLockInput{
		TripID: tripID,
		Scope:  ScopeItinerary,
		UserID: firstUserID,
		TTL:    time.Second,
	}); err != nil {
		t.Fatalf("first acquire: %v", err)
	}

	now = now.Add(2 * time.Second)
	replaced, err := manager.AcquireOrRenew(context.Background(), AcquireLockInput{
		TripID: tripID,
		Scope:  ScopeItinerary,
		UserID: secondUserID,
		TTL:    time.Minute,
	})
	if err != nil {
		t.Fatalf("replace expired acquire: %v", err)
	}
	if !replaced.Acquired || replaced.BlockedByOther || replaced.Lock == nil ||
		replaced.Lock.LockedByUserID != secondUserID {
		t.Fatalf("expected expired lock replacement, got %+v", replaced)
	}

	otherTripID := uuid.New()
	if _, err := manager.AcquireOrRenew(context.Background(), AcquireLockInput{
		TripID: otherTripID,
		Scope:  ScopeItinerary,
		UserID: firstUserID,
		TTL:    time.Second,
	}); err != nil {
		t.Fatalf("other acquire: %v", err)
	}
	now = now.Add(2 * time.Second)
	manager.CleanupExpired(now)
	lock, err := manager.Get(context.Background(), otherTripID, ScopeItinerary, firstUserID)
	if err != nil {
		t.Fatalf("get after cleanup: %v", err)
	}
	if lock != nil {
		t.Fatalf("expected cleanup to remove expired lock, got %+v", lock)
	}
}

func TestManagerInvalidScope(t *testing.T) {
	manager := NewManager()
	_, err := manager.AcquireOrRenew(context.Background(), AcquireLockInput{
		TripID: uuid.New(),
		Scope:  "day",
		UserID: uuid.New(),
		TTL:    time.Minute,
	})
	if err != ErrInvalidScope {
		t.Fatalf("expected invalid scope error, got %v", err)
	}
}

func TestManagerConcurrentAcquireOnlyOneWins(t *testing.T) {
	manager := NewManager()
	tripID := uuid.New()
	const users = 24

	var wg sync.WaitGroup
	results := make(chan AcquireLockResult, users)
	errs := make(chan error, users)
	for i := 0; i < users; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := manager.AcquireOrRenew(context.Background(), AcquireLockInput{
				TripID: tripID,
				Scope:  ScopeItinerary,
				UserID: uuid.New(),
				Role:   "editor",
				TTL:    time.Minute,
			})
			if err != nil {
				errs <- err
				return
			}
			results <- result
		}()
	}
	wg.Wait()
	close(results)
	close(errs)

	for err := range errs {
		t.Fatalf("unexpected acquire error: %v", err)
	}
	acquired := 0
	blocked := 0
	for result := range results {
		if result.Acquired {
			acquired++
		}
		if result.BlockedByOther {
			blocked++
		}
	}
	if acquired != 1 || blocked != users-1 {
		t.Fatalf("expected one acquire and %d blocked, got acquired=%d blocked=%d", users-1, acquired, blocked)
	}
}
