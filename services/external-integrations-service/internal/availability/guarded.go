package availability

import (
	"context"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

type guardedProvider struct {
	guard         *providerlimits.Guard
	providerName  string
	next          AvailabilityProvider
	fallback      AvailabilityProvider
	allowFallback bool
	log           *zap.Logger
}

func newGuardedProvider(
	guard *providerlimits.Guard,
	providerName string,
	next AvailabilityProvider,
	fallback AvailabilityProvider,
	allowFallback bool,
	log *zap.Logger,
) AvailabilityProvider {
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

func (p *guardedProvider) Name() string { return p.next.Name() }

func (p *guardedProvider) DisplayName() string { return providerDisplayName(p.next) }

func (p *guardedProvider) SearchAvailability(ctx context.Context, req AvailabilitySearchRequest) (*AvailabilitySearchResult, error) {
	decision, _ := p.guard.CheckAndReserve(ctx, providerlimits.ProviderCall{
		Provider:      p.providerName,
		Operation:     providerlimits.OpAvailabilitySearch,
		Cost:          1,
		AllowFallback: p.allowFallback,
	})
	if decision.Allowed {
		return p.next.SearchAvailability(ctx, req)
	}
	if p.allowFallback && p.fallback != nil {
		result, err := p.fallback.SearchAvailability(ctx, req)
		if err != nil {
			return nil, providerlimits.LimitErrorFrom(decision)
		}
		p.guard.RecordFallback(ctx, p.providerName, providerlimits.OpAvailabilitySearch, decision.Reason)
		result.Provider = p.fallback.Name()
		result.ProviderDisplayName = providerDisplayName(p.fallback)
		result.FallbackUsed = true
		result.Result = ProviderResultFallback
		result.Warnings = append([]string{
			"Using estimated fallback availability. Confirm details on the booking website.",
		}, result.Warnings...)
		p.log.Info("availability_fallback_used",
			zap.String("provider", p.providerName),
			zap.String("fallbackProvider", p.fallback.Name()),
			zap.String("reason", decision.Reason),
		)
		return result, nil
	}
	return nil, providerlimits.LimitErrorFrom(decision)
}
