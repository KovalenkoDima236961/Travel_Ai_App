// Package errs holds application-level errors raised by use cases.
package errs

import "fmt"

// InvalidInputError signals that the caller supplied invalid input. The HTTP
// layer maps it to 400.
type InvalidInputError struct {
	Message string
}

func (e *InvalidInputError) Error() string { return e.Message }

// NewInvalidInput builds an InvalidInputError from a format string.
func NewInvalidInput(format string, args ...any) *InvalidInputError {
	return &InvalidInputError{Message: fmt.Sprintf(format, args...)}
}
