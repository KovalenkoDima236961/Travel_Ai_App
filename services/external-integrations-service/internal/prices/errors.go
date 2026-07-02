package prices

import "strings"

const providerErrorUnavailable = "unavailable"

var ErrUnsupportedCurrency = &ProviderError{Kind: "unsupported_currency"}

type ProviderError struct {
	Provider string
	Kind     string
}

func (e *ProviderError) Error() string {
	if e == nil {
		return "price provider error"
	}
	if strings.TrimSpace(e.Provider) == "" {
		return "price provider error: " + e.Kind
	}
	return "price provider " + e.Provider + ": " + e.Kind
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
