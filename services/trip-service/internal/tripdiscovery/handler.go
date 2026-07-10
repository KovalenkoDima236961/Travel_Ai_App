package tripdiscovery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
	tripresponse "github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/response"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/planningconstraints"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/trip-discovery/suggestions", h.Discover)
	r.Post("/trip-discovery/surprise-me", h.Surprise)
	r.Get("/trip-discovery/sessions", h.List)
	r.Get("/trip-discovery/sessions/{sessionId}", h.Get)
	r.Get("/trip-discovery/sessions/{sessionId}/votes", h.GetSuggestionVotes)
	r.Post("/trip-discovery/sessions/{sessionId}/suggestions/{suggestionId}/vote", h.VoteSuggestion)
	r.Post("/trip-discovery/{sessionId}/refine", h.Refine)
	r.Post(
		"/trip-discovery/{sessionId}/suggestions/{suggestionId}/create-trip",
		h.CreateTrip,
	)
}

func (h *Handler) Discover(w http.ResponseWriter, r *http.Request) {
	var input DiscoverInput
	if !decodeJSON(w, r, &input) {
		return
	}
	session, err := h.service.Discover(r.Context(), input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, session)
}

func (h *Handler) Surprise(w http.ResponseWriter, r *http.Request) {
	var input DiscoverInput
	if !decodeJSON(w, r, &input) {
		return
	}
	session, err := h.service.Surprise(r.Context(), input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, session)
}

func (h *Handler) Refine(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := parseID(w, r, "sessionId")
	if !ok {
		return
	}
	var input RefineInput
	if !decodeJSON(w, r, &input) {
		return
	}
	session, err := h.service.Refine(r.Context(), sessionID, input)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, session)
}

func (h *Handler) CreateTrip(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := parseID(w, r, "sessionId")
	if !ok {
		return
	}
	var input CreateTripInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := h.service.CreateTrip(
		r.Context(),
		sessionID,
		chi.URLParam(r, "suggestionId"),
		input,
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	body := map[string]any{
		"trip":          tripresponse.NewTrip(result.Trip),
		"generationJob": nil,
	}
	if result.GenerationJob != nil {
		body["generationJob"] = generationjobs.NewJobResponse(result.GenerationJob)
	}
	writeJSON(w, http.StatusCreated, body)
}

func (h *Handler) VoteSuggestion(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := parseID(w, r, "sessionId")
	if !ok {
		return
	}
	var input VoteSuggestionInput
	if !decodeJSON(w, r, &input) {
		return
	}
	result, err := h.service.VoteSuggestion(
		r.Context(),
		sessionID,
		chi.URLParam(r, "suggestionId"),
		input,
	)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetSuggestionVotes(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := parseID(w, r, "sessionId")
	if !ok {
		return
	}
	result, err := h.service.SuggestionVotes(r.Context(), sessionID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := parseID(w, r, "sessionId")
	if !ok {
		return
	}
	session, err := h.service.Get(r.Context(), sessionID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			writeError(w, http.StatusBadRequest, "limit must be a positive integer")
			return
		}
		limit = parsed
	}
	sessions, err := h.service.List(r.Context(), limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": sessions, "limit": min(limit, 100)})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func parseID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+name)
		return uuid.Nil, false
	}
	return id, true
}

func writeServiceError(w http.ResponseWriter, err error) {
	var invalid *apperrs.InvalidInputError
	var conflict *apperrs.ConflictError
	var dependency *apperrs.DependencyError
	var planningBlocking *planningconstraints.BlockingError
	switch {
	case errors.As(err, &planningBlocking):
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":       "planning_constraints_blocked",
			"message":     planningBlocking.Error(),
			"constraints": planningBlocking.Constraints,
			"warnings":    planningBlocking.Constraints.Warnings,
			"blockers":    planningBlocking.Constraints.Blockers,
		})
	case errors.As(err, &invalid):
		writeError(w, http.StatusBadRequest, invalid.Error())
	case errors.Is(err, apperrs.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, domainerrs.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.As(err, &conflict):
		writeError(w, http.StatusConflict, conflict.Error())
	case errors.As(err, &dependency):
		writeJSON(w, http.StatusBadGateway, map[string]string{
			"error":   "trip_discovery_failed",
			"message": dependency.Error(),
		})
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
