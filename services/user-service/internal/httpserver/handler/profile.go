// Package handler exposes user profile and preferences HTTP endpoints.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/service"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/httpserver/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/httpserver/dto/response"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/pkg/validation"
)

// Handler wires user use cases to HTTP.
type Handler struct {
	svc       *service.Service
	validator validation.Validator
	log       *zap.Logger
}

// New constructs the user HTTP handler.
func New(svc *service.Service, validator validation.Validator, log *zap.Logger) *Handler {
	return &Handler{svc: svc, validator: validator, log: log}
}

// RegisterRoutes mounts the user routes onto the given chi router.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/users/me", func(r chi.Router) {
		r.Get("/profile", h.GetProfile)
		r.Put("/profile", h.UpdateProfile)
		r.Get("/preferences", h.GetPreferences)
		r.Patch("/preferences", h.PatchPreferences)
	})
}

// GetProfile handles GET /users/me/profile.
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	profile, err := h.svc.GetProfile(r.Context())
	if err != nil {
		recordUserProfileRequest("get", "error")
		h.writeServiceError(w, err)
		return
	}
	recordUserProfileRequest("get", "success")
	writeJSON(w, http.StatusOK, response.NewProfile(profile))
}

// UpdateProfile handles PUT /users/me/profile.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var req request.UpdateProfile
	if err := decodeJSON(r, &req); err != nil {
		recordUserProfileRequest("update", "invalid_request")
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		recordUserProfileRequest("update", "validation_error")
		h.writeValidationError(w, err)
		return
	}

	profile, err := h.svc.UpdateProfile(r.Context(), req.ToInput())
	if err != nil {
		recordUserProfileRequest("update", "error")
		h.writeServiceError(w, err)
		return
	}
	recordUserProfileRequest("update", "success")
	writeJSON(w, http.StatusOK, response.NewProfile(profile))
}

// GetPreferences handles GET /users/me/preferences.
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	preferences, err := h.svc.GetPreferences(r.Context())
	if err != nil {
		recordUserPreferencesRequest("get", "error")
		h.writeServiceError(w, err)
		return
	}
	recordUserPreferencesRequest("get", "success")
	writeJSON(w, http.StatusOK, response.NewPreferences(preferences))
}

// PatchPreferences handles PATCH /users/me/preferences.
func (h *Handler) PatchPreferences(w http.ResponseWriter, r *http.Request) {
	var req request.PatchPreferences
	if err := decodeJSON(r, &req); err != nil {
		recordUserPreferencesRequest("patch", "invalid_request")
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		recordUserPreferencesRequest("patch", "validation_error")
		h.writeValidationError(w, err)
		return
	}

	preferences, err := h.svc.PatchPreferences(r.Context(), req.ToInput())
	if err != nil {
		recordUserPreferencesRequest("patch", "error")
		h.writeServiceError(w, err)
		return
	}
	recordUserPreferencesRequest("patch", "success")
	writeJSON(w, http.StatusOK, response.NewPreferences(preferences))
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
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
		writeError(w, http.StatusNotFound, "user resource not found")
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
