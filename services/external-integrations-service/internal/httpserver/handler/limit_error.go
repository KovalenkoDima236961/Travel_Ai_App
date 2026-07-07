package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

// writeProviderLimitError writes a controlled provider-limit response and returns
// true when err is a provider limit error. It is safe for users and workers: it
// exposes only the bounded error code, a friendly message, the provider and
// operation, and a retry hint — never API keys or quota internals.
//
// provider_rate_limited and provider_quota_exceeded map to HTTP 429;
// provider_limits_unavailable maps to HTTP 503.
func writeProviderLimitError(w http.ResponseWriter, err error) bool {
	var limitErr *providerlimits.LimitError
	if !errors.As(err, &limitErr) {
		return false
	}
	status := http.StatusTooManyRequests
	if limitErr.Code == providerlimits.CodeLimitsUnavailable {
		status = http.StatusServiceUnavailable
	}
	if limitErr.RetryAfterSeconds > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(limitErr.RetryAfterSeconds))
	}
	writeJSON(w, status, map[string]any{
		"error":             limitErr.Code,
		"message":           limitErr.Message,
		"provider":          limitErr.Provider,
		"operation":         limitErr.Operation,
		"retryAfterSeconds": limitErr.RetryAfterSeconds,
	})
	return true
}
