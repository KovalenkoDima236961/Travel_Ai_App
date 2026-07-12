package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/response"
)

func (h *Handler) GetTripAvailability(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	result, err := h.svc.GetTripAvailability(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.TripAvailability(result))
}

func (h *Handler) UpsertMyTripAvailability(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.UpsertTripAvailability
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.UpsertMyTripAvailability(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.TripAvailabilityResponse(result))
}

func (h *Handler) DeleteMyTripAvailability(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteMyTripAvailability(r.Context(), id); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) GetTripDateOptions(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	input, ok := parseDateOptionsQuery(w, r)
	if !ok {
		return
	}
	result, err := h.svc.GetTripDateOptions(r.Context(), id, input.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.DateOptionsResponse(result))
}

func (h *Handler) GenerateTripDateOptions(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.GenerateDateOptions
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.GenerateTripDateOptions(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.DateOptionsResponse(result))
}

func (h *Handler) ApplyTripDateOption(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	optionID := chi.URLParam(r, "optionId")
	var req request.ApplyDateOption
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.ApplyTripDateOption(r.Context(), id, optionID, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	var jobResponse *generationjobs.JobResponse
	if req.RegenerateItinerary && h.generationJobs != nil {
		job, err := h.generationJobs.Create(r.Context(), id, generationjobs.CreateRequest{
			JobType:                   entity.GenerationJobTypeFullGeneration,
			ExpectedItineraryRevision: &result.ExpectedItineraryRevision,
		})
		if err != nil {
			h.writeGenerationJobError(w, err)
			return
		}
		resp := generationjobs.NewJobResponse(job)
		jobResponse = &resp
	}
	writeJSON(w, http.StatusOK, response.NewApplyDateOptionResponse(result, jobResponse))
}

func (h *Handler) CreateDateOptionsPoll(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.CreateDateOptionsPoll
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	info, err := h.svc.CreateDateOptionsPoll(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response.NewTripPoll(info))
}

func (h *Handler) RequestTripAvailability(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.RequestAvailability
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	summary, err := h.svc.RequestTripAvailability(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.TripAvailabilitySummary(summary))
}

func parseDateOptionsQuery(w http.ResponseWriter, r *http.Request) (request.GenerateDateOptions, bool) {
	minDays, ok := parseOptionalQueryInt(w, r, "minDays")
	if !ok {
		return request.GenerateDateOptions{}, false
	}
	maxDays, ok := parseOptionalQueryInt(w, r, "maxDays")
	if !ok {
		return request.GenerateDateOptions{}, false
	}
	limit, ok := parseQueryInt(w, r, "limit")
	if !ok {
		return request.GenerateDateOptions{}, false
	}
	preferWeekends, ok := parseOptionalQueryBool(w, r, "preferWeekends")
	if !ok {
		return request.GenerateDateOptions{}, false
	}
	return request.GenerateDateOptions{
		MinDays:         minDays,
		MaxDays:         maxDays,
		SearchStartDate: r.URL.Query().Get("searchStartDate"),
		SearchEndDate:   r.URL.Query().Get("searchEndDate"),
		PreferWeekends:  preferWeekends,
		Limit:           limit,
	}, true
}

func parseOptionalQueryInt(w http.ResponseWriter, r *http.Request, key string) (*int, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+key)
		return nil, false
	}
	return &value, true
}

func parseOptionalQueryBool(w http.ResponseWriter, r *http.Request, key string) (*bool, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil, true
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid "+key)
		return nil, false
	}
	return &value, true
}
