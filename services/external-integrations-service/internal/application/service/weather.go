package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

// WeatherProvider is implemented by each weather provider. v1 ships only a
// deterministic mock provider; real providers can be added behind this port.
type WeatherProvider interface {
	GetForecast(ctx context.Context, req entity.WeatherForecastRequest) (*entity.WeatherForecast, error)
}

// WeatherService contains weather forecast use cases over the configured
// provider. The handler owns transport validation.
type WeatherService struct {
	provider WeatherProvider
	log      *zap.Logger
}

func NewWeatherService(provider WeatherProvider, log *zap.Logger) *WeatherService {
	if log == nil {
		log = zap.NewNop()
	}
	return &WeatherService{provider: provider, log: log}
}

// GetForecast delegates to the provider and logs one structured line per
// forecast. No user payloads or auth material are logged.
func (s *WeatherService) GetForecast(ctx context.Context, req entity.WeatherForecastRequest) (*entity.WeatherForecast, error) {
	start := time.Now()

	forecast, err := s.provider.GetForecast(ctx, req)
	if err != nil {
		s.log.Warn("weather_forecast",
			zap.String("action", "weather_forecast"),
			zap.String("destination", req.Destination),
			zap.Int("days", req.Days),
			zap.Int64("durationMs", time.Since(start).Milliseconds()),
			zap.Bool("success", false),
			zap.Error(err),
		)
		return nil, err
	}

	s.log.Info("weather_forecast",
		zap.String("action", "weather_forecast"),
		zap.String("provider", forecast.Provider),
		zap.String("destination", forecast.Destination),
		zap.Int("days", len(forecast.Days)),
		zap.Int64("durationMs", time.Since(start).Milliseconds()),
		zap.Bool("success", true),
	)

	return forecast, nil
}
