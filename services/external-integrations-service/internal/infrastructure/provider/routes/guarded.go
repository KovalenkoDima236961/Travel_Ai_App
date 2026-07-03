package routes

import (
	"context"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

// guardedRouteProvider enforces the per-provider rate limit and daily quota
// before the real route provider is called. When limited it falls back to the
// mock provider (if fallback is enabled) or returns a controlled limit error.
// It is wrapped by the cache decorator, so cache hits never reach the guard and
// therefore never consume quota.
type guardedRouteProvider struct {
	guard         *providerlimits.Guard
	providerName  string
	next          service.RouteProvider
	fallback      service.RouteProvider
	allowFallback bool
	log           *zap.Logger
}

func newGuardedRouteProvider(
	guard *providerlimits.Guard,
	providerName string,
	next service.RouteProvider,
	fallback service.RouteProvider,
	allowFallback bool,
	log *zap.Logger,
) service.RouteProvider {
	if guard == nil {
		return next
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &guardedRouteProvider{
		guard:         guard,
		providerName:  providerName,
		next:          next,
		fallback:      fallback,
		allowFallback: allowFallback,
		log:           log,
	}
}

func (p *guardedRouteProvider) EstimateRoute(ctx context.Context, req entity.RouteEstimateRequest) (*entity.RouteEstimate, error) {
	decision, _ := p.guard.CheckAndReserve(ctx, providerlimits.ProviderCall{
		Provider:      p.providerName,
		Operation:     providerlimits.OpRouteEstimate,
		Cost:          1,
		AllowFallback: p.allowFallback,
	})
	if decision.Allowed {
		return p.next.EstimateRoute(ctx, req)
	}

	// Limited (rate limit, quota, or unavailable-with-fail-closed).
	if p.allowFallback && p.fallback != nil {
		estimate, err := p.fallback.EstimateRoute(ctx, req)
		if err != nil {
			return nil, providerlimits.LimitErrorFrom(decision)
		}
		p.guard.RecordFallback(ctx, p.providerName, providerlimits.OpRouteEstimate, decision.Reason)
		estimate.Provider = "mock"
		estimate.FallbackUsed = true
		return estimate, nil
	}
	return nil, providerlimits.LimitErrorFrom(decision)
}
