package weather

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/cache"
)

type countingWeatherProvider struct {
	calls    int
	forecast entity.WeatherForecast
}

func (p *countingWeatherProvider) GetForecast(context.Context, entity.WeatherForecastRequest) (*entity.WeatherForecast, error) {
	p.calls++
	result := p.forecast
	return &result, nil
}

func weatherCacheReq() entity.WeatherForecastRequest {
	return entity.WeatherForecastRequest{
		Destination: "Rome",
		StartDate:   time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC),
		Days:        3,
	}
}

func TestCachingWeatherProviderHitAvoidsSecondCall(t *testing.T) {
	counter := &countingWeatherProvider{forecast: entity.WeatherForecast{Provider: "openweathermap", Destination: "Rome"}}
	provider := newCachingWeatherProvider("openweathermap", "metric", counter, cache.New(0), time.Minute, zap.NewNop())

	if _, err := provider.GetForecast(context.Background(), weatherCacheReq()); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := provider.GetForecast(context.Background(), weatherCacheReq()); err != nil {
		t.Fatalf("second call: %v", err)
	}

	if counter.calls != 1 {
		t.Fatalf("expected underlying provider called once, got %d", counter.calls)
	}
}

func TestCachingWeatherProviderDoesNotCacheFallbackResults(t *testing.T) {
	counter := &countingWeatherProvider{forecast: entity.WeatherForecast{Provider: "mock", Destination: "Rome", FallbackUsed: true}}
	provider := newCachingWeatherProvider("openweathermap", "metric", counter, cache.New(0), time.Minute, zap.NewNop())

	_, _ = provider.GetForecast(context.Background(), weatherCacheReq())
	_, _ = provider.GetForecast(context.Background(), weatherCacheReq())

	if counter.calls != 2 {
		t.Fatalf("expected fallback results not cached, got %d calls", counter.calls)
	}
}

func TestWeatherCacheKeyStable(t *testing.T) {
	want := "weather:openweathermap:rome:2026-08-10:3:metric"
	if got := weatherCacheKey("openweathermap", "metric", weatherCacheReq()); got != want {
		t.Fatalf("unexpected cache key:\n got %q\nwant %q", got, want)
	}
}
