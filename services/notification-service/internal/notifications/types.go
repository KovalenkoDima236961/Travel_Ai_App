// Package notifications holds the Notification Service use cases: creating
// notifications (via the internal batch endpoint) and serving a user's own
// notification list, unread count, and read-state changes.
package notifications

// Notification type constants. Use these instead of scattered string literals so
// the vocabulary stays consistent across the create path, the API, and tests.
// They mirror the Trip Service activity vocabulary where the two overlap.
const (
	// Collaboration.
	TypeCollaborationInvited   = "collaboration_invited"
	TypeCollaborationAccepted  = "collaboration_accepted"
	TypeCollaboratorRoleChange = "collaborator_role_changed"
	TypeCollaboratorRemoved    = "collaborator_removed"

	// Comments.
	TypeCommentCreated = "comment_created"

	// Itinerary.
	TypeItineraryUpdated      = "itinerary_updated"
	TypeItineraryGenerated    = "itinerary_generated"
	TypeDayRegenerated        = "day_regenerated"
	TypeItemRegenerated       = "item_regenerated"
	TypeVersionRestored       = "version_restored"
	TypeGenerationJobFailed   = "generation_job_failed"
	TypeDateOptionApplied     = "date_option_applied"
	TypeAvailabilityRequested = "availability_requested"
	TypePreTripReminderDue    = "pre_trip_reminder_due"
	TypeReminderAssigned      = "reminder_assigned"
	TypeExpenseAdded          = "expense_added"
	TypeSettlementPaid        = "settlement_paid"

	// Budget optimization.
	TypeBudgetOptimizationReady  = "budget_optimization_ready"
	TypeBudgetOptimizationFailed = "budget_optimization_failed"

	// Workspace budgets.
	TypeWorkspaceBudgetCreated   = "workspace_budget_created"
	TypeWorkspaceBudgetUpdated   = "workspace_budget_updated"
	TypeWorkspaceBudgetArchived  = "workspace_budget_archived"
	TypeWorkspaceBudgetExceeded  = "workspace_budget_exceeded"
	TypeWorkspaceBudgetNearLimit = "workspace_budget_nearing_limit"

	// Workspaces.
	TypeWorkspaceInvited            = "workspace_invited"
	TypeWorkspaceInvitationAccepted = "workspace_invitation_accepted"
	TypeWorkspaceInvitationDeclined = "workspace_invitation_declined"
	TypeWorkspaceMemberRemoved      = "workspace_member_removed"
	TypeWorkspaceRoleChanged        = "workspace_role_changed"
	TypeWorkspaceTripCreated        = "workspace_trip_created"

	// Approval workflow.
	TypeTripSubmittedForApproval = "trip_submitted_for_approval"
	TypeTripApproved             = "trip_approved"
	TypeTripChangesRequested     = "trip_changes_requested"
	TypeTripApprovalCancelled    = "trip_approval_cancelled"
	TypeTripApprovalResetToDraft = "trip_approval_reset_to_draft"
)

// Entity type constants describe the kind of object a notification refers to.
const (
	EntityTrip             = "trip"
	EntityComment          = "comment"
	EntityCollaborator     = "collaborator"
	EntityItinerary        = "itinerary"
	EntityItineraryDay     = "itinerary_day"
	EntityItineraryItem    = "itinerary_item"
	EntityItineraryVersion = "itinerary_version"
	EntityWorkspace        = "workspace"
	EntityWorkspaceBudget  = "workspace_budget"
	EntityAvailability     = "availability"
	EntityDateOption       = "date_option"
	EntityTripReminder     = "trip_reminder"
	EntityTripExpense      = "trip_expense"
	EntityTripSettlement   = "trip_settlement"
)

// knownTypes is the set of notification types this version recognises. Unknown
// types are still accepted by the create path for forward compatibility, but
// this helper remains useful for renderers/tests that need to distinguish known
// vocabulary from future types.
var knownTypes = map[string]struct{}{
	TypeCollaborationInvited:        {},
	TypeCollaborationAccepted:       {},
	TypeCollaboratorRoleChange:      {},
	TypeCollaboratorRemoved:         {},
	TypeCommentCreated:              {},
	TypeItineraryUpdated:            {},
	TypeItineraryGenerated:          {},
	TypeDayRegenerated:              {},
	TypeItemRegenerated:             {},
	TypeVersionRestored:             {},
	TypeGenerationJobFailed:         {},
	TypeDateOptionApplied:           {},
	TypeAvailabilityRequested:       {},
	TypePreTripReminderDue:          {},
	TypeReminderAssigned:            {},
	TypeExpenseAdded:                {},
	TypeSettlementPaid:              {},
	TypeBudgetOptimizationReady:     {},
	TypeBudgetOptimizationFailed:    {},
	TypeWorkspaceBudgetCreated:      {},
	TypeWorkspaceBudgetUpdated:      {},
	TypeWorkspaceBudgetArchived:     {},
	TypeWorkspaceBudgetExceeded:     {},
	TypeWorkspaceBudgetNearLimit:    {},
	TypeWorkspaceInvited:            {},
	TypeWorkspaceInvitationAccepted: {},
	TypeWorkspaceInvitationDeclined: {},
	TypeWorkspaceMemberRemoved:      {},
	TypeWorkspaceRoleChanged:        {},
	TypeWorkspaceTripCreated:        {},
	TypeTripSubmittedForApproval:    {},
	TypeTripApproved:                {},
	TypeTripChangesRequested:        {},
	TypeTripApprovalCancelled:       {},
	TypeTripApprovalResetToDraft:    {},
}

// IsKnownType reports whether the notification type is part of the recognised
// vocabulary.
func IsKnownType(notificationType string) bool {
	_, ok := knownTypes[notificationType]
	return ok
}
