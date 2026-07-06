package availability

import (
	"errors"
	"fmt"
	"strings"
)

const (
	ErrorValidationFailed    = "availability_validation_failed"
	ErrorRateLimited         = "availability_provider_rate_limited"
	ErrorQuotaExceeded       = "availability_provider_quota_exceeded"
	ErrorProviderUnavailable = "availability_provider_unavailable"
	ErrorNoOptionsFound      = "availability_no_options_found"
	ErrorMalformedResponse   = "availability_provider_malformed_response"
	ErrorUnsupportedCurrency = "unsupported_currency"
)

// providerError* are the internal error kinds a real availability provider can
// classify a failure as. They mirror the weather/place provider vocabulary so
// logging, metrics, and fallback behaviour stay consistent across the service.
const (
	providerErrorUnavailable = "unavailable"
	providerErrorMalformed   = "malformed_response"
	providerErrorAuthConfig  = "auth_config"
	providerErrorRateLimit   = "rate_limited"
	providerErrorTimeout     = "timeout"
	providerErrorRequest     = "request_failed"
	providerErrorBadResponse = "bad_response"
)

var ErrUnsupportedCurrency = &ProviderError{Kind: ErrorUnsupportedCurrency}

// ProviderError classifies an availability-provider failure without leaking raw
// upstream payloads to HTTP clients. StatusCode and Err are optional and are set
// by real HTTP providers; the zero value keeps the mock and normalization paths
// working unchanged.
type ProviderError struct {
	Provider   string
	Kind       string
	StatusCode int
	Err        error
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "availability provider error"
	}
	prefix := "availability provider error: " + e.Kind
	if strings.TrimSpace(e.Provider) != "" {
		prefix = "availability provider " + e.Provider + ": " + e.Kind
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s (status %d)", prefix, e.StatusCode)
	}
	return prefix
}

func (e *ProviderError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func providerErrorKind(err error) string {
	if err == nil {
		return ""
	}
	var providerErr *ProviderError
	if errors.As(err, &providerErr) && providerErr.Kind != "" {
		return providerErr.Kind
	}
	return "unknown"
}
