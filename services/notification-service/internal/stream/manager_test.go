package stream

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestManagerRegisterIncreasesCount(t *testing.T) {
	userID := uuid.New()
	manager := NewManager(Config{MaxConnectionsPerUser: 2}, nil)
	client := NewClient(userID)

	if err := manager.Register(userID, client); err != nil {
		t.Fatalf("register: %v", err)
	}
	if got := manager.CountForUser(userID); got != 1 {
		t.Fatalf("expected count 1, got %d", got)
	}
}

func TestManagerUnregisterRemovesClient(t *testing.T) {
	userID := uuid.New()
	manager := NewManager(Config{MaxConnectionsPerUser: 2}, nil)
	client := NewClient(userID)
	if err := manager.Register(userID, client); err != nil {
		t.Fatalf("register: %v", err)
	}

	manager.Unregister(userID, client.ID)

	if got := manager.CountForUser(userID); got != 0 {
		t.Fatalf("expected count 0, got %d", got)
	}
	if _, ok := <-client.Send; ok {
		t.Fatal("expected client channel to be closed")
	}
}

func TestManagerPublishSendsEventToRegisteredClient(t *testing.T) {
	userID := uuid.New()
	manager := NewManager(Config{MaxConnectionsPerUser: 2}, nil)
	client := NewClient(userID)
	if err := manager.Register(userID, client); err != nil {
		t.Fatalf("register: %v", err)
	}

	manager.PublishToUser(context.Background(), userID, StreamEvent{Name: EventNotificationCreated, Data: "ok"})

	select {
	case got := <-client.Send:
		if got.Name != EventNotificationCreated || got.Data != "ok" {
			t.Fatalf("unexpected event: %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for event")
	}
}

func TestManagerPublishToNoClientsDoesNotError(t *testing.T) {
	manager := NewManager(Config{MaxConnectionsPerUser: 2}, nil)
	manager.PublishToUser(context.Background(), uuid.New(), StreamEvent{Name: EventNotificationCreated})
}

func TestManagerMultipleClientsForSameUserReceiveEvent(t *testing.T) {
	userID := uuid.New()
	manager := NewManager(Config{MaxConnectionsPerUser: 2}, nil)
	clients := []*Client{NewClient(userID), NewClient(userID)}
	for _, client := range clients {
		if err := manager.Register(userID, client); err != nil {
			t.Fatalf("register: %v", err)
		}
	}

	manager.PublishToUser(context.Background(), userID, StreamEvent{Name: EventNotificationCreated})

	for _, client := range clients {
		select {
		case got := <-client.Send:
			if got.Name != EventNotificationCreated {
				t.Fatalf("unexpected event for client %s: %+v", client.ID, got)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for event for client %s", client.ID)
		}
	}
}

func TestManagerDifferentUsersDoNotReceiveEachOthersEvents(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	manager := NewManager(Config{MaxConnectionsPerUser: 2}, nil)
	client := NewClient(userID)
	otherClient := NewClient(otherUserID)
	if err := manager.Register(userID, client); err != nil {
		t.Fatalf("register user: %v", err)
	}
	if err := manager.Register(otherUserID, otherClient); err != nil {
		t.Fatalf("register other: %v", err)
	}

	manager.PublishToUser(context.Background(), userID, StreamEvent{Name: EventNotificationCreated})

	select {
	case <-client.Send:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for user event")
	}
	select {
	case got := <-otherClient.Send:
		t.Fatalf("other user received event: %+v", got)
	default:
	}
}

func TestManagerFullClientChannelDropsWithoutBlocking(t *testing.T) {
	userID := uuid.New()
	manager := NewManager(Config{MaxConnectionsPerUser: 1}, nil)
	client := NewClientWithBuffer(userID, 1)
	if err := manager.Register(userID, client); err != nil {
		t.Fatalf("register: %v", err)
	}
	client.Send <- StreamEvent{Name: "existing"}

	done := make(chan struct{})
	go func() {
		manager.PublishToUser(context.Background(), userID, StreamEvent{Name: EventNotificationCreated})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publish blocked on full client channel")
	}

	got := <-client.Send
	if got.Name != "existing" {
		t.Fatalf("expected original queued event to remain, got %+v", got)
	}
	select {
	case got := <-client.Send:
		t.Fatalf("expected dropped event, got %+v", got)
	default:
	}
}

func TestManagerMaxConnectionsPerUserEnforced(t *testing.T) {
	userID := uuid.New()
	manager := NewManager(Config{MaxConnectionsPerUser: 1}, nil)
	if err := manager.Register(userID, NewClient(userID)); err != nil {
		t.Fatalf("register first: %v", err)
	}

	err := manager.Register(userID, NewClient(userID))
	if !errors.Is(err, ErrMaxConnectionsExceeded) {
		t.Fatalf("expected ErrMaxConnectionsExceeded, got %v", err)
	}
}
