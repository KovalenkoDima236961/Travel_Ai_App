package copilot

import "errors"

var (
	ErrDisabled          = errors.New("copilot is disabled")
	ErrRateLimitExceeded = errors.New("copilot rate limit exceeded")
	ErrResponseInvalid   = errors.New("copilot response validation failed")
)
