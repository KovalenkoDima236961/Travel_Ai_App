package weathercontext

import (
	"fmt"
	"net/http"
	"time"
)

// Config controls the External Integrations Service weather client.
type Config struct {
	BaseURL        string
	TimeoutSeconds int
}

// New constructs a timeout-bound weather forecast client.
func New(cfg Config) (*Client, error) {
	if cfg.TimeoutSeconds <= 0 {
		return nil, fmt.Errorf("WEATHER_CONTEXT_TIMEOUT_SECONDS must be greater than 0")
	}

	httpClient := &http.Client{
		Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
	}
	return NewClient(cfg.BaseURL, httpClient)
}
