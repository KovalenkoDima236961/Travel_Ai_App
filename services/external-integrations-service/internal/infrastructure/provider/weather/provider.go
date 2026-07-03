package weather

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

// New selects the configured weather provider adapter and wires the quota guard,
// fallback, and caching around it. Mock remains the default and fallback;
// OpenWeatherMap is opt-in. Unsupported providers fail fast at startup. The
// decorator order is cache -> guard -> provider so cache hits never consume
// provider quota.
func New(cfg *config.Config, guard *providerlimits.Guard, log *zap.Logger) (service.WeatherProvider, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.WeatherProvider.Provider))
	if provider == "" {
		provider = config.WeatherProviderMock
	}

	selected, err := selectWeatherProvider(provider, cfg.WeatherProvider, log)
	if err != nil {
		return nil, err
	}

	selected = newGuardedWeatherProvider(guard, provider, selected, NewMockWeatherProvider(), cfg.WeatherProvider.FallbackToMock, log)

	if cfg.WeatherProvider.CacheEnabled {
		ttl := time.Duration(cfg.WeatherProvider.CacheTTLSeconds) * time.Second
		if ttl <= 0 {
			ttl = time.Hour
		}
		units := strings.ToLower(strings.TrimSpace(cfg.WeatherProvider.OpenWeatherUnits))
		if units == "" {
			units = openWeatherDefaultUnits
		}
		selected = newCachingWeatherProvider(provider, units, selected, cache.New(weatherCacheMaxEntries), ttl, log)
	}

	return selected, nil
}

func selectWeatherProvider(provider string, cfg config.WeatherProviderConfig, log *zap.Logger) (service.WeatherProvider, error) {
	switch provider {
	case config.WeatherProviderMock:
		log.Info("weather provider configured", zap.String("provider", config.WeatherProviderMock))
		return NewMockWeatherProvider(), nil

	case config.WeatherProviderOpenWeather:
		openWeatherProvider, err := NewOpenWeatherProvider(cfg, log)
		if err != nil {
			if cfg.FallbackToMock {
				log.Warn("falling back to mock weather provider",
					zap.String("provider", config.WeatherProviderOpenWeather),
					zap.String("fallbackProvider", config.WeatherProviderMock),
					zap.Bool("fallbackUsed", true),
					zap.String("errorType", providerErrorKind(err)),
					zap.Error(err),
				)
				return NewMockWeatherProvider(), nil
			}
			return nil, err
		}

		log.Info("weather provider configured",
			zap.String("provider", config.WeatherProviderOpenWeather),
			zap.Bool("fallbackToMock", cfg.FallbackToMock),
		)
		if cfg.FallbackToMock {
			return newFallbackWeatherProvider(config.WeatherProviderOpenWeather, openWeatherProvider, NewMockWeatherProvider(), log), nil
		}
		return openWeatherProvider, nil

	default:
		return nil, fmt.Errorf("unsupported WEATHER_PROVIDER %q: supported providers: mock, openweathermap", cfg.Provider)
	}
}
