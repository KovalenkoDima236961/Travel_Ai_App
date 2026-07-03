package push

import "errors"

var (
	// ErrPushRejected is returned when a push service returns a non-success
	// status that is not classified as gone/invalid.
	ErrPushRejected = errors.New("push service rejected notification")
)
