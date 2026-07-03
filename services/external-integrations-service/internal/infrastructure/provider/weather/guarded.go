package weather

import (
	"context"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

// guardedWeatherProvider enforces the per-provider rate limit and daily quota
// for weather_forecast before the real provider is called. It sits below the
// cache decorator so cache hits never consume quota.
type guardedWeatherProvider struct {
	guard         *providerlimits.Guard
	providerName  string
	next          service.WeatherProvider
	fallback      service.WeatherProvider
	allowFallback bool
	log           *zap.Logger
}

func newGuardedWeatherProvider(
	guard *providerlimits.Guard,
	providerName string,
	next service.WeatherProvider,
	fallback service.WeatherProvider,
	allowFallback bool,
	log *zap.Logger,
) service.WeatherProvider {
	if guard == nil {
		return next
	}
	if log == nil {
		log = zap.NewNop()
	}
	return &guardedWeatherProvider{
		guard:         guard,
		providerName:  providerName,
		next:          next,
		fallback:      fallback,
		allowFallback: allowFallback,
		log:           log,
	}
}

func (p *guardedWeatherProvider) GetForecast(ctx context.Context, req entity.WeatherForecastRequest) (*entity.WeatherForecast, error) {
	decision, _ := p.guard.CheckAndReserve(ctx, providerlimits.ProviderCall{
		Provider:      p.providerName,
		Operation:     providerlimits.OpWeatherForecast,
		Cost:          1,
		AllowFallback: p.allowFallback,
	})
	if decision.Allowed {
		return p.next.GetForecast(ctx, req)
	}
	if p.allowFallback && p.fallback != nil {
		forecast, err := p.fallback.GetForecast(ctx, req)
		if err != nil {
			return nil, providerlimits.LimitErrorFrom(decision)
		}
		p.guard.RecordFallback(ctx, p.providerName, providerlimits.OpWeatherForecast, decision.Reason)
		forecast.Provider = "mock"
		forecast.FallbackUsed = true
		return forecast, nil
	}
	return nil, providerlimits.LimitErrorFrom(decision)
}
