package usercontext

import (
	"fmt"
	"net/http"
	"time"
)

// Config controls the User Service client used by Trip Service during
// itinerary generation.
type Config struct {
	BaseURL        string
	TimeoutSeconds int
}

// New constructs a timeout-bound User Service client.
func New(cfg Config) (*Client, error) {
	if cfg.TimeoutSeconds <= 0 {
		return nil, fmt.Errorf("USER_CONTEXT_TIMEOUT_SECONDS must be greater than 0")
	}

	httpClient := &http.Client{
		Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
	}
	return NewClient(cfg.BaseURL, httpClient)
}
