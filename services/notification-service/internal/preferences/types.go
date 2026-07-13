// Package preferences holds the Notification Service use case for per-user
// notification preferences. Preferences are global per user and category-based.
// They are split across independent channels (in-app, email, and push) so a
// user can, for example, keep in-app collaboration notifications while turning
// off the matching emails or browser push notifications.
//
// Preferences only gate the creation of future notifications: disabling a
// category never deletes existing notifications, never touches the activity
// feed, and never affects core collaboration data (invitations, comments).
package preferences

import "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"

// Channel constants. A preference always targets exactly one channel. Use these
// instead of scattered string literals.
const (
	// ChannelInApp controls whether an in-app notification row is created.
	ChannelInApp = "in_app"
	// ChannelEmail controls whether an email is sent.
	ChannelEmail = "email"
	// ChannelPush controls whether a browser Web Push notification is sent.
	ChannelPush = "push"
)

// Category constants. Notification types are grouped into a small set of
// user-facing categories so the settings UI stays manageable.
const (
	CategoryCollaboration      = "collaboration"
	CategoryComments           = "comments"
	CategoryTripUpdates        = "trip_updates"
	CategoryRoleChanges        = "role_changes"
	CategoryPreTripReminders   = "pre_trip_reminders"
	CategoryChecklistReminders = "checklist_reminders"
)

// AllChannels lists the channels in display order. Range over this (never a map)
// when building the full preference matrix so the output order is deterministic.
var AllChannels = []string{ChannelInApp, ChannelEmail, ChannelPush}

// AllCategories lists the categories in display order. Range over this (never a
// map) when building the full preference matrix so the output order is stable.
var AllCategories = []string{
	CategoryCollaboration,
	CategoryComments,
	CategoryRoleChanges,
	CategoryTripUpdates,
	CategoryPreTripReminders,
	CategoryChecklistReminders,
}

// knownChannels and knownCategories are the recognised vocabularies. The update
// endpoint rejects anything outside them.
var (
	knownChannels = map[string]struct{}{
		ChannelInApp: {},
		ChannelEmail: {},
		ChannelPush:  {},
	}
	knownCategories = map[string]struct{}{
		CategoryCollaboration:      {},
		CategoryComments:           {},
		CategoryTripUpdates:        {},
		CategoryRoleChanges:        {},
		CategoryPreTripReminders:   {},
		CategoryChecklistReminders: {},
	}
)

// IsKnownChannel reports whether the channel is part of the recognised vocabulary.
func IsKnownChannel(channel string) bool {
	_, ok := knownChannels[channel]
	return ok
}

// IsKnownCategory reports whether the category is part of the recognised vocabulary.
func IsKnownCategory(category string) bool {
	_, ok := knownCategories[category]
	return ok
}

// typeToCategory maps each known notification type to its preference category.
// It mirrors the Notification Service notification vocabulary; adding a new type
// there should also add it here so it is governed by a category.
var typeToCategory = map[string]string{
	notifications.TypeCollaborationInvited:        CategoryCollaboration,
	notifications.TypeCollaborationAccepted:       CategoryCollaboration,
	notifications.TypeCommentCreated:              CategoryComments,
	notifications.TypeCollaboratorRoleChange:      CategoryRoleChanges,
	notifications.TypeCollaboratorRemoved:         CategoryRoleChanges,
	notifications.TypeItineraryUpdated:            CategoryTripUpdates,
	notifications.TypeItineraryGenerated:          CategoryTripUpdates,
	notifications.TypeDayRegenerated:              CategoryTripUpdates,
	notifications.TypeItemRegenerated:             CategoryTripUpdates,
	notifications.TypeVersionRestored:             CategoryTripUpdates,
	notifications.TypeGenerationJobFailed:         CategoryTripUpdates,
	notifications.TypeDateOptionApplied:           CategoryTripUpdates,
	notifications.TypeAvailabilityRequested:       CategoryCollaboration,
	notifications.TypeBudgetOptimizationReady:     CategoryTripUpdates,
	notifications.TypeBudgetOptimizationFailed:    CategoryTripUpdates,
	notifications.TypeWorkspaceBudgetCreated:      CategoryTripUpdates,
	notifications.TypeWorkspaceBudgetUpdated:      CategoryTripUpdates,
	notifications.TypeWorkspaceBudgetArchived:     CategoryTripUpdates,
	notifications.TypeWorkspaceBudgetExceeded:     CategoryTripUpdates,
	notifications.TypeWorkspaceBudgetNearLimit:    CategoryTripUpdates,
	notifications.TypeWorkspaceInvited:            CategoryCollaboration,
	notifications.TypeWorkspaceInvitationAccepted: CategoryCollaboration,
	notifications.TypeWorkspaceInvitationDeclined: CategoryCollaboration,
	notifications.TypeWorkspaceMemberRemoved:      CategoryRoleChanges,
	notifications.TypeWorkspaceRoleChanged:        CategoryRoleChanges,
	notifications.TypeWorkspaceTripCreated:        CategoryTripUpdates,
	notifications.TypeTripSubmittedForApproval:    CategoryCollaboration,
	notifications.TypeTripApproved:                CategoryCollaboration,
	notifications.TypeTripChangesRequested:        CategoryCollaboration,
	notifications.TypeTripApprovalCancelled:       CategoryCollaboration,
	notifications.TypeTripApprovalResetToDraft:    CategoryCollaboration,
	notifications.TypePreTripReminderDue:          CategoryPreTripReminders,
	notifications.TypeReminderAssigned:            CategoryChecklistReminders,
}

// CategoryForNotificationType maps a notification type to its preference
// category. The second return value is false for an unknown type; callers must
// then fall back to the documented unknown-type defaults (in-app allowed, email
// not) rather than dropping the notification.
func CategoryForNotificationType(notificationType string) (string, bool) {
	category, ok := typeToCategory[notificationType]
	return category, ok
}
