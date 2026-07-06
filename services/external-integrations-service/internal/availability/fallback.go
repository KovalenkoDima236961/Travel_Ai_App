package availability

import (
	"context"

	"go.uber.org/zap"
)

type fallbackProvider struct {
	providerName         string
	fallbackProviderName string
	primary              AvailabilityProvider
	fallback             AvailabilityProvider
	log                  *zap.Logger
}

func newFallbackProvider(providerName string, primary AvailabilityProvider, fallback AvailabilityProvider, log *zap.Logger) AvailabilityProvider {
	if log == nil {
		log = zap.NewNop()
	}
	return &fallbackProvider{
		providerName:         providerName,
		fallbackProviderName: fallback.Name(),
		primary:              primary,
		fallback:             fallback,
		log:                  log,
	}
}

func (p *fallbackProvider) Name() string { return p.primary.Name() }

func (p *fallbackProvider) DisplayName() string { return providerDisplayName(p.primary) }

func (p *fallbackProvider) SupportsItem(item AvailabilityItem) bool {
	return providerSupportsItem(p.primary, item)
}

func (p *fallbackProvider) SearchAvailability(ctx context.Context, req AvailabilitySearchRequest) (*AvailabilitySearchResult, error) {
	result, err := p.primary.SearchAvailability(ctx, req)
	if err == nil {
		return result, nil
	}

	p.log.Warn("availability provider fallback used",
		zap.String("operation", availabilityOperation),
		zap.String("provider", p.providerName),
		zap.String("fallbackProvider", p.fallbackProviderName),
		zap.Bool("fallbackUsed", true),
		zap.String("errorType", providerErrorKind(err)),
		zap.Error(err),
	)

	fallbackResult, fallbackErr := p.fallback.SearchAvailability(ctx, req)
	if fallbackErr != nil {
		p.log.Warn("availability provider fallback failed",
			zap.String("operation", availabilityOperation),
			zap.String("provider", p.providerName),
			zap.String("fallbackProvider", p.fallbackProviderName),
			zap.String("errorType", providerErrorKind(fallbackErr)),
			zap.Error(fallbackErr),
		)
		return nil, err
	}
	if fallbackResult != nil {
		fallbackResult.Provider = p.fallbackProviderName
		fallbackResult.ProviderDisplayName = providerDisplayName(p.fallback)
		fallbackResult.FallbackUsed = true
		fallbackResult.Result = ProviderResultFallback
		fallbackResult.Warnings = append([]string{
			"Using estimated fallback availability. Confirm details on the booking website.",
		}, fallbackResult.Warnings...)
	}
	return fallbackResult, nil
}
