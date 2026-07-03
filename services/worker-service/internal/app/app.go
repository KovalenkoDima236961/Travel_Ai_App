package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appservice "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	tripauth "github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	tripconfig "github.com/KovalenkoDima236961/Travel_Ai_App/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/exchangerateclient"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/generator"
	triprepo "github.com/KovalenkoDima236961/Travel_Ai_App/internal/infrastructure/repository/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/jobqueue"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
	tripops "github.com/KovalenkoDima236961/Travel_Ai_App/internal/ops"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/placecontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/placeenrichment"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/priceclient"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/priceenrichment"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/logger"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/observability"
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
		server:    newHTTPServer(cfg, db, consumer, log),
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
	cfg *workerconfig.Config,
	db *postgres.DB,
	consumer *rabbitmq.Consumer,
	log *zap.Logger,
) *http.Server {
	r := chi.NewRouter()
	startedAt := time.Now().UTC()
	r.Use(observability.RequestIDMiddleware)
	r.Use(chimiddleware.Recoverer)
	r.Use(observability.HTTPMetricsMiddleware(observability.DefaultHTTPMetrics("worker-service")))
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
	r.Handle("/metrics", observability.MetricsHandler(nil))
	if cfg.Trip.Ops.DashboardEnabled {
		devUserID, err := uuid.Parse(cfg.Trip.Auth.DevUserID)
		if err != nil {
			log.Panic("invalid dev user id", zap.String("dev_user_id", cfg.Trip.Auth.DevUserID), zap.Error(err))
		}
		mgmt, err := rabbitmq.NewManagementClient(rabbitmq.ManagementConfig{
			URL:      cfg.RabbitMQManagement.URL,
			User:     cfg.RabbitMQManagement.User,
			Password: cfg.RabbitMQManagement.Password,
			AMQPURL:  cfg.Trip.GenerationJobs.RabbitMQURL,
			Queue:    generationQueueConfig(cfg.Trip),
		})
		if err != nil {
			log.Warn("rabbitmq management client disabled", zap.Error(err))
		}
		opsHandler := &workerOpsHandler{
			cfg:        cfg,
			db:         db,
			consumer:   consumer,
			management: mgmt,
			startedAt:  startedAt,
			log:        log,
		}
		r.Group(func(r chi.Router) {
			r.Use(tripauth.Middleware(tripauth.MiddlewareConfig{
				Required:        true,
				JWTAccessSecret: cfg.Trip.Auth.JWTAccessSecret,
				HeaderName:      cfg.Trip.Auth.HeaderName,
				DevUserID:       devUserID,
			}))
			r.Use(tripops.NewAdminChecker(cfg.Trip.Ops, log).Middleware)
			r.Get("/ops/worker/status", opsHandler.status)
			r.Get("/ops/queues/status", opsHandler.queueStatus)
			r.Get("/ops/dlq/messages", opsHandler.listDLQ)
			r.Post("/ops/dlq/messages/{messageId}/requeue", opsHandler.requeueDLQ)
			r.Post("/ops/dlq/messages/{messageId}/discard", opsHandler.discardDLQ)
		})
	}

	return &http.Server{
		Addr:         cfg.Runtime.HTTPAddress,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
}

func requestLogger(log *zap.Logger) func(http.Handler) http.Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			fields := []zap.Field{
				zap.String("service", "worker-service"),
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.String("route", observability.RoutePattern(r)),
				zap.Int("status", status),
				zap.Float64("durationMs", float64(time.Since(start).Microseconds())/1000),
			}
			fields = append(fields, observability.RequestIDFields(r.Context())...)
			log.Info("http_request", fields...)
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

type workerOpsHandler struct {
	cfg        *workerconfig.Config
	db         *postgres.DB
	consumer   *rabbitmq.Consumer
	management *rabbitmq.ManagementClient
	startedAt  time.Time
	log        *zap.Logger
}

func (h *workerOpsHandler) status(w http.ResponseWriter, r *http.Request) {
	dbConnected := false
	if h.db != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		err := h.db.Ping(ctx)
		cancel()
		dbConnected = err == nil
	}
	rabbitConnected := h.consumer != nil && h.consumer.Ready()
	writeJSON(w, http.StatusOK, map[string]any{
		"service":           "worker-service",
		"enabled":           h.cfg.Runtime.Enabled,
		"healthy":           dbConnected && rabbitConnected,
		"rabbitmqConnected": rabbitConnected,
		"dbConnected":       dbConnected,
		"concurrency":       h.cfg.Runtime.Concurrency,
		"prefetch":          h.consumerPrefetch(),
		"activeJobs":        h.activeJobs(),
		"startedAt":         h.startedAt,
		"version":           "local",
	})
}

