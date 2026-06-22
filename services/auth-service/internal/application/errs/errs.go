package errs

import (
	"errors"
	"fmt"
)

var (
	ErrEmailAlreadyExists  = errors.New("email already exists")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrInvalidAccessToken  = errors.New("invalid access token")
)

// InvalidInputError signals that the caller supplied invalid input.
type InvalidInputError struct {
	Message string
}

func (e *InvalidInputError) Error() string { return e.Message }

// NewInvalidInput builds an InvalidInputError from a format string.
func NewInvalidInput(format string, args ...any) *InvalidInputError {
	return &InvalidInputError{Message: fmt.Sprintf(format, args...)}
}
