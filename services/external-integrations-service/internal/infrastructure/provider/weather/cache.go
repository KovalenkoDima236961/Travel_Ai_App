package weather

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/cache"
	extobs "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/observability"
)

// weatherCacheMaxEntries bounds the in-memory weather cache.
const weatherCacheMaxEntries = 2048

// cachingWeatherProvider memoises successful forecasts from the wrapped provider.
// A cache hit skips both the geocoding and forecast upstream calls. Only real,
// non-fallback results are cached; errors are never cached.
type cachingWeatherProvider struct {
	providerName string
	units        string
	next         service.WeatherProvider
	cache        *cache.TTLCache
	ttl          time.Duration
	log          *zap.Logger
}

func newCachingWeatherProvider(
	providerName string,
	units string,
	next service.WeatherProvider,
	store *cache.TTLCache,
	ttl time.Duration,
	log *zap.Logger,
) service.WeatherProvider {
	if log == nil {
		log = zap.NewNop()
	}
	return &cachingWeatherProvider{
		providerName: strings.ToLower(strings.TrimSpace(providerName)),
		units:        strings.ToLower(strings.TrimSpace(units)),
		next:         next,
		cache:        store,
		ttl:          ttl,
		log:          log,
	}
}

func (p *cachingWeatherProvider) GetForecast(ctx context.Context, req entity.WeatherForecastRequest) (*entity.WeatherForecast, error) {
	key := weatherCacheKey(p.providerName, p.units, req)

	if cached, ok := p.cache.Get(key); ok {
		if forecast, ok := cached.(entity.WeatherForecast); ok {
			extobs.RecordProviderCacheHit(p.providerName, "weather_forecast")
			p.log.Info("weather cache lookup",
				zap.String("endpoint", "weather"),
				zap.String("provider", p.providerName),
				zap.Bool("cacheHit", true),
			)
			result := forecast
			return &result, nil
		}
	}

	forecast, err := p.next.GetForecast(ctx, req)
	if err != nil {
		return nil, err
	}
	extobs.RecordProviderCacheMiss(p.providerName, "weather_forecast")

	p.log.Info("weather cache lookup",
		zap.String("endpoint", "weather"),
		zap.String("provider", p.providerName),
		zap.Bool("cacheHit", false),
	)

	if !forecast.FallbackUsed {
		p.cache.Set(key, *forecast, p.ttl)
	}
	return forecast, nil
}

// weatherCacheKey is provider + normalised destination + start date + days +
// units, e.g. weather:openweathermap:rome:2026-08-10:5:metric.
func weatherCacheKey(provider, units string, req entity.WeatherForecastRequest) string {
	return fmt.Sprintf(
		"weather:%s:%s:%s:%d:%s",
		provider,
		normalizeDestination(req.Destination),
		req.StartDate.Format("2006-01-02"),
		req.Days,
		units,
	)
}
