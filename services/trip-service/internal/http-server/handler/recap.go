package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/response"
)

func (h *Handler) GetTripRecapStatus(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	result, err := h.svc.GetTripRecapStatus(r.Context(), tripID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetTripRecap(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	result, err := h.svc.GetTripRecap(r.Context(), tripID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GenerateTripRecap(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req struct {
		ForceRegenerate bool   `json:"forceRegenerate"`
		GenerateEarly   bool   `json:"generateEarly"`
		Language        string `json:"language"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.GenerateTripRecap(r.Context(), tripID, appdto.GenerateTripRecapInput{ForceRegenerate: req.ForceRegenerate, GenerateEarly: req.GenerateEarly, Language: req.Language})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) UpdateTripRecap(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req struct {
		Recap appdto.RecapJSON `json:"recap"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.UpdateTripRecap(r.Context(), tripID, req.Recap)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) FinalizeTripRecap(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	result, err := h.svc.FinalizeTripRecap(r.Context(), tripID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) ArchiveTripRecap(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.ArchiveTripRecap(r.Context(), tripID); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) SubmitTripRecapFeedback(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req struct {
		FeedbackType               string         `json:"feedbackType"`
		EntityType                 string         `json:"entityType"`
		EntityID                   string         `json:"entityId"`
		Label                      string         `json:"label"`
		Value                      string         `json:"value"`
		ApprovedForPersonalization bool           `json:"approvedForPersonalization"`
		Metadata                   map[string]any `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.SubmitTripRecapFeedback(r.Context(), tripID, appdto.SubmitRecapFeedbackInput{FeedbackType: req.FeedbackType, EntityType: req.EntityType, EntityID: req.EntityID, Label: req.Label, Value: req.Value, ApprovedForPersonalization: req.ApprovedForPersonalization, Metadata: req.Metadata})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) ApplyTripRecapLearning(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req struct {
		FeedbackIDs        []string                   `json:"feedbackIds"`
		LearningCandidates []appdto.LearningCandidate `json:"learningCandidates"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ids := make([]uuid.UUID, 0, len(req.FeedbackIDs))
	for _, value := range req.FeedbackIDs {
		id, err := uuid.Parse(value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid feedback id")
			return
		}
		ids = append(ids, id)
	}
	result, err := h.svc.ApplyTripRecapLearning(r.Context(), tripID, appdto.ApplyRecapLearningInput{FeedbackIDs: ids, LearningCandidates: req.LearningCandidates})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"feedback": result})
}

func (h *Handler) CreateTemplateFromTripRecap(w http.ResponseWriter, r *http.Request) {
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req struct {
		Title           string   `json:"title"`
		Description     string   `json:"description"`
		Visibility      string   `json:"visibility"`
		Tags            []string `json:"tags"`
		UseRecapLessons bool     `json:"useRecapLessons"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.CreateTemplateFromTripRecap(r.Context(), tripID, appdto.CreateTemplateFromRecapInput{Title: req.Title, Description: req.Description, Visibility: entity.TripTemplateVisibility(req.Visibility), Tags: req.Tags, UseRecapLessons: req.UseRecapLessons})
	if err != nil {
		h.writeTemplateServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response.NewTripTemplateDetail(*result))
}
