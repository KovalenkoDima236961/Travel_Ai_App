package preferences

// MaxUpdateItems caps how many preference items a single update request may
// carry. Three channels times four categories is twelve; the cap leaves headroom
// while bounding the request.
const MaxUpdateItems = 60

// PreferenceInput is one (channel, category, enabled) triple in an update
// request, already mapped from transport. Channel/category are validated by the
// service against the known vocabulary.
type PreferenceInput struct {
	Channel      string
	Category     string
	Enabled      bool
	DeliveryMode string
}

// PreferenceItem is one entry in the effective preference matrix returned to the
// caller. The HTTP layer maps it to the JSON response shape.
type PreferenceItem struct {
	Channel      string
	Category     string
	Enabled      bool
	DeliveryMode string
}

// PreferencesResult is the full effective preference matrix for a user. It
// always contains len(AllChannels) * len(AllCategories) items (12 in v1), in a
// stable channel-then-category order.
type PreferencesResult struct {
	Items    []PreferenceItem
	Settings NotificationSettings
}

type NotificationSettings struct {
	QuietHoursEnabled        bool
	QuietHoursStart          string
	QuietHoursEnd            string
	QuietHoursTimezone       string
	UrgentBypassesQuietHours bool
	DailyDigestTime          string
	WeeklyDigestDay          int
	WeeklyDigestTime         string
}

func DefaultSettings() NotificationSettings {
	return NotificationSettings{
		QuietHoursStart: "22:00", QuietHoursEnd: "08:00", QuietHoursTimezone: "UTC",
		UrgentBypassesQuietHours: true, DailyDigestTime: "08:00", WeeklyDigestDay: 1,
		WeeklyDigestTime: "08:00",
	}
}
