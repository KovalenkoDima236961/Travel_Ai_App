package preferences

// MaxUpdateItems caps how many preference items a single update request may
// carry. Three channels times four categories is twelve; the cap leaves headroom
// while bounding the request.
const MaxUpdateItems = 20

// PreferenceInput is one (channel, category, enabled) triple in an update
// request, already mapped from transport. Channel/category are validated by the
// service against the known vocabulary.
type PreferenceInput struct {
	Channel  string
	Category string
	Enabled  bool
}

// PreferenceItem is one entry in the effective preference matrix returned to the
// caller. The HTTP layer maps it to the JSON response shape.
type PreferenceItem struct {
	Channel  string
	Category string
	Enabled  bool
}

// PreferencesResult is the full effective preference matrix for a user. It
// always contains len(AllChannels) * len(AllCategories) items (12 in v1), in a
// stable channel-then-category order.
type PreferencesResult struct {
	Items []PreferenceItem
}
