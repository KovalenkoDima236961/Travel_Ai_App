package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activitystream"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/aiobservability"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/aivalidation"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetconfidence"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/calendarclient"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/editlocks"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/exchangerateclient"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
	httpserver "github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/handler"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/generator"
	triprepo "github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/jobqueue"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/placecontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/placeenrichment"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/closer"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/validation"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/presence"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/priceclient"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/priceenrichment"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/receipts"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/search"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/transportclient"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/tripdiscovery"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/triphealth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/users"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
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
	db, err := postgres.New(ctx, cfg.Postgres, log)
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
	repairClient, err := generator.NewGenerationRepairClient(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("init ai generation repair client: %w", err)
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
	var priceEnrichmentSvc interface {
		EnrichItinerary(context.Context, priceenrichment.EnrichItineraryInput) (*priceenrichment.EnrichItineraryResult, error)
	}
	if cfg.PriceEnrichment.Enabled {
		priceClient, err := priceclient.New(priceclient.Config{
			BaseURL:        cfg.PriceEnrichment.ExternalIntegrationsServiceURL,
			Token:          cfg.PriceEnrichment.InternalServiceToken,
			TimeoutSeconds: cfg.PriceEnrichment.TimeoutSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("init price client: %w", err)
		}
		priceEnrichmentSvc = priceenrichment.New(priceClient, priceenrichment.Config{
			Enabled:              cfg.PriceEnrichment.Enabled,
			FailOpen:             cfg.PriceEnrichment.FailOpen,
			OverwriteAICosts:     cfg.PriceEnrichment.OverwriteAICosts,
			OverwriteManualCosts: cfg.PriceEnrichment.OverwriteManualCosts,
			MinMatchConfidence:   cfg.PriceEnrichment.MinMatchConfidence,
			MaxItems:             cfg.PriceEnrichment.MaxItems,
			DefaultCurrency:      cfg.PriceEnrichment.DefaultCurrency,
		})
	}
	userLookupClient, err := users.New(users.Config{
		BaseURL:        cfg.UserLookup.AuthServiceURL,
		Token:          cfg.UserLookup.InternalServiceToken,
		TimeoutSeconds: cfg.UserLookup.TimeoutSeconds,
	})
	if err != nil {
		return nil, fmt.Errorf("init user lookup client: %w", err)
	}
	var workspaceClient interface {
		AccessCheck(context.Context, uuid.UUID, uuid.UUID) (*workspaces.Access, error)
		ListForUser(context.Context, uuid.UUID) ([]workspaces.UserWorkspace, error)
		ListMembers(context.Context, uuid.UUID) ([]workspaces.WorkspaceMember, error)
		BatchInfo(context.Context, []uuid.UUID) ([]workspaces.WorkspaceInfo, error)
	}
	if cfg.Workspaces.Enabled {
		workspaceClient, err = workspaces.New(workspaces.Config{
			BaseURL:        cfg.Workspaces.UserServiceURL,
			Token:          cfg.Workspaces.ServiceToken,
			TimeoutSeconds: cfg.Workspaces.TimeoutSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("init workspace client: %w", err)
		}
	}
	activityStreamCfg := activitystream.Normalize(activitystream.Config{
		Enabled:                      cfg.ActivityStream.Enabled,
		HeartbeatInterval:            cfg.ActivityStreamHeartbeatInterval(),
		WriteTimeout:                 cfg.ActivityStreamWriteTimeout(),
		MaxConnectionsPerUserPerTrip: cfg.ActivityStream.MaxConnectionsPerUserPerTrip,
		ClientBufferSize:             cfg.ActivityStream.ClientBufferSize,
	})
	activityStreamManager := activitystream.NewManager(activityStreamCfg, log)
	activitySvc := activity.New(repo, log, activity.WithPublisher(activityStreamManager))
	policyRepo := workspacepolicies.NewRepository(db)
	policySvc := workspacepolicies.New(policyRepo, workspaceClient)

	var notificationClient *notifications.Client
	if cfg.Notifications.Enabled {
		notificationClient, err = notifications.New(notifications.Config{
			BaseURL:        cfg.Notifications.NotificationServiceURL,
			Token:          cfg.Notifications.NotificationServiceToken,
			TimeoutSeconds: cfg.Notifications.TimeoutSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("init notification client: %w", err)
		}
	}

	opts := []service.Option{
		service.WithUserContext(
			userContextClient,
			cfg.UserContext.Enabled,
			cfg.UserContext.FailOpen,
		),
		service.WithWeatherContext(
			weatherContextClient,
			cfg.WeatherContext.Enabled,
			cfg.WeatherContext.FailOpen,
		),
		service.WithPlaceEnrichment(
			placeEnrichmentSvc,
			cfg.PlaceEnrichment.Enabled,
			cfg.PlaceEnrichment.FailOpen,
		),
		service.WithPriceEnrichment(
			priceEnrichmentSvc,
			cfg.PriceEnrichment.Enabled,
			cfg.PriceEnrichment.FailOpen,
		),
		service.WithPublicSharing(
			cfg.PublicSharing.Enabled,
			cfg.PublicSharing.PublicWebBaseURL,
			cfg.PublicSharing.ShareTokenBytes,
			cfg.PublicSharing.PublicShareAccessSecret,
			cfg.PublicSharing.PublicShareAccessTTLMinutes,
		),
		service.WithUserLookup(userLookupClient),
		service.WithActivity(activitySvc),
		service.WithWorkspaces(workspaceClient, cfg.Workspaces.Enabled),
		service.WithWorkspacePolicies(policySvc),
		service.WithTripHealthConfig(triphealth.Config{
			Enabled:                         cfg.TripHealth.Enabled,
			IncludeDebug:                    cfg.TripHealth.IncludeDebug,
			LargeExpenseReceiptThreshold:    cfg.TripHealth.LargeExpenseReceiptThreshold,
			DefaultMaxWalkingKmPerDay:       cfg.TripHealth.DefaultMaxWalkingKmPerDay,
			DefaultMaxTransferMinutesPerDay: cfg.TripHealth.DefaultMaxTransferMinutesPerDay,
		}),
		service.WithBudgetConfidenceConfig(budgetconfidence.Config{
			Enabled:                         cfg.BudgetConfidence.Enabled,
			FailOpen:                        cfg.BudgetConfidence.FailOpen,
			LargeExpenseReceiptThreshold:    cfg.BudgetConfidence.LargeExpenseReceiptThreshold,
			ActualSpendHighThresholdPercent: cfg.BudgetConfidence.ActualSpendHighThresholdPercent,
			PlannedActualGapWarningPercent:  cfg.BudgetConfidence.PlannedActualGapWarningPercent,
			PlannedActualGapHighPercent:     cfg.BudgetConfidence.PlannedActualGapHighPercent,
		}),
		service.WithSummaryCache(
			cfg.SummaryCache.Enabled,
			time.Duration(cfg.SummaryCache.TTLSeconds)*time.Second,
			cfg.SummaryCache.MaxItems,
			time.Duration(cfg.SummaryCache.EndpointTimeoutSeconds)*time.Second,
		),
	}
	aiValidationCfg := aivalidation.Config{
		Enabled:                    cfg.AIValidation.Enabled,
		RepairEnabled:              cfg.AIValidation.RepairEnabled,
		MaxRepairAttempts:          cfg.AIValidation.MaxRepairAttempts,
		BlockOnSchemaErrors:        cfg.AIValidation.BlockOnSchemaErrors,
		BlockOnPolicyBlockers:      cfg.AIValidation.BlockOnPolicyBlockers,
		BlockOnCriticalRouteErrors: cfg.AIValidation.BlockOnCriticalRouteErrors,
		BlockOnBudgetErrors:        cfg.AIValidation.BlockOnBudgetErrors,
		FailOpen:                   cfg.AIValidation.FailOpen,
	}
	if notificationClient != nil {
		opts = append(opts, service.WithNotifications(
			notificationClient,
			cfg.Notifications.Enabled,
			cfg.Notifications.FailOpen,
		))
	}
	if cfg.CalendarSync.Enabled || cfg.CalendarSync.FreeBusyImportEnabled {
		calendarTimeout := cfg.CalendarSync.TimeoutSeconds
		if cfg.CalendarSync.FreeBusyImportEnabled && cfg.CalendarSync.FreeBusyImportTimeoutSeconds > calendarTimeout {
			calendarTimeout = cfg.CalendarSync.FreeBusyImportTimeoutSeconds
		}
		calendarClient, err := calendarclient.New(calendarclient.Config{
			BaseURL:        cfg.CalendarSync.ExternalIntegrationsServiceURL,
			Token:          cfg.CalendarSync.InternalServiceToken,
			TimeoutSeconds: calendarTimeout,
		})
		if err != nil {
			return nil, fmt.Errorf("init calendar sync client: %w", err)
		}
		if cfg.CalendarSync.Enabled {
			opts = append(opts, service.WithCalendarSync(
				calendarClient,
				cfg.CalendarSync.Enabled,
				cfg.PublicSharing.PublicWebBaseURL,
				cfg.CalendarSync.DefaultTimeZone,
			))
		}
		if cfg.CalendarSync.FreeBusyImportEnabled {
			opts = append(opts, service.WithCalendarAvailabilityImport(
				calendarClient,
				cfg.CalendarSync.FreeBusyImportEnabled,
				cfg.CalendarSync.FreeBusyImportFailOpen,
				cfg.CalendarSync.DefaultTimeZone,
			))
		}
	}
	if cfg.BudgetConversion.Enabled {
		exchangeRateClient, err := exchangerateclient.New(exchangerateclient.Config{
			BaseURL:        cfg.BudgetConversion.ExternalIntegrationsServiceURL,
			Token:          cfg.BudgetConversion.InternalServiceToken,
			TimeoutSeconds: cfg.BudgetConversion.TimeoutSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("init exchange rate client: %w", err)
		}
		opts = append(opts, service.WithBudgetConversion(
			exchangeRateClient,
			cfg.BudgetConversion.Enabled,
			cfg.BudgetConversion.FailOpen,
		))
	}
	if cfg.TransportSearch.Enabled {
		transportClient, err := transportclient.New(transportclient.Config{
			BaseURL:        cfg.TransportSearch.ExternalIntegrationsServiceURL,
			Token:          cfg.TransportSearch.InternalServiceToken,
			TimeoutSeconds: cfg.TransportSearch.TimeoutSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("init transport search client: %w", err)
		}
		opts = append(opts, service.WithTransportSearch(
			transportClient,
			cfg.TransportSearch.Enabled,
			cfg.TransportSearch.FailOpen,
		))
	}
	receiptStorage, err := receipts.NewLocalStorage(cfg.Receipts.LocalDir)
	if err != nil {
		return nil, fmt.Errorf("init receipt storage: %w", err)
	}
	receiptOCRProvider := receipts.NewMockOCRProvider()
	opts = append(opts, service.WithReceipts(
		receiptStorage,
		receiptOCRProvider,
		receipts.Config{
			StorageProvider:   cfg.Receipts.StorageProvider,
			LocalDir:          cfg.Receipts.LocalDir,
			MaxFileSizeMB:     cfg.Receipts.MaxFileSizeMB,
			MaxFileSizeBytes:  cfg.Receipts.UploadMaxBytes,
			AllowedMIMEs:      splitCSV(cfg.Receipts.UploadAllowedMIME),
			AllowedExtensions: splitCSV(cfg.Receipts.UploadAllowedExt),
			ScanningEnabled:   cfg.Receipts.FileScanningEnabled,
			ScanningFailOpen:  cfg.Receipts.FileScanningFailOpen,
			OCREnabled:        cfg.Receipts.OCREnabled,
			OCRProvider:       receiptOCRProvider.Name(),
			OCRTimeout:        time.Duration(cfg.Receipts.OCRTimeoutSeconds) * time.Second,
			OCRFailOpen:       cfg.Receipts.OCRFailOpen,
			StoreRawText:      cfg.Receipts.OCRStoreRawText,
		},
	))
	opts = append(opts, service.WithFileScanner(receipts.NoopFileScanner{}))
	svc := service.New(repo, gen, log, opts...)
	reliability := aivalidation.NewPipeline(
		aivalidation.NewValidator(aiValidationCfg),
		repairClient,
		svc,
		aiValidationCfg,
		log,
	)
	opts = append(opts, service.WithGenerationReliability(reliability))
	svc = service.New(repo, gen, log, opts...)
	generationJobsCfg := generationjobs.NormalizeConfig(generationjobs.Config{
		Enabled:               cfg.GenerationJobs.Enabled,
		WorkerEnabled:         cfg.GenerationJobs.WorkerEnabled,
		DispatchMode:          cfg.GenerationJobs.DispatchMode,
		PollInterval:          cfg.GenerationJobWorkerPollInterval(),
		MaxConcurrent:         cfg.GenerationJobs.WorkerMaxConcurrent,
		MaxRunning:            cfg.GenerationJobMaxRunning(),
		PublishTimeout:        cfg.GenerationJobPublishTimeout(),
		PublishFailOpen:       cfg.GenerationJobs.PublishFailOpen,
		FailOpenNotifications: cfg.GenerationJobs.FailOpenNotifications,
	})
	generationJobOptions := []generationjobs.Option{}
	if generationJobsCfg.QueueMode() {
		publisher, err := jobqueue.NewRabbitMQPublisher(context.Background(), jobqueue.Config{
			URL:                  cfg.GenerationJobs.RabbitMQURL,
			Exchange:             cfg.GenerationJobs.RabbitMQExchange,
			DLX:                  cfg.GenerationJobs.RabbitMQDLX,
			QueueName:            cfg.GenerationJobs.QueueName,
			RoutingKey:           cfg.GenerationJobs.RoutingKey,
			DeadLetterQueueName:  cfg.GenerationJobs.DeadLetterQueueName,
			DeadLetterRoutingKey: cfg.GenerationJobs.DeadLetterRoutingKey,
			RetryQueueName:       cfg.GenerationJobs.RetryQueueName,
			RetryRoutingKey:      cfg.GenerationJobs.RetryRoutingKey,
			RetryDelay:           time.Duration(cfg.GenerationJobs.RetryDelaySeconds) * time.Second,
			PublishTimeout:       cfg.GenerationJobPublishTimeout(),
		}, log)
		if err != nil {
			return nil, fmt.Errorf("init generation job publisher: %w", err)
		}
		closer.Add("generation-job-publisher", func(context.Context) error {
			return publisher.Close()
		})
		generationJobOptions = append(generationJobOptions, generationjobs.WithPublisher(publisher))
	}
	generationJobSvc := generationjobs.NewService(repo, svc, generationJobsCfg, generationJobOptions...)
	aiTraceService := aiobservability.New(db, aiobservability.Config{
		Enabled:                   cfg.AIObservability.Enabled,
		TraceEventsEnabled:        cfg.AIObservability.TraceEventsEnabled,
		StoreRedactedPrompts:      cfg.AIObservability.StoreRedactedPrompts,
		StoreRedactedResponses:    cfg.AIObservability.StoreRedactedResponses,
		MaxPromptSnapshotChars:    cfg.AIObservability.MaxPromptSnapshotChars,
		RetentionDays:             cfg.AIObservability.RetentionDays,
		FailOpen:                  cfg.AIObservability.FailOpen,
		DebugLocalOnly:            cfg.AIObservability.DebugLocalOnly,
		PromptLoggingEnabled:      cfg.AIObservability.PromptLoggingEnabled,
		PromptLoggingRedactedOnly: cfg.AIObservability.PromptLoggingRedactedOnly,
		RedactionEnabled:          cfg.AIObservability.RedactionEnabled,
		Provider:                  providerForAIMode(cfg.ItineraryGenerator.Mode),
		Mode:                      cfg.ItineraryGenerator.Mode,
	}, log)
	closer.Add("ai-generation-trace-cleanup", aiobservability.StartCleanupLoop(context.Background(), aiTraceService, 24*time.Hour, log))
	discoveryAIClient, err := tripdiscovery.NewHTTPAIClient(
		cfg.ItineraryGenerator.AIPlanningServiceURL,
		time.Duration(cfg.TripDiscovery.AITimeoutSeconds)*time.Second,
	)
	if err != nil {
		return nil, fmt.Errorf("init trip discovery AI client: %w", err)
	}
	discoverySvc := tripdiscovery.NewService(
		repo,
		discoveryAIClient,
		svc,
		generationJobSvc,
		userContextClient,
		workspaceClient,
		policySvc,
		tripdiscovery.Config{
			Enabled:                cfg.TripDiscovery.Enabled,
			MaxPreviousTrips:       cfg.TripDiscovery.MaxPreviousTrips,
			DefaultSuggestionCount: cfg.TripDiscovery.DefaultSuggestionCount,
		},
		log,
	)
	discoveryHandler := tripdiscovery.NewHandler(discoverySvc)
	searchHandler := search.NewModule(db, workspaceClient, search.Config{
		Enabled:          cfg.Search.Enabled,
		DefaultLimit:     cfg.Search.DefaultLimit,
		MaxLimit:         cfg.Search.MaxLimit,
		PerCategoryLimit: cfg.Search.PerCategoryLimit,
		MinQueryLength:   cfg.Search.MinQueryLength,
		QueryTimeout:     time.Duration(cfg.Search.QueryTimeoutSeconds) * time.Second,
	}, log)
	generationJobWorker := generationjobs.NewWorker(repo, svc, generationJobsCfg, log, generationjobs.WithTracer(aiTraceService))
	closer.Add(
		"generation-job-worker",
		generationJobWorker.Start(context.Background()),
	)
	presenceCfg := presence.Normalize(presence.Config{
		Enabled:                      cfg.Presence.Enabled,
		HeartbeatInterval:            cfg.PresenceHeartbeatInterval(),
		StaleAfter:                   cfg.PresenceStaleAfter(),
		MaxConnectionsPerUserPerTrip: cfg.Presence.MaxConnectionsPerUserPerTrip,
		SendFullSnapshot:             cfg.Presence.SendFullSnapshot,
	})
	presenceManager := presence.NewManager(presenceCfg, log)
	closer.Add("trip-presence-cleanup", presence.StartCleanupLoop(context.Background(), presenceManager, presenceCfg, log))

	editLocksCfg := editlocks.Normalize(editlocks.Config{
		Enabled:         cfg.EditLocks.Enabled,
		TTL:             cfg.EditLockTTL(),
		RenewalInterval: cfg.EditLockRenewalInterval(),
		CleanupInterval: cfg.EditLockCleanupInterval(),
	})
	editLockManager := editlocks.NewManager()
	closer.Add("trip-edit-lock-cleanup", editlocks.StartCleanupLoop(context.Background(), editLockManager, editLocksCfg, log))

	tripHandler := handler.New(svc, validator, log).
		EnableSecurityLimits(
			cfg.PublicSharing.UnlockRateLimitPerMinute,
			cfg.PublicSharing.AccessRateLimitPerMinute,
			cfg.Receipts.UploadRateLimitPerMinute,
		).
		EnablePresence(presenceManager, presenceCfg).
		EnableActivityStream(activityStreamManager, activityStreamCfg).
		EnableEditLocks(editLockManager, editLocksCfg).
		EnableGenerationJobs(generationJobSvc).
		EnableAIObservability(aiTraceService).
		EnableWorkspacePolicies(policySvc)
	readinessHandler := httpserver.NewReadinessHandler(
		db,
		cfg.ItineraryGenerator.Mode,
		cfg.ItineraryGenerator.AIPlanningServiceURL,
		time.Duration(cfg.ItineraryGenerator.AIPlanningTimeoutSeconds)*time.Second,
		log,
	)
	router := httpserver.NewRouter(
		log,
		tripHandler,
		readinessHandler,
		cfg.CORS,
		cfg.Auth,
		cfg.Ops,
		discoveryHandler,
		searchHandler,
	)

	return &container{
		db:     db,
		router: router,
	}, nil
}

func providerForAIMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "mock":
		return "mock"
	case "http", "ollama":
		return "ollama"
	case "disabled":
		return "other"
	default:
		return "other"
	}
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
