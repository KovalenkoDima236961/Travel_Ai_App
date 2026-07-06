// Package approvals holds the pure, dependency-free core of the Workspace
// Approval Workflow: the approval status vocabulary, the allowed status
// transitions, and the submission checklist calculator. It performs no I/O and
// no permission checks — the trip use case (internal/application/service) gathers
// signals, enforces permissions, and persists state; this package only computes.
package approvals

// Status is a trip's approval status. Personal trips are always StatusNotRequired;
// workspace trips move through the draft -> pending -> approved/changes lifecycle.
type Status string

const (
	StatusNotRequired      Status = "not_required"
	StatusDraft            Status = "draft"
	StatusPendingApproval  Status = "pending_approval"
	StatusChangesRequested Status = "changes_requested"
	StatusApproved         Status = "approved"
	StatusCancelled        Status = "cancelled"
)

// Valid reports whether s is a recognised approval status.
func (s Status) Valid() bool {
	switch s {
	case StatusNotRequired, StatusDraft, StatusPendingApproval,
		StatusChangesRequested, StatusApproved, StatusCancelled:
		return true
	default:
		return false
	}
}

// EventType names an entry in the approval history (trip_approval_events).
type EventType string

const (
	EventSubmitted        EventType = "submitted"
	EventApproved         EventType = "approved"
	EventChangesRequested EventType = "changes_requested"
	EventCancelled        EventType = "cancelled"
	EventResetToDraft     EventType = "reset_to_draft"
)

// CanSubmitFrom reports whether a submit action is allowed from the given status.
// A submit is valid from draft, changes_requested, or a previously cancelled
// submission; it is never valid from pending_approval, approved, or not_required.
func CanSubmitFrom(s Status) bool {
	return s == StatusDraft || s == StatusChangesRequested || s == StatusCancelled
}

// CanApproveFrom reports whether an approve action is allowed from the given status.
func CanApproveFrom(s Status) bool { return s == StatusPendingApproval }

// CanRequestChangesFrom reports whether a request-changes action is allowed.
// v1 only permits it while pending (see task recommendation), keeping the flow
// simple rather than allowing approval revocation.
func CanRequestChangesFrom(s Status) bool { return s == StatusPendingApproval }

// CanCancelFrom reports whether a pending submission may be cancelled.
func CanCancelFrom(s Status) bool { return s == StatusPendingApproval }
