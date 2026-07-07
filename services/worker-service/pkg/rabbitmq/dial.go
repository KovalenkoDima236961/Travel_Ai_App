package rabbitmq

import (
	"context"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// DialFunc is the AMQP dial function used by DialWithRetry.
type DialFunc func(url string) (*amqp.Connection, error)

// DialRetryConfig controls RabbitMQ connection retry behavior.
type DialRetryConfig struct {
	Attempts     int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Dial         DialFunc
}

// DialWithRetry connects to RabbitMQ with bounded exponential backoff.
func DialWithRetry(ctx context.Context, url string, cfg DialRetryConfig) (*amqp.Connection, error) {
	if cfg.Attempts < 1 {
		cfg.Attempts = 1
	}
	if cfg.InitialDelay <= 0 {
		cfg.InitialDelay = 500 * time.Millisecond
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 5 * time.Second
	}
	if cfg.Dial == nil {
		cfg.Dial = amqp.Dial
	}

	delay := cfg.InitialDelay
	var lastErr error
	for attempt := 0; attempt < cfg.Attempts; attempt++ {
		conn, err := cfg.Dial(url)
		if err == nil {
			return conn, nil
		}
		lastErr = err
		if attempt == cfg.Attempts-1 {
			break
		}

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("connect rabbitmq: %w", ctx.Err())
		case <-timer.C:
		}
		if delay < cfg.MaxDelay {
			delay *= 2
			if delay > cfg.MaxDelay {
				delay = cfg.MaxDelay
			}
		}
	}
	return nil, fmt.Errorf("connect rabbitmq: %w", lastErr)
}
