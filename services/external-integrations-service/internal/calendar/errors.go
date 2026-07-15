package calendar

import "errors"

var (
	ErrCalendarDisabled                    = errors.New("calendar disabled")
	ErrCalendarNotConnected                = errors.New("calendar not connected")
	ErrCalendarReauthRequired              = errors.New("calendar reauth required")
	ErrInvalidOAuthState                   = errors.New("invalid oauth state")
	ErrCalendarFreeBusyDisabled            = errors.New("calendar free busy disabled")
	ErrCalendarFreeBusyInvalidRange        = errors.New("calendar free busy invalid range")
	ErrCalendarFreeBusyRangeTooLarge       = errors.New("calendar free busy range too large")
	ErrCalendarFreeBusyInvalidTimeZone     = errors.New("calendar free busy invalid timezone")
	ErrCalendarFreeBusyUnsupportedCalendar = errors.New("calendar free busy unsupported calendar")
	ErrCalendarFreeBusyUnavailable         = errors.New("calendar free busy unavailable")
	ErrCalendarFreeBusyMalformedResponse   = errors.New("calendar free busy malformed response")
)
