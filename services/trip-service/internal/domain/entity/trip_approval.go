package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TripApprovalFields is the set of approval columns stored on a trip. It is
// loaded independently of the main Trip entity (via GetTripApprovalFields) so the
// hot trip read/scan path stays untouched and personal trips carry no approval
// weight beyond a single status string.
type TripApprovalFields struct {
	TripID                    uuid.UUID
	WorkspaceID               *uuid.UUID
	Status                    string
	SubmittedAt               *time.Time
	SubmittedByUserID         *uuid.UUID
	ApprovedAt                *time.Time
	ApprovedByUserID          *uuid.UUID
	ChangesRequestedAt        *time.Time
	ChangesRequestedByUserID  *uuid.UUID
	CancelledAt               *time.Time
	CancelledByUserID         *uuid.UUID
	Note                      *string
	DecisionNote              *string
	LastStatusChangedAt       *time.Time
	LastStatusChangedByUserID *uuid.UUID
}

// TripApprovalEvent is one row of approval history (trip_approval_events). It is
// kept separate from the generic trip activity feed so approval decisions have a
// durable, queryable trail that can carry a checklist snapshot.
type TripApprovalEvent struct {
	ID                uuid.UUID
	TripID            uuid.UUID
	WorkspaceID       uuid.UUID
	ActorUserID       uuid.UUID
	EventType         string
	FromStatus        *string
	ToStatus          string
	Note              *string
	ChecklistSnapshot json.RawMessage
	CreatedAt         time.Time
}

// ListWorkspaceApprovalsParams filters and paginates the workspace approvals
// queue. An empty Statuses slice means "no status filter" (the service decides
// the default active set before calling the repository).
type ListWorkspaceApprovalsParams struct {
	WorkspaceID     uuid.UUID
	Statuses        []string
	Limit           int
	Offset          int
	SubmittedAfter  *time.Time
	SubmittedBefore *time.Time
}

// WorkspaceApprovalCounts is the per-status tally shown above the approvals queue.
type WorkspaceApprovalCounts struct {
	PendingApproval  int
	ChangesRequested int
	Approved         int
	Draft            int
	Cancelled        int
}

// ApprovalResetResult reports the outcome of an atomic reset-on-edit. Reset is
// false when the trip was not in a resettable state (not a workspace trip, or not
// approved/pending), in which case the other fields are zero values.
type ApprovalResetResult struct {
	Reset       bool
	FromStatus  string
	ToStatus    string
	WorkspaceID uuid.UUID
}

// WorkspaceApprovalRow is a lightweight projection of a workspace trip used to
// build the approvals queue. It carries only the trip fields the queue needs; the
// service enriches each row with a checklist status and estimated total.
type WorkspaceApprovalRow struct {
	TripID            uuid.UUID
	WorkspaceID       uuid.UUID
	Destination       string
	StartDate         *time.Time
	BudgetAmount      *float64
	BudgetCurrency    string
	ApprovalStatus    string
	SubmittedAt       *time.Time
	SubmittedByUserID *uuid.UUID
	ApprovedAt        *time.Time
	UpdatedAt         time.Time
	Itinerary         json.RawMessage
}
