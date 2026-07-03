package availability

import "strings"

const (
	ErrorValidationFailed    = "availability_validation_failed"
	ErrorRateLimited         = "availability_provider_rate_limited"
	ErrorQuotaExceeded       = "availability_provider_quota_exceeded"
	ErrorProviderUnavailable = "availability_provider_unavailable"
	ErrorNoOptionsFound      = "availability_no_options_found"
	ErrorMalformedResponse   = "availability_provider_malformed_response"
	ErrorUnsupportedCurrency = "unsupported_currency"
)

const (
	providerErrorUnavailable = "unavailable"
	providerErrorMalformed   = "malformed_response"
)

var ErrUnsupportedCurrency = &ProviderError{Kind: ErrorUnsupportedCurrency}

type ProviderError struct {
	Provider string
	Kind     string
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "availability provider error"
	}
	if strings.TrimSpace(e.Provider) == "" {
		return "availability provider error: " + e.Kind
	}
	return "availability provider " + e.Provider + ": " + e.Kind
}

func providerErrorKind(err error) string {
	if err == nil {
		return ""
	}
	if providerErr, ok := err.(*ProviderError); ok && providerErr.Kind != "" {
		return providerErr.Kind
	}
	return "unknown"
}
