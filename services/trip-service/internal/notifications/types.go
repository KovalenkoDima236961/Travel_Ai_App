package notifications

// Notification type constants. These must match the vocabulary the Notification
// Service recognises (it rejects unknown types), so they are defined here as the
// single source of truth for the values Trip Service sends.
const (
	TypeCollaborationInvited     = "collaboration_invited"
	TypeCollaborationAccepted    = "collaboration_accepted"
	TypeCollaboratorRoleChange   = "collaborator_role_changed"
	TypeCollaboratorRemoved      = "collaborator_removed"
	TypeCommentCreated           = "comment_created"
	TypeTripPollCreated          = "trip_poll_created"
	TypeTripPollClosed           = "trip_poll_closed"
	TypeItineraryUpdated         = "itinerary_updated"
	TypeItineraryGenerated       = "itinerary_generated"
	TypeDayRegenerated           = "day_regenerated"
	TypeItemRegenerated          = "item_regenerated"
	TypeVersionRestored          = "version_restored"
	TypeGenerationJobFailed      = "generation_job_failed"
	TypeBudgetOptimizationReady  = "budget_optimization_ready"
	TypeBudgetOptimizationFailed = "budget_optimization_failed"
	TypeWorkspaceBudgetCreated   = "workspace_budget_created"
	TypeWorkspaceBudgetUpdated   = "workspace_budget_updated"
	TypeWorkspaceBudgetArchived  = "workspace_budget_archived"
	TypeWorkspaceBudgetExceeded  = "workspace_budget_exceeded"
	TypeWorkspaceBudgetNearLimit = "workspace_budget_nearing_limit"

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
	EntityTripPoll         = "trip_poll"
	EntityCollaborator     = "collaborator"
	EntityItinerary        = "itinerary"
	EntityItineraryDay     = "itinerary_day"
	EntityItineraryItem    = "itinerary_item"
	EntityItineraryVersion = "itinerary_version"
	EntityWorkspaceBudget  = "workspace_budget"
)
