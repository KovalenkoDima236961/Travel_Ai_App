package activity

// Event type constants. Use these instead of scattered string literals so the
// vocabulary stays consistent across recording call sites, the API, and tests.
const (
	// Trip.
	EventTripCreated                         = "trip_created"
	EventTripBudgetUpdated                   = "trip_budget_updated"
	EventRouteUpdated                        = "route_updated"
	EventTransportOptionAttached             = "transport_option_attached"
	EventTransportOptionRemoved              = "transport_option_removed"
	EventRouteAlternativesGenerated          = "route_alternatives_generated"
	EventRouteAlternativeRefined             = "route_alternative_refined"
	EventTripCreatedFromRouteAlternative     = "trip_created_from_route_alternative"
	EventRouteAlternativeApplied             = "route_alternative_applied"
	EventRouteAlternativesPollCreated        = "route_alternatives_poll_created"
	EventTripCreatedFromTemplate             = "trip_created_from_template"
	EventTripCreatedFromAITemplateAdaptation = "trip_created_from_ai_template_adaptation"
	EventTripTravelerAdded                   = "trip_traveler_added"
	EventTripTravelerUpdated                 = "trip_traveler_updated"
	EventTripTravelerRemoved                 = "trip_traveler_removed"
	EventChecklistGenerated                  = "checklist_generated"
	EventChecklistRegenerated                = "checklist_regenerated"
	EventChecklistItemAdded                  = "checklist_item_added"
	EventChecklistItemAssigned               = "checklist_item_assigned"
	EventChecklistItemDeleted                = "checklist_item_deleted"
	EventRemindersGenerated                  = "reminders_generated"
	EventReminderCreated                     = "reminder_created"
	EventReminderUpdated                     = "reminder_updated"
	EventReminderAssigned                    = "reminder_assigned"
	EventReminderDisabled                    = "reminder_disabled"
	EventReminderDeleted                     = "reminder_deleted"
	EventAvailabilitySubmitted               = "availability_submitted"
	EventAvailabilityUpdated                 = "availability_updated"
	EventAvailabilityRemoved                 = "availability_removed"
	EventAvailabilityRequested               = "availability_requested"
	EventAvailabilityImportedFromCalendar    = "availability_imported_from_calendar"
	EventDateOptionApplied                   = "date_option_applied"
	EventDateOptionsPollCreated              = "date_options_poll_created"
	EventExpenseCreated                      = "expense_created"
	EventExpenseUpdated                      = "expense_updated"
	EventExpenseDeleted                      = "expense_deleted"
	EventReceiptUploaded                     = "receipt_uploaded"
	EventReceiptExtracted                    = "receipt_extracted"
	EventReceiptExtractionFailed             = "receipt_extraction_failed"
	EventExpenseCreatedFromReceipt           = "expense_created_from_receipt"
	EventReceiptAttached                     = "receipt_attached"
	EventReceiptDeleted                      = "receipt_deleted"
	EventSettlementMarkedPaid                = "settlement_marked_paid"
	EventSettlementCancelled                 = "settlement_cancelled"
	EventGroupReadinessNudgeSent             = "group_readiness_nudge_sent"
	EventVerificationRefreshed               = "verification_refreshed"
	EventTripRecapGenerated                  = "trip_recap_generated"
	EventTripRecapUpdated                    = "trip_recap_updated"
	EventTripRecapFinalized                  = "trip_recap_finalized"
	EventTripRecapLearningApplied            = "trip_recap_learning_applied"
	EventTripTemplateCreatedFromRecap        = "trip_template_created_from_recap"

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
	EventItineraryItemStatusUpdated  = "itinerary_item_status_updated"
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

	// Decisions.
	EventTripPollCreated  = "trip_poll_created"
	EventTripPollClosed   = "trip_poll_closed"
	EventTripPollArchived = "trip_poll_archived"

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
	EntityTrip               = "trip"
	EntityTripTraveler       = "trip_traveler"
	EntityTripTemplate       = "trip_template"
	EntityAccommodation      = "accommodation"
	EntityItinerary          = "itinerary"
	EntityItineraryDay       = "itinerary_day"
	EntityItineraryItem      = "itinerary_item"
	EntityItineraryVersion   = "itinerary_version"
	EntityComment            = "comment"
	EntityTripPoll           = "trip_poll"
	EntityCollaborator       = "collaborator"
	EntityShare              = "share"
	EntityCalendarSync       = "calendar_sync"
	EntityChecklist          = "checklist"
	EntityChecklistItem      = "checklist_item"
	EntityReminder           = "trip_reminder"
	EntityAvailability       = "availability"
	EntityDateOption         = "date_option"
	EntityTripExpense        = "trip_expense"
	EntityTripExpenseReceipt = "trip_expense_receipt"
	EntityTripSettlement     = "trip_settlement"
	EntityTripRecap          = "trip_recap"
)

// knownEventTypes is the set of event types this version recognises. Recording
// an unknown type is allowed (forward-compat) but is logged so typos surface.
var knownEventTypes = map[string]struct{}{
	EventTripCreated:                         {},
	EventTripBudgetUpdated:                   {},
	EventRouteUpdated:                        {},
	EventTransportOptionAttached:             {},
	EventTransportOptionRemoved:              {},
	EventRouteAlternativesGenerated:          {},
	EventRouteAlternativeRefined:             {},
	EventTripCreatedFromRouteAlternative:     {},
	EventRouteAlternativeApplied:             {},
	EventRouteAlternativesPollCreated:        {},
	EventTripCreatedFromTemplate:             {},
	EventTripCreatedFromAITemplateAdaptation: {},
	EventTripTravelerAdded:                   {},
	EventTripTravelerUpdated:                 {},
	EventTripTravelerRemoved:                 {},
	EventChecklistGenerated:                  {},
	EventChecklistRegenerated:                {},
	EventChecklistItemAdded:                  {},
	EventChecklistItemAssigned:               {},
	EventChecklistItemDeleted:                {},
	EventRemindersGenerated:                  {},
	EventReminderCreated:                     {},
	EventReminderUpdated:                     {},
	EventReminderAssigned:                    {},
	EventReminderDisabled:                    {},
	EventReminderDeleted:                     {},
	EventAvailabilitySubmitted:               {},
	EventAvailabilityUpdated:                 {},
	EventAvailabilityRemoved:                 {},
	EventAvailabilityRequested:               {},
	EventAvailabilityImportedFromCalendar:    {},
	EventDateOptionApplied:                   {},
	EventDateOptionsPollCreated:              {},
	EventExpenseCreated:                      {},
	EventExpenseUpdated:                      {},
	EventExpenseDeleted:                      {},
	EventReceiptUploaded:                     {},
	EventReceiptExtracted:                    {},
	EventReceiptExtractionFailed:             {},
	EventExpenseCreatedFromReceipt:           {},
	EventReceiptAttached:                     {},
	EventReceiptDeleted:                      {},
	EventSettlementMarkedPaid:                {},
	EventSettlementCancelled:                 {},
	EventGroupReadinessNudgeSent:             {},
	EventVerificationRefreshed:               {},
	EventTripRecapGenerated:                  {},
	EventTripRecapUpdated:                    {},
	EventTripRecapFinalized:                  {},
	EventTripRecapLearningApplied:            {},
	EventTripTemplateCreatedFromRecap:        {},
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
	EventItineraryItemStatusUpdated:          {},
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
	EventTripPollCreated:                     {},
	EventTripPollClosed:                      {},
	EventTripPollArchived:                    {},
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
