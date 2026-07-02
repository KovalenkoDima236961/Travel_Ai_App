package prices

import (
	"context"

	"go.uber.org/zap"
)

type fallbackProvider struct {
	providerName         string
	fallbackProviderName string
	primary              PriceProvider
	fallback             PriceProvider
	log                  *zap.Logger
}

func newFallbackProvider(providerName string, primary PriceProvider, fallback PriceProvider, log *zap.Logger) PriceProvider {
	if log == nil {
		log = zap.NewNop()
	}
	return &fallbackProvider{
		providerName:         providerName,
		fallbackProviderName: mockProviderName,
		primary:              primary,
		fallback:             fallback,
		log:                  log,
	}
}

func (p *fallbackProvider) EstimatePrice(ctx context.Context, input PriceEstimateInput) (*PriceEstimateResult, error) {
	result, err := p.primary.EstimatePrice(ctx, input)
	if err == nil {
		return result, nil
	}

	p.log.Warn("price provider fallback used",
		zap.String("action", "price_estimate"),
		zap.String("provider", p.providerName),
		zap.String("fallbackProvider", p.fallbackProviderName),
		zap.Bool("fallbackUsed", true),
		zap.String("destination", input.Destination),
		zap.String("errorType", providerErrorKind(err)),
		zap.Error(err),
	)

	fallbackResult, fallbackErr := p.fallback.EstimatePrice(ctx, input)
	if fallbackErr != nil {
		p.log.Warn("price provider fallback failed",
			zap.String("action", "price_estimate"),
			zap.String("provider", p.providerName),
			zap.String("fallbackProvider", p.fallbackProviderName),
			zap.String("errorType", providerErrorKind(fallbackErr)),
			zap.Error(fallbackErr),
		)
		return nil, err
	}
	if fallbackResult != nil {
		fallbackResult.Provider = p.fallbackProviderName
		fallbackResult.FallbackUsed = true
	}
	return fallbackResult, nil
}
