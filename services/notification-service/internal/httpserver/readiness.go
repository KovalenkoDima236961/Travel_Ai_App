package httpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type readinessDB interface {
	Ping(ctx context.Context) error
}

// ReadinessHandler checks dependencies required to serve the auth flow.
type ReadinessHandler struct {
	db  readinessDB
	log *zap.Logger
}

func NewReadinessHandler(db readinessDB, log *zap.Logger) *ReadinessHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &ReadinessHandler{db: db, log: log}
}

func (h *ReadinessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	checks := map[string]string{}
	ready := true

	if h.db == nil {
		checks["postgres"] = "failed"
		ready = false
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		err := h.db.Ping(ctx)
		cancel()
		if err != nil {
			checks["postgres"] = "failed"
			ready = false
			h.log.Warn("postgres readiness check failed", zap.Error(err))
		} else {
			checks["postgres"] = "ok"
		}
	}

	status := "ready"
	httpStatus := http.StatusOK
	if !ready {
		status = "not_ready"
		httpStatus = http.StatusServiceUnavailable
	}

	h.log.Info("readiness check completed",
		zap.String("status", status),
		zap.Any("checks", checks),
		zap.Duration("duration", time.Since(startedAt)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": status,
		"checks": checks,
	})
}
