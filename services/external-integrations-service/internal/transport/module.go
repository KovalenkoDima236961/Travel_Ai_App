package transport

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	appservice "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/cache"
)

func New(cfg *config.Config, guard *providerlimits.Guard, routeProvider appservice.RouteProvider, log *zap.Logger) (*Service, error) {
	if log == nil {
		log = zap.NewNop()
	}
	providerName := strings.ToLower(strings.TrimSpace(cfg.TransportProvider.Provider))
	if providerName == "" {
		providerName = config.TransportProviderMock
	}
	mock := NewMockProvider(cfg.TransportProvider.MaxOptionsPerMode)
	provider, err := selectProvider(providerName, cfg.TransportProvider, routeProvider, mock, log)
	if err != nil {
		return nil, err
	}
	provider = newGuardedProvider(guard, providerName, provider, mock, cfg.TransportProvider.FallbackToMock, log)
	if cfg.TransportProvider.CacheEnabled {
		ttl := time.Duration(cfg.TransportProvider.CacheTTLSeconds) * time.Second
		if ttl <= 0 {
			ttl = time.Hour
		}
		provider = newCachingProvider(providerName, provider, cache.New(transportCacheMaxEntries), ttl, log)
	}
	timeout := time.Duration(cfg.TransportProvider.TimeoutSeconds) * time.Second
	return NewService(provider, timeout, log), nil
}

func selectProvider(
	providerName string,
	cfg config.TransportProviderConfig,
	routeProvider appservice.RouteProvider,
	mock *MockProvider,
	log *zap.Logger,
) (TransportProvider, error) {
	switch providerName {
	case config.TransportProviderMock:
		log.Info("transport provider configured", zap.String("provider", config.TransportProviderMock))
		return mock, nil
	case config.TransportProviderRouteEstimate:
		primary := NewRouteEstimateProvider(routeProvider, mock, cfg.FallbackToMock, cfg.MaxOptionsPerMode, log)
		if cfg.FallbackToMock {
			return newFallbackProvider(providerName, primary, mock, log), nil
		}
		return primary, nil
	case config.TransportProviderGTFSStatic,
		config.TransportProviderAmadeus,
		config.TransportProviderSkyscanner,
		config.TransportProviderRome2Rio,
		config.TransportProviderNationalRail,
		config.TransportProviderFerry,
		config.TransportProviderManual:
		primary := &unconfiguredRealProvider{name: providerName}
		if cfg.FallbackToMock {
			log.Warn("falling back to mock transport provider",
				zap.String("provider", providerName),
				zap.String("fallbackProvider", config.TransportProviderMock),
				zap.Bool("fallbackUsed", true),
			)
			return newFallbackProvider(providerName, primary, mock, log), nil
		}
		return primary, nil
	default:
		return nil, fmt.Errorf("unsupported TRANSPORT_PROVIDER %q: supported providers: mock, route_estimate, gtfs_static, amadeus, skyscanner, rome2rio, national_rail, ferry_provider, manual", cfg.Provider)
	}
}

type unconfiguredRealProvider struct {
	name string
}

func (p *unconfiguredRealProvider) SearchTransportOptions(context.Context, TransportSearchRequest) (TransportSearchResponse, error) {
	return TransportSearchResponse{}, &ProviderError{Provider: p.name, Kind: providerErrorUnavailable}
}
