package places

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

// New selects the configured place provider adapter and wraps it with the quota
// guard. When the active provider is limited the guard falls back to mock (if
// enabled) or returns a controlled limit error.
func New(cfg *config.Config, guard *providerlimits.Guard, log *zap.Logger) (service.PlaceProvider, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.PlaceProvider.Provider))
	if provider == "" {
		provider = config.PlaceProviderMock
	}

	selected, err := selectPlaceProvider(provider, cfg, log)
	if err != nil {
		return nil, err
	}
	return newGuardedPlaceProvider(guard, provider, selected, NewMockPlaceProvider(), cfg.PlaceProvider.FallbackToMock, log), nil
}

func selectPlaceProvider(provider string, cfg *config.Config, log *zap.Logger) (service.PlaceProvider, error) {
	switch provider {
	case config.PlaceProviderMock:
		log.Info("place provider configured", zap.String("provider", config.PlaceProviderMock))
		return NewMockPlaceProvider(), nil
	case config.PlaceProviderFoursquare:
		foursquareProvider, err := NewFoursquarePlaceProvider(cfg.PlaceProvider, log)
		if err != nil {
			if cfg.PlaceProvider.FallbackToMock {
				log.Warn("falling back to mock place provider",
					zap.String("provider", config.PlaceProviderFoursquare),
					zap.String("fallbackProvider", config.PlaceProviderMock),
					zap.Bool("fallbackUsed", true),
					zap.String("errorType", providerErrorKind(err)),
					zap.Error(err),
				)
				return NewMockPlaceProvider(), nil
			}
			return nil, err
		}

		log.Info("place provider configured",
			zap.String("provider", config.PlaceProviderFoursquare),
			zap.Bool("fallbackToMock", cfg.PlaceProvider.FallbackToMock),
		)
		if cfg.PlaceProvider.FallbackToMock {
			return newFallbackPlaceProvider(
				config.PlaceProviderFoursquare,
				foursquareProvider,
				NewMockPlaceProvider(),
				log,
			), nil
		}
		return foursquareProvider, nil
	default:
		return nil, fmt.Errorf("unsupported PLACE_PROVIDER %q: supported providers: mock, foursquare", cfg.PlaceProvider.Provider)
	}
}
