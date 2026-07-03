package exchangerates

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/cache"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

// New selects the configured exchange-rate provider adapter and wires the quota
// guard, fallback, and caching around it. Mock remains the default and fallback
// for local development. Real provider names are reserved for future adapters;
// with fallback enabled they degrade safely to mock. The decorator order is
// cache -> guard -> provider so cache hits never consume provider quota.
func New(cfg *config.Config, guard *providerlimits.Guard, log *zap.Logger) (service.ExchangeRateProvider, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.ExchangeRateProvider.Provider))
	if provider == "" {
		provider = config.ExchangeRateProviderMock
	}

	selected, err := selectExchangeRateProvider(provider, cfg.ExchangeRateProvider, log)
	if err != nil {
		return nil, err
	}

	selected = newGuardedExchangeRateProvider(guard, provider, selected, NewMockExchangeRateProvider(), cfg.ExchangeRateProvider.FallbackToMock, log)

	if cfg.ExchangeRateProvider.CacheEnabled {
		ttl := time.Duration(cfg.ExchangeRateProvider.CacheTTLSeconds) * time.Second
		if ttl <= 0 {
			ttl = 6 * time.Hour
		}
		selected = newCachingExchangeRateProvider(provider, selected, cache.New(exchangeRateCacheMaxEntries), ttl, log)
	}
	return selected, nil
}

func selectExchangeRateProvider(provider string, cfg config.ExchangeRateProviderConfig, log *zap.Logger) (service.ExchangeRateProvider, error) {
	switch provider {
	case config.ExchangeRateProviderMock:
		log.Info("exchange rate provider configured", zap.String("provider", config.ExchangeRateProviderMock))
		return NewMockExchangeRateProvider(), nil
	case config.ExchangeRateProviderHost, config.ExchangeRateProviderOpenExchangeRates, config.ExchangeRateProviderAPI:
		realProvider := &unconfiguredRealProvider{name: provider}
		if cfg.FallbackToMock {
			log.Info("exchange rate provider configured",
				zap.String("provider", provider),
				zap.Bool("fallbackToMock", true),
				zap.String("realProviderStatus", "not_configured"),
			)
			return newFallbackExchangeRateProvider(provider, realProvider, NewMockExchangeRateProvider(), log), nil
		}
		return realProvider, nil
	default:
		return nil, fmt.Errorf("unsupported EXCHANGE_RATE_PROVIDER %q: supported providers: mock, exchangerate_host, openexchangerates, exchangerate_api", cfg.Provider)
	}
}

type unconfiguredRealProvider struct {
	name string
}

func (p *unconfiguredRealProvider) Latest(context.Context, string) (*entity.ExchangeRateTable, error) {
	return nil, &ProviderError{Provider: p.name, Kind: providerErrorUnavailable}
}

func (p *unconfiguredRealProvider) Convert(context.Context, float64, string, string) (*entity.CurrencyConversionResult, error) {
	return nil, &ProviderError{Provider: p.name, Kind: providerErrorUnavailable}
}
