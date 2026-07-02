package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appservice "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	tripconfig "github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/exchangerateclient"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/generator"
	triprepo "github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/jobqueue"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/placecontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/placeenrichment"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/priceclient"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/priceenrichment"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/logger"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
	workerconfig "github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/rabbitmq"
)

type App struct {
	cfg       *workerconfig.Config
	log       *zap.Logger
	db        *postgres.DB
	server    *http.Server
	consumer  *rabbitmq.Consumer
	publisher *jobqueue.RabbitMQPublisher
}

func New(configPath string) *App {
	cfg := workerconfig.MustLoad(configPath)
	log := logger.InitLogger()
	log.Info("configuration loaded",
		zap.String("service", cfg.Runtime.ServiceName),
		zap.String("http_address", cfg.Runtime.HTTPAddress),
	)

	processor, db, publisher, err := buildProcessor(context.Background(), cfg.Trip, log)
	if err != nil {
		log.Fatal("failed to build worker", zap.Error(err))
	}

	queueCfg := generationQueueConfig(cfg.Trip)
	consumer := rabbitmq.NewConsumer(
		queueCfg,
		cfg.Trip.GenerationJobs.Prefetch,
		cfg.Trip.GenerationJobs.MaxAttempts,
		processor,
		publisher,
		log,
	)

	return &App{
		cfg:       cfg,
		log:       log,
		db:        db,
		server:    newHTTPServer(cfg.Runtime.HTTPAddress, db, consumer, log),
		consumer:  consumer,
		publisher: publisher,
	}
}

func (a *App) Run() {
	if !a.cfg.Runtime.Enabled {
		a.log.Info("worker disabled")
		return
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	consumerCtx, cancelConsumer := context.WithCancel(rootCtx)
	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- a.consumer.Run(consumerCtx)
	}()

	serverDone := make(chan error, 1)
	go func() {
		a.log.Info("worker http server starting", zap.String("address", a.server.Addr))
		err := a.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			serverDone <- err
			return
		}
		serverDone <- nil
	}()

	consumerStopped := false
	select {
	case <-rootCtx.Done():
		a.log.Info("shutdown signal received")
	case err := <-consumerDone:
		consumerStopped = true
		if err != nil {
			a.log.Error("consumer stopped", zap.Error(err))
		}
	case err := <-serverDone:
		if err != nil {
			a.log.Error("http server stopped", zap.Error(err))
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), a.cfg.ShutdownTimeout())
	defer cancel()
	cancelConsumer()

	if !consumerStopped {
		select {
		case err := <-consumerDone:
			if err != nil {
				a.log.Warn("consumer stopped during shutdown", zap.Error(err))
			}
		case <-shutdownCtx.Done():
			a.log.Warn("consumer shutdown timed out", zap.Error(shutdownCtx.Err()))
		}
	}

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		a.log.Warn("http server shutdown failed", zap.Error(err))
	}
	if a.consumer != nil {
		_ = a.consumer.Close()
	}
	if a.publisher != nil {
		_ = a.publisher.Close()
	}
	if a.db != nil {
		a.db.Close()
	}
	a.log.Info("shutdown complete")
	_ = a.log.Sync()
}

