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
		if err := mountOpsRoutes(r, cfg, db, consumer, startedAt, log); err != nil {
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
