package availability

import "context"

type AvailabilityProvider interface {
	Name() string
	SearchAvailability(ctx context.Context, req AvailabilitySearchRequest) (*AvailabilitySearchResult, error)
}

type displayNameProvider interface {
	DisplayName() string
}

func providerDisplayName(provider AvailabilityProvider) string {
	if named, ok := provider.(displayNameProvider); ok {
		return named.DisplayName()
	}
	return provider.Name()
}

// itemSupportChecker lets a provider declare which item types it can serve. The
// service consults it *before* the cache/quota/rate-limit chain so unsupported
// item types (e.g. a museum for the events-only Ticketmaster adapter, or a rest
// stop) never consume provider quota. Decorators forward the call to their inner
// provider so the check reaches the real adapter through the whole chain.
type itemSupportChecker interface {
	SupportsItem(item AvailabilityItem) bool
}

// providerSupportsItem reports whether the provider can serve the item. Providers
// that do not implement itemSupportChecker (mock, unconfigured placeholders) are
// treated as supporting everything, preserving existing behaviour.
func providerSupportsItem(provider AvailabilityProvider, item AvailabilityItem) bool {
	if checker, ok := provider.(itemSupportChecker); ok {
		return checker.SupportsItem(item)
	}
	return true
}
