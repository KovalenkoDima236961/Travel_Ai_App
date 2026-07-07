package app

import (
	"context"
	"fmt"
	"time"

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
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/storage/postgres"
	workerconfig "github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/httpserver"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/rabbitmq"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/pkg/closer"
)

type container struct {
	server   server
	consumer *rabbitmq.Consumer
}

func buildContainer(
	ctx context.Context,
	cfg *workerconfig.Config,
	log *zap.Logger,
	shutdown *closer.Stack,
) (*container, error) {
	processor, db, publisher, err := buildProcessor(ctx, cfg.Trip, log)
	if err != nil {
		return nil, err
	}
	shutdown.Add("postgres", func(context.Context) error {
		db.Close()
		return nil
	})
	shutdown.Add("rabbitmq-publisher", func(context.Context) error {
		return publisher.Close()
	})

	queueCfg := rabbitmq.GenerationQueueConfig(cfg.Trip)
	consumer := rabbitmq.NewConsumer(
		queueCfg,
		cfg.Trip.GenerationJobs.Prefetch,
		cfg.Trip.GenerationJobs.MaxAttempts,
		processor,
		publisher,
		log,
	)
	shutdown.Add("rabbitmq-consumer", func(context.Context) error {
		return consumer.Close()
	})

	httpServer, err := httpserver.New(cfg, db, consumer, log)
	if err != nil {
		return nil, fmt.Errorf("init http server: %w", err)
	}
	shutdown.Add("http-server", httpServer.Shutdown)

	return &container{
		server:   httpServer,
		consumer: consumer,
	}, nil
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

	publisher, err := jobqueue.NewRabbitMQPublisher(ctx, rabbitmq.GenerationQueueConfig(cfg), log)
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
