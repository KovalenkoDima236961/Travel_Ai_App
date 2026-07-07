package places

import (
	"errors"
	"fmt"
)

const (
	providerErrorAuthConfig  = "auth_config"
	providerErrorRateLimit   = "rate_limited"
	providerErrorUnavailable = "unavailable"
	providerErrorResponse    = "bad_response"
	providerErrorRequest     = "request_failed"
)

// ProviderError classifies upstream provider failures without exposing raw
// provider payloads to HTTP clients.
type ProviderError struct {
	Provider   string
	Kind       string
	StatusCode int
	Err        error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "place provider error"
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s place provider %s error (status %d)", e.Provider, e.Kind, e.StatusCode)
	}
	return fmt.Sprintf("%s place provider %s error", e.Provider, e.Kind)
}

func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func providerErrorKind(err error) string {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Kind
	}
	return "unknown"
}
