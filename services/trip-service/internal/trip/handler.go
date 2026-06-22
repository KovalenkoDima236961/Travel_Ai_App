package trip

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/validation"
)

// Handler exposes the trip HTTP endpoints. It is responsible only for request
// decoding, validation, response encoding, and status-code mapping.
type Handler struct {
	service   *Service
	validator validation.Validator
	log       *zap.Logger
}

// NewHandler constructs the trip HTTP handler.
func NewHandler(service *Service, validator validation.Validator, log *zap.Logger) *Handler {
	return &Handler{service: service, validator: validator, log: log}
}

// RegisterRoutes mounts the trip routes onto the given chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/trips", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/{id}", h.Get)
		r.Post("/{id}/generate", h.Generate)
	})
}

// Create handles POST /trips.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTripRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validator.Validate(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	created, err := h.service.Create(r.Context(), req.toInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newTripResponse(created))
}

// Get handles GET /trips/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	t, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTripResponse(t))
}

// Generate handles POST /trips/{id}/generate.
func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	t, err := h.service.Generate(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTripResponse(t))
}

func (h *Handler) parseID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid trip id")
		return uuid.Nil, false
	}
	return id, true
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
	switch {
	case errors.Is(err, ErrNotFound):
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
