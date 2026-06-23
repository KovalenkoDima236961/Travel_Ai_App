package httpserver

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// ReadinessHandler checks dependencies required to serve place traffic. v1 has
// no database or external network dependency when running the mock provider.
type ReadinessHandler struct {
	log *zap.Logger
}

// NewReadinessHandler creates the /ready handler.
func NewReadinessHandler(log *zap.Logger) *ReadinessHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &ReadinessHandler{log: log}
}

func (h *ReadinessHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	startedAt := time.Now()
	checks := map[string]string{"place_provider": "ok"}

	h.log.Info(
		"readiness check completed",
		zap.String("status", "ready"),
		zap.Any("checks", checks),
		zap.Duration("duration", time.Since(startedAt)),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "ready",
		"checks": checks,
	})
}
