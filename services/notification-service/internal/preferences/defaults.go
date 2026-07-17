package preferences

// defaultModeMatrix is used when no stored override exists. It deliberately
// keeps urgent/action channels useful while digesting routine email and muting
// noisy push categories.
var defaultModeMatrix = map[string]map[string]string{
	ChannelInApp: {
		CategoryCollaboration:      ModeInstant,
		CategoryComments:           ModeInstant,
		CategoryRoleChanges:        ModeInstant,
		CategoryTripUpdates:        ModeDailyDigest,
		CategoryChecklist:          ModeInstant,
		CategoryChecklistReminders: ModeInstant,
		CategoryReminders:          ModeInstant,
		CategoryPreTripReminders:   ModeInstant,
		CategoryExpenses:           ModeDailyDigest,
		CategorySettlements:        ModeInstant,
		CategoryApproval:           ModeInstant,
		CategoryBudget:             ModeDailyDigest,
		CategoryHealth:             ModeDailyDigest,
		CategoryOfflineSync:        ModeInstant,
		CategoryCalendar:           ModeInstant,
		CategoryAIGeneration:       ModeInstant,
		CategorySecurity:           ModeInstant,
		CategorySystem:             ModeInstant,
	},
	ChannelEmail: {
		CategoryCollaboration:      ModeInstant,
		CategoryComments:           ModeDailyDigest,
		CategoryRoleChanges:        ModeInstant,
		CategoryTripUpdates:        ModeDailyDigest,
		CategoryChecklist:          ModeDailyDigest,
		CategoryChecklistReminders: ModeDailyDigest,
		CategoryReminders:          ModeInstant,
		CategoryPreTripReminders:   ModeInstant,
		CategoryExpenses:           ModeDailyDigest,
		CategorySettlements:        ModeInstant,
		CategoryApproval:           ModeInstant,
		CategoryBudget:             ModeDailyDigest,
		CategoryHealth:             ModeDailyDigest,
		CategoryOfflineSync:        ModeInstant,
		CategoryCalendar:           ModeInstant,
		CategoryAIGeneration:       ModeDailyDigest,
		CategorySecurity:           ModeInstant,
		CategorySystem:             ModeMuted,
	},
	ChannelPush: {
		CategoryCollaboration:      ModeInstant,
		CategoryComments:           ModeMuted,
		CategoryRoleChanges:        ModeInstant,
		CategoryTripUpdates:        ModeMuted,
		CategoryChecklist:          ModeInstant,
		CategoryChecklistReminders: ModeInstant,
		CategoryReminders:          ModeInstant,
		CategoryPreTripReminders:   ModeInstant,
		CategoryExpenses:           ModeMuted,
		CategorySettlements:        ModeInstant,
		CategoryApproval:           ModeInstant,
		CategoryBudget:             ModeMuted,
		CategoryHealth:             ModeInstant,
		CategoryOfflineSync:        ModeInstant,
		CategoryCalendar:           ModeInstant,
		CategoryAIGeneration:       ModeInstant,
		CategorySecurity:           ModeInstant,
		CategorySystem:             ModeMuted,
	},
}

func defaultDeliveryMode(channel, category string) string {
	if byCategory, ok := defaultModeMatrix[channel]; ok {
		if mode, ok := byCategory[category]; ok {
			return mode
		}
	}
	return ModeMuted
}

func defaultEnabled(channel, category string) bool {
	return defaultDeliveryMode(channel, category) != ModeMuted
}

func defaultModeMatrixCopy() map[string]map[string]string {
	out := make(map[string]map[string]string, len(defaultModeMatrix))
	for channel, byCategory := range defaultModeMatrix {
		copied := make(map[string]string, len(byCategory))
		for category, mode := range byCategory {
			copied[category] = mode
		}
		out[channel] = copied
	}
	return out
}
