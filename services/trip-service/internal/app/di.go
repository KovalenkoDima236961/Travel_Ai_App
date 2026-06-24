package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	httpserver "github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/handler"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/generator"
	triprepo "github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/placecontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/placeenrichment"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/users"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/closer"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/validation"
)

// container holds the wired dependencies. It is a small, explicit composition
// root — no DI framework — assembled in buildContainer.
type container struct {
	db     *postgres.DB
	router http.Handler
}

// buildContainer constructs and wires all dependencies in order:
// postgres (with auto-migrations) -> validator -> repository -> generator ->
// service -> handler -> router. Long-lived resources register themselves with
// the closer.
func buildContainer(ctx context.Context, cfg *config.Config, log *zap.Logger) (*container, error) {
	db, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}
	closer.Add("postgres", func(context.Context) error {
		db.Close()
		return nil
	})

	validator, err := validation.NewValidator(
		validation.BeforeNowTag(),
		validation.OriginTag(),
	)
	if err != nil {
		return nil, fmt.Errorf("init validator: %w", err)
	}

	repo := triprepo.New(db)
	gen, err := generator.NewItineraryGenerator(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("init itinerary generator: %w", err)
	}
	var userContextClient interface {
		GetUserContext(context.Context, string) (*usercontext.UserContext, error)
	}
	if cfg.UserContext.Enabled {
		userContextClient, err = usercontext.New(usercontext.Config{
			BaseURL:        cfg.UserContext.UserServiceURL,
			TimeoutSeconds: cfg.UserContext.TimeoutSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("init user context client: %w", err)
		}
	}
	var weatherContextClient interface {
		GetForecast(context.Context, string, string, int) (*weathercontext.WeatherForecast, error)
	}
	if cfg.WeatherContext.Enabled {
		weatherContextClient, err = weathercontext.New(weathercontext.Config{
			BaseURL:        cfg.WeatherContext.ExternalIntegrationsServiceURL,
			TimeoutSeconds: cfg.WeatherContext.TimeoutSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("init weather context client: %w", err)
		}
	}
	var placeEnrichmentSvc interface {
		EnrichItinerary(context.Context, placeenrichment.EnrichItineraryInput) (*placeenrichment.EnrichItineraryResult, error)
	}
	if cfg.PlaceEnrichment.Enabled {
		placeClient, err := placecontext.New(placecontext.Config{
			BaseURL:        cfg.PlaceEnrichment.ExternalIntegrationsServiceURL,
			TimeoutSeconds: cfg.PlaceEnrichment.TimeoutSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("init place context client: %w", err)
		}
		placeEnrichmentSvc = placeenrichment.New(placeClient, placeenrichment.Config{
			MinConfidence:     cfg.PlaceEnrichment.MinConfidence,
			MaxItems:          cfg.PlaceEnrichment.MaxItems,
			OverwriteExisting: cfg.PlaceEnrichment.OverwriteExisting,
			FailOpen:          cfg.PlaceEnrichment.FailOpen,
		})
	}
	userLookupClient, err := users.New(users.Config{
		BaseURL:        cfg.UserLookup.AuthServiceURL,
		TimeoutSeconds: cfg.UserLookup.TimeoutSeconds,
	})
	if err != nil {
		return nil, fmt.Errorf("init user lookup client: %w", err)
	}
	svc := service.New(repo, gen, log, service.WithUserContext(
		userContextClient,
		cfg.UserContext.Enabled,
		cfg.UserContext.FailOpen,
	), service.WithWeatherContext(
		weatherContextClient,
		cfg.WeatherContext.Enabled,
		cfg.WeatherContext.FailOpen,
	), service.WithPlaceEnrichment(
		placeEnrichmentSvc,
		cfg.PlaceEnrichment.Enabled,
		cfg.PlaceEnrichment.FailOpen,
	), service.WithPublicSharing(
		cfg.PublicSharing.Enabled,
		cfg.PublicSharing.PublicWebBaseURL,
		cfg.PublicSharing.ShareTokenBytes,
		cfg.PublicSharing.PublicShareAccessSecret,
		cfg.PublicSharing.PublicShareAccessTTLMinutes,
	), service.WithUserLookup(userLookupClient))
	tripHandler := handler.New(svc, validator, log)
	readinessHandler := httpserver.NewReadinessHandler(
		db,
		cfg.ItineraryGenerator.Mode,
		cfg.ItineraryGenerator.AIPlanningServiceURL,
		time.Duration(cfg.ItineraryGenerator.AIPlanningTimeoutSeconds)*time.Second,
		log,
	)
	router := httpserver.NewRouter(log, tripHandler, readinessHandler, cfg.CORS, cfg.Auth)

	return &container{
		db:     db,
		router: router,
	}, nil
}
