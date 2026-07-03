package app

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	appservice "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/calendar"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	tokencrypto "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/crypto"
	httpserver "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/http-server"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/http-server/handler"
	exchangerateprovider "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/provider/exchangerates"
	placeprovider "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/provider/places"
	routeprovider "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/provider/routes"
	weatherprovider "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/provider/weather"
	calendarrepo "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/repository/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/prices"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/closer"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/storage/postgres"
)

// container holds the wired dependencies. It is a small, explicit composition
// root, matching Auth Service, Trip Service, and User Service style.
type container struct {
	db     *postgres.DB
	router http.Handler
}

// buildContainer constructs and wires dependencies in order:
// provider -> service -> handler -> router. There is no database in v1.
func buildContainer(_ context.Context, cfg *config.Config, log *zap.Logger) (*container, error) {
	ctx := context.Background()
	db, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		return nil, fmt.Errorf("init postgres: %w", err)
	}
	closer.Add("postgres", func(context.Context) error {
		db.Close()
		return nil
	})

	provider, err := placeprovider.New(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("init place provider: %w", err)
	}

	routeProvider, err := routeprovider.New(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("init route provider: %w", err)
	}

	weatherProvider, err := weatherprovider.New(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("init weather provider: %w", err)
	}
	exchangeRateProvider, err := exchangerateprovider.New(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("init exchange rate provider: %w", err)
	}
	priceSvc, err := prices.New(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("init price provider: %w", err)
	}

	svc := appservice.New(provider, log)
	routesSvc := appservice.NewRoutesService(routeProvider, log)
	weatherSvc := appservice.NewWeatherService(weatherProvider, log)
	exchangeRateSvc := appservice.NewExchangeRateService(exchangeRateProvider, log)
	placesHandler := handler.NewPlacesHandler(svc, log, cfg.PlaceProvider.Provider)
	routesHandler := handler.NewRoutesHandler(routesSvc, log, cfg.RouteProvider.Provider)
	weatherHandler := handler.NewWeatherHandler(weatherSvc, log)
	exchangeRateHandler := handler.NewExchangeRateHandler(exchangeRateSvc, log)
	priceHandler := prices.NewHandler(priceSvc, log, cfg.PriceProvider.DefaultCurrency)
	cipher, err := tokencrypto.NewStringCipher(cfg.Calendar.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("init calendar token encryption: %w", err)
	}
	var calendarProvider calendar.CalendarProvider
	switch cfg.Calendar.Provider {
	case config.CalendarProviderGoogle:
		calendarProvider = calendar.NewGoogleCalendarProvider(cfg.Calendar)
	default:
		calendarProvider = calendar.NewMockCalendarProvider(cfg.Calendar)
	}
	calendarRepo := calendarrepo.New(db)
	calendarSvc := calendar.NewService(calendarRepo, calendarProvider, cipher, calendar.Config{
		Enabled:          cfg.Calendar.Enabled,
		StateTTL:         cfg.Calendar.StateTTL(),
		PublicWebBaseURL: cfg.Calendar.PublicWebBaseURL,
		DefaultTimeZone:  cfg.Calendar.DefaultTimeZone,
	}, log)
	calendarHandler := handler.NewCalendarHandler(calendarSvc, log)
	internalCalendarHandler := handler.NewInternalCalendarHandler(calendarSvc, log)
	providerOpsHandler := handler.NewProviderOpsHandler(cfg, log)
	readinessHandler := httpserver.NewReadinessHandler(log)
	router := httpserver.NewRouter(
		log,
		placesHandler,
		routesHandler,
		weatherHandler,
		exchangeRateHandler,
		priceHandler,
		calendarHandler,
		internalCalendarHandler,
		providerOpsHandler,
		readinessHandler,
		cfg.CORS,
		cfg.Auth,
		cfg.Internal,
		cfg.Ops,
	)

	return &container{db: db, router: router}, nil
}
