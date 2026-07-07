package rabbitmq

import (
	"context"
	"errors"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func TestDialWithRetryRetriesConfiguredAttempts(t *testing.T) {
	wantErr := errors.New("dial failed")
	attempts := 0

	_, err := DialWithRetry(context.Background(), "amqp://example", DialRetryConfig{
		Attempts:     3,
		InitialDelay: time.Nanosecond,
		Dial: func(string) (*amqp.Connection, error) {
			attempts++
			return nil, wantErr
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped dial error, got %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
}

func TestDialWithRetryStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	attempts := 0

	_, err := DialWithRetry(ctx, "amqp://example", DialRetryConfig{
		Attempts:     3,
		InitialDelay: time.Hour,
		Dial: func(string) (*amqp.Connection, error) {
			attempts++
			return nil, errors.New("dial failed")
		},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}
