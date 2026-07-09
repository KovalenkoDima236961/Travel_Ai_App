package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/request"
)

// GetApproval handles GET /trips/{id}/approval.
func (h *Handler) GetApproval(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	state, err := h.svc.GetTripApproval(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// GetApprovalRisk handles GET /trips/{id}/approval-risk.
func (h *Handler) GetApprovalRisk(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	risk, err := h.svc.GetTripApprovalRisk(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, risk)
}

// SubmitApproval handles POST /trips/{id}/approval/submit.
func (h *Handler) SubmitApproval(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.SubmitApproval
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	state, err := h.svc.SubmitTripApproval(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// ApproveTrip handles POST /trips/{id}/approval/approve.
func (h *Handler) ApproveTrip(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.ApprovalDecision
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	state, err := h.svc.ApproveTrip(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// RequestTripChanges handles POST /trips/{id}/approval/request-changes.
func (h *Handler) RequestTripChanges(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.ApprovalDecision
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	state, err := h.svc.RequestTripChanges(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// CancelApproval handles POST /trips/{id}/approval/cancel.
func (h *Handler) CancelApproval(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.CancelApproval
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	state, err := h.svc.CancelTripApproval(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, state)
}

// ListApprovalEvents handles GET /trips/{id}/approval/events.
func (h *Handler) ListApprovalEvents(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	events, err := h.svc.ListTripApprovalEvents(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// ListWorkspaceApprovals handles GET /workspaces/{workspaceId}/approvals.
func (h *Handler) ListWorkspaceApprovals(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return
	}
	limit, ok := parseQueryInt(w, r, "limit")
	if !ok {
		return
	}
	offset, ok := parseQueryInt(w, r, "offset")
	if !ok {
		return
	}
	response, err := h.svc.ListWorkspaceApprovals(r.Context(), workspaceID, appdto.ListWorkspaceApprovalsInput{
		Status: strings.TrimSpace(r.URL.Query().Get("status")),
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}
