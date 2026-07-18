package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/request"
)

// GetTravelDay returns the authenticated, private mobile execution summary.
// Public share routers do not register this handler.
func (h *Handler) GetTravelDay(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	summary, err := h.svc.GetTravelDay(r.Context(), id, strings.TrimSpace(r.URL.Query().Get("date")))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (h *Handler) UpdateTravelItemStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	dayNumber, ok := parseURLInt(w, r, "dayNumber")
	if !ok {
		return
	}
	itemIndex, ok := parseURLInt(w, r, "itemIndex")
	if !ok {
		return
	}
	var req request.UpdateTravelItemStatus
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	trip, status, err := h.svc.UpdateTravelItemStatus(r.Context(), id, dayNumber, itemIndex, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":            status.Status,
		"updatedAt":         status.UpdatedAt,
		"updatedByUserId":   status.UpdatedByUserID,
		"note":              status.Note,
		"itineraryRevision": trip.ItineraryRevision,
	})
}
