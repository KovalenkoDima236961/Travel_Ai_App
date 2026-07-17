package notifications

const (
	PriorityLow    = "low"
	PriorityNormal = "normal"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)

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
	TypeChecklistItemAssigned    = "checklist_item_assigned"
	TypeChecklistItemCompleted   = "checklist_item_completed"
	TypeChecklistItemOverdue     = "checklist_item_overdue"
	TypeChecklistGenerated       = "checklist_generated"
	TypePreTripReminderDue       = "pre_trip_reminder_due"
	TypeReminderAssigned         = "reminder_assigned"
	TypeAvailabilityRequested    = "availability_requested"
	TypeDateOptionApplied        = "date_option_applied"
	TypeExpenseAdded             = "expense_added"
	TypeSettlementPaid           = "settlement_paid"
	TypeSettlementPending        = "settlement_pending"
	TypeSettlementOverdue        = "settlement_overdue"
	TypeGroupReadinessNudge      = "group_readiness_nudge"
	TypeAvailabilityNudge        = "availability_nudge"
	TypeChecklistAssignmentNudge = "checklist_assignment_nudge"
	TypeReminderTaskNudge        = "reminder_task_nudge"
	TypePollVoteNudge            = "poll_vote_nudge"
	TypeSettlementNudge          = "settlement_nudge"
	TypeRouteChanged             = "route_changed"
	TypeBudgetConfidenceChanged  = "budget_confidence_changed"
	TypeTripHealthIssue          = "trip_health_issue"
	TypeOfflineSyncConflict      = "offline_sync_conflict"
	TypeCalendarSyncFailed       = "calendar_sync_failed"
	TypeShareSecurityChanged     = "share_security_changed"

	// Approval workflow.
	TypeTripSubmittedForApproval = "trip_submitted_for_approval"
	TypeTripApproved             = "trip_approved"
	TypeTripChangesRequested     = "trip_changes_requested"
	TypeTripApprovalCancelled    = "trip_approval_cancelled"
	TypeTripApprovalResetToDraft = "trip_approval_reset_to_draft"
)

func defaultPriority(notificationType string) string {
	switch notificationType {
	case TypeGenerationJobFailed, TypeBudgetOptimizationFailed, TypeSettlementOverdue,
		TypeOfflineSyncConflict, TypeCalendarSyncFailed, TypeShareSecurityChanged:
		return PriorityUrgent
	case TypeCollaborationInvited, TypeTripSubmittedForApproval, TypeTripChangesRequested,
		TypePreTripReminderDue, TypeChecklistItemOverdue, TypeSettlementPending, TypeRouteChanged:
		return PriorityHigh
	case TypeChecklistItemCompleted:
		return PriorityLow
	default:
		return PriorityNormal
	}
}

func defaultCategory(notificationType string) string {
	switch notificationType {
	case TypeCollaborationInvited, TypeCollaborationAccepted, TypeAvailabilityRequested, TypeGroupReadinessNudge, TypeAvailabilityNudge, TypePollVoteNudge:
		return "collaboration"
	case TypeCollaboratorRoleChange, TypeCollaboratorRemoved:
		return "role_changes"
	case TypeCommentCreated:
		return "comments"
	case TypeChecklistItemAssigned, TypeChecklistItemCompleted, TypeChecklistItemOverdue,
		TypeChecklistGenerated, TypeChecklistAssignmentNudge, TypeReminderTaskNudge, TypeReminderAssigned:
		return "checklist"
	case TypePreTripReminderDue:
		return "reminders"
	case TypeExpenseAdded:
		return "expenses"
	case TypeSettlementPaid, TypeSettlementPending, TypeSettlementOverdue, TypeSettlementNudge:
		return "settlements"
	case TypeTripSubmittedForApproval, TypeTripApproved, TypeTripChangesRequested, TypeTripApprovalCancelled, TypeTripApprovalResetToDraft:
		return "approval"
	case TypeBudgetOptimizationReady, TypeBudgetOptimizationFailed, TypeWorkspaceBudgetCreated, TypeWorkspaceBudgetUpdated, TypeWorkspaceBudgetArchived, TypeWorkspaceBudgetExceeded, TypeWorkspaceBudgetNearLimit:
		return "budget"
	case TypeGenerationJobFailed, TypeItineraryGenerated:
		return "ai_generation"
	case TypeBudgetConfidenceChanged:
		return "budget"
	case TypeTripHealthIssue:
		return "health"
	case TypeOfflineSyncConflict:
		return "offline_sync"
	case TypeCalendarSyncFailed:
		return "calendar"
	case TypeShareSecurityChanged:
		return "security"
	default:
		return "trip_updates"
	}
}

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
	EntityChecklist        = "checklist"
	EntityChecklistItem    = "checklist_item"
	EntityTripReminder     = "trip_reminder"
	EntityAvailability     = "availability"
	EntityDateOption       = "date_option"
	EntityTripExpense      = "trip_expense"
	EntityTripSettlement   = "trip_settlement"
)
