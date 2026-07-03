package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/request"
)

func (h *Handler) ListWorkspaceBudgets(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return
	}
	budgets, err := h.svc.ListWorkspaceBudgets(r.Context(), workspaceID, r.URL.Query().Get("status"))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, appdto.NewWorkspaceBudgetsEnvelope(budgets))
}

func (h *Handler) CreateWorkspaceBudget(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return
	}
	var req request.CreateWorkspaceBudget
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input, err := req.ToInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	created, err := h.svc.CreateWorkspaceBudget(r.Context(), workspaceID, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, appdto.NewWorkspaceBudgetEnvelope(created))
}

func (h *Handler) GetPrimaryWorkspaceBudgetSummary(w http.ResponseWriter, r *http.Request) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return
	}
	summary, err := h.svc.GetPrimaryWorkspaceBudgetSummary(r.Context(), workspaceID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) GetWorkspaceBudget(w http.ResponseWriter, r *http.Request) {
	workspaceID, budgetID, ok := parseWorkspaceBudgetIDs(w, r)
	if !ok {
		return
	}
	budget, err := h.svc.GetWorkspaceBudget(r.Context(), workspaceID, budgetID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, appdto.NewWorkspaceBudgetEnvelope(budget))
}

func (h *Handler) UpdateWorkspaceBudget(w http.ResponseWriter, r *http.Request) {
	workspaceID, budgetID, ok := parseWorkspaceBudgetIDs(w, r)
	if !ok {
		return
	}
	input, err := request.DecodeUpdateWorkspaceBudget(r.Body)
	if err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	updated, err := h.svc.UpdateWorkspaceBudget(r.Context(), workspaceID, budgetID, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, appdto.NewWorkspaceBudgetEnvelope(updated))
}

func (h *Handler) ArchiveWorkspaceBudget(w http.ResponseWriter, r *http.Request) {
	workspaceID, budgetID, ok := parseWorkspaceBudgetIDs(w, r)
	if !ok {
		return
	}
	var req request.ArchiveWorkspaceBudget
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	_ = strings.TrimSpace(req.Reason)
	archived, err := h.svc.ArchiveWorkspaceBudget(r.Context(), workspaceID, budgetID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, appdto.NewWorkspaceBudgetEnvelope(archived))
}

func (h *Handler) MakeWorkspaceBudgetPrimary(w http.ResponseWriter, r *http.Request) {
	workspaceID, budgetID, ok := parseWorkspaceBudgetIDs(w, r)
	if !ok {
		return
	}
	updated, err := h.svc.MakeWorkspaceBudgetPrimary(r.Context(), workspaceID, budgetID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, appdto.NewWorkspaceBudgetEnvelope(updated))
}

func (h *Handler) GetWorkspaceBudgetSummary(w http.ResponseWriter, r *http.Request) {
	workspaceID, budgetID, ok := parseWorkspaceBudgetIDs(w, r)
	if !ok {
		return
	}
	summary, err := h.svc.GetWorkspaceBudgetSummary(r.Context(), workspaceID, budgetID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func parseWorkspaceBudgetIDs(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	workspaceID, ok := parseUUIDParam(w, r, "workspaceId", "invalid workspace id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	budgetID, ok := parseUUIDParam(w, r, "budgetId", "invalid budget id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return workspaceID, budgetID, true
}
