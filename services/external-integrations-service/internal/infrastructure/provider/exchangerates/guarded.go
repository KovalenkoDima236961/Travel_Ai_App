package exchangerates

import (
	"context"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

// guardedExchangeRateProvider enforces the per-provider rate limit and daily
// quota for exchange_rate_latest and exchange_rate_convert. It sits below the
// cache decorator so cache hits never consume quota.
type guardedExchangeRateProvider struct {
	guard         *providerlimits.Guard
	providerName  string
	next          service.ExchangeRateProvider
	fallback      service.ExchangeRateProvider
	allowFallback bool
	log           *zap.Logger
}

func newGuardedExchangeRateProvider(
	guard *providerlimits.Guard,
	providerName string,
	next service.ExchangeRateProvider,
	fallback service.ExchangeRateProvider,
	allowFallback bool,
	log *zap.Logger,
) service.ExchangeRateProvider {
	if guard == nil {
		return next
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &guardedExchangeRateProvider{
		guard:         guard,
		providerName:  providerName,
		next:          next,
		fallback:      fallback,
		allowFallback: allowFallback,
		log:           log,
	}
}

func (p *guardedExchangeRateProvider) Latest(ctx context.Context, base string) (*entity.ExchangeRateTable, error) {
	decision, _ := p.guard.CheckAndReserve(ctx, providerlimits.ProviderCall{
		Provider:      p.providerName,
		Operation:     providerlimits.OpExchangeRateLatest,
		Cost:          1,
		AllowFallback: p.allowFallback,
	})
	if decision.Allowed {
		return p.next.Latest(ctx, base)
	}
	if p.allowFallback && p.fallback != nil {
		table, err := p.fallback.Latest(ctx, base)
		if err != nil {
			return nil, providerlimits.LimitErrorFrom(decision)
		}
		p.guard.RecordFallback(ctx, p.providerName, providerlimits.OpExchangeRateLatest, decision.Reason)
		table.Provider = "mock"
		table.FallbackUsed = true
		return table, nil
	}
	return nil, providerlimits.LimitErrorFrom(decision)
}

func (p *guardedExchangeRateProvider) Convert(ctx context.Context, amount float64, from string, to string) (*entity.CurrencyConversionResult, error) {
	decision, _ := p.guard.CheckAndReserve(ctx, providerlimits.ProviderCall{
		Provider:      p.providerName,
		Operation:     providerlimits.OpExchangeRateConvert,
		Cost:          1,
		AllowFallback: p.allowFallback,
	})
	if decision.Allowed {
		return p.next.Convert(ctx, amount, from, to)
	}
	if p.allowFallback && p.fallback != nil {
		result, err := p.fallback.Convert(ctx, amount, from, to)
		if err != nil {
			return nil, providerlimits.LimitErrorFrom(decision)
		}
		p.guard.RecordFallback(ctx, p.providerName, providerlimits.OpExchangeRateConvert, decision.Reason)
		result.Provider = "mock"
		result.FallbackUsed = true
		return result, nil
	}
	return nil, providerlimits.LimitErrorFrom(decision)
}
