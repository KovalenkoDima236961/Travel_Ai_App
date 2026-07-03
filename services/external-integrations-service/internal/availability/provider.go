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
