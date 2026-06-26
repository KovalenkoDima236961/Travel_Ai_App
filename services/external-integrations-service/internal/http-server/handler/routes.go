package handler

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	appservice "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

const (
	maxStopNameLength = 200
	minRouteStops     = 2
	maxRouteStops     = 25
)

// RoutesHandler wires route-estimation use cases to HTTP.
type RoutesHandler struct {
	svc            *appservice.RoutesService
	log            *zap.Logger
	supportedModes map[string]struct{}
	modesLabel     string
}

// NewRoutesHandler builds the handler. The set of accepted travel modes depends
// on the configured provider: the mock provider estimates walking only, while
// the real ORS provider also supports driving and cycling. This keeps the API
// honest about what the active provider can actually estimate.
func NewRoutesHandler(svc *appservice.RoutesService, log *zap.Logger, providerName string) *RoutesHandler {
	if log == nil {
		log = zap.NewNop()
	}
	modes := supportedRouteModesFor(providerName)
	return &RoutesHandler{
		svc:            svc,
		log:            log,
		supportedModes: modes,
		modesLabel:     sortedModeLabel(modes),
	}
}

// supportedRouteModesFor returns the closed set of travel modes the active
// provider can estimate. Unknown providers default to walking only.
func supportedRouteModesFor(providerName string) map[string]struct{} {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case config.RouteProviderORS:
		return map[string]struct{}{"walking": {}, "driving": {}, "cycling": {}}
	default:
		return map[string]struct{}{"walking": {}}
	}
}

func sortedModeLabel(modes map[string]struct{}) string {
	names := make([]string, 0, len(modes))
	for mode := range modes {
		names = append(names, mode)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// RegisterRoutes mounts route-estimation routes onto the given chi router.
func (h *RoutesHandler) RegisterRoutes(r chi.Router) {
	r.Post("/routes/estimate", h.Estimate)
}

// Estimate handles POST /routes/estimate.
func (h *RoutesHandler) Estimate(w http.ResponseWriter, r *http.Request) {
	var req entity.RouteEstimateRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body must be valid JSON")
		return
	}

	req.Mode = strings.ToLower(strings.TrimSpace(req.Mode))
	if message, ok := h.validateRouteEstimateRequest(req); !ok {
		writeError(w, http.StatusBadRequest, message)
		return
	}

	estimate, err := h.svc.EstimateRoute(r.Context(), req)
	if err != nil {
		// Validation already passed, so any error here is an upstream provider
		// failure. Return a safe, generic provider-unavailable response.
		h.log.Warn("route estimate failed", zap.String("mode", req.Mode), zap.Int("stop_count", len(req.Stops)), zap.Error(err))
		writeError(w, http.StatusBadGateway, "route_provider_unavailable")
		return
	}

	writeJSON(w, http.StatusOK, estimate)
}

// validateRouteEstimateRequest returns a human-readable message and false when
// the request is invalid. The mode is expected to be normalised by the caller.
func (h *RoutesHandler) validateRouteEstimateRequest(req entity.RouteEstimateRequest) (string, bool) {
	if req.Mode == "" {
		return "mode is required", false
	}
	if _, ok := h.supportedModes[req.Mode]; !ok {
		return "unsupported mode: supported modes are " + h.modesLabel, false
	}
	if len(req.Stops) < minRouteStops {
		return "at least 2 stops are required", false
	}
	if len(req.Stops) > maxRouteStops {
		return "at most 25 stops are supported", false
	}

	for _, stop := range req.Stops {
		if strings.TrimSpace(stop.Name) == "" {
			return "each stop requires a name", false
		}
		if len(stop.Name) > maxStopNameLength {
			return "stop name must be at most 200 characters", false
		}
		if stop.Latitude < -90 || stop.Latitude > 90 {
			return "stop latitude must be between -90 and 90", false
		}
		if stop.Longitude < -180 || stop.Longitude > 180 {
			return "stop longitude must be between -180 and 180", false
		}
	}

	return "", true
}
