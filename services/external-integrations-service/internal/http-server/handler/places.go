package handler

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	appservice "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	extobs "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/observability"
)

const (
	maxQueryLength       = 200
	maxDestinationLength = 100
)

// PlacesHandler wires place use cases to HTTP.
type PlacesHandler struct {
	svc          *appservice.PlacesService
	log          *zap.Logger
	providerName string
}

func NewPlacesHandler(svc *appservice.PlacesService, log *zap.Logger, providerName string) *PlacesHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &PlacesHandler{
		svc:          svc,
		log:          log,
		providerName: strings.ToLower(strings.TrimSpace(providerName)),
	}
}

// RegisterRoutes mounts place routes onto the given chi router.
func (h *PlacesHandler) RegisterRoutes(r chi.Router) {
	r.Get("/places/search", h.Search)
	r.Get("/places/{placeId}", h.Details)
}

// Search handles GET /places/search.
func (h *PlacesHandler) Search(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	destination := strings.TrimSpace(r.URL.Query().Get("destination"))

	if query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}
	if len(query) > maxQueryLength {
		writeError(w, http.StatusBadRequest, "query must be at most 200 characters")
		return
	}
	if len(destination) > maxDestinationLength {
		writeError(w, http.StatusBadRequest, "destination must be at most 100 characters")
		return
	}

	items, err := h.svc.Search(r.Context(), query, destination)
	if err != nil {
		extobs.RecordProviderRequest(h.providerName, "place_search", "error", time.Since(start))
		extobs.RecordProviderFailure(h.providerName, "place_search", "provider_error")
		if writeProviderLimitError(w, err) {
			return
		}
		h.log.Warn("place search failed", zap.Int("query_length", len(query)), zap.String("destination", destination), zap.Error(err))
		writeError(w, http.StatusInternalServerError, "place search failed")
		return
	}
	extobs.RecordProviderRequest(h.providerName, "place_search", "success", time.Since(start))

	h.log.Info("place search completed",
		zap.Int("query_length", len(query)),
		zap.String("destination", destination),
		zap.Int("result_count", len(items)),
		zap.String("provider", h.providerName),
	)

	writeJSON(w, http.StatusOK, SearchPlacesResponse{Items: items})
}

// Details handles GET /places/{placeId}.
func (h *PlacesHandler) Details(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	placeID := strings.TrimSpace(chi.URLParam(r, "placeId"))
	if placeID == "" {
		writeError(w, http.StatusBadRequest, "placeId is required")
		return
	}

	place, err := h.svc.Details(r.Context(), placeID)
	if err != nil {
		extobs.RecordProviderRequest(h.providerName, "place_details", "error", time.Since(start))
		extobs.RecordProviderFailure(h.providerName, "place_details", "provider_error")
		if writeProviderLimitError(w, err) {
			return
		}
		h.log.Warn("place details failed", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "place details failed")
		return
	}
	if place == nil {
		extobs.RecordProviderRequest(h.providerName, "place_details", "not_found", time.Since(start))
		writeError(w, http.StatusNotFound, "place not found")
		return
	}
	extobs.RecordProviderRequest(h.providerName, "place_details", "success", time.Since(start))

	writeJSON(w, http.StatusOK, place)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
