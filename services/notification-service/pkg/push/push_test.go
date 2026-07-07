package push

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestNewSenderSelectsMockWhenDisabledOrUnconfigured(t *testing.T) {
	cases := map[string]Config{
		"disabled":    {Enabled: false},
		"missing key": {Enabled: true, VAPIDPublicKey: "public"},
	}

	for name, cfg := range cases {
		t.Run(name, func(t *testing.T) {
			sender, err := NewSender(cfg, nil)
			if err != nil {
				t.Fatalf("NewSender returned error: %v", err)
			}
			if _, ok := sender.(*MockSender); !ok {
				t.Fatalf("expected *MockSender, got %T", sender)
			}
		})
	}
}

func TestNewSenderSelectsWebPushWhenConfigured(t *testing.T) {
	sender, err := NewSender(Config{
		Enabled:         true,
		VAPIDPublicKey:  "public",
		VAPIDPrivateKey: "private",
		Subject:         "mailto:test@example.com",
	}, nil)
	if err != nil {
		t.Fatalf("NewSender returned error: %v", err)
	}
	if _, ok := sender.(*WebPushSender); !ok {
		t.Fatalf("expected *WebPushSender, got %T", sender)
	}
}

func TestNewWebPushSenderRequiresKeysAndSubject(t *testing.T) {
	if _, err := NewWebPushSender(Config{Subject: "mailto:test@example.com"}, nil); err == nil {
		t.Fatal("expected missing VAPID keys to error")
	}
	if _, err := NewWebPushSender(Config{VAPIDPublicKey: "public", VAPIDPrivateKey: "private"}, nil); err == nil {
		t.Fatal("expected missing subject to error")
	}
}

func TestMockSenderReportsAccepted(t *testing.T) {
	sender := NewMockSender(nil)
	result, err := sender.Send(context.Background(), PushSubscription{
		ID:       uuid.New(),
		UserID:   uuid.New(),
		Endpoint: "https://push.example.test/subscription/1",
		P256DH:   "p256dh",
		Auth:     "auth",
	}, PushPayload{
		Title:    "Trip update",
		Body:     "Open the app for details.",
		URL:      "/notifications",
		Type:     "comment_created",
		Category: "comments",
	})
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if result == nil || result.StatusCode != 202 || result.SubscriptionGone {
		t.Fatalf("unexpected result %+v", result)
	}
}

func TestEndpointHashIsStableAndShort(t *testing.T) {
	endpoint := "https://push.example.test/subscription/1"
	first := EndpointHash(endpoint)
	second := EndpointHash(endpoint)
	if first != second {
		t.Fatalf("hash is not stable: %q != %q", first, second)
	}
	if len(first) != 16 {
		t.Fatalf("expected 16 hex chars, got %q", first)
	}
}
