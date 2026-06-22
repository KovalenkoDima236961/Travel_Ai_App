package errs

import "errors"

var (
	// ErrNotFound is returned when a requested auth domain record does not exist.
	ErrNotFound = errors.New("auth record not found")
	// ErrAlreadyExists is returned when a unique auth domain record already exists.
	ErrAlreadyExists = errors.New("auth record already exists")
)
