package transport

import (
	"errors"
	"fmt"
	"strings"
)

const (
	ErrorValidationFailed      = "validation_failed"
	ErrorRateLimited           = "provider_rate_limited"
	ErrorQuotaExceeded         = "provider_quota_exceeded"
	ErrorProviderUnavailable   = "transport_provider_unavailable"
	ErrorMalformedResponse     = "transport_provider_malformed_response"
	providerErrorUnavailable   = "unavailable"
	providerErrorMalformed     = "malformed_response"
	providerErrorConfiguration = "configuration"
)

type ProviderError struct {
	Provider string
	Kind     string
	Err      error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "transport provider error"
	}
	provider := strings.TrimSpace(e.Provider)
	if provider == "" {
		return "transport provider error: " + e.Kind
	}
	return fmt.Sprintf("transport provider %s: %s", provider, e.Kind)
}

func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func providerErrorKind(err error) string {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) && providerErr.Kind != "" {
		return providerErr.Kind
	}
	return "unknown"
}
