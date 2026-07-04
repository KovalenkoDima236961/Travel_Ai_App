package activity

// Event type constants. Use these instead of scattered string literals so the
// vocabulary stays consistent across recording call sites, the API, and tests.
const (
	// Trip.
	EventTripCreated             = "trip_created"
	EventTripBudgetUpdated       = "trip_budget_updated"
	EventTripCreatedFromTemplate = "trip_created_from_template"

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
	EventGenerationJobFailed         = "generation_job_failed"
	EventBudgetOptimizationRequested = "budget_optimization_requested"
	EventBudgetOptimizationProposed  = "budget_optimization_proposed"
	EventBudgetOptimizationApplied   = "budget_optimization_applied"
	EventBudgetOptimizationDiscarded = "budget_optimization_discarded"
	EventBudgetOptimizationFailed    = "budget_optimization_failed"

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
)

// Entity type constants describe the kind of object an event refers to.
const (
	EntityTrip             = "trip"
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
	EventTripCreated:                 {},
	EventTripBudgetUpdated:           {},
	EventTripCreatedFromTemplate:     {},
	EventTemplateCreated:             {},
	EventTemplateArchived:            {},
	EventAccommodationAdded:          {},
	EventAccommodationUpdated:        {},
	EventAccommodationRemoved:        {},
	EventItineraryGenerated:          {},
	EventItineraryUpdated:            {},
	EventDayRegenerated:              {},
	EventItemRegenerated:             {},
	EventVersionRestored:             {},
	EventGenerationJobFailed:         {},
	EventBudgetOptimizationRequested: {},
	EventBudgetOptimizationProposed:  {},
	EventBudgetOptimizationApplied:   {},
	EventBudgetOptimizationDiscarded: {},
	EventBudgetOptimizationFailed:    {},
	EventCommentCreated:              {},
	EventCommentUpdated:              {},
	EventCommentDeleted:              {},
	EventCollaboratorInvited:         {},
	EventCollaboratorAccepted:        {},
	EventCollaboratorDeclined:        {},
	EventCollaboratorRoleChanged:     {},
	EventCollaboratorRemoved:         {},
	EventShareCreated:                {},
	EventShareUpdated:                {},
	EventShareDisabled:               {},
	EventSharePasswordEnabled:        {},
	EventSharePasswordDisabled:       {},
	EventShareExpirationUpdated:      {},
	EventCalendarSynced:              {},
	EventCalendarSyncRemoved:         {},
}

// IsKnownEventType reports whether the event type is part of the recognised
// vocabulary.
func IsKnownEventType(eventType string) bool {
	_, ok := knownEventTypes[eventType]
	return ok
}
