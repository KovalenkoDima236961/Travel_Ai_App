package calendar

import "errors"

var (
	ErrCalendarDisabled       = errors.New("calendar disabled")
	ErrCalendarNotConnected   = errors.New("calendar not connected")
	ErrCalendarReauthRequired = errors.New("calendar reauth required")
	ErrInvalidOAuthState      = errors.New("invalid oauth state")
)
