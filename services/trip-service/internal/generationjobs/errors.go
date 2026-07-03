package generationjobs

import "errors"

var (
	ErrDisabled           = errors.New("generation jobs are disabled")
	ErrNotCancellable     = errors.New("generation job cannot be cancelled")
	ErrJobDispatchFailed  = errors.New("generation job dispatch failed")
	ErrJobAlreadyFinished = errors.New("generation job already finished")
	ErrOpsInvalidAction   = errors.New("ops action is not allowed for this job")
	ErrOpsJobNotStale     = errors.New("generation job is not stale")
)
