package weather

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
)

// New selects the configured weather provider adapter.
func New(cfg *config.Config, log *zap.Logger) (service.WeatherProvider, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.WeatherProvider.Provider))
	if provider == "" {
		provider = config.WeatherProviderMock
	}

	switch provider {
	case config.WeatherProviderMock:
		log.Info("weather provider configured", zap.String("provider", config.WeatherProviderMock))
		return NewMockWeatherProvider(), nil
	default:
		return nil, fmt.Errorf("unsupported WEATHER_PROVIDER %q: supported providers: mock", cfg.WeatherProvider.Provider)
	}
}
