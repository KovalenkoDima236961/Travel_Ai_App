package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"

	tripauth "github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	tripops "github.com/KovalenkoDima236961/Travel_Ai_App/internal/ops"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/cleanup"
	workerconfig "github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/rabbitmq"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/internal/version"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/worker-service/pkg/observability"
)

// New builds the worker HTTP server: liveness/readiness, metrics, and optional
// ops routes. It does not start listening.
func New(
	cfg *workerconfig.Config,
	db *postgres.DB,
	consumer *rabbitmq.Consumer,
	cleanupRunner *cleanup.Runner,
	log *zap.Logger,
) (*http.Server, error) {
	if log == nil {
		log = zap.NewNop()
	}

	r := chi.NewRouter()
	startedAt := time.Now().UTC()
	r.Use(observability.RequestIDMiddleware)
	r.Use(chimiddleware.Recoverer)
	r.Use(observability.HTTPMetricsMiddleware(observability.DefaultHTTPMetrics("worker-service")))
	r.Use(requestLogger(log))
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "worker-service"})
	})
	r.Get("/version", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, version.Info())
	})
	r.Get("/ready", readinessHandler(db, consumer, log))
	r.Handle("/metrics", observability.MetricsHandler(nil))

	if cfg.Trip.Ops.DashboardEnabled {
		if err := mountOpsRoutes(r, cfg, db, consumer, cleanupRunner, startedAt, log); err != nil {
			return nil, err
		}
	}

	return &http.Server{
		Addr:         cfg.Runtime.HTTPAddress,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}, nil
}

func readinessHandler(db *postgres.DB, consumer *rabbitmq.Consumer, log *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
			"status":       status,
			"service":      "worker-service",
			"dependencies": checks,
			"checks":       checks,
		})
	}
}

func mountOpsRoutes(
	r chi.Router,
	cfg *workerconfig.Config,
	db *postgres.DB,
	consumer *rabbitmq.Consumer,
	cleanupRunner *cleanup.Runner,
	startedAt time.Time,
	log *zap.Logger,
) error {
	devUserID, err := uuid.Parse(cfg.Trip.Auth.DevUserID)
	if err != nil {
		return fmt.Errorf("invalid dev user id %q: %w", cfg.Trip.Auth.DevUserID, err)
	}

	mgmt, err := rabbitmq.NewManagementClient(rabbitmq.ManagementConfig{
		URL:      cfg.RabbitMQManagement.URL,
		User:     cfg.RabbitMQManagement.User,
		Password: cfg.RabbitMQManagement.Password,
		AMQPURL:  cfg.Trip.GenerationJobs.RabbitMQURL,
		Queue:    rabbitmq.GenerationQueueConfig(cfg.Trip),
	})
	if err != nil {
		log.Warn("rabbitmq management client disabled", zap.Error(err))
	}
	opsHandler := &workerOpsHandler{
		cfg:        cfg,
		db:         db,
		consumer:   consumer,
		cleanup:    cleanupRunner,
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
		r.Get("/ops/cleanup/tasks", opsHandler.cleanupTasks)
		r.Get("/ops/cleanup/runs", opsHandler.cleanupRuns)
		r.Get("/ops/cleanup/runs/{runId}", opsHandler.cleanupRun)
		r.Post("/ops/cleanup/run", opsHandler.runCleanup)
		r.Get("/ops/storage/summary", opsHandler.storageSummary)
	})
	return nil
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
	cleanup    *cleanup.Runner
	management *rabbitmq.ManagementClient
	startedAt  time.Time
	log        *zap.Logger
}

func (h *workerOpsHandler) cleanupTasks(w http.ResponseWriter, _ *http.Request) {
	if h.cleanup == nil {
		writeError(w, http.StatusServiceUnavailable, "cleanup_disabled")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": h.cleanup.Tasks()})
}

func (h *workerOpsHandler) cleanupRuns(w http.ResponseWriter, r *http.Request) {
	if h.cleanup == nil {
		writeError(w, http.StatusServiceUnavailable, "cleanup_disabled")
		return
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			writeError(w, http.StatusBadRequest, "cleanup_invalid_scope")
			return
		}
		limit = parsed
	}
	runs, err := h.cleanup.Runs(r.Context(), limit)
	if err != nil {
		h.log.Warn("ops cleanup list failed", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "cleanup_internal_error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"runs": runs})
}

func (h *workerOpsHandler) cleanupRun(w http.ResponseWriter, r *http.Request) {
	if h.cleanup == nil {
		writeError(w, http.StatusServiceUnavailable, "cleanup_disabled")
		return
	}
	run, err := h.cleanup.RunByID(r.Context(), chi.URLParam(r, "runId"))
	if err != nil {
		writeError(w, http.StatusNotFound, "cleanup_task_not_found")
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (h *workerOpsHandler) runCleanup(w http.ResponseWriter, r *http.Request) {
	if h.cleanup == nil || !h.cfg.Cleanup.Enabled {
		writeError(w, http.StatusServiceUnavailable, "cleanup_disabled")
		return
	}
	var req struct {
		TaskName   string `json:"taskName"`
		DryRun     *bool  `json:"dryRun"`
		BatchSize  int    `json:"batchSize"`
		MaxBatches int    `json:"maxBatches"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "cleanup_invalid_scope")
		return
	}
	if strings.TrimSpace(req.TaskName) == "" || req.DryRun == nil {
		writeError(w, http.StatusBadRequest, "cleanup_invalid_scope")
		return
	}
	params := cleanup.Params{DryRun: *req.DryRun, BatchSize: req.BatchSize, MaxBatches: req.MaxBatches, StartedBy: "ops", RequestID: r.Header.Get("X-Request-ID")}
	run, err := h.cleanup.Run(r.Context(), req.TaskName, params)
	if err != nil {
		switch err.Error() {
		case "cleanup_task_not_found":
			writeError(w, http.StatusNotFound, err.Error())
		case "cleanup_already_running":
			writeError(w, http.StatusConflict, err.Error())
		case "cleanup_invalid_scope":
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusBadGateway, "cleanup_internal_error")
		}
		return
	}
	writeJSON(w, http.StatusOK, run)
}

// storageSummary intentionally reports only aggregates. It never returns file
// names, users, prompts, receipt text, or other sensitive lifecycle data.
func (h *workerOpsHandler) storageSummary(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		writeError(w, http.StatusServiceUnavailable, "cleanup_disabled")
		return
	}
	type count struct {
		Name   string     `json:"name"`
		Count  int64      `json:"count"`
		Oldest *time.Time `json:"oldest,omitempty"`
	}
	queries := []struct{ name, sql string }{{"cleanup_runs", "SELECT COUNT(*), MIN(started_at) FROM cleanup_runs"}, {"generation_jobs", "SELECT COUNT(*), MIN(created_at) FROM trip_generation_jobs"}, {"activity_events", "SELECT COUNT(*), MIN(created_at) FROM trip_activity_events"}, {"export_jobs", "SELECT COUNT(*), MIN(created_at) FROM data_export_jobs"}, {"receipt_ocr_results", "SELECT COUNT(*), MIN(created_at) FROM receipt_ocr_results"}}
	items := make([]count, 0, len(queries))
	for _, query := range queries {
		var item count
		item.Name = query.name
		if err := h.db.QueryRow(r.Context(), query.sql).Scan(&item.Count, &item.Oldest); err != nil {
			h.log.Warn("ops storage summary query failed", zap.String("category", query.name), zap.Error(err))
			continue
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": items})
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
		"version":           version.Version,
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
