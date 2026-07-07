package places

import (
	"context"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

// guardedPlaceProvider enforces the per-provider rate limit and daily quota for
// place_search and place_details before the real provider is called. When
// limited it falls back to mock (if enabled) or returns a controlled limit
// error. It is wrapped by the cache decorator, so cache hits never consume
// quota.
type guardedPlaceProvider struct {
	guard         *providerlimits.Guard
	providerName  string
	next          service.PlaceProvider
	fallback      service.PlaceProvider
	allowFallback bool
	log           *zap.Logger
}

func newGuardedPlaceProvider(
	guard *providerlimits.Guard,
	providerName string,
	next service.PlaceProvider,
	fallback service.PlaceProvider,
	allowFallback bool,
	log *zap.Logger,
) service.PlaceProvider {
	if guard == nil {
		return next
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &guardedPlaceProvider{
		guard:         guard,
		providerName:  providerName,
		next:          next,
		fallback:      fallback,
		allowFallback: allowFallback,
		log:           log,
	}
}

func (p *guardedPlaceProvider) SearchPlaces(ctx context.Context, query string, destination string) ([]entity.Place, error) {
	decision, _ := p.guard.CheckAndReserve(ctx, providerlimits.ProviderCall{
		Provider:      p.providerName,
		Operation:     providerlimits.OpPlaceSearch,
		Cost:          1,
		AllowFallback: p.allowFallback,
	})
	if decision.Allowed {
		return p.next.SearchPlaces(ctx, query, destination)
	}
	if p.allowFallback && p.fallback != nil {
		items, err := p.fallback.SearchPlaces(ctx, query, destination)
		if err != nil {
			return nil, providerlimits.LimitErrorFrom(decision)
		}
		p.guard.RecordFallback(ctx, p.providerName, providerlimits.OpPlaceSearch, decision.Reason)
		return items, nil
	}
	return nil, providerlimits.LimitErrorFrom(decision)
}

func (p *guardedPlaceProvider) GetPlaceDetails(ctx context.Context, providerPlaceID string) (*entity.Place, error) {
	decision, _ := p.guard.CheckAndReserve(ctx, providerlimits.ProviderCall{
		Provider:      p.providerName,
		Operation:     providerlimits.OpPlaceDetails,
		Cost:          1,
		AllowFallback: p.allowFallback,
	})
	if decision.Allowed {
		return p.next.GetPlaceDetails(ctx, providerPlaceID)
	}
	if p.allowFallback && p.fallback != nil {
		place, err := p.fallback.GetPlaceDetails(ctx, providerPlaceID)
		if err != nil {
			return nil, providerlimits.LimitErrorFrom(decision)
		}
		p.guard.RecordFallback(ctx, p.providerName, providerlimits.OpPlaceDetails, decision.Reason)
		return place, nil
	}
	return nil, providerlimits.LimitErrorFrom(decision)
}