func (h *workerOpsHandler) queueStatus(w http.ResponseWriter, r *http.Request) {
	if h.management == nil {
		rabbitmq.RecordOpsQueueStatusRequest("unavailable")
		writeError(w, http.StatusServiceUnavailable, "rabbitmq management API unavailable")
		return
	}
	queues, err := h.management.QueueStatuses(r.Context())
	if err != nil {
		rabbitmq.RecordOpsQueueStatusRequest("error")
		h.log.Warn("ops queue status failed", zap.Error(err))
		writeError(w, http.StatusServiceUnavailable, "rabbitmq management API unavailable")
		return
	}
	rabbitmq.RecordOpsQueueStatusRequest("success")
	writeJSON(w, http.StatusOK, map[string]any{"queues": queues})
}

func (h *workerOpsHandler) listDLQ(w http.ResponseWriter, r *http.Request) {
	if h.management == nil {
		writeError(w, http.StatusServiceUnavailable, "rabbitmq management API unavailable")
		return
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = parsed
	}
	messages, err := h.management.ListDLQMessages(r.Context(), limit)
	if err != nil {
		h.log.Warn("ops list dlq failed", zap.Error(err))
		writeError(w, http.StatusServiceUnavailable, "rabbitmq management API unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": messages})
}

func (h *workerOpsHandler) requeueDLQ(w http.ResponseWriter, r *http.Request) {
	h.dlqAction(w, r, "requeue")
}

func (h *workerOpsHandler) discardDLQ(w http.ResponseWriter, r *http.Request) {
	h.dlqAction(w, r, "discard")
}

func (h *workerOpsHandler) dlqAction(w http.ResponseWriter, r *http.Request, action string) {
	if h.management == nil {
		rabbitmq.RecordOpsDLQAction(action, "unavailable")
		writeError(w, http.StatusServiceUnavailable, "rabbitmq management API unavailable")
		return
	}
	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		rabbitmq.RecordOpsDLQAction(action, "invalid")
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		rabbitmq.RecordOpsDLQAction(action, "invalid")
		writeError(w, http.StatusBadRequest, "reason is required")
		return
	}
	messageID := chi.URLParam(r, "messageId")
	var err error
	switch action {
	case "requeue":
		err = h.management.RequeueDLQMessage(r.Context(), messageID, reason)
	case "discard":
		err = h.management.DiscardDLQMessage(r.Context(), messageID, reason)
	default:
		err = fmt.Errorf("unsupported action")
	}
	if err != nil {
		if errors.Is(err, rabbitmq.ErrMessageNotFound) {
			rabbitmq.RecordOpsDLQAction(action, "not_found")
			writeError(w, http.StatusNotFound, "message not found")
			return
		}
		rabbitmq.RecordOpsDLQAction(action, "error")
		h.log.Warn("ops dlq action failed",
			zap.String("action", action),
			zap.String("messageId", messageID),
			zap.Error(err),
		)
		writeError(w, http.StatusServiceUnavailable, "rabbitmq management API unavailable")
		return
	}
	rabbitmq.RecordOpsDLQAction(action, "success")
	key := "discarded"
	if action == "requeue" {
		key = "requeued"
	}
	writeJSON(w, http.StatusOK, map[string]bool{key: true})
}

func (h *workerOpsHandler) activeJobs() []rabbitmq.ActiveJob {
	if h.consumer == nil {
		return []rabbitmq.ActiveJob{}
	}
	return h.consumer.ActiveJobs()
}

func (h *workerOpsHandler) consumerPrefetch() int {
	if h.consumer == nil {
		return h.cfg.Trip.GenerationJobs.Prefetch
	}
	return h.consumer.Prefetch()
}
