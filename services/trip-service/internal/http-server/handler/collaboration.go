package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/http-server/dto/response"
)

func (h *Handler) CreateTripInvitation(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	if !h.allowCollaborationInvite(w, r, id.String()) {
		return
	}
	var req request.CreateTripInvitation
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	info, err := h.svc.CreateTripInvitation(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response.NewTripInvitation(info))
}

func (h *Handler) ListTripInvitations(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	invitations, err := h.svc.ListTripInvitations(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewListTripInvitations(invitations))
}

func (h *Handler) ResendTripInvitation(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	invitationID, ok := parseUUIDParam(w, r, "invitationId", "invalid invitation id")
	if !ok {
		return
	}
	var req request.ResendTripInvitation
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	info, err := h.svc.ResendTripInvitation(r.Context(), id, invitationID, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripInvitation(info))
}

func (h *Handler) RevokeTripInvitation(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	invitationID, ok := parseUUIDParam(w, r, "invitationId", "invalid invitation id")
	if !ok {
		return
	}
	if err := h.svc.RevokeTripInvitation(r.Context(), id, invitationID); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) AcceptTripInvitation(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	invitationID, ok := parseUUIDParam(w, r, "invitationId", "invalid invitation id")
	if !ok {
		return
	}
	info, err := h.svc.AcceptTripInvitation(r.Context(), id, invitationID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripInvitation(info))
}

func (h *Handler) DeclineTripInvitation(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	invitationID, ok := parseUUIDParam(w, r, "invitationId", "invalid invitation id")
	if !ok {
		return
	}
	if err := h.svc.DeclineTripInvitation(r.Context(), id, invitationID); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) ListTripMembers(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	members, err := h.svc.ListTripMembers(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewListTripMembers(members))
}

func (h *Handler) TransferTripOwnership(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.TransferTripOwnership
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	trip, err := h.svc.TransferTripOwnership(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTrip(trip))
}

func (h *Handler) LeaveTrip(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	if err := h.svc.LeaveTrip(r.Context(), id); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) CreateTripSuggestion(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.CreateTripSuggestion
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	info, err := h.svc.CreateTripSuggestion(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response.NewTripSuggestion(info))
}

func (h *Handler) ListTripSuggestions(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	status, ok := parseTripSuggestionStatusQuery(w, r)
	if !ok {
		return
	}
	suggestions, err := h.svc.ListTripSuggestions(r.Context(), id, status)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewListTripSuggestions(suggestions))
}

func (h *Handler) AcceptTripSuggestion(w http.ResponseWriter, r *http.Request) {
	id, suggestionID, ok := parseTripSuggestionIDs(w, r)
	if !ok {
		return
	}
	var req request.UpdateTripSuggestionStatus
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	info, err := h.svc.AcceptTripSuggestion(r.Context(), id, suggestionID, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripSuggestion(info))
}

func (h *Handler) RejectTripSuggestion(w http.ResponseWriter, r *http.Request) {
	id, suggestionID, ok := parseTripSuggestionIDs(w, r)
	if !ok {
		return
	}
	info, err := h.svc.RejectTripSuggestion(r.Context(), id, suggestionID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripSuggestion(info))
}

func (h *Handler) ResolveTripSuggestion(w http.ResponseWriter, r *http.Request) {
	id, suggestionID, ok := parseTripSuggestionIDs(w, r)
	if !ok {
		return
	}
	info, err := h.svc.ResolveTripSuggestion(r.Context(), id, suggestionID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripSuggestion(info))
}

func (h *Handler) SetTripVote(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	var req request.SetTripVote
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	summary, err := h.svc.SetTripVote(r.Context(), id, req.ToInput())
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripVoteSummary(summary))
}

func (h *Handler) ListTripVotes(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	in := appdto.ListTripVotesInput{
		TargetType: entity.TripVoteTargetType(strings.TrimSpace(r.URL.Query().Get("targetType"))),
		TargetID:   strings.TrimSpace(r.URL.Query().Get("targetId")),
	}
	summaries, err := h.svc.ListTripVotes(r.Context(), id, in)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewListTripVoteSummaries(summaries))
}

func (h *Handler) DeleteTripVote(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	voteID, ok := parseUUIDParam(w, r, "voteId", "invalid vote id")
	if !ok {
		return
	}
	if err := h.svc.DeleteTripVote(r.Context(), id, voteID); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func parseTripSuggestionIDs(w http.ResponseWriter, r *http.Request) (tripID, suggestionID uuid.UUID, ok bool) {
	tripID, ok = parseUUIDParam(w, r, "id", "invalid trip id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	suggestionID, ok = parseUUIDParam(w, r, "suggestionId", "invalid suggestion id")
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return tripID, suggestionID, true
}

func parseTripSuggestionStatusQuery(w http.ResponseWriter, r *http.Request) (*entity.TripSuggestionStatus, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("status"))
	if raw == "" {
		return nil, true
	}
	status := entity.TripSuggestionStatus(raw)
	switch status {
	case entity.TripSuggestionStatusOpen,
		entity.TripSuggestionStatusAccepted,
		entity.TripSuggestionStatusRejected,
		entity.TripSuggestionStatusResolved:
		return &status, true
	default:
		writeError(w, http.StatusBadRequest, "invalid status")
		return nil, false
	}
}
