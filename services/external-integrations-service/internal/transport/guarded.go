package transport

import (
	"context"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

type guardedProvider struct {
	guard         *providerlimits.Guard
	providerName  string
	next          TransportProvider
	fallback      TransportProvider
	allowFallback bool
	log           *zap.Logger
}

func newGuardedProvider(
	guard *providerlimits.Guard,
	providerName string,
	next TransportProvider,
	fallback TransportProvider,
	allowFallback bool,
	log *zap.Logger,
) TransportProvider {
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

func (p *guardedProvider) SearchTransportOptions(ctx context.Context, req TransportSearchRequest) (TransportSearchResponse, error) {
	decision, _ := p.guard.CheckAndReserve(ctx, providerlimits.ProviderCall{
		Provider:      p.providerName,
		Operation:     providerlimits.OpTransportSearch,
		Cost:          1,
		AllowFallback: p.allowFallback,
	})
	if decision.Allowed {
		return p.next.SearchTransportOptions(ctx, req)
	}
	if p.allowFallback && p.fallback != nil {
		result, err := p.fallback.SearchTransportOptions(ctx, req)
		if err != nil {
			return TransportSearchResponse{}, providerlimits.LimitErrorFrom(decision)
		}
		p.guard.RecordFallback(ctx, p.providerName, providerlimits.OpTransportSearch, decision.Reason)
		result.Summary.Provider = ProviderMock
		result.Summary.FallbackUsed = true
		return result, nil
	}
	return TransportSearchResponse{}, providerlimits.LimitErrorFrom(decision)
}
