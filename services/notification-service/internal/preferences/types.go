package preferences

import "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"

const (
	ChannelInApp = "in_app"
	ChannelEmail = "email"
	ChannelPush  = "push"
)

const (
	ModeInstant      = "instant"
	ModeHourlyDigest = "hourly_digest"
	ModeDailyDigest  = "daily_digest"
	ModeWeeklyDigest = "weekly_digest"
	ModeMuted        = "muted"
)

const (
	CategoryCollaboration      = notifications.CategoryCollaboration
	CategoryComments           = notifications.CategoryComments
	CategoryTripUpdates        = notifications.CategoryTripUpdates
	CategoryRoleChanges        = notifications.CategoryRoleChanges
	CategoryChecklist          = notifications.CategoryChecklist
	CategoryChecklistReminders = "checklist_reminders"
	CategoryReminders          = notifications.CategoryReminders
	CategoryPreTripReminders   = "pre_trip_reminders"
	CategoryExpenses           = notifications.CategoryExpenses
	CategorySettlements        = notifications.CategorySettlements
	CategoryApproval           = notifications.CategoryApproval
	CategoryBudget             = notifications.CategoryBudget
	CategoryHealth             = notifications.CategoryHealth
	CategoryOfflineSync        = notifications.CategoryOfflineSync
	CategoryCalendar           = notifications.CategoryCalendar
	CategoryAIGeneration       = notifications.CategoryAIGeneration
	CategorySecurity           = notifications.CategorySecurity
	CategorySystem             = notifications.CategorySystem
)

var AllChannels = []string{ChannelInApp, ChannelEmail, ChannelPush}
var AllDeliveryModes = []string{ModeInstant, ModeHourlyDigest, ModeDailyDigest, ModeWeeklyDigest, ModeMuted}
var AllCategories = []string{
	CategoryCollaboration, CategoryComments, CategoryRoleChanges, CategoryTripUpdates,
	CategoryChecklist, CategoryChecklistReminders, CategoryReminders, CategoryPreTripReminders,
	CategoryExpenses, CategorySettlements, CategoryApproval, CategoryBudget, CategoryHealth,
	CategoryOfflineSync, CategoryCalendar, CategoryAIGeneration, CategorySecurity, CategorySystem,
}

var knownChannels = stringSet(AllChannels)
var knownModes = stringSet(AllDeliveryModes)
var knownCategories = stringSet(AllCategories)

func IsKnownChannel(channel string) bool   { _, ok := knownChannels[channel]; return ok }
func IsKnownDeliveryMode(mode string) bool { _, ok := knownModes[mode]; return ok }
func IsKnownCategory(category string) bool { _, ok := knownCategories[category]; return ok }

func CategoryForNotificationType(notificationType string) (string, bool) {
	if !notifications.IsKnownType(notificationType) {
		return "", false
	}
	category := notifications.DefaultCategory(notificationType)
	if category == notifications.CategoryChecklist && notificationType == notifications.TypeReminderAssigned {
		return CategoryChecklistReminders, true
	}
	if category == notifications.CategoryReminders {
		return CategoryPreTripReminders, true
	}
	return category, true
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}
