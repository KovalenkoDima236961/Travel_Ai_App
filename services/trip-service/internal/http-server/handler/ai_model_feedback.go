package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/aimodel"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
)

type aiModelFeedbackRequest struct {
	GenerationJobID     *uuid.UUID `json:"generationJobId"`
	ItineraryVersionID  *uuid.UUID `json:"itineraryVersionId"`
	RequestAssignmentID *uuid.UUID `json:"requestAssignmentId"`
	DeploymentID        *uuid.UUID `json:"deploymentId"`
	Feedback            string     `json:"feedback"`
	Note                string     `json:"note"`
}

func (h *Handler) SubmitAIModelFeedback(w http.ResponseWriter, r *http.Request) {
	if h.aiModelFeedback == nil || !h.aiModelFeedback.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "ai model feedback is not configured")
		return
	}
	tripID, ok := h.parseID(w, r)
	if !ok {
		return
	}
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	access, err := h.svc.GetTripAccess(r.Context(), tripID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	if !access.CanView() {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req aiModelFeedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Feedback) == "" {
		writeError(w, http.StatusBadRequest, "feedback is required")
		return
	}
	record, err := h.aiModelFeedback.RecordFeedback(r.Context(), aimodel.FeedbackInput{
		TripID:              tripID,
		GenerationJobID:     req.GenerationJobID,
		ItineraryVersionID:  req.ItineraryVersionID,
		RequestAssignmentID: req.RequestAssignmentID,
		DeploymentID:        req.DeploymentID,
		UserID:              user.ID,
		Feedback:            strings.TrimSpace(req.Feedback),
		Note:                req.Note,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, record)
}
