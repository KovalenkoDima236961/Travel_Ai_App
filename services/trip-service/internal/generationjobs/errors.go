package generationjobs

import "errors"

var (
	ErrDisabled       = errors.New("generation jobs are disabled")
	ErrNotCancellable = errors.New("generation job cannot be cancelled")
)
