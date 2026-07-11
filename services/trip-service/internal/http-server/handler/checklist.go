package handler

import (
	"encoding/json"
	"net/http"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/request"
)

func (h *Handler) GetChecklist(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	result, err := h.svc.GetTripChecklist(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GenerateChecklist(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.GenerateChecklist
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.GenerateTripChecklist(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) CreateChecklistItem(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.CreateChecklistItem
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input, err := req.ToInput()
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	item, err := h.svc.CreateTripChecklistItem(r.Context(), id, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) UpdateChecklistItem(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	itemID, ok := parseUUIDParam(w, r, "itemId", "invalid checklist item id")
	if !ok {
		return
	}
	var req request.UpdateChecklistItem
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input, err := req.ToInput()
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	item, err := h.svc.UpdateTripChecklistItem(r.Context(), id, itemID, input)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) DeleteChecklistItem(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	itemID, ok := parseUUIDParam(w, r, "itemId", "invalid checklist item id")
	if !ok {
		return
	}
	if err := h.svc.DeleteTripChecklistItem(r.Context(), id, itemID); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) CheckChecklistItem(w http.ResponseWriter, r *http.Request) {
	h.setChecklistItemChecked(w, r, true)
}

func (h *Handler) UncheckChecklistItem(w http.ResponseWriter, r *http.Request) {
	h.setChecklistItemChecked(w, r, false)
}

func (h *Handler) setChecklistItemChecked(w http.ResponseWriter, r *http.Request, checked bool) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	itemID, ok := parseUUIDParam(w, r, "itemId", "invalid checklist item id")
	if !ok {
		return
	}
	item, err := h.svc.SetTripChecklistItemChecked(r.Context(), id, itemID, checked)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) ReorderChecklistItems(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.ReorderChecklistItems
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.ReorderTripChecklistItems(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
