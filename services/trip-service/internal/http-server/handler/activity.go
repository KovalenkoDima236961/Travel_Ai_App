package handler

import (
	"net/http"
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/response"
)

// ListActivity handles GET /trips/{id}/activity?limit=&cursor=. It returns a
// newest-first page of the trip's activity feed for the owner or an accepted
// collaborator. Permission and pagination bounds are enforced by the service;
// the handler only parses query parameters. This route is mounted only in the
// authenticated group, so public share viewers can never reach it.
func (h *Handler) ListActivity(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	limit, ok := parseQueryInt(w, r, "limit")
	if !ok {
		return
	}
	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))

	result, err := h.svc.ListActivity(r.Context(), id, limit, cursor)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewTripActivity(result))
}
