package notifications

import (
	"fmt"
	"net/http"
	"time"
)

// Config controls the Notification Service client.
type Config struct {
	BaseURL        string
	Token          string
	TimeoutSeconds int
}

// New constructs a timeout-bound Notification Service client.
func New(cfg Config) (*Client, error) {
	if cfg.TimeoutSeconds <= 0 {
		return nil, fmt.Errorf("NOTIFICATION_SERVICE_TIMEOUT_SECONDS must be greater than 0")
	}

	httpClient := &http.Client{
		Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
	}
	return NewClient(cfg.BaseURL, cfg.Token, httpClient)
}
