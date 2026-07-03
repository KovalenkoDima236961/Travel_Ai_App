package prices

import (
	"context"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

// guardedProvider enforces the per-provider rate limit and daily quota for
// price_estimate before the real provider is called. It sits below the cache
// decorator so cache hits never consume quota. The mock price provider is
// unlimited by default, so this guard is a no-op cost path until a real price
// provider is configured.
type guardedProvider struct {
	guard         *providerlimits.Guard
	providerName  string
	next          PriceProvider
	fallback      PriceProvider
	allowFallback bool
	log           *zap.Logger
}

func newGuardedProvider(
	guard *providerlimits.Guard,
	providerName string,
	next PriceProvider,
	fallback PriceProvider,
	allowFallback bool,
	log *zap.Logger,
) PriceProvider {
	if guard == nil {
		return next
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &guardedProvider{
		guard:         guard,
		providerName:  providerName,
		next:          next,
		fallback:      fallback,
		allowFallback: allowFallback,
		log:           log,
	}
}

func (p *guardedProvider) EstimatePrice(ctx context.Context, input PriceEstimateInput) (*PriceEstimateResult, error) {
	decision, _ := p.guard.CheckAndReserve(ctx, providerlimits.ProviderCall{
		Provider:      p.providerName,
		Operation:     providerlimits.OpPriceEstimate,
		Cost:          1,
		AllowFallback: p.allowFallback,
	})
	if decision.Allowed {
		return p.next.EstimatePrice(ctx, input)
	}
	if p.allowFallback && p.fallback != nil {
		result, err := p.fallback.EstimatePrice(ctx, input)
		if err != nil {
			return nil, providerlimits.LimitErrorFrom(decision)
		}
		p.guard.RecordFallback(ctx, p.providerName, providerlimits.OpPriceEstimate, decision.Reason)
		result.Provider = "mock"
		result.FallbackUsed = true
		return result, nil
	}
	return nil, providerlimits.LimitErrorFrom(decision)
}