func buildProcessor(
	ctx context.Context,
	cfg *tripconfig.Config,
	log *zap.Logger,
) (*generationjobs.Worker, *postgres.DB, *jobqueue.RabbitMQPublisher, error) {
	db, err := postgres.New(ctx, cfg.Postgres)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("init postgres: %w", err)
	}

	repo := triprepo.New(db)
	gen, err := generator.NewItineraryGenerator(cfg, log)
	if err != nil {
		db.Close()
		return nil, nil, nil, fmt.Errorf("init itinerary generator: %w", err)
	}

	opts, err := tripServiceOptions(cfg, repo, log)
	if err != nil {
		db.Close()
		return nil, nil, nil, err
	}
	tripSvc := appservice.New(repo, gen, log, opts...)

	jobCfg := generationjobs.NormalizeConfig(generationjobs.Config{
		Enabled:               cfg.GenerationJobs.Enabled,
		WorkerEnabled:         true,
		DispatchMode:          generationjobs.DispatchModeQueue,
		PollInterval:          cfg.GenerationJobWorkerPollInterval(),
		MaxConcurrent:         cfg.GenerationJobs.WorkerMaxConcurrent,
		MaxRunning:            cfg.GenerationJobMaxRunning(),
		PublishTimeout:        cfg.GenerationJobPublishTimeout(),
		PublishFailOpen:       cfg.GenerationJobs.PublishFailOpen,
		FailOpenNotifications: cfg.GenerationJobs.FailOpenNotifications,
	})

	processor := generationjobs.NewWorker(repo, tripSvc, jobCfg, log)
	if err := failStaleRunningJobs(ctx, repo, jobCfg, log); err != nil {
		db.Close()
		return nil, nil, nil, err
	}

	publisher, err := jobqueue.NewRabbitMQPublisher(ctx, generationQueueConfig(cfg), log)
	if err != nil {
		db.Close()
		return nil, nil, nil, fmt.Errorf("init rabbitmq publisher: %w", err)
	}

	return processor, db, publisher, nil
}

