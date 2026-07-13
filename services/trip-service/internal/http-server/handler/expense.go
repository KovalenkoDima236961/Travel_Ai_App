package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/request"
)

func (h *Handler) CreateTripExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.CreateTripExpense
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input, err := req.ToInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	expense, err := h.svc.CreateTripExpense(r.Context(), id, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, expense)
}

func (h *Handler) ListTripExpenses(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	filters, ok := parseExpenseFilters(w, r)
	if !ok {
		return
	}
	expenses, err := h.svc.ListTripExpenses(r.Context(), id, filters)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, expenses)
}

func (h *Handler) GetTripExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	expenseID, ok := parseUUIDParam(w, r, "expenseId", "invalid expense id")
	if !ok {
		return
	}
	expense, err := h.svc.GetTripExpense(r.Context(), id, expenseID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, expense)
}

func (h *Handler) UpdateTripExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	expenseID, ok := parseUUIDParam(w, r, "expenseId", "invalid expense id")
	if !ok {
		return
	}
	var req request.UpdateTripExpense
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input, err := req.ToInput()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	expense, err := h.svc.UpdateTripExpense(r.Context(), id, expenseID, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, expense)
}

func (h *Handler) DeleteTripExpense(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	expenseID, ok := parseUUIDParam(w, r, "expenseId", "invalid expense id")
	if !ok {
		return
	}
	if err := h.svc.DeleteTripExpense(r.Context(), id, expenseID); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) GetTripExpenseSummary(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	currency, ok := parseCurrencyQuery(w, r)
	if !ok {
		return
	}
	summary, err := h.svc.GetTripExpenseSummary(r.Context(), id, currency)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) GetTripSettlements(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	currency, ok := parseCurrencyQuery(w, r)
	if !ok {
		return
	}
	result, err := h.svc.GetTripSettlements(r.Context(), id, currency)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) RecalculateTripSettlements(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	currency, ok := parseCurrencyQuery(w, r)
	if !ok {
		return
	}
	result, err := h.svc.RecalculateTripSettlements(r.Context(), id, currency)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) MarkTripSettlementPaid(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	settlementID := strings.TrimSpace(chi.URLParam(r, "settlementId"))
	if settlementID == "" {
		writeError(w, http.StatusBadRequest, "invalid settlement id")
		return
	}
	var req request.MarkSettlementPaid
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.MarkTripSettlementPaid(r.Context(), id, settlementID, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) CancelTripSettlement(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	settlementID, ok := parseUUIDParam(w, r, "settlementId", "invalid settlement id")
	if !ok {
		return
	}
	result, err := h.svc.CancelTripSettlement(r.Context(), id, settlementID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func parseExpenseFilters(w http.ResponseWriter, r *http.Request) (appdto.ListExpensesInput, bool) {
	var filters appdto.ListExpensesInput
	query := r.URL.Query()
	if rawCategory := strings.TrimSpace(query.Get("category")); rawCategory != "" {
		category := entity.ExpenseCategory(rawCategory)
		filters.Category = &category
	}
	if rawPaidBy := strings.TrimSpace(query.Get("paidByUserId")); rawPaidBy != "" {
		parsed, err := uuid.Parse(rawPaidBy)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid paidByUserId")
			return filters, false
		}
		filters.PaidByUserID = &parsed
	}
	from, ok := parseDateQuery(w, r, "fromDate")
	if !ok {
		return filters, false
	}
	to, ok := parseDateQuery(w, r, "toDate")
	if !ok {
		return filters, false
	}
	linkedOnly, ok := parseBoolQuery(w, r, "linkedOnly")
	if !ok {
		return filters, false
	}
	filters.FromDate = from
	filters.ToDate = to
	filters.LinkedOnly = linkedOnly
	return filters, true
}
