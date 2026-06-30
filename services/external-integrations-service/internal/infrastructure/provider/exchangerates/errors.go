package exchangerates

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

var ErrUnsupportedCurrency = errors.New("unsupported_currency")

// ProviderError classifies upstream exchange-rate provider failures without
// exposing raw provider payloads or API keys to clients.
type ProviderError struct {
	Provider   string
	Kind       string
	StatusCode int
	Message    string
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "exchange rate provider error"
	}
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s exchange rate provider %s error (status %d)", e.Provider, e.Kind, e.StatusCode)
	}
	return fmt.Sprintf("%s exchange rate provider %s error", e.Provider, e.Kind)
}

func providerErrorKind(err error) string {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Kind
	}
	if errors.Is(err, ErrUnsupportedCurrency) {
		return "unsupported_currency"
	}
	return "unknown"
}
