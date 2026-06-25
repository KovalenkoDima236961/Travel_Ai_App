// Package errs holds domain-level sentinel errors.
package errs

import "errors"

// ErrNotFound is returned when a requested notification does not exist or does
// not belong to the requesting user. Infrastructure adapters produce it; the
// HTTP layer maps it to 404.
var ErrNotFound = errors.New("notification not found")

// ErrConflict is returned when a unique persistence constraint is hit.
var ErrConflict = errors.New("conflict")
