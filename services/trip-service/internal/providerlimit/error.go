// Package providerlimit defines the typed provider rate-limit/quota errors that
// External Integrations Service returns, and helpers for Trip Service clients to
// detect them. Keeping the codes in one place lets the generation-job worker
// classify limit failures consistently (transient rate limits are retryable;
// exhausted daily quotas are terminal until the next day).
package providerlimit

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Controlled error codes returned by External Integrations Service.
const (
	CodeRateLimited       = "provider_rate_limited"
	CodeQuotaExceeded     = "provider_quota_exceeded"
	CodeLimitsUnavailable = "provider_limits_unavailable"
)

// Error is a typed provider-limit error surfaced by a client. It carries only
// safe, bounded fields — never provider credentials.
type Error struct {
	Code              string
	Provider          string
	Operation         string
	RetryAfterSeconds int
	Message           string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Code
}

// IsLimitCode reports whether a code is a known provider-limit code.
func IsLimitCode(code string) bool {
	switch code {
	case CodeRateLimited, CodeQuotaExceeded, CodeLimitsUnavailable:
		return true
	default:
		return false
	}
}

// Parse inspects an External Integrations Service error response body and
// returns a typed Error when it carries a known provider-limit code. It returns
// nil otherwise, so callers can fall back to their existing error handling.
func Parse(statusCode int, body []byte) *Error {
	var payload struct {
		Error             string `json:"error"`
		Message           string `json:"message"`
		Provider          string `json:"provider"`
		Operation         string `json:"operation"`
		RetryAfterSeconds int    `json:"retryAfterSeconds"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	if !IsLimitCode(payload.Error) {
		return nil
	}
	retryAfter := payload.RetryAfterSeconds
	if retryAfter <= 0 && statusCode == http.StatusTooManyRequests {
		retryAfter = 60
	}
	return &Error{
		Code:              payload.Error,
		Provider:          payload.Provider,
		Operation:         payload.Operation,
		RetryAfterSeconds: retryAfter,
		Message:           payload.Message,
	}
}

// As returns the typed provider-limit Error in err's chain, if any.
func As(err error) (*Error, bool) {
	var e *Error
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}
