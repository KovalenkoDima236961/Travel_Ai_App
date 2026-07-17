// Package notifications holds the Notification Service use cases: creating
// notifications (via the internal batch endpoint) and serving a user's own
// notification list, unread count, and read-state changes.
package notifications

import "strings"

const (
	PriorityLow    = "low"
	PriorityNormal = "normal"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)

const (
	CategoryCollaboration = "collaboration"
	CategoryComments      = "comments"
	CategoryTripUpdates   = "trip_updates"
	CategoryRoleChanges   = "role_changes"
	CategoryChecklist     = "checklist"
	CategoryReminders     = "reminders"
	CategoryExpenses      = "expenses"
	CategorySettlements   = "settlements"
	CategoryApproval      = "approval"
	CategoryBudget        = "budget"
	CategoryHealth        = "health"
	CategoryOfflineSync   = "offline_sync"
	CategoryCalendar      = "calendar"
	CategoryAIGeneration  = "ai_generation"
	CategorySecurity      = "security"
	CategorySystem        = "system"
)

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
	TypeCommentCreated  = "comment_created"
	TypeTripPollCreated = "trip_poll_created"
	TypeTripPollClosed  = "trip_poll_closed"

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
	TypeGroupReadinessNudge   = "group_readiness_nudge"
	TypeAvailabilityNudge     = "availability_nudge"
	TypeChecklistNudge        = "checklist_assignment_nudge"
	TypeReminderTaskNudge     = "reminder_task_nudge"
	TypePollVoteNudge         = "poll_vote_nudge"
	TypeSettlementNudge       = "settlement_nudge"

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

	// Digest/noise-control vocabulary used by newer producers.
	TypeRouteChanged            = "route_changed"
	TypeChecklistItemAssigned   = "checklist_item_assigned"
	TypeChecklistItemCompleted  = "checklist_item_completed"
	TypeChecklistItemOverdue    = "checklist_item_overdue"
	TypeChecklistGenerated      = "checklist_generated"
	TypeSettlementPending       = "settlement_pending"
	TypeSettlementOverdue       = "settlement_overdue"
	TypeBudgetConfidenceChanged = "budget_confidence_changed"
	TypeTripHealthIssue         = "trip_health_issue"
	TypeOfflineSyncConflict     = "offline_sync_conflict"
	TypeCalendarSyncFailed      = "calendar_sync_failed"
	TypeShareSecurityChanged    = "share_security_changed"
	TypeNotificationDigest      = "notification_digest"
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
	TypeTripPollCreated:             {},
	TypeTripPollClosed:              {},
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
	TypeGroupReadinessNudge:         {},
	TypeAvailabilityNudge:           {},
	TypeChecklistNudge:              {},
	TypeReminderTaskNudge:           {},
	TypePollVoteNudge:               {},
	TypeSettlementNudge:             {},
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
	TypeRouteChanged:                {},
	TypeChecklistItemAssigned:       {},
	TypeChecklistItemCompleted:      {},
	TypeChecklistItemOverdue:        {},
	TypeChecklistGenerated:          {},
	TypeSettlementPending:           {},
	TypeSettlementOverdue:           {},
	TypeBudgetConfidenceChanged:     {},
	TypeTripHealthIssue:             {},
	TypeOfflineSyncConflict:         {},
	TypeCalendarSyncFailed:          {},
	TypeShareSecurityChanged:        {},
	TypeNotificationDigest:          {},
}

// IsKnownType reports whether the notification type is part of the recognised
// vocabulary.
func IsKnownType(notificationType string) bool {
	_, ok := knownTypes[notificationType]
	return ok
}

// DefaultPriority is the deterministic fallback for producers that have not
// yet adopted the priority field.
func DefaultPriority(notificationType string) string {
	switch notificationType {
	case TypeGenerationJobFailed, TypeBudgetOptimizationFailed, TypeOfflineSyncConflict,
		TypeCalendarSyncFailed, TypeShareSecurityChanged, TypeSettlementOverdue:
		return PriorityUrgent
	case TypeCollaborationInvited, TypeTripSubmittedForApproval, TypeTripChangesRequested,
		TypeChecklistItemOverdue, TypePreTripReminderDue, TypeSettlementPending,
		TypeRouteChanged:
		return PriorityHigh
	case TypeChecklistItemCompleted:
		return PriorityLow
	default:
		return PriorityNormal
	}
}

func IsPriority(value string) bool {
	switch value {
	case PriorityLow, PriorityNormal, PriorityHigh, PriorityUrgent:
		return true
	default:
		return false
	}
}

// DefaultCategory keeps old producers governed by the new category controls.
func DefaultCategory(notificationType string) string {
	switch notificationType {
	case TypeCollaborationInvited, TypeCollaborationAccepted, TypeAvailabilityRequested,
		TypeGroupReadinessNudge, TypeAvailabilityNudge, TypePollVoteNudge,
		TypeTripPollCreated, TypeTripPollClosed:
		return CategoryCollaboration
	case TypeCollaboratorRoleChange, TypeCollaboratorRemoved:
		return CategoryRoleChanges
	case TypeCommentCreated:
		return CategoryComments
	case TypeChecklistItemAssigned, TypeChecklistItemCompleted, TypeChecklistItemOverdue, TypeChecklistGenerated,
		TypeChecklistNudge, TypeReminderTaskNudge, TypeReminderAssigned:
		return CategoryChecklist
	case TypePreTripReminderDue:
		return CategoryReminders
	case TypeExpenseAdded:
		return CategoryExpenses
	case TypeSettlementPaid, TypeSettlementPending, TypeSettlementOverdue, TypeSettlementNudge:
		return CategorySettlements
	case TypeTripSubmittedForApproval, TypeTripApproved, TypeTripChangesRequested,
		TypeTripApprovalCancelled, TypeTripApprovalResetToDraft:
		return CategoryApproval
	case TypeBudgetOptimizationReady, TypeBudgetOptimizationFailed,
		TypeWorkspaceBudgetCreated, TypeWorkspaceBudgetUpdated, TypeWorkspaceBudgetArchived,
		TypeWorkspaceBudgetExceeded, TypeWorkspaceBudgetNearLimit, TypeBudgetConfidenceChanged:
		return CategoryBudget
	case TypeTripHealthIssue:
		return CategoryHealth
	case TypeOfflineSyncConflict:
		return CategoryOfflineSync
	case TypeCalendarSyncFailed:
		return CategoryCalendar
	case TypeGenerationJobFailed, TypeItineraryGenerated:
		return CategoryAIGeneration
	case TypeShareSecurityChanged:
		return CategorySecurity
	case TypeNotificationDigest:
		return CategorySystem
	default:
		return CategoryTripUpdates
	}
}

func DefaultDigestKey(notificationType, category, tripID string) string {
	if strings.TrimSpace(tripID) != "" {
		return "trip:" + tripID + ":" + category
	}
	return "category:" + category + ":" + notificationType
}

func IsProtectedFromTripMute(notificationType, category string) bool {
	return category == CategorySecurity || notificationType == TypeOfflineSyncConflict ||
		notificationType == TypeTripHealthIssue || notificationType == TypePreTripReminderDue ||
		notificationType == TypeTripSubmittedForApproval
}
