// Package errs holds application-level errors raised by use cases.
package errs

import "fmt"

// InvalidInputError signals that the caller supplied invalid input. The HTTP
// layer maps it to 400. It lets the use case enforce business rules
// independently of the transport-layer validator, which keeps it unit-testable.
type InvalidInputError struct {
	Message string
}

func (e *InvalidInputError) Error() string { return e.Message }

// NewInvalidInput builds an InvalidInputError from a format string.
func NewInvalidInput(format string, args ...any) *InvalidInputError {
	return &InvalidInputError{Message: fmt.Sprintf(format, args...)}
}

// DependencyError signals that an upstream dependency required by the use case
// is unavailable or returned unusable data.
type DependencyError struct {
	Message string
}

func (e *DependencyError) Error() string { return e.Message }

// NewDependencyError builds a DependencyError from a format string.
func NewDependencyError(format string, args ...any) *DependencyError {
	return &DependencyError{Message: fmt.Sprintf(format, args...)}
}
