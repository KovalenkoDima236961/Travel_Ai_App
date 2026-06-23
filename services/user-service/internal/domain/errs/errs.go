// Package errs holds domain-level sentinel errors.
package errs

import "errors"

// ErrNotFound is returned when a requested profile/preferences row does not exist.
var ErrNotFound = errors.New("user resource not found")
