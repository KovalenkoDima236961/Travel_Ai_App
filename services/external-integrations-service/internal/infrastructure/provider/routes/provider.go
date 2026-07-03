package routes

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/cache"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

// New selects the configured route provider adapter and wires the quota guard,
// fallback, and caching around it. Mock remains the default and fallback; ORS is
// opt-in. Unsupported providers fail fast at startup, mirroring the place
// provider. The decorator order is cache -> guard -> provider so cache hits
// never consume provider quota.
func New(cfg *config.Config, guard *providerlimits.Guard, log *zap.Logger) (service.RouteProvider, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.RouteProvider.Provider))
	if provider == "" {
		provider = config.RouteProviderMock
	}

	selected, err := selectRouteProvider(provider, cfg.RouteProvider, log)
	if err != nil {
		return nil, err
	}

	selected = newGuardedRouteProvider(guard, provider, selected, NewMockRouteProvider(), cfg.RouteProvider.FallbackToMock, log)

	if cfg.RouteProvider.CacheEnabled {
		ttl := time.Duration(cfg.RouteProvider.CacheTTLSeconds) * time.Second
		if ttl <= 0 {
			ttl = 6 * time.Hour
		}
		selected = newCachingRouteProvider(provider, selected, cache.New(routeCacheMaxEntries), ttl, log)
	}

	return selected, nil
}

func selectRouteProvider(provider string, cfg config.RouteProviderConfig, log *zap.Logger) (service.RouteProvider, error) {
	switch provider {
	case config.RouteProviderMock:
		log.Info("route provider configured", zap.String("provider", config.RouteProviderMock))
		return NewMockRouteProvider(), nil

	case config.RouteProviderORS:
		orsProvider, err := NewOpenRouteServiceProvider(cfg, log)
		if err != nil {
			if cfg.FallbackToMock {
				log.Warn("falling back to mock route provider",
					zap.String("provider", config.RouteProviderORS),
					zap.String("fallbackProvider", config.RouteProviderMock),
					zap.Bool("fallbackUsed", true),
					zap.String("errorType", providerErrorKind(err)),
					zap.Error(err),
				)
				return NewMockRouteProvider(), nil
			}
			return nil, err
		}

		log.Info("route provider configured",
			zap.String("provider", config.RouteProviderORS),
			zap.Bool("fallbackToMock", cfg.FallbackToMock),
		)
		if cfg.FallbackToMock {
			return newFallbackRouteProvider(config.RouteProviderORS, orsProvider, NewMockRouteProvider(), log), nil
		}
		return orsProvider, nil

	default:
		return nil, fmt.Errorf("unsupported ROUTE_PROVIDER %q: supported providers: mock, ors", cfg.Provider)
	}
}
