package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/response"
)

// ListComments handles GET /trips/{id}/comments. When both dayNumber and
// itemIndex query parameters are present it returns comments for that single
// item; with neither it returns all active comments for the trip. Supplying only
// one of the two is rejected as a bad request.
func (h *Handler) ListComments(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	dayRaw := strings.TrimSpace(r.URL.Query().Get("dayNumber"))
	itemRaw := strings.TrimSpace(r.URL.Query().Get("itemIndex"))
	hasDay := dayRaw != ""
	hasItem := itemRaw != ""

	if hasDay != hasItem {
		writeError(w, http.StatusBadRequest, "dayNumber and itemIndex must be provided together")
		return
	}

	if hasDay && hasItem {
		dayNumber, ok := parseQueryInt(w, r, "dayNumber")
		if !ok {
			return
		}
		itemIndex, ok := parseQueryInt(w, r, "itemIndex")
		if !ok {
			return
		}
		infos, err := h.svc.ListItemComments(r.Context(), id, dayNumber, itemIndex)
		if err != nil {
			h.writeServiceError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, response.NewListComments(infos))
		return
	}

	infos, err := h.svc.ListComments(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewListComments(infos))
}

// ListCommentCounts handles GET /trips/{id}/comments/counts.
func (h *Handler) ListCommentCounts(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	counts, err := h.svc.ListCommentCounts(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewCommentCounts(counts))
}

// CreateComment handles POST /trips/{id}/comments.
func (h *Handler) CreateComment(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}

	var req request.CreateComment
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	info, err := h.svc.CreateComment(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response.NewItineraryComment(info))
}

// UpdateComment handles PATCH /trips/{id}/comments/{commentId}.
func (h *Handler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	commentID, ok := parseUUIDParam(w, r, "commentId", "invalid comment id")
	if !ok {
		return
	}

	var req request.UpdateComment
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validator.Validate(req); err != nil {
		h.writeValidationError(w, err)
		return
	}

	info, err := h.svc.UpdateComment(r.Context(), id, commentID, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewItineraryComment(info))
}

// DeleteComment handles DELETE /trips/{id}/comments/{commentId}.
func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	commentID, ok := parseUUIDParam(w, r, "commentId", "invalid comment id")
	if !ok {
		return
	}

	if err := h.svc.DeleteComment(r.Context(), id, commentID); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
