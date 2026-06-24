// Package errs holds domain-level sentinel errors.
package errs

import "errors"

// ErrNotFound is returned when a requested trip does not exist. Infrastructure
// adapters produce it; the HTTP layer maps it to 404.
var ErrNotFound = errors.New("trip not found")

// ErrConflict is returned when a unique persistence constraint is hit.
var ErrConflict = errors.New("conflict")
