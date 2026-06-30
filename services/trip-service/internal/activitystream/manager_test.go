package activitystream

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
)

func TestManagerRegisterAndUnregister(t *testing.T) {
	manager := NewManager(Config{MaxConnectionsPerUserPerTrip: 2}, nil)
	tripID := uuid.New()
	userID := uuid.New()
	connectionID := uuid.NewString()

	events, err := manager.Register(context.Background(), RegisterClientInput{
		ConnectionID: connectionID,
		TripID:       tripID,
		UserID:       userID,
		Role:         "owner",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if got := manager.ClientCount(tripID); got != 1 {
		t.Fatalf("expected one client, got %d", got)
	}

	manager.Unregister(tripID, connectionID)

	if got := manager.ClientCount(tripID); got != 0 {
		t.Fatalf("expected no clients, got %d", got)
	}
	assertActivityChannelCloses(t, events)
}

func TestManagerPublishSendsEventToRegisteredClient(t *testing.T) {
	manager := NewManager(Config{MaxConnectionsPerUserPerTrip: 2}, nil)
	tripID := uuid.New()
	events, err := manager.Register(context.Background(), RegisterClientInput{
		TripID: tripID,
		UserID: uuid.New(),
		Role:   "viewer",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	event := activity.EventDTO{ID: uuid.New(), TripID: tripID, EventType: activity.EventCommentCreated}
	manager.Publish(context.Background(), tripID, event)

	got := readActivityStreamEvent(t, events)
	if got.Name != EventActivityCreated {
		t.Fatalf("expected activity.created, got %+v", got)
	}
	payload, ok := got.Data.(ActivityCreatedPayload)
	if !ok {
		t.Fatalf("expected ActivityCreatedPayload, got %T", got.Data)
	}
	if payload.Event.ID != event.ID || payload.Event.EventType != event.EventType {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestManagerPublishIsTripScoped(t *testing.T) {
	manager := NewManager(Config{MaxConnectionsPerUserPerTrip: 2}, nil)
	tripID := uuid.New()
	otherTripID := uuid.New()
	events, err := manager.Register(context.Background(), RegisterClientInput{
		TripID: tripID,
		UserID: uuid.New(),
		Role:   "owner",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	manager.Publish(context.Background(), otherTripID, activity.EventDTO{
		ID:        uuid.New(),
		TripID:    otherTripID,
		EventType: activity.EventCommentCreated,
	})

	select {
	case event := <-events:
		t.Fatalf("unexpected event for another trip: %+v", event)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestManagerMultipleClientsReceiveEvent(t *testing.T) {
	manager := NewManager(Config{MaxConnectionsPerUserPerTrip: 2}, nil)
	tripID := uuid.New()
	first, err := manager.Register(context.Background(), RegisterClientInput{
		TripID: tripID,
		UserID: uuid.New(),
		Role:   "owner",
	})
	if err != nil {
		t.Fatalf("register first: %v", err)
	}
	second, err := manager.Register(context.Background(), RegisterClientInput{
		TripID: tripID,
		UserID: uuid.New(),
		Role:   "editor",
	})
	if err != nil {
		t.Fatalf("register second: %v", err)
	}

	manager.Publish(context.Background(), tripID, activity.EventDTO{
		ID:        uuid.New(),
		TripID:    tripID,
		EventType: activity.EventItineraryUpdated,
	})

	if readActivityStreamEvent(t, first).Name != EventActivityCreated {
		t.Fatal("first client did not receive activity event")
	}
	if readActivityStreamEvent(t, second).Name != EventActivityCreated {
		t.Fatal("second client did not receive activity event")
	}
}

func TestManagerMaxConnectionsPerUserTripEnforced(t *testing.T) {
	manager := NewManager(Config{MaxConnectionsPerUserPerTrip: 1}, nil)
	tripID := uuid.New()
	userID := uuid.New()
	if _, err := manager.Register(context.Background(), RegisterClientInput{
		TripID: tripID,
		UserID: userID,
		Role:   "owner",
	}); err != nil {
		t.Fatalf("register first: %v", err)
	}

	_, err := manager.Register(context.Background(), RegisterClientInput{
		TripID: tripID,
		UserID: userID,
		Role:   "owner",
	})
	if !errors.Is(err, ErrMaxConnectionsExceeded) {
		t.Fatalf("expected ErrMaxConnectionsExceeded, got %v", err)
	}
}

func TestManagerPublishToFullChannelDoesNotBlock(t *testing.T) {
	manager := NewManager(Config{ClientBufferSize: 1, MaxConnectionsPerUserPerTrip: 1}, nil)
	tripID := uuid.New()
	if _, err := manager.Register(context.Background(), RegisterClientInput{
		TripID: tripID,
		UserID: uuid.New(),
		Role:   "owner",
	}); err != nil {
		t.Fatalf("register: %v", err)
	}

	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			manager.Publish(context.Background(), tripID, activity.EventDTO{
				ID:        uuid.New(),
				TripID:    tripID,
				EventType: activity.EventCommentCreated,
			})
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publish blocked on full channel")
	}
}

func TestManagerConcurrentPublishRegister(t *testing.T) {
	manager := NewManager(Config{ClientBufferSize: 4, MaxConnectionsPerUserPerTrip: 10}, nil)
	tripID := uuid.New()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := manager.Register(context.Background(), RegisterClientInput{
				TripID: tripID,
				UserID: uuid.New(),
				Role:   "viewer",
			}); err != nil {
				t.Errorf("register: %v", err)
			}
		}()
	}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			manager.Publish(context.Background(), tripID, activity.EventDTO{
				ID:        uuid.New(),
				TripID:    tripID,
				EventType: activity.EventCommentCreated,
			})
		}()
	}
	wg.Wait()
}

func readActivityStreamEvent(t *testing.T, events <-chan ActivityStreamEvent) ActivityStreamEvent {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for activity stream event")
		return ActivityStreamEvent{}
	}
}

func assertActivityChannelCloses(t *testing.T, events <-chan ActivityStreamEvent) {
	t.Helper()
	select {
	case _, ok := <-events:
		if ok {
			t.Fatal("expected channel to close")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for activity stream channel to close")
	}
}
