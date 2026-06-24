package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/response"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/validation"
)

// Handler wires the trip use case to HTTP.
type Handler struct {
	svc       *service.Service
	validator validation.Validator
	log       *zap.Logger
}

// New constructs the trip HTTP handler.
func New(svc *service.Service, validator validation.Validator, log *zap.Logger) *Handler {
	return &Handler{svc: svc, validator: validator, log: log}
}

// RegisterRoutes mounts the trip routes onto the given chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/trips", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/", h.List)
		r.Get("/{id}", h.Get)
		r.Get("/{id}/share", h.GetShare)
		r.Post("/{id}/share", h.CreateShare)
		r.Delete("/{id}/share", h.DisableShare)
		r.Post("/{id}/generate", h.Generate)
		r.Put("/{id}/itinerary", h.UpdateItinerary)
		r.Get("/{id}/itinerary/versions", h.ListItineraryVersions)
		r.Get("/{id}/itinerary/versions/{versionId}", h.GetItineraryVersion)
		r.Post("/{id}/itinerary/versions/{versionId}/restore", h.RestoreItineraryVersion)
		r.Post("/{id}/itinerary/days/{dayNumber}/regenerate", h.RegenerateDay)
		r.Post("/{id}/itinerary/days/{dayNumber}/items/{itemIndex}/regenerate", h.RegenerateItem)
	})
}

// RegisterPublicRoutes mounts unauthenticated read-only public routes.
func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	r.Get("/public/trips/{shareToken}", h.GetPublicTrip)
}

// Create handles POST /trips.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req request.CreateTrip
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validator.Validate(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	created, err := h.svc.Create(r.Context(), req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, response.NewTrip(created))
}

// List handles GET /trips?limit=&offset=. Pagination defaults and bounds are
// enforced by the service; the handler only parses the query parameters.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, ok := parseQueryInt(w, r, "limit")
	if !ok {
		return
	}
	offset, ok := parseQueryInt(w, r, "offset")
	if !ok {
		return
	}

	trips, appliedLimit, appliedOffset, err := h.svc.List(r.Context(), limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewListTrips(trips, appliedLimit, appliedOffset))
}

// Get handles GET /trips/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	t, err := h.svc.Get(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTrip(t))
}

// GetShare handles GET /trips/{id}/share.
func (h *Handler) GetShare(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	share, err := h.svc.GetTripShare(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTripShareInfo(share))
}

// CreateShare handles POST /trips/{id}/share.
func (h *Handler) CreateShare(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	share, err := h.svc.CreateOrEnableTripShare(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTripShareInfo(share))
}

// DisableShare handles DELETE /trips/{id}/share.
func (h *Handler) DisableShare(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	if err := h.svc.DisableTripShare(r.Context(), id); err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GetPublicTrip handles GET /public/trips/{shareToken}.
func (h *Handler) GetPublicTrip(w http.ResponseWriter, r *http.Request) {
	shareToken := strings.TrimSpace(chi.URLParam(r, "shareToken"))

	t, share, err := h.svc.GetPublicTripByShareToken(r.Context(), shareToken)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			writeError(w, http.StatusNotFound, "shared trip not found")
			return
		}
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewPublicTrip(t, share.CreatedAt))
}

// Generate handles POST /trips/{id}/generate.
func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	t, err := h.svc.Generate(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTrip(t))
}

// UpdateItinerary handles PUT /trips/{id}/itinerary.
func (h *Handler) UpdateItinerary(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req request.UpdateTripItinerary
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	t, err := h.svc.UpdateItinerary(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTrip(t))
}

// RegenerateDay handles POST /trips/{id}/itinerary/days/{dayNumber}/regenerate.
func (h *Handler) RegenerateDay(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	dayNumber, ok := parseURLInt(w, r, "dayNumber")
	if !ok {
		return
	}

	req, ok := decodeRegenerateRequest(w, r)
	if !ok {
		return
	}

	t, err := h.svc.RegenerateDay(r.Context(), id, dayNumber, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTrip(t))
}

