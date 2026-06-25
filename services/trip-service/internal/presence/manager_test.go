package presence

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestManagerRegisterAddsSessionAndPublishesSnapshot(t *testing.T) {
	manager := NewManager(Config{MaxConnectionsPerUserPerTrip: 2}, nil)
	tripID := uuid.New()
	userID := uuid.New()

	events, err := manager.Register(context.Background(), PresenceSession{
		TripID: tripID,
		UserID: userID,
		Role:   "owner",
		State:  PresenceStateViewing,
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	got := readPresenceEvent(t, events)
	if got.Name != EventPresenceSnapshot {
		t.Fatalf("expected snapshot event, got %+v", got)
	}
	snapshot := manager.Snapshot(tripID)
	if len(snapshot.Users) != 1 || snapshot.Users[0].UserID != userID.String() {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
}

func TestManagerUnregisterRemovesSession(t *testing.T) {
	manager := NewManager(Config{MaxConnectionsPerUserPerTrip: 2}, nil)
	tripID := uuid.New()
	sessionID := uuid.NewString()
	userID := uuid.New()
	events, err := manager.Register(context.Background(), PresenceSession{
		SessionID: sessionID,
		TripID:    tripID,
		UserID:    userID,
		Role:      "owner",
		State:     PresenceStateViewing,
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	_ = readPresenceEvent(t, events)

	manager.Unregister(tripID, sessionID)

	if got := manager.Snapshot(tripID); len(got.Users) != 0 {
		t.Fatalf("expected no users after unregister, got %+v", got)
	}
	assertChannelClosesAfterDrain(t, events)
}

func TestManagerSnapshotCollapsesSessionsAndEditingWins(t *testing.T) {
	manager := NewManager(Config{MaxConnectionsPerUserPerTrip: 5}, nil)
	tripID := uuid.New()
	userID := uuid.New()
	base := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)

	for _, session := range []PresenceSession{
		{
			SessionID:   "first",
			TripID:      tripID,
			UserID:      userID,
			Role:        "editor",
			State:       PresenceStateViewing,
			ConnectedAt: base.Add(2 * time.Minute),
			LastSeenAt:  base.Add(2 * time.Minute),
		},
		{
			SessionID:   "second",
			TripID:      tripID,
			UserID:      userID,
			Role:        "editor",
			State:       PresenceStateEditing,
			ConnectedAt: base,
			LastSeenAt:  base.Add(5 * time.Minute),
		},
	} {
		if _, err := manager.Register(context.Background(), session); err != nil {
			t.Fatalf("register %s: %v", session.SessionID, err)
		}
	}

	snapshot := manager.Snapshot(tripID)
	if len(snapshot.Users) != 1 {
		t.Fatalf("expected one collapsed user, got %+v", snapshot)
	}
	user := snapshot.Users[0]
	if user.State != PresenceStateEditing {
		t.Fatalf("expected editing state to win, got %+v", user)
	}
	if user.ConnectedAt != base.Format(time.RFC3339Nano) {
		t.Fatalf("expected earliest connectedAt, got %s", user.ConnectedAt)
	}
	if user.LastSeenAt != base.Add(5*time.Minute).Format(time.RFC3339Nano) {
		t.Fatalf("expected latest lastSeenAt, got %s", user.LastSeenAt)
	}
}

func TestManagerUpdateStateChangesAllUserSessions(t *testing.T) {
	manager := NewManager(Config{MaxConnectionsPerUserPerTrip: 5}, nil)
	tripID := uuid.New()
	userID := uuid.New()
	for _, sessionID := range []string{"first", "second"} {
		if _, err := manager.Register(context.Background(), PresenceSession{
			SessionID: sessionID,
			TripID:    tripID,
			UserID:    userID,
			Role:      "editor",
			State:     PresenceStateViewing,
		}); err != nil {
			t.Fatalf("register %s: %v", sessionID, err)
		}
	}

	if err := manager.UpdateState(tripID, userID, PresenceStateEditing); err != nil {
		t.Fatalf("update state: %v", err)
	}

	snapshot := manager.Snapshot(tripID)
	if len(snapshot.Users) != 1 || snapshot.Users[0].State != PresenceStateEditing {
		t.Fatalf("expected editing snapshot, got %+v", snapshot)
	}
}

func TestManagerUpdateInvalidStateReturnsError(t *testing.T) {
	manager := NewManager(Config{}, nil)
	err := manager.UpdateState(uuid.New(), uuid.New(), "away")
	if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}
}

func TestManagerMultipleUsersReceiveSnapshot(t *testing.T) {
	manager := NewManager(Config{MaxConnectionsPerUserPerTrip: 2}, nil)
	tripID := uuid.New()
	firstEvents, err := manager.Register(context.Background(), PresenceSession{
		TripID: tripID,
		UserID: uuid.New(),
		Role:   "owner",
		State:  PresenceStateViewing,
	})
	if err != nil {
		t.Fatalf("register first: %v", err)
	}
	_ = readPresenceEvent(t, firstEvents)

	secondEvents, err := manager.Register(context.Background(), PresenceSession{
		TripID: tripID,
		UserID: uuid.New(),
		Role:   "viewer",
		State:  PresenceStateViewing,
	})
	if err != nil {
		t.Fatalf("register second: %v", err)
	}

	gotFirst := readPresenceEvent(t, firstEvents)
	gotSecond := readPresenceEvent(t, secondEvents)
	if gotFirst.Name != EventPresenceSnapshot || gotSecond.Name != EventPresenceSnapshot {
		t.Fatalf("expected snapshot events, got first=%+v second=%+v", gotFirst, gotSecond)
	}
}

func TestManagerMaxConnectionsPerUserTripEnforced(t *testing.T) {
	manager := NewManager(Config{MaxConnectionsPerUserPerTrip: 1}, nil)
	tripID := uuid.New()
	userID := uuid.New()
	if _, err := manager.Register(context.Background(), PresenceSession{
		TripID: tripID,
		UserID: userID,
		Role:   "owner",
		State:  PresenceStateViewing,
	}); err != nil {
		t.Fatalf("register first: %v", err)
	}

	_, err := manager.Register(context.Background(), PresenceSession{
		TripID: tripID,
		UserID: userID,
		Role:   "owner",
		State:  PresenceStateViewing,
	})
	if !errors.Is(err, ErrMaxConnectionsExceeded) {
		t.Fatalf("expected ErrMaxConnectionsExceeded, got %v", err)
	}
}

func TestManagerCleanupStaleRemovesOldSessions(t *testing.T) {
	manager := NewManager(Config{StaleAfter: time.Minute, MaxConnectionsPerUserPerTrip: 2}, nil)
	tripID := uuid.New()
	sessionID := uuid.NewString()
	userID := uuid.New()
	now := time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC)
	events, err := manager.Register(context.Background(), PresenceSession{
		SessionID:   sessionID,
		TripID:      tripID,
		UserID:      userID,
		Role:        "owner",
		State:       PresenceStateViewing,
		ConnectedAt: now.Add(-2 * time.Minute),
		LastSeenAt:  now.Add(-2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	manager.CleanupStale(now)

	if got := manager.Snapshot(tripID); len(got.Users) != 0 {
		t.Fatalf("expected stale session removed, got %+v", got)
	}
	assertChannelClosesAfterDrain(t, events)
}

func TestManagerPublishToFullChannelDoesNotBlock(t *testing.T) {
	manager := NewManager(Config{MaxConnectionsPerUserPerTrip: 1}, nil)
	tripID := uuid.New()
	events, err := manager.Register(context.Background(), PresenceSession{
		TripID: tripID,
		UserID: uuid.New(),
		Role:   "owner",
		State:  PresenceStateViewing,
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	_ = events

	done := make(chan struct{})
	go func() {
		for i := 0; i < DefaultClientBufferSize*2; i++ {
			manager.PublishSnapshot(tripID)
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publish blocked on full channel")
	}
}

func readPresenceEvent(t *testing.T, events <-chan PresenceEvent) PresenceEvent {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for presence event")
		return PresenceEvent{}
	}
}

func assertChannelClosesAfterDrain(t *testing.T, events <-chan PresenceEvent) {
	t.Helper()
	for {
		select {
		case _, ok := <-events:
			if !ok {
				return
			}
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for presence channel to close")
		}
	}
}
