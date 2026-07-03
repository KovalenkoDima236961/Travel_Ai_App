package providerlimits

import "fmt"

// Controlled error codes returned to callers when a provider limit is hit.
// These are safe to surface to workers and (with a friendly message) to users;
// they never leak provider account, credential, or quota internals.
const (
	CodeRateLimited       = "provider_rate_limited"
	CodeQuotaExceeded     = "provider_quota_exceeded"
	CodeLimitsUnavailable = "provider_limits_unavailable"
)

// LimitError is a controlled, safe error describing a provider limit outcome.
// It carries the bounded provider/operation names and a retry hint but no
// sensitive provider details.
type LimitError struct {
	Code              string
	Provider          string
	Operation         string
	RetryAfterSeconds int
	// Message is a short, user-safe explanation.
	Message string
}

func (e *LimitError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("%s: provider=%s operation=%s", e.Code, e.Provider, e.Operation)
}

// safeMessage returns a friendly, non-sensitive message for the given code.
func safeMessage(code, category string) string {
	label := category
	if label == "" {
		label = "provider"
	}
	switch code {
	case CodeRateLimited:
		return fmt.Sprintf("The %s provider is temporarily rate limited. Please try again later.", label)
	case CodeQuotaExceeded:
		return fmt.Sprintf("The %s provider daily quota has been reached. Please try again later.", label)
	case CodeLimitsUnavailable:
		return fmt.Sprintf("The %s provider limit service is temporarily unavailable. Please try again later.", label)
	default:
		return "The provider is temporarily unavailable. Please try again later."
	}
}

// LimitErrorFrom builds a LimitError from a non-allowed Decision. It returns nil
// when the decision is allowed or does not represent a limit condition.
func LimitErrorFrom(d Decision) *LimitError {
	switch {
	case d.Limited:
		return &LimitError{
			Code:              CodeRateLimited,
			Provider:          d.Provider,
			Operation:         d.Operation,
			RetryAfterSeconds: d.RetryAfterSeconds,
			Message:           safeMessage(CodeRateLimited, d.Category),
		}
	case d.QuotaExceeded:
		return &LimitError{
			Code:              CodeQuotaExceeded,
			Provider:          d.Provider,
			Operation:         d.Operation,
			RetryAfterSeconds: d.RetryAfterSeconds,
			Message:           safeMessage(CodeQuotaExceeded, d.Category),
		}
	case d.Unavailable:
		return &LimitError{
			Code:              CodeLimitsUnavailable,
			Provider:          d.Provider,
			Operation:         d.Operation,
			RetryAfterSeconds: d.RetryAfterSeconds,
			Message:           safeMessage(CodeLimitsUnavailable, d.Category),
		}
	default:
		return nil
	}
}
