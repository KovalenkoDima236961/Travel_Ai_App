// Package handler exposes user profile and preferences HTTP endpoints.
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/dataexport"
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
		r.Get("/preferences/completeness", h.GetPreferenceCompleteness)
		r.Post("/export", h.CreateAccountExport)
		r.Get("/export/{exportId}", h.GetAccountExport)
		r.Get("/export/{exportId}/download", h.DownloadAccountExport)
		r.Post("/account-cleanup/request-deletion", h.RequestAccountCleanup)
	})
	// Short aliases match the portability API contract while the established
	// /users/me routes remain the canonical browser API.
	r.Post("/me/export", h.CreateAccountExport)
	r.Get("/me/export/{exportId}", h.GetAccountExport)
	r.Get("/me/export/{exportId}/download", h.DownloadAccountExport)
	r.Post("/me/account-cleanup/request-deletion", h.RequestAccountCleanup)
}

func (h *Handler) CreateAccountExport(w http.ResponseWriter, r *http.Request) {
	var request service.AccountExportRequest
	if err := decodeJSON(r, &request); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	job, err := h.svc.CreateAccountExport(r.Context(), request)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, accountExportResponse(job))
}

func (h *Handler) GetAccountExport(w http.ResponseWriter, r *http.Request) {
	exportID, ok := parseAccountExportID(w, r)
	if !ok {
		return
	}
	job, err := h.svc.GetAccountExport(r.Context(), exportID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, accountExportResponse(job))
}

func (h *Handler) DownloadAccountExport(w http.ResponseWriter, r *http.Request) {
	exportID, ok := parseAccountExportID(w, r)
	if !ok {
		return
	}
	file, err := h.svc.OpenAccountExport(r.Context(), exportID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	defer file.Reader.Close()
	w.Header().Set("Content-Type", file.ContentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": file.Filename}))
	w.Header().Set("Cache-Control", "private, no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if file.SizeBytes > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(file.SizeBytes, 10))
	}
	_, _ = io.Copy(w, file.Reader)
}

func (h *Handler) RequestAccountCleanup(w http.ResponseWriter, r *http.Request) {
	var request service.AccountCleanupRequest
	if err := decodeJSON(r, &request); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.RequestAccountCleanup(r.Context(), request); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "received", "message": "Account deletion is not automatic in this version. Your request has been recorded."})
}

func parseAccountExportID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "exportId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid export id")
		return uuid.Nil, false
	}
	return id, true
}

func accountExportResponse(job *dataexport.Job) map[string]any {
	result := map[string]any{"exportId": job.ID.String(), "status": job.Status, "createdAt": job.CreatedAt}
	if job.FileName != nil {
		result["fileName"] = *job.FileName
	}
	if job.SizeBytes != nil {
		result["sizeBytes"] = *job.SizeBytes
	}
	if job.ChecksumSHA256 != nil {
		result["checksumSha256"] = *job.ChecksumSHA256
	}
	if job.ExpiresAt != nil {
		result["expiresAt"] = *job.ExpiresAt
	}
	if job.ErrorCode != nil {
		result["errorCode"] = *job.ErrorCode
	}
	if job.ErrorMessageSafe != nil {
		result["errorMessageSafe"] = *job.ErrorMessageSafe
	}
	if job.Status == dataexport.Completed {
		result["downloadUrl"] = "/users/me/export/" + job.ID.String() + "/download"
	}
	return result
}

// GetPreferenceCompleteness handles GET /users/me/preferences/completeness.
func (h *Handler) GetPreferenceCompleteness(w http.ResponseWriter, r *http.Request) {
	result, err := h.svc.GetPreferenceCompleteness(r.Context())
	if err != nil {
		recordUserPreferencesRequest("completeness", "error")
		h.writeServiceError(w, err)
		return
	}
	recordUserPreferencesRequest("completeness", "success")
	writeJSON(w, http.StatusOK, result)
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
			Code:   "validation_error",
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
	Code   string            `json:"code,omitempty"`
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
