package weather

import (
	"context"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

// fallbackWeatherProvider tries the primary (real) provider and, on failure,
// falls back to the mock provider so local development and transient outages or
// out-of-range trip dates keep working. It mirrors the place-provider model.
type fallbackWeatherProvider struct {
	providerName         string
	fallbackProviderName string
	primary              service.WeatherProvider
	fallback             service.WeatherProvider
	log                  *zap.Logger
}

func newFallbackWeatherProvider(
	providerName string,
	primary service.WeatherProvider,
	fallback service.WeatherProvider,
	log *zap.Logger,
) service.WeatherProvider {
	if log == nil {
		log = zap.NewNop()
	}
	return &fallbackWeatherProvider{
		providerName:         providerName,
		fallbackProviderName: "mock",
		primary:              primary,
		fallback:             fallback,
		log:                  log,
	}
}

func (p *fallbackWeatherProvider) GetForecast(ctx context.Context, req entity.WeatherForecastRequest) (*entity.WeatherForecast, error) {
	forecast, err := p.primary.GetForecast(ctx, req)
	if err == nil {
		return forecast, nil
	}

	p.log.Warn("weather provider fallback used",
		zap.String("action", "weather_forecast"),
		zap.String("provider", p.providerName),
		zap.String("fallbackProvider", p.fallbackProviderName),
		zap.Bool("fallbackUsed", true),
		zap.String("destination", req.Destination),
		zap.String("errorType", providerErrorKind(err)),
		zap.Error(err),
	)

	fallbackForecast, fallbackErr := p.fallback.GetForecast(ctx, req)
	if fallbackErr != nil {
		p.log.Warn("weather provider fallback failed",
			zap.String("action", "weather_forecast"),
			zap.String("provider", p.providerName),
			zap.String("fallbackProvider", p.fallbackProviderName),
			zap.String("errorType", providerErrorKind(fallbackErr)),
			zap.Error(fallbackErr),
		)
		// Return the original provider error so the handler reports a safe
		// provider-unavailable response.
		return nil, err
	}

	fallbackForecast.Provider = p.fallbackProviderName
	fallbackForecast.FallbackUsed = true
	return fallbackForecast, nil
}
