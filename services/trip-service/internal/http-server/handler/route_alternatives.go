package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/response"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/routealternatives"
)

type createTripFromRouteAlternativeResponse struct {
	Trip *response.Trip              `json:"trip"`
	Job  *generationjobs.JobResponse `json:"generationJob,omitempty"`
}

func (h *Handler) SuggestRouteAlternatives(w http.ResponseWriter, r *http.Request) {
	var req routealternatives.SuggestInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.SuggestRouteAlternatives(r.Context(), req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) SuggestTripRouteAlternatives(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req routealternatives.ExistingTripSuggestInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.SuggestTripRouteAlternatives(r.Context(), tripID, req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) ListRouteAlternativeSessions(w http.ResponseWriter, r *http.Request) {
	var tripID *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("tripId")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid trip id")
			return
		}
		tripID = &parsed
	}
	limit, ok := parseQueryInt(w, r, "limit")
	if !ok {
		return
	}
	result, err := h.svc.ListRouteAlternativeSessions(r.Context(), tripID, limit)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetRouteAlternativeSession(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := parseUUIDParam(w, r, "sessionId", "invalid session id")
	if !ok {
		return
	}
	result, err := h.svc.GetRouteAlternativeSession(r.Context(), sessionID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) RefineRouteAlternativeSession(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := parseUUIDParam(w, r, "sessionId", "invalid session id")
	if !ok {
		return
	}
	var req routealternatives.RefineInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.RefineRouteAlternativeSession(r.Context(), sessionID, req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) CreateTripFromRouteAlternative(w http.ResponseWriter, r *http.Request) {
	sessionID, ok := parseUUIDParam(w, r, "sessionId", "invalid session id")
	if !ok {
		return
	}
	alternativeID := strings.TrimSpace(chiURLParam(r, "alternativeId"))
	if alternativeID == "" {
		writeError(w, http.StatusBadRequest, "invalid alternative id")
		return
	}
	var req routealternatives.CreateTripInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.AutoGenerateItinerary && h.generationJobs == nil {
		writeError(w, http.StatusServiceUnavailable, "generation jobs are not configured")
		return
	}
	trip, err := h.svc.CreateTripFromRouteAlternative(r.Context(), sessionID, alternativeID, req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	out := createTripFromRouteAlternativeResponse{Trip: ptrTripResponse(response.NewTrip(trip))}
	if req.AutoGenerateItinerary {
		expectedRevision := trip.ItineraryRevision
		var instruction *string
		if trip.CreationMetadata != nil {
			if raw, _ := trip.CreationMetadata["suggestedPromptForItinerary"].(string); strings.TrimSpace(raw) != "" {
				trimmed := strings.TrimSpace(raw)
				instruction = &trimmed
			}
		}
		job, err := h.generationJobs.Create(r.Context(), trip.ID, generationjobs.CreateRequest{
			JobType:                   entity.GenerationJobTypeFullGeneration,
			ExpectedItineraryRevision: &expectedRevision,
			Instruction:               instruction,
		})
		if err != nil {
			h.writeGenerationJobError(w, err)
			return
		}
		jobResponse := generationjobs.NewJobResponse(job)
		out.Job = &jobResponse
		writeJSON(w, http.StatusAccepted, out)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *Handler) ApplyRouteAlternative(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	sessionID, ok := parseUUIDParam(w, r, "sessionId", "invalid session id")
	if !ok {
		return
	}
	alternativeID := strings.TrimSpace(chiURLParam(r, "alternativeId"))
	if alternativeID == "" {
		writeError(w, http.StatusBadRequest, "invalid alternative id")
		return
	}
	var req routealternatives.ApplyInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	trip, err := h.svc.ApplyRouteAlternative(r.Context(), tripID, sessionID, alternativeID, req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTrip(trip))
}

func (h *Handler) CreateRouteAlternativesPoll(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	sessionID, ok := parseUUIDParam(w, r, "sessionId", "invalid session id")
	if !ok {
		return
	}
	var req routealternatives.CreatePollInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	info, err := h.svc.CreateRouteAlternativesPoll(r.Context(), tripID, sessionID, req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response.NewTripPoll(info))
}

func ptrTripResponse(trip response.Trip) *response.Trip {
	return &trip
}

func chiURLParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}
