package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

func (h *Handler) GetWorkspacePolicy(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return
	}
	if h.workspacePolicies == nil {
		writeError(w, http.StatusServiceUnavailable, "workspace policies are unavailable")
		return
	}
	result, err := h.workspacePolicies.Get(r.Context(), workspaceID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) UpsertWorkspacePolicy(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return
	}
	if h.workspacePolicies == nil {
		writeError(w, http.StatusServiceUnavailable, "workspace policies are unavailable")
		return
	}
	var input workspacepolicies.UpsertInput
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	policy, err := h.workspacePolicies.Upsert(r.Context(), workspaceID, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"policy": policy})
}

func (h *Handler) ArchiveWorkspacePolicy(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return
	}
	if h.workspacePolicies == nil {
		writeError(w, http.StatusServiceUnavailable, "workspace policies are unavailable")
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&struct{}{}); err != nil &&
		!errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	policy, err := h.workspacePolicies.Archive(r.Context(), workspaceID)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			writeError(w, http.StatusNotFound, "active workspace policy not found")
			return
		}
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"policy": policy})
}

func (h *Handler) EvaluateTripPolicy(w http.ResponseWriter, r *http.Request) {
	h.evaluateTripPolicy(w, r)
}

func (h *Handler) GetTripPolicyEvaluation(w http.ResponseWriter, r *http.Request) {
	h.evaluateTripPolicy(w, r)
}

func (h *Handler) evaluateTripPolicy(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	result, err := h.svc.EvaluateTripPolicy(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
