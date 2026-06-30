package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activitystream"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
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

// StreamActivity handles GET /trips/{id}/activity/stream. It is an
// authenticated, private-trip-only SSE stream for newly persisted activity
// events. The persisted GET /activity endpoint remains the source of truth.
func (h *Handler) StreamActivity(w http.ResponseWriter, r *http.Request) {
	id, ok := h.parseID(w, r)
	if !ok {
		return
	}
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !h.activityStreamAvailable(w) {
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
	connectionID := uuid.NewString()
	events, err := h.activityStream.Register(r.Context(), activitystream.RegisterClientInput{
		ConnectionID: connectionID,
		TripID:       id,
		UserID:       user.ID,
		Role:         string(access.Level),
		ConnectedAt:  now,
		LastSeenAt:   now,
	})
	if err != nil {
		if errors.Is(err, activitystream.ErrMaxConnectionsExceeded) {
			writeError(w, http.StatusTooManyRequests, "too many active trip activity streams")
			return
		}
		h.log.Error("register trip activity stream",
			zap.String("trip_id", id.String()),
			zap.String("user_id", user.ID.String()),
			zap.Error(err),
		)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	defer h.activityStream.Unregister(id, connectionID)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher.Flush()

	controller := http.NewResponseController(w)

	writeEvent := func(event activitystream.ActivityStreamEvent) bool {
		if h.activityStreamCfg.WriteTimeout > 0 {
			_ = controller.SetWriteDeadline(time.Now().Add(h.activityStreamCfg.WriteTimeout))
		}
		if err := activitystream.WriteSSE(w, event.Name, event.Data); err != nil {
			h.log.Debug("write trip activity stream event failed",
				zap.String("trip_id", id.String()),
				zap.String("user_id", user.ID.String()),
				zap.String("connection_id", connectionID),
				zap.String("event", event.Name),
				zap.Error(err),
			)
			return false
		}
		flusher.Flush()
		return true
	}

	ticker := time.NewTicker(h.activityStreamCfg.HeartbeatInterval)
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
			if !writeEvent(activitystream.HeartbeatEvent()) {
				return
			}
		}
	}
}

func (h *Handler) activityStreamAvailable(w http.ResponseWriter) bool {
	if h.activityStream == nil || !h.activityStreamCfg.Enabled {
		writeError(w, http.StatusServiceUnavailable, "activity_stream_disabled")
		return false
	}
	return true
}
