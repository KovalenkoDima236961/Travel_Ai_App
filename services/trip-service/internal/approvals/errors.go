package approvals

import "errors"

// Sentinel errors describing why an approval action is not currently allowed.
// They are intentionally transport-agnostic; the trip use case maps them onto
// application errors (400/403/409) before they reach the HTTP layer.
var (
	// ErrNotWorkspaceTrip is returned when an approval action targets a personal
	// trip. Approval only applies to workspace trips.
	ErrNotWorkspaceTrip = errors.New("approval is only available for workspace trips")

	// ErrInvalidTransition is returned when an action is not permitted from the
	// trip's current approval status (e.g. approving a draft, submitting a
	// pending trip).
	ErrInvalidTransition = errors.New("approval action is not allowed from the current status")

	// ErrBlockedByChecklist is returned when a submission is blocked by a failing
	// blocker check (a missing itinerary in v1).
	ErrBlockedByChecklist = errors.New("submission is blocked by an unmet requirement")
)
