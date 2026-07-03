package preferences

// defaultMatrix is the per-channel, per-category default applied when a user has
// no stored override for a given (channel, category) pair.
//
// Rationale:
//   - In-app notifications can surface all useful activity, so every in-app
//     category is enabled by default.
//   - Email should avoid noisy trip updates by default, so trip_updates email is
//     off while the other email categories are on.
var defaultMatrix = map[string]map[string]bool{
	ChannelInApp: {
		CategoryCollaboration: true,
		CategoryComments:      true,
		CategoryRoleChanges:   true,
		CategoryTripUpdates:   true,
	},
	ChannelEmail: {
		CategoryCollaboration: true,
		CategoryComments:      true,
		CategoryRoleChanges:   true,
		CategoryTripUpdates:   false,
	},
	ChannelPush: {
		CategoryCollaboration: true,
		CategoryComments:      true,
		CategoryRoleChanges:   true,
		CategoryTripUpdates:   true,
	},
}

// defaultEnabled returns the default enabled state for a (channel, category)
// pair. An unrecognised pair defaults to false (defence in depth; callers
// validate the vocabulary before reaching here).
func defaultEnabled(channel, category string) bool {
	if byCategory, ok := defaultMatrix[channel]; ok {
		if enabled, ok := byCategory[category]; ok {
			return enabled
		}
	}
	return false
}
