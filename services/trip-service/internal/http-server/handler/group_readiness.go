package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/groupreadiness"
)

// GetGroupReadiness handles GET /trips/{id}/group-readiness. Public share
// routes never mount this endpoint, so readiness stays private to trip viewers.
func (h *Handler) GetGroupReadiness(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	includeDetails := true
	if strings.TrimSpace(r.URL.Query().Get("includeDetails")) != "" {
		includeDetails, ok = parseBoolQuery(w, r, "includeDetails")
		if !ok {
			return
		}
	}
	includeDebug, ok := parseBoolQuery(w, r, "includeDebug")
	if !ok {
		return
	}

	readiness, err := h.svc.GetGroupReadiness(r.Context(), id, groupreadiness.Options{
		IncludeDetails: includeDetails,
		IncludeDebug:   includeDebug,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, readiness)
}

func (h *Handler) SendGroupReadinessNudge(w http.ResponseWriter, r *http.Request) {
	h.sendGroupReadinessNudge(w, r, nil)
}

func (h *Handler) NudgeMissingAvailability(w http.ResponseWriter, r *http.Request) {
	category := groupreadiness.CategoryAvailability
	h.sendGroupReadinessNudge(w, r, &category)
}

func (h *Handler) NudgeAssignedTasks(w http.ResponseWriter, r *http.Request) {
	category := groupreadiness.CategoryChecklist
	h.sendGroupReadinessNudge(w, r, &category)
}

func (h *Handler) NudgePendingVotes(w http.ResponseWriter, r *http.Request) {
	category := groupreadiness.CategoryPolls
	h.sendGroupReadinessNudge(w, r, &category)
}

func (h *Handler) NudgePendingSettlements(w http.ResponseWriter, r *http.Request) {
	category := groupreadiness.CategorySettlements
	h.sendGroupReadinessNudge(w, r, &category)
}

func (h *Handler) sendGroupReadinessNudge(
	w http.ResponseWriter,
	r *http.Request,
	forcedCategory *groupreadiness.Category,
) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req groupreadiness.NudgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if forcedCategory != nil {
		req.Categories = []groupreadiness.Category{*forcedCategory}
	}

	result, err := h.svc.SendGroupReadinessNudge(r.Context(), id, req)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
