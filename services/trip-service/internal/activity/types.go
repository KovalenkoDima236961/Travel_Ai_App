package activity

// Event type constants. Use these instead of scattered string literals so the
// vocabulary stays consistent across recording call sites, the API, and tests.
const (
	// Trip.
	EventTripCreated                         = "trip_created"
	EventTripBudgetUpdated                   = "trip_budget_updated"
	EventTripCreatedFromTemplate             = "trip_created_from_template"
	EventTripCreatedFromAITemplateAdaptation = "trip_created_from_ai_template_adaptation"
	EventTripTravelerAdded                   = "trip_traveler_added"
	EventTripTravelerUpdated                 = "trip_traveler_updated"
	EventTripTravelerRemoved                 = "trip_traveler_removed"

	// Templates.
	EventTemplateCreated  = "template_created"
	EventTemplateArchived = "template_archived"

	// Accommodation.
	EventAccommodationAdded   = "accommodation_added"
	EventAccommodationUpdated = "accommodation_updated"
	EventAccommodationRemoved = "accommodation_removed"

	// Itinerary.
	EventItineraryGenerated          = "itinerary_generated"
	EventItineraryUpdated            = "itinerary_updated"
	EventDayRegenerated              = "day_regenerated"
	EventItemRegenerated             = "item_regenerated"
	EventVersionRestored             = "version_restored"
	EventCostSplitUpdated            = "cost_split_updated"
	EventAccommodationSplitUpdated   = "accommodation_split_updated"
	EventGenerationJobFailed         = "generation_job_failed"
	EventBudgetOptimizationRequested = "budget_optimization_requested"
	EventBudgetOptimizationProposed  = "budget_optimization_proposed"
	EventBudgetOptimizationApplied   = "budget_optimization_applied"
	EventBudgetOptimizationDiscarded = "budget_optimization_discarded"
	EventBudgetOptimizationFailed    = "budget_optimization_failed"
	EventTripRepairJobCreated        = "trip_repair_job_created"
	EventTripRepairProposalCreated   = "trip_repair_proposal_created"
	EventTripRepairProposalApplied   = "trip_repair_proposal_applied"
	EventTripRepairProposalDiscarded = "trip_repair_proposal_discarded"
	EventTripRepairProposalExpired   = "trip_repair_proposal_expired"

	// Comments.
	EventCommentCreated = "comment_created"
	EventCommentUpdated = "comment_updated"
	EventCommentDeleted = "comment_deleted"

	// Collaboration.
	EventCollaboratorInvited     = "collaborator_invited"
	EventCollaboratorAccepted    = "collaborator_accepted"
	EventCollaboratorDeclined    = "collaborator_declined"
	EventCollaboratorRoleChanged = "collaborator_role_changed"
	EventCollaboratorRemoved     = "collaborator_removed"

	// Sharing.
	EventShareCreated           = "share_created"
	EventShareUpdated           = "share_updated"
	EventShareDisabled          = "share_disabled"
	EventSharePasswordEnabled   = "share_password_enabled"
	EventSharePasswordDisabled  = "share_password_disabled"
	EventShareExpirationUpdated = "share_expiration_updated"

	// Calendar sync.
	EventCalendarSynced      = "calendar_synced"
	EventCalendarSyncRemoved = "calendar_sync_removed"

	// Approval workflow.
	EventTripSubmittedForApproval = "trip_submitted_for_approval"
	EventTripApproved             = "trip_approved"
	EventTripChangesRequested     = "trip_changes_requested"
	EventTripApprovalCancelled    = "trip_approval_cancelled"
	EventTripApprovalResetToDraft = "trip_approval_reset_to_draft"
)

// Entity type constants describe the kind of object an event refers to.
const (
	EntityTrip             = "trip"
	EntityTripTraveler     = "trip_traveler"
	EntityTripTemplate     = "trip_template"
	EntityAccommodation    = "accommodation"
	EntityItinerary        = "itinerary"
	EntityItineraryDay     = "itinerary_day"
	EntityItineraryItem    = "itinerary_item"
	EntityItineraryVersion = "itinerary_version"
	EntityComment          = "comment"
	EntityCollaborator     = "collaborator"
	EntityShare            = "share"
	EntityCalendarSync     = "calendar_sync"
)

// knownEventTypes is the set of event types this version recognises. Recording
// an unknown type is allowed (forward-compat) but is logged so typos surface.
var knownEventTypes = map[string]struct{}{
	EventTripCreated:                         {},
	EventTripBudgetUpdated:                   {},
	EventTripCreatedFromTemplate:             {},
	EventTripCreatedFromAITemplateAdaptation: {},
	EventTripTravelerAdded:                   {},
	EventTripTravelerUpdated:                 {},
	EventTripTravelerRemoved:                 {},
	EventTemplateCreated:                     {},
	EventTemplateArchived:                    {},
	EventAccommodationAdded:                  {},
	EventAccommodationUpdated:                {},
	EventAccommodationRemoved:                {},
	EventItineraryGenerated:                  {},
	EventItineraryUpdated:                    {},
	EventDayRegenerated:                      {},
	EventItemRegenerated:                     {},
	EventVersionRestored:                     {},
	EventCostSplitUpdated:                    {},
	EventAccommodationSplitUpdated:           {},
	EventGenerationJobFailed:                 {},
	EventBudgetOptimizationRequested:         {},
	EventBudgetOptimizationProposed:          {},
	EventBudgetOptimizationApplied:           {},
	EventBudgetOptimizationDiscarded:         {},
	EventBudgetOptimizationFailed:            {},
	EventTripRepairJobCreated:                {},
	EventTripRepairProposalCreated:           {},
	EventTripRepairProposalApplied:           {},
	EventTripRepairProposalDiscarded:         {},
	EventTripRepairProposalExpired:           {},
	EventCommentCreated:                      {},
	EventCommentUpdated:                      {},
	EventCommentDeleted:                      {},
	EventCollaboratorInvited:                 {},
	EventCollaboratorAccepted:                {},
	EventCollaboratorDeclined:                {},
	EventCollaboratorRoleChanged:             {},
	EventCollaboratorRemoved:                 {},
	EventShareCreated:                        {},
	EventShareUpdated:                        {},
	EventShareDisabled:                       {},
	EventSharePasswordEnabled:                {},
	EventSharePasswordDisabled:               {},
	EventShareExpirationUpdated:              {},
	EventCalendarSynced:                      {},
	EventCalendarSyncRemoved:                 {},
	EventTripSubmittedForApproval:            {},
	EventTripApproved:                        {},
	EventTripChangesRequested:                {},
	EventTripApprovalCancelled:               {},
	EventTripApprovalResetToDraft:            {},
}

// IsKnownEventType reports whether the event type is part of the recognised
// vocabulary.
func IsKnownEventType(eventType string) bool {
	_, ok := knownEventTypes[eventType]
	return ok
}
