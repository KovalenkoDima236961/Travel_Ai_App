package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/aimodel"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/observability"
)

func (h *Handler) registerOpsAIModelRoutes(r chi.Router) {
	r.Post("/ai/model-deployments", h.OpsRegisterAIModelDeployment)
	r.Post("/ai/model-deployments/{deploymentId}/enable-shadow", h.OpsEnableAIModelShadow)
	r.Post("/ai/model-deployments/{deploymentId}/pause", h.OpsPauseAIModelDeployment)
	r.Post("/ai/model-deployments/{deploymentId}/rollback", h.OpsRollbackAIModelDeployment)
	r.Get("/ai/model-deployments/{deploymentId}/online-summary", h.OpsAIModelOnlineSummary)
}

func (h *Handler) OpsRegisterAIModelDeployment(w http.ResponseWriter, r *http.Request) {
	if !h.opsAIModelAvailable(w) {
		return
	}
	var req aimodel.RegisterDeploymentInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.ActorUserID = opsActorID(r)
	req.RequestID = observability.RequestIDFromContext(r.Context())
	deployment, err := h.aiModelOps.RegisterDeployment(r.Context(), req)
	if err != nil {
		h.writeAIModelOpsError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, deployment)
}

func (h *Handler) OpsEnableAIModelShadow(w http.ResponseWriter, r *http.Request) {
	if !h.opsAIModelAvailable(w) {
		return
	}
	deploymentID, ok := parseUUIDParam(w, r, "deploymentId", "invalid deployment id")
	if !ok {
		return
	}
	var req aimodel.ShadowRolloutInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.ActorUserID = opsActorID(r)
	req.RequestID = observability.RequestIDFromContext(r.Context())
	deployment, err := h.aiModelOps.EnableShadow(r.Context(), deploymentID, req)
	if err != nil {
		h.writeAIModelOpsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, deployment)
}

func (h *Handler) OpsPauseAIModelDeployment(w http.ResponseWriter, r *http.Request) {
	h.opsAIModelDeploymentAction(w, r, h.aiModelOps.PauseDeployment)
}

func (h *Handler) OpsRollbackAIModelDeployment(w http.ResponseWriter, r *http.Request) {
	h.opsAIModelDeploymentAction(w, r, h.aiModelOps.RollbackDeployment)
}

func (h *Handler) OpsAIModelOnlineSummary(w http.ResponseWriter, r *http.Request) {
	if !h.opsAIModelAvailable(w) {
		return
	}
	deploymentID, ok := parseUUIDParam(w, r, "deploymentId", "invalid deployment id")
	if !ok {
		return
	}
	summary, err := h.aiModelOps.OnlineSummary(r.Context(), deploymentID)
	if err != nil {
		h.writeAIModelOpsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) opsAIModelDeploymentAction(w http.ResponseWriter, r *http.Request, action func(context.Context, uuid.UUID, aimodel.DeploymentActionInput) (*aimodel.Deployment, error)) {
	if !h.opsAIModelAvailable(w) {
		return
	}
	deploymentID, ok := parseUUIDParam(w, r, "deploymentId", "invalid deployment id")
	if !ok {
		return
	}
	var req aimodel.DeploymentActionInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.ActorUserID = opsActorID(r)
	req.RequestID = observability.RequestIDFromContext(r.Context())
	deployment, err := action(r.Context(), deploymentID, req)
	if err != nil {
		h.writeAIModelOpsError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, deployment)
}

func (h *Handler) opsAIModelAvailable(w http.ResponseWriter) bool {
	if h.aiModelOps == nil || !h.aiModelOps.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "ai model ops are not configured")
		return false
	}
	return true
}

func (h *Handler) writeAIModelOpsError(w http.ResponseWriter, err error) {
	if errors.Is(err, domainerrs.ErrNotFound) {
		writeError(w, http.StatusNotFound, "ai model deployment not found")
		return
	}
	writeError(w, http.StatusBadRequest, err.Error())
}
