package places

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
)

// New selects the configured place provider adapter.
func New(cfg *config.Config, log *zap.Logger) (service.PlaceProvider, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.PlaceProvider.Provider))
	if provider == "" {
		provider = config.PlaceProviderMock
	}

	switch provider {
	case config.PlaceProviderMock:
		log.Info("place provider configured", zap.String("provider", config.PlaceProviderMock))
		return NewMockPlaceProvider(), nil
	default:
		return nil, fmt.Errorf("unsupported PLACE_PROVIDER %q: supported providers: mock", cfg.PlaceProvider.Provider)
	}
}
