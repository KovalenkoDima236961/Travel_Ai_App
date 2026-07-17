package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/response"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/personalization"
)

func (h *Handler) SubmitPersonalizationFeedback(w http.ResponseWriter, r *http.Request) {
	var input personalization.SubmitFeedbackInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.TripID != nil && *input.TripID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "invalid trip id")
		return
	}
	created, err := h.svc.SubmitPersonalizationFeedback(r.Context(), input)
	if err != nil {
		if strings.Contains(err.Error(), "invalid feedback") {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) GetRecommendedTemplates(w http.ResponseWriter, r *http.Request) {
	var workspaceID *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("workspaceId")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid workspace id")
			return
		}
		workspaceID = &parsed
	}
	limit, ok := parseQueryInt(w, r, "limit")
	if !ok {
		return
	}
	items, err := h.svc.RecommendedTemplates(r.Context(), workspaceID, limit)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	type item struct {
		Template       response.TripTemplate          `json:"template"`
		FitScore       int                            `json:"fitScore"`
		WhyThisFitsYou personalization.WhyThisFitsYou `json:"whyThisFitsYou"`
		FitTags        []string                       `json:"fitTags"`
	}
	out := make([]item, 0, len(items))
	for _, value := range items {
		out = append(out, item{Template: response.NewTripTemplate(value.Template), FitScore: value.FitScore, WhyThisFitsYou: value.WhyThisFitsYou, FitTags: value.FitTags})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (h *Handler) GetPersonalizedBudgetSuggestion(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	result, err := h.svc.GetPersonalizedBudgetSuggestion(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetPersonalizationFeedbackSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.svc.PersonalizationFeedbackSummary(r.Context())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) ClearPersonalizationFeedback(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.ClearPersonalizationFeedback(r.Context()); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetPersonalizationContext(w http.ResponseWriter, r *http.Request) {
	var tripID *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("tripId")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid trip id")
			return
		}
		tripID = &parsed
	}
	context, err := h.svc.GetPersonalizationContext(r.Context(), personalization.SourceSettings, tripID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, context)
}
