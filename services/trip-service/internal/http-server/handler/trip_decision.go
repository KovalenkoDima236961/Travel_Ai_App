package handler

import (
	"encoding/json"
	"net/http"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/response"
)

func (h *Handler) CreateTripPoll(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.CreateTripPoll
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	info, err := h.svc.CreateTripPoll(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response.NewTripPoll(info))
}

func (h *Handler) ListTripPolls(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	infos, err := h.svc.ListTripPolls(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewListTripPolls(infos))
}

func (h *Handler) GetTripPoll(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	pollID, ok := parseUUIDParam(w, r, "pollId", "invalid poll id")
	if !ok {
		return
	}
	info, err := h.svc.GetTripPoll(r.Context(), id, pollID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripPoll(info))
}

func (h *Handler) VoteTripPoll(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	pollID, ok := parseUUIDParam(w, r, "pollId", "invalid poll id")
	if !ok {
		return
	}
	var req request.VoteTripPoll
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	info, err := h.svc.VoteTripPoll(r.Context(), id, pollID, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripPoll(info))
}

func (h *Handler) CloseTripPoll(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	pollID, ok := parseUUIDParam(w, r, "pollId", "invalid poll id")
	if !ok {
		return
	}
	info, err := h.svc.CloseTripPoll(r.Context(), id, pollID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripPoll(info))
}

func (h *Handler) ArchiveTripPoll(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	pollID, ok := parseUUIDParam(w, r, "pollId", "invalid poll id")
	if !ok {
		return
	}
	info, err := h.svc.ArchiveTripPoll(r.Context(), id, pollID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripPoll(info))
}

func (h *Handler) SetItineraryItemReaction(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.SetItineraryItemReaction
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	summary, err := h.svc.SetItineraryItemReaction(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewItineraryItemReactionSummary(summary))
}

func (h *Handler) ListItineraryItemReactions(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	summaries, err := h.svc.ListItineraryItemReactions(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewListItineraryItemReactionSummaries(summaries))
}

func (h *Handler) GetItineraryItemReactions(w http.ResponseWriter, r *http.Request) {
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
	summary, err := h.svc.ListItineraryItemReactionsByItem(r.Context(), id, dayNumber, itemIndex)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewItineraryItemReactionSummary(summary))
}

func (h *Handler) DeleteMyItineraryItemReaction(w http.ResponseWriter, r *http.Request) {
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
	if err := h.svc.DeleteMyItineraryItemReaction(r.Context(), id, dayNumber, itemIndex); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) GetGroupPreferences(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	summary, err := h.svc.GetGroupPreferences(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
