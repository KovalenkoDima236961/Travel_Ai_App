package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

type readinessDB interface {
	Ping(ctx context.Context) error
}

// ReadinessHandler checks dependencies required to serve the full trip flow.
type ReadinessHandler struct {
	db                   readinessDB
	generatorMode        string
	aiPlanningServiceURL string
	client               *http.Client
	log                  *zap.Logger
}

// NewReadinessHandler creates the /ready handler. The AI Planning Service check
// is active only when the Trip Service is configured with the HTTP generator.
func NewReadinessHandler(
	db readinessDB,
	generatorMode string,
	aiPlanningServiceURL string,
	timeout time.Duration,
	log *zap.Logger,
) *ReadinessHandler {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if log == nil {
		log = zap.NewNop()
	}

	return &ReadinessHandler{
		db:                   db,
		generatorMode:        strings.ToLower(strings.TrimSpace(generatorMode)),
		aiPlanningServiceURL: strings.TrimSpace(aiPlanningServiceURL),
		client:               &http.Client{Timeout: timeout},
		log:                  log,
	}
}

func (h *ReadinessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startedAt := time.Now()
	checks := map[string]string{}
	ready := true

	if h.db == nil {
		checks["postgres"] = "failed"
		ready = false
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), h.client.Timeout)
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

	if h.generatorMode == "http" {
		if err := h.checkAIPlanningService(r.Context()); err != nil {
			checks["aiPlanningService"] = "failed"
			ready = false
			h.log.Warn("ai planning service readiness check failed", zap.Error(err))
		} else {
			checks["aiPlanningService"] = "ok"
		}
	}

	status := "ready"
	httpStatus := http.StatusOK
	if !ready {
		status = "not_ready"
		httpStatus = http.StatusServiceUnavailable
	}

	h.log.Info(
		"readiness check completed",
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

func (h *ReadinessHandler) checkAIPlanningService(ctx context.Context) error {
	if h.aiPlanningServiceURL == "" {
		return fmt.Errorf("AI_PLANNING_SERVICE_URL is empty")
	}

	endpoint, err := url.JoinPath(strings.TrimRight(h.aiPlanningServiceURL, "/"), "health")
	if err != nil {
		return fmt.Errorf("build ai planning health URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create ai planning health request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("call ai planning health endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("ai planning health endpoint returned HTTP %d", resp.StatusCode)
	}

	return nil
}
