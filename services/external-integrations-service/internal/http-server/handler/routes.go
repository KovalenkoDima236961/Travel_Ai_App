package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	appservice "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

const (
	maxStopNameLength = 200
	minRouteStops     = 2
	maxRouteStops     = 25
)

// supportedRouteModes is the closed set of travel modes the v1 routing API
// accepts. Walking is the only mode the mock provider estimates today.
var supportedRouteModes = map[string]struct{}{
	"walking": {},
}

// RoutesHandler wires route-estimation use cases to HTTP.
type RoutesHandler struct {
	svc *appservice.RoutesService
	log *zap.Logger
}

func NewRoutesHandler(svc *appservice.RoutesService, log *zap.Logger) *RoutesHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &RoutesHandler{svc: svc, log: log}
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
	if message, ok := validateRouteEstimateRequest(req); !ok {
		writeError(w, http.StatusBadRequest, message)
		return
	}

	estimate, err := h.svc.EstimateRoute(r.Context(), req)
	if err != nil {
		h.log.Warn("route estimate failed", zap.String("mode", req.Mode), zap.Int("stop_count", len(req.Stops)), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "route estimate failed")
		return
	}

	writeJSON(w, http.StatusOK, estimate)
}

// validateRouteEstimateRequest returns a human-readable message and false when
// the request is invalid. The mode is expected to be normalised by the caller.
func validateRouteEstimateRequest(req entity.RouteEstimateRequest) (string, bool) {
	if req.Mode == "" {
		return "mode is required", false
	}
	if _, ok := supportedRouteModes[req.Mode]; !ok {
		return "unsupported mode: only walking is supported", false
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