func tripServiceOptions(
	cfg *tripconfig.Config,
	repo *triprepo.Repository,
	log *zap.Logger,
) ([]appservice.Option, error) {
	opts := []appservice.Option{
		appservice.WithActivity(activity.New(repo, log)),
	}

	if cfg.UserContext.Enabled {
		client, err := usercontext.New(usercontext.Config{
			BaseURL:        cfg.UserContext.UserServiceURL,
			TimeoutSeconds: cfg.UserContext.TimeoutSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("init user context client: %w", err)
		}
		opts = append(opts, appservice.WithUserContext(
			client,
			cfg.UserContext.Enabled,
			cfg.UserContext.FailOpen,
		))
	}

	if cfg.WeatherContext.Enabled {
		client, err := weathercontext.New(weathercontext.Config{
			BaseURL:        cfg.WeatherContext.ExternalIntegrationsServiceURL,
			TimeoutSeconds: cfg.WeatherContext.TimeoutSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("init weather context client: %w", err)
		}
		opts = append(opts, appservice.WithWeatherContext(
			client,
			cfg.WeatherContext.Enabled,
			cfg.WeatherContext.FailOpen,
		))
	}

	if cfg.PlaceEnrichment.Enabled {
		client, err := placecontext.New(placecontext.Config{
			BaseURL:        cfg.PlaceEnrichment.ExternalIntegrationsServiceURL,
			TimeoutSeconds: cfg.PlaceEnrichment.TimeoutSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("init place context client: %w", err)
		}
		opts = append(opts, appservice.WithPlaceEnrichment(
			placeenrichment.New(client, placeenrichment.Config{
				MinConfidence:     cfg.PlaceEnrichment.MinConfidence,
				MaxItems:          cfg.PlaceEnrichment.MaxItems,
				OverwriteExisting: cfg.PlaceEnrichment.OverwriteExisting,
				FailOpen:          cfg.PlaceEnrichment.FailOpen,
			}),
			cfg.PlaceEnrichment.Enabled,
			cfg.PlaceEnrichment.FailOpen,
		))
	}

	if cfg.PriceEnrichment.Enabled {
		client, err := priceclient.New(priceclient.Config{
			BaseURL:        cfg.PriceEnrichment.ExternalIntegrationsServiceURL,
			Token:          cfg.PriceEnrichment.InternalServiceToken,
			TimeoutSeconds: cfg.PriceEnrichment.TimeoutSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("init price client: %w", err)
		}
		opts = append(opts, appservice.WithPriceEnrichment(
			priceenrichment.New(client, priceenrichment.Config{
				Enabled:              cfg.PriceEnrichment.Enabled,
				FailOpen:             cfg.PriceEnrichment.FailOpen,
				OverwriteAICosts:     cfg.PriceEnrichment.OverwriteAICosts,
				OverwriteManualCosts: cfg.PriceEnrichment.OverwriteManualCosts,
				MinMatchConfidence:   cfg.PriceEnrichment.MinMatchConfidence,
				MaxItems:             cfg.PriceEnrichment.MaxItems,
				DefaultCurrency:      cfg.PriceEnrichment.DefaultCurrency,
			}),
			cfg.PriceEnrichment.Enabled,
			cfg.PriceEnrichment.FailOpen,
		))
	}

	if cfg.Notifications.Enabled {
		client, err := notifications.New(notifications.Config{
			BaseURL:        cfg.Notifications.NotificationServiceURL,
			Token:          cfg.Notifications.NotificationServiceToken,
			TimeoutSeconds: cfg.Notifications.TimeoutSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("init notification client: %w", err)
		}
		opts = append(opts, appservice.WithNotifications(
			client,
			cfg.Notifications.Enabled,
			cfg.Notifications.FailOpen,
		))
	}

	if cfg.BudgetConversion.Enabled {
		client, err := exchangerateclient.New(exchangerateclient.Config{
			BaseURL:        cfg.BudgetConversion.ExternalIntegrationsServiceURL,
			Token:          cfg.BudgetConversion.InternalServiceToken,
			TimeoutSeconds: cfg.BudgetConversion.TimeoutSeconds,
		})
		if err != nil {
			return nil, fmt.Errorf("init exchange rate client: %w", err)
		}
		opts = append(opts, appservice.WithBudgetConversion(
			client,
			cfg.BudgetConversion.Enabled,
			cfg.BudgetConversion.FailOpen,
		))
	}

	return opts, nil
}

func failStaleRunningJobs(
	ctx context.Context,
	repo *triprepo.Repository,
	cfg generationjobs.Config,
	log *zap.Logger,
) error {
	startedBefore := time.Now().Add(-cfg.MaxRunning)
	count, err := repo.MarkStaleRunningGenerationJobsFailed(
		ctx,
		startedBefore,
		generationjobs.ErrorWorkerInterrupted,
		"Generation job was interrupted by worker shutdown.",
	)
	if err != nil {
		return fmt.Errorf("mark stale running generation jobs failed: %w", err)
	}
	if count > 0 {
		log.Warn("stale running generation jobs marked failed", zap.Int64("count", count))
	}
	return nil
}

func generationQueueConfig(cfg *tripconfig.Config) jobqueue.Config {
	return jobqueue.Config{
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
	}
}

func newHTTPServer(
	address string,
	db *postgres.DB,
	consumer *rabbitmq.Consumer,
	log *zap.Logger,
) *http.Server {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(requestLogger(log))
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		checks := map[string]string{}
		ready := true

		if db == nil {
			checks["postgres"] = "failed"
			ready = false
		} else {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			err := db.Ping(ctx)
			cancel()
			if err != nil {
				checks["postgres"] = "failed"
				ready = false
				log.Warn("postgres readiness check failed", zap.Error(err))
			} else {
				checks["postgres"] = "ok"
			}
		}

		if consumer == nil || !consumer.Ready() {
			checks["rabbitmq"] = "failed"
			ready = false
		} else {
			checks["rabbitmq"] = "ok"
		}

		status := "ready"
		httpStatus := http.StatusOK
		if !ready {
			status = "not_ready"
			httpStatus = http.StatusServiceUnavailable
		}
		writeJSON(w, httpStatus, map[string]any{
			"status": status,
			"checks": checks,
		})
	})

	return &http.Server{
		Addr:         address,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func requestLogger(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			log.Info("http_request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.Status()),
				zap.Duration("duration", time.Since(start)),
				zap.String("request_id", middleware.GetReqID(r.Context())),
			)
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
