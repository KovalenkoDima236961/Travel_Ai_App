package exchangerates

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/cache"
)

const exchangeRateCacheMaxEntries = 512

type cachingExchangeRateProvider struct {
	providerName string
	next         service.ExchangeRateProvider
	cache        *cache.TTLCache
	ttl          time.Duration
	log          *zap.Logger
}

func newCachingExchangeRateProvider(
	providerName string,
	next service.ExchangeRateProvider,
	store *cache.TTLCache,
	ttl time.Duration,
	log *zap.Logger,
) service.ExchangeRateProvider {
	if log == nil {
		log = zap.NewNop()
	}
	return &cachingExchangeRateProvider{
		providerName: strings.ToLower(strings.TrimSpace(providerName)),
		next:         next,
		cache:        store,
		ttl:          ttl,
		log:          log,
	}
}

func (p *cachingExchangeRateProvider) Latest(ctx context.Context, base string) (*entity.ExchangeRateTable, error) {
	base = normalizeCurrency(base)
	key := exchangeRateCacheKey(p.providerName, base)
	if cached, ok := p.cache.Get(key); ok {
		if table, ok := cached.(entity.ExchangeRateTable); ok {
			p.log.Info("exchange rate cache lookup",
				zap.String("endpoint", "exchange_rates"),
				zap.String("provider", p.providerName),
				zap.String("base", base),
				zap.Bool("cacheHit", true),
			)
			result := table
			result.Rates = cloneRates(table.Rates)
			return &result, nil
		}
	}

	table, err := p.next.Latest(ctx, base)
	if err != nil {
		return nil, err
	}
	p.log.Info("exchange rate cache lookup",
		zap.String("endpoint", "exchange_rates"),
		zap.String("provider", p.providerName),
		zap.String("base", base),
		zap.Bool("cacheHit", false),
	)
	if !table.FallbackUsed {
		cacheValue := *table
		cacheValue.Rates = cloneRates(table.Rates)
		p.cache.Set(key, cacheValue, p.ttl)
	}
	return table, nil
}

func (p *cachingExchangeRateProvider) Convert(ctx context.Context, amount float64, from string, to string) (*entity.CurrencyConversionResult, error) {
	from = normalizeCurrency(from)
	to = normalizeCurrency(to)
	if from == to {
		return identityConversion(amount, from, to), nil
	}
	table, err := p.Latest(ctx, from)
	if err != nil {
		return nil, err
	}
	rate, ok := table.Rates[to]
	if !ok {
		return nil, ErrUnsupportedCurrency
	}
	return &entity.CurrencyConversionResult{
		Provider:        table.Provider,
		From:            from,
		To:              to,
		Amount:          amount,
		ConvertedAmount: round2(amount * rate),
		Rate:            rate,
		AsOf:            table.AsOf,
		FallbackUsed:    table.FallbackUsed,
	}, nil
}

func exchangeRateCacheKey(provider string, base string) string {
	return fmt.Sprintf("exchange_rates:%s:%s", strings.ToLower(strings.TrimSpace(provider)), normalizeCurrency(base))
}

func cloneRates(in map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