// RegenerateItem handles POST
// /trips/{id}/itinerary/days/{dayNumber}/items/{itemIndex}/regenerate.
func (h *Handler) RegenerateItem(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	dayNumber, ok := parseURLInt(w, r, "dayNumber")
	if !ok {
		return
	}
	itemIndex, ok := parseURLInt(w, r, "itemIndex")
	if !ok {
		return
	}

	req, ok := decodeRegenerateRequest(w, r)
	if !ok {
		return
	}

	t, err := h.svc.RegenerateItem(r.Context(), id, dayNumber, itemIndex, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTrip(t))
}

// ListItineraryVersions handles GET /trips/{id}/itinerary/versions.
func (h *Handler) ListItineraryVersions(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	limit, ok := parseQueryInt(w, r, "limit")
	if !ok {
		return
	}
	offset, ok := parseQueryInt(w, r, "offset")
	if !ok {
		return
	}

	versions, appliedLimit, appliedOffset, err := h.svc.ListItineraryVersions(r.Context(), id, limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewListItineraryVersions(versions, appliedLimit, appliedOffset))
}

// GetItineraryVersion handles GET /trips/{id}/itinerary/versions/{versionId}.
func (h *Handler) GetItineraryVersion(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	versionID, ok := parseUUIDParam(w, r, "versionId", "invalid version id")
	if !ok {
		return
	}

	version, err := h.svc.GetItineraryVersion(r.Context(), id, versionID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewItineraryVersionDetail(version))
}

// RestoreItineraryVersion handles
// POST /trips/{id}/itinerary/versions/{versionId}/restore.
func (h *Handler) RestoreItineraryVersion(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	versionID, ok := parseUUIDParam(w, r, "versionId", "invalid version id")
	if !ok {
		return
	}

	t, err := h.svc.RestoreItineraryVersion(r.Context(), id, versionID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTrip(t))
}

func (h *Handler) parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return parseUUIDParam(w, r, "id", "invalid trip id")
}

func parseUUIDParam(w http.ResponseWriter, r *http.Request, key, errorMessage string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, key))
	if err != nil {
		writeError(w, http.StatusBadRequest, errorMessage)
		return uuid.Nil, false
	}
	return id, true
}

func parseURLInt(w http.ResponseWriter, r *http.Request, key string) (int, bool) {
	raw := strings.TrimSpace(chi.URLParam(r, key))
	v, err := strconv.Atoi(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid %s", key))
		return 0, false
	}
	return v, true
}

func decodeRegenerateRequest(w http.ResponseWriter, r *http.Request) (request.RegenerateItineraryPart, bool) {
	var req request.RegenerateItineraryPart
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return request.RegenerateItineraryPart{}, false
	}
	return req, true
}

// parseQueryInt reads an integer query parameter. A missing/empty value yields 0
// (so the service can apply its default); a non-integer value is a 400.
func parseQueryInt(w http.ResponseWriter, r *http.Request, key string) (int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, true
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid %s", key))
		return 0, false
	}
	return v, true
}

func (h *Handler) writeValidationError(w http.ResponseWriter, err error) {
	var ve *validation.ValidationError
	if errors.As(err, &ve) {
		writeJSON(w, http.StatusBadRequest, errorBody{
			Error:  "validation failed",
			Fields: ve.Fields(),
		})
		return
	}
	writeError(w, http.StatusBadRequest, err.Error())
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	var invalid *apperrs.InvalidInputError
	var dependency *apperrs.DependencyError
	switch {
	case errors.As(err, &invalid):
		writeError(w, http.StatusBadRequest, invalid.Error())
	case errors.As(err, &dependency):
		writeError(w, http.StatusBadGateway, dependency.Error())
	case errors.Is(err, domainerrs.ErrNotFound):
		writeError(w, http.StatusNotFound, "trip not found")
	default:
		h.log.Error("unhandled service error", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

// errorBody is the uniform error envelope. Fields is populated only for
// validation failures.
type errorBody struct {
	Error  string            `json:"error"`
	Fields map[string]string `json:"fields,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorBody{Error: message})
}
