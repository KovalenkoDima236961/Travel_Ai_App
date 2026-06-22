// Package handler exposes the trip HTTP endpoints. It is responsible only for
// request decoding, validation, response encoding, and status-code mapping.
package handler

import (
	"encoding/json"
	"errors"
	"fmt"
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
		r.Post("/{id}/generate", h.Generate)
	})
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

func (h *Handler) parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid trip id")
		return uuid.Nil, false
	}
	return id, true
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
	switch {
	case errors.As(err, &invalid):
		writeError(w, http.StatusBadRequest, invalid.Error())
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
