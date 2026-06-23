package routes

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
)

// New selects the configured route provider adapter. Unsupported providers fail
// fast at startup with a clear, actionable error, mirroring the place provider.
func New(cfg *config.Config, log *zap.Logger) (service.RouteProvider, error) {
	provider := strings.ToLower(strings.TrimSpace(cfg.RouteProvider.Provider))
	if provider == "" {
		provider = config.RouteProviderMock
	}

	switch provider {
	case config.RouteProviderMock:
		log.Info("route provider configured", zap.String("provider", config.RouteProviderMock))
		return NewMockRouteProvider(), nil
	default:
		return nil, fmt.Errorf("unsupported ROUTE_PROVIDER %q: supported providers: mock", cfg.RouteProvider.Provider)
	}
}
