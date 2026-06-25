package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server/dto/response"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/preferences"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/stream"
)

// notificationService is the user-facing port. The concrete notifications.Service
// satisfies it; tests substitute a fake.
type notificationService interface {
	List(ctx context.Context, in notifications.ListInput) (*notifications.ListResult, error)
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
	MarkRead(ctx context.Context, id, userID uuid.UUID) (*entity.Notification, error)
	MarkAllRead(ctx context.Context, userID uuid.UUID) (int, error)
}

// preferenceService is the user-facing preference port. The concrete
// preferences.Service satisfies it; tests can substitute a fake.
type preferenceService interface {
	GetPreferences(ctx context.Context, userID uuid.UUID) (*preferences.PreferencesResult, error)
	UpdatePreferences(ctx context.Context, userID uuid.UUID, items []preferences.PreferenceInput) (*preferences.PreferencesResult, error)
}

// Handler serves a user's own notifications. All routes it registers must be
// mounted behind JWT auth so user_id always comes from a validated token.
type Handler struct {
	svc         notificationService
	preferences preferenceService
	streams     stream.Manager
	streamCfg   stream.Config
	log         *zap.Logger
}

// New constructs the user-facing notification HTTP handler.
func New(svc notificationService, log *zap.Logger, preferenceSvc ...preferenceService) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	var prefs preferenceService
	if len(preferenceSvc) > 0 {
		prefs = preferenceSvc[0]
	}
	return &Handler{svc: svc, preferences: prefs, log: log}
}

// EnableStream wires the optional SSE stream endpoint onto the handler.
func (h *Handler) EnableStream(manager stream.Manager, cfg stream.Config) *Handler {
	h.streams = manager
	h.streamCfg = stream.Normalize(cfg)
	return h
}

// RegisterRoutes mounts the user-facing notification routes. The caller is
// responsible for wrapping these in JWT auth middleware.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/notifications", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/unread-count", h.UnreadCount)
		if h.streams != nil {
			r.Get("/stream", h.Stream)
		}
		if h.preferences != nil {
			r.Get("/preferences", h.GetPreferences)
			r.Put("/preferences", h.UpdatePreferences)
		}
		r.Patch("/read-all", h.MarkAllRead)
		r.Patch("/{id}/read", h.MarkRead)
	})
}

// Stream handles GET /notifications/stream. It keeps an authenticated SSE
// response open for the current user and receives events from the in-memory
// stream manager.
func (h *Handler) Stream(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !h.streamCfg.Enabled {
		writeError(w, http.StatusServiceUnavailable, "notification stream disabled")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	client := stream.NewClient(user.ID)
	if err := h.streams.Register(user.ID, client); err != nil {
		if errors.Is(err, stream.ErrMaxConnectionsExceeded) {
			writeError(w, http.StatusTooManyRequests, "too many active notification streams")
			return
		}
		h.log.Error("register notification stream client",
			zap.String("user_id", user.ID.String()),
			zap.Error(err),
		)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	defer h.streams.Unregister(user.ID, client.ID)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	controller := http.NewResponseController(w)
	_ = controller.SetWriteDeadline(time.Time{})

	writeEvent := func(event stream.StreamEvent) bool {
		if h.streamCfg.WriteTimeout > 0 {
			_ = controller.SetWriteDeadline(time.Now().Add(h.streamCfg.WriteTimeout))
		}
		if err := stream.WriteSSE(w, event.Name, event.Data); err != nil {
			h.log.Debug("write notification stream event failed",
				zap.String("user_id", user.ID.String()),
				zap.String("client_id", client.ID),
				zap.String("event", event.Name),
				zap.Error(err),
			)
			return false
		}
		flusher.Flush()
		if h.streamCfg.WriteTimeout > 0 {
			_ = controller.SetWriteDeadline(time.Time{})
		}
		return true
	}

	if !writeEvent(heartbeatEvent("connected")) {
		return
	}

	ticker := time.NewTicker(h.streamCfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-client.Send:
			if !ok {
				return
			}
			if !writeEvent(event) {
				return
			}
		case <-ticker.C:
			if !writeEvent(heartbeatEvent("")) {
				return
			}
		}
	}
}

// List handles GET /notifications. It returns the current user's notifications
// newest first with cursor pagination.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit, ok := parseLimit(w, r.URL.Query().Get("limit"))
	if !ok {
		return
	}

	cursorCreatedAt, cursorID, err := notifications.DecodeCursor(strings.TrimSpace(r.URL.Query().Get("cursor")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid cursor")
		return
	}

	result, err := h.svc.List(r.Context(), notifications.ListInput{
		UserID:          user.ID,
		Limit:           limit,
		CursorCreatedAt: cursorCreatedAt,
		CursorID:        cursorID,
	})
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewNotificationList(result.Notifications, result.NextCursor))
}

// UnreadCount handles GET /notifications/unread-count.
func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	count, err := h.svc.CountUnread(r.Context(), user.ID)
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusOK, response.UnreadCount{Count: count})
}

// GetPreferences handles GET /notifications/preferences.
func (h *Handler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if h.preferences == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	result, err := h.preferences.GetPreferences(r.Context(), user.ID)
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewNotificationPreferences(result))
}

// UpdatePreferences handles PUT /notifications/preferences. The user id comes
// only from the JWT subject; request bodies never carry a user id.
func (h *Handler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if h.preferences == nil {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	var req request.UpdateNotificationPreferences
	if !decodeJSON(w, r, &req) {
		return
	}
	inputs, err := req.ToInputs()
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}

	result, err := h.preferences.UpdatePreferences(r.Context(), user.ID, inputs)
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusOK, response.NewNotificationPreferences(result))
}

// MarkRead handles PATCH /notifications/{id}/read. It is idempotent and only
// affects a notification owned by the current user.
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "id")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid notification id")
		return
	}

	if _, err := h.svc.MarkRead(r.Context(), id, user.ID); err != nil {
		writeServiceError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusOK, response.Success{Success: true})
}

// MarkAllRead handles PATCH /notifications/read-all.
func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if _, err := h.svc.MarkAllRead(r.Context(), user.ID); err != nil {
		writeServiceError(w, h.log, err)
		return
	}

	writeJSON(w, http.StatusOK, response.Success{Success: true})
}

// parseLimit parses the optional ?limit= query value. Empty is allowed (the
// service applies the default); a non-numeric or out-of-range value is rejected.
func parseLimit(w http.ResponseWriter, raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, true
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit < 1 || limit > notifications.MaxLimit {
		writeError(w, http.StatusBadRequest, "limit must be between 1 and "+strconv.Itoa(notifications.MaxLimit))
		return 0, false
	}
	return limit, true
}

func heartbeatEvent(status string) stream.StreamEvent {
	data := map[string]string{
		"ts": time.Now().UTC().Format(time.RFC3339Nano),
	}
	if status != "" {
		data["status"] = status
	}
	return stream.StreamEvent{Name: stream.EventHeartbeat, Data: data}
}
