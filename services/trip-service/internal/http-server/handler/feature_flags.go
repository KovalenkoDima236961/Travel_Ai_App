package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/featureflags"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/observability"
)

// EnableFeatureFlags wires the central, database-backed runtime controls. A
// nil service keeps older focused handler tests backwards compatible; app
// wiring always supplies the service in normal operation.
func (h *Handler) EnableFeatureFlags(service *featureflags.Service) *Handler {
	h.featureFlags = service
	return h
}

func (h *Handler) PublicFeatureFlags(w http.ResponseWriter, r *http.Request) {
	if h.featureFlags == nil {
		writeJSON(w, http.StatusOK, map[string]any{"flags": map[string]bool{}, "environment": "local"})
		return
	}
	flags, err := h.featureFlags.ListFlags(r.Context(), "", true)
	if err != nil {
		h.writeFeatureFlagError(w, err)
		return
	}
	values := make(map[string]bool, len(flags))
	var updatedAt *time.Time
	for _, flag := range flags {
		values[flag.Key] = flag.Value
		if flag.Metadata.UpdatedAt != nil && (updatedAt == nil || flag.Metadata.UpdatedAt.After(*updatedAt)) {
			updated := *flag.Metadata.UpdatedAt
			updatedAt = &updated
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"flags": values, "environment": h.featureFlagEnvironment(), "updatedAt": updatedAt,
	})
}

func (h *Handler) InternalEvaluateFeatureFlag(w http.ResponseWriter, r *http.Request) {
	if h.featureFlags == nil {
		writeFeatureDisabled(w, r, "runtime_controls_unavailable")
		return
	}
	var request struct {
		Key         string `json:"key"`
		Environment string `json:"environment"`
		ServiceName string `json:"serviceName"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	flag, err := h.featureFlags.GetFlag(r.Context(), strings.TrimSpace(request.Key), featureflags.EvaluationContext{
		Environment: request.Environment, ServiceName: request.ServiceName, RequestID: observability.RequestIDFromContext(r.Context()),
	})
	if err != nil {
		h.writeFeatureFlagError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, flag)
}

func (h *Handler) OpsListFeatureFlags(w http.ResponseWriter, r *http.Request) {
	flags, err := h.featureFlags.ListFlags(r.Context(), "", false)
	if err != nil {
		h.writeFeatureFlagError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"flags": flags, "environment": h.featureFlagEnvironment()})
}

func (h *Handler) OpsGetFeatureFlag(w http.ResponseWriter, r *http.Request) {
	flag, err := h.featureFlags.GetFlag(r.Context(), chi.URLParam(r, "key"), featureflags.EvaluationContext{RequestID: observability.RequestIDFromContext(r.Context())})
	if err != nil {
		h.writeFeatureFlagError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, flag)
}

func (h *Handler) OpsUpdateFeatureFlag(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Value  *bool  `json:"value"`
		Reason string `json:"reason"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil || request.Value == nil {
		writeError(w, http.StatusBadRequest, "value must be a boolean")
		return
	}
	user, _ := auth.UserFromContext(r.Context())
	flag, err := h.featureFlags.UpdateGlobal(r.Context(), chi.URLParam(r, "key"), "", *request.Value, request.Reason, observability.RequestIDFromContext(r.Context()), &user.ID)
	if err != nil {
		h.writeFeatureFlagError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, flag)
}

func (h *Handler) OpsResetFeatureFlag(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Reason string `json:"reason"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	user, _ := auth.UserFromContext(r.Context())
	if err := h.featureFlags.ResetGlobal(r.Context(), chi.URLParam(r, "key"), "", request.Reason, observability.RequestIDFromContext(r.Context()), &user.ID); err != nil {
		h.writeFeatureFlagError(w, err)
		return
	}
	flag, err := h.featureFlags.GetFlag(r.Context(), chi.URLParam(r, "key"), featureflags.EvaluationContext{RequestID: observability.RequestIDFromContext(r.Context())})
	if err != nil {
		h.writeFeatureFlagError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, flag)
}

func (h *Handler) OpsListFeatureFlagAudit(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 200 {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 200")
			return
		}
		limit = parsed
	}
	events, err := h.featureFlags.ListAudit(r.Context(), chi.URLParam(r, "key"), limit)
	if err != nil {
		h.writeFeatureFlagError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (h *Handler) gateFeature(flag string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.featureFlags == nil {
			next(w, r)
			return
		}
		enabled, metadata, err := h.featureFlags.IsEnabled(r.Context(), flag, featureflags.EvaluationContext{RequestID: observability.RequestIDFromContext(r.Context()), ServiceName: "trip-service"})
		if err != nil && h.log != nil {
			h.log.Warn("feature flag evaluation failed while serving request", append([]zap.Field{zap.String("flag", flag)}, observability.RequestIDFields(r.Context())...)...)
		}
		if enabled {
			next(w, r)
			return
		}
		route := observability.RoutePattern(r)
		featureflags.RecordDisabledRequest(flag, route)
		if h.log != nil {
			h.log.Info("feature disabled request blocked", append([]zap.Field{zap.String("flag", flag), zap.String("route", route), zap.String("source", metadata.Source)}, observability.RequestIDFields(r.Context())...)...)
		}
		writeFeatureDisabled(w, r, flag)
	}
}

func (h *Handler) featureMiddleware(flag string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return h.gateFeature(flag, next.ServeHTTP)
	}
}

func (h *Handler) featureFlagEnvironment() string {
	if h.featureFlags == nil {
		return "local"
	}
	return h.featureFlags.Environment()
}

func (h *Handler) writeFeatureFlagError(w http.ResponseWriter, err error) {
	if errors.Is(err, featureflags.ErrUnknownFlag) {
		writeError(w, http.StatusNotFound, "feature flag not found")
		return
	}
	writeError(w, http.StatusBadRequest, err.Error())
}

func writeFeatureDisabled(w http.ResponseWriter, r *http.Request, flag string) {
	writeJSON(w, http.StatusForbidden, map[string]any{"error": map[string]any{
		"code": "feature_disabled", "message": "This feature is currently disabled.",
		"details": map[string]string{"feature": flag}, "requestId": observability.RequestIDFromContext(r.Context()),
	}})
}
