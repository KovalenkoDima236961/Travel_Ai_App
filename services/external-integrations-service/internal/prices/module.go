package prices

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/cache"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

func New(cfg *config.Config, guard *providerlimits.Guard, log *zap.Logger) (*Service, error) {
	if log == nil {
		log = zap.NewNop()
	}
	providerName := strings.ToLower(strings.TrimSpace(cfg.PriceProvider.Provider))
	if providerName == "" {
		providerName = config.PriceProviderMock
	}

	provider, err := selectProvider(providerName, cfg.PriceProvider, log)
	if err != nil {
		return nil, err
	}

	provider = newGuardedProvider(guard, providerName, provider, NewMockPriceProvider(), cfg.PriceProvider.FallbackToMock, log)

	if cfg.PriceProvider.CacheEnabled {
		ttl := time.Duration(cfg.PriceProvider.CacheTTLSeconds) * time.Second
		if ttl <= 0 {
			ttl = 24 * time.Hour
		}
		provider = newCachingProvider(providerName, provider, cache.New(priceCacheMaxEntries), ttl, log)
	}

	return NewService(provider, log), nil
}

func selectProvider(provider string, cfg config.PriceProviderConfig, log *zap.Logger) (PriceProvider, error) {
	switch provider {
	case config.PriceProviderMock:
		log.Info("price provider configured", zap.String("provider", config.PriceProviderMock))
		return NewMockPriceProvider(), nil
	case config.PriceProviderAPI:
		primary := &unconfiguredRealProvider{name: provider}
		if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.BaseURL) == "" {
			if cfg.FallbackToMock {
				log.Warn("falling back to mock price provider",
					zap.String("provider", provider),
					zap.String("fallbackProvider", config.PriceProviderMock),
					zap.Bool("fallbackUsed", true),
					zap.String("errorType", providerErrorUnavailable),
				)
				return newFallbackProvider(provider, primary, NewMockPriceProvider(), log), nil
			}
			return nil, fmt.Errorf("PRICE_API_BASE_URL and PRICE_API_KEY are required when PRICE_PROVIDER=api and fallback is disabled")
		}
		if cfg.FallbackToMock {
			return newFallbackProvider(provider, primary, NewMockPriceProvider(), log), nil
		}
		return primary, nil
	default:
		return nil, fmt.Errorf("unsupported PRICE_PROVIDER %q: supported providers: mock, api", cfg.Provider)
	}
}

type unconfiguredRealProvider struct {
	name string
}

func (p *unconfiguredRealProvider) EstimatePrice(context.Context, PriceEstimateInput) (*PriceEstimateResult, error) {
	return nil, &ProviderError{Provider: p.name, Kind: providerErrorUnavailable}
}
