package availability

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
	providerName := strings.ToLower(strings.TrimSpace(cfg.Availability.Provider))
	if providerName == "" {
		providerName = config.AvailabilityProviderMock
	}
	if cfg.IsProduction() && providerName == config.AvailabilityProviderMock {
		log.Warn("mock availability provider enabled in production",
			zap.String("provider", providerName),
			zap.Bool("realAvailabilityEnabled", false),
		)
	}

	provider, err := selectProvider(providerName, cfg.Availability, log)
	if err != nil {
		return nil, err
	}

	provider = newGuardedProvider(
		guard,
		providerName,
		provider,
		NewMockAvailabilityProvider(),
		cfg.Availability.FallbackToMock,
		log,
	)

	if cfg.Availability.CacheEnabled {
		ttl := time.Duration(cfg.Availability.CacheTTLSeconds) * time.Second
		if ttl <= 0 {
			ttl = 15 * time.Minute
		}
		provider = newCachingProvider(providerName, provider, cache.New(availabilityCacheMaxEntries), ttl, log)
	}

	return NewService(provider, log, cfg.Availability.Enabled), nil
}

func selectProvider(provider string, cfg config.AvailabilityConfig, log *zap.Logger) (AvailabilityProvider, error) {
	switch provider {
	case config.AvailabilityProviderMock:
		log.Info("availability provider configured", zap.String("provider", config.AvailabilityProviderMock))
		return NewMockAvailabilityProvider(), nil
	case config.AvailabilityProviderGetYourGuide, config.AvailabilityProviderViator, config.AvailabilityProviderTiqets:
		primary := &unconfiguredRealProvider{name: provider}
		if cfg.FallbackToMock {
			return newFallbackProvider(provider, primary, NewMockAvailabilityProvider(), log), nil
		}
		return primary, nil
	default:
		return nil, fmt.Errorf("unsupported AVAILABILITY_PROVIDER %q: supported providers: mock, getyourguide, viator, tiqets", cfg.Provider)
	}
}

type unconfiguredRealProvider struct {
	name string
}

func (p *unconfiguredRealProvider) Name() string { return p.name }

func (p *unconfiguredRealProvider) DisplayName() string {
	switch p.name {
	case config.AvailabilityProviderGetYourGuide:
		return "GetYourGuide"
	case config.AvailabilityProviderViator:
		return "Viator"
	case config.AvailabilityProviderTiqets:
		return "Tiqets"
	default:
		return p.name
	}
}

func (p *unconfiguredRealProvider) SearchAvailability(context.Context, AvailabilitySearchRequest) (*AvailabilitySearchResult, error) {
	return nil, &ProviderError{Provider: p.name, Kind: providerErrorUnavailable}
}
