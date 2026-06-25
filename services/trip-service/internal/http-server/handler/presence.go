package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/presence"
)

type updatePresenceStateRequest struct {
	State string `json:"state"`
}

// StreamPresence handles GET /trips/{id}/presence/stream.
func (h *Handler) StreamPresence(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !h.presenceAvailable(w) {
		return
	}

	access, err := h.svc.GetTripAccess(r.Context(), id)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	now := time.Now().UTC()
	sessionID := uuid.NewString()
	events, err := h.presence.Register(r.Context(), presence.PresenceSession{
		SessionID:   sessionID,
		TripID:      id,
		UserID:      user.ID,
		Role:        string(access.Level),
		State:       presence.PresenceStateViewing,
		ConnectedAt: now,
		LastSeenAt:  now,
	})
	if err != nil {
		if errors.Is(err, presence.ErrMaxConnectionsExceeded) {
			writeError(w, http.StatusTooManyRequests, "too many active trip presence streams")
			return
		}
		if errors.Is(err, presence.ErrInvalidState) {
			writeError(w, http.StatusBadRequest, "invalid presence state")
			return
		}
		h.log.Error("register trip presence session",
			zap.String("trip_id", id.String()),
			zap.String("user_id", user.ID.String()),
			zap.Error(err),
		)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	defer h.presence.Unregister(id, sessionID)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	controller := http.NewResponseController(w)
	_ = controller.SetWriteDeadline(time.Time{})

	writeEvent := func(event presence.PresenceEvent) bool {
		if err := presence.WriteSSE(w, event.Name, event.Data); err != nil {
			h.log.Debug("write trip presence stream event failed",
				zap.String("trip_id", id.String()),
				zap.String("user_id", user.ID.String()),
				zap.String("session_id", sessionID),
				zap.String("event", event.Name),
				zap.Error(err),
			)
			return false
		}
		flusher.Flush()
		return true
	}

	ticker := time.NewTicker(h.presenceCfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-events:
			if !ok {
				return
			}
			if !writeEvent(event) {
				return
			}
		case <-ticker.C:
			if !writeEvent(presence.HeartbeatEvent()) {
				return
			}
		}
	}
}

// UpdatePresenceState handles POST /trips/{id}/presence/state.
func (h *Handler) UpdatePresenceState(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !h.presenceAvailable(w) {
		return
	}

	var req updatePresenceStateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if !presence.IsValidState(req.State) {
		writeError(w, http.StatusBadRequest, "invalid presence state")
		return
	}
	if _, err := h.svc.GetTripAccess(r.Context(), id); err != nil {
		h.writeServiceError(w, err)
		return
	}
	if err := h.presence.UpdateState(id, user.ID, req.State); err != nil {
		if errors.Is(err, presence.ErrInvalidState) {
			writeError(w, http.StatusBadRequest, "invalid presence state")
			return
		}
		h.log.Error("update trip presence state",
			zap.String("trip_id", id.String()),
			zap.String("user_id", user.ID.String()),
			zap.String("state", req.State),
			zap.Error(err),
		)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

// GetPresenceSnapshot handles GET /trips/{id}/presence.
func (h *Handler) GetPresenceSnapshot(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	if _, err := auth.MustUserFromContext(r.Context()); err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !h.presenceAvailable(w) {
		return
	}
	if _, err := h.svc.GetTripAccess(r.Context(), id); err != nil {
		h.writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, h.presence.Snapshot(id))
}

func (h *Handler) presenceAvailable(w http.ResponseWriter) bool {
	if h.presence == nil || !h.presenceCfg.Enabled {
		writeError(w, http.StatusServiceUnavailable, "trip presence disabled")
		return false
	}
	return true
}
