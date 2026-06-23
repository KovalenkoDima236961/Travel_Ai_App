package app

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	appservice "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	httpserver "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/http-server"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/http-server/handler"
	placeprovider "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/provider/places"
	routeprovider "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/provider/routes"
)

// container holds the wired dependencies. It is a small, explicit composition
// root, matching Auth Service, Trip Service, and User Service style.
type container struct {
	router http.Handler
}

// buildContainer constructs and wires dependencies in order:
// provider -> service -> handler -> router. There is no database in v1.
func buildContainer(_ context.Context, cfg *config.Config, log *zap.Logger) (*container, error) {
	provider, err := placeprovider.New(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("init place provider: %w", err)
	}

	routeProvider, err := routeprovider.New(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("init route provider: %w", err)
	}

	svc := appservice.New(provider, log)
	routesSvc := appservice.NewRoutesService(routeProvider, log)
	placesHandler := handler.NewPlacesHandler(svc, log, cfg.PlaceProvider.Provider)
	routesHandler := handler.NewRoutesHandler(routesSvc, log)
	readinessHandler := httpserver.NewReadinessHandler(log)
	router := httpserver.NewRouter(log, placesHandler, routesHandler, readinessHandler, cfg.CORS)

	return &container{router: router}, nil
}
