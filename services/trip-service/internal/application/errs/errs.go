// Package errs holds application-level errors raised by use cases.
package errs

import (
	"errors"
	"fmt"
)

// ErrForbidden signals that the authenticated caller exists but lacks the
// permission required for this operation. The HTTP layer maps it to 403.
var ErrForbidden = errors.New("forbidden")

// ExpectedItineraryRevisionRequiredError signals that an itinerary-changing
// request omitted the revision it was based on.
type ExpectedItineraryRevisionRequiredError struct{}

func (e *ExpectedItineraryRevisionRequiredError) Error() string {
	return "expectedItineraryRevision is required."
}

// ErrExpectedItineraryRevisionRequired is mapped by the HTTP layer to a 400
// response with a stable machine-readable code.
var ErrExpectedItineraryRevisionRequired = &ExpectedItineraryRevisionRequiredError{}

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

// ItineraryConflictError signals that the persisted itinerary revision no
// longer matches the client's expected revision.
type ItineraryConflictError struct {
	CurrentItineraryRevision int
}

func (e *ItineraryConflictError) Error() string {
	return "This itinerary was changed by someone else."
}

// NewItineraryConflict builds an itinerary conflict error with the current
// server revision so callers can reload the latest trip separately.
func NewItineraryConflict(currentItineraryRevision int) *ItineraryConflictError {
	return &ItineraryConflictError{CurrentItineraryRevision: currentItineraryRevision}
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

// BudgetConversionError signals that budget summary conversion was required but
// one or more costs could not be converted. The HTTP layer maps it to 502 with
// a stable machine-readable code.
type BudgetConversionError struct{}

func (e *BudgetConversionError) Error() string {
	return "Some costs could not be converted."
}

var ErrBudgetConversionFailed = &BudgetConversionError{}
