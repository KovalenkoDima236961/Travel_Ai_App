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
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/controls"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/httpserver/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/httpserver/dto/response"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/preferences"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/push"
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

type controlsService interface {
	GetSettings(ctx context.Context, userID uuid.UUID) (*entity.NotificationSettings, error)
	UpdateSettings(ctx context.Context, input controls.SettingsInput) (*entity.NotificationSettings, error)
	ListTripMutes(ctx context.Context, userID, tripID uuid.UUID) ([]entity.NotificationTripMute, error)
	UpsertTripMute(ctx context.Context, input controls.TripMuteInput) (*entity.NotificationTripMute, error)
	DeleteTripMute(ctx context.Context, id, userID uuid.UUID) error
}

type userDigestService interface {
	ListPending(ctx context.Context, userID uuid.UUID, limit int) ([]entity.NotificationDigestBatch, error)
	ListHistory(ctx context.Context, userID uuid.UUID, limit int) ([]entity.NotificationDigestBatch, error)
	Get(ctx context.Context, id, userID uuid.UUID) (*entity.NotificationDigestBatch, error)
}

// pushService is the user-facing browser push port.
type pushService interface {
	PublicKey() push.PublicKeyResult
	Subscribe(ctx context.Context, input push.SubscribeInput) (bool, error)
	Unsubscribe(ctx context.Context, userID uuid.UUID, endpoint string) error
	Status(ctx context.Context, userID uuid.UUID) (*push.StatusResult, error)
}

// Handler serves a user's own notifications. All routes it registers must be
// mounted behind JWT auth so user_id always comes from a validated token.
type Handler struct {
	svc         notificationService
	preferences preferenceService
	controls    controlsService
	digests     userDigestService
	push        pushService
	streams     stream.Manager
	streamCfg   stream.Config
	log         *zap.Logger
}

func (h *Handler) EnableControls(service controlsService) *Handler  { h.controls = service; return h }
func (h *Handler) EnableDigests(service userDigestService) *Handler { h.digests = service; return h }

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

// EnablePush wires browser push endpoints onto the handler.
func (h *Handler) EnablePush(pushSvc pushService) *Handler {
	h.push = pushSvc
	return h
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
		if h.controls != nil {
			r.Get("/trip-mutes", h.ListTripMutes)
			r.Put("/trip-mutes", h.UpsertTripMute)
			r.Delete("/trip-mutes/{muteId}", h.DeleteTripMute)
		}
		if h.digests != nil {
			r.Get("/digests/pending", h.ListPendingDigests)
			r.Get("/digests/history", h.ListDigestHistory)
			r.Get("/digests/{digestId}", h.GetDigest)
		}
		if h.push != nil {
			r.Post("/push/subscribe", h.SubscribePush)
			r.Delete("/push/unsubscribe", h.UnsubscribePush)
			r.Get("/push/status", h.PushStatus)
		}
		r.Patch("/read-all", h.MarkAllRead)
		r.Patch("/read-trip", h.MarkTripRead)
		r.Post("/cleanup", h.Cleanup)
		r.Patch("/{id}/read", h.MarkRead)
	})
}

// RegisterPublicRoutes mounts notification routes that do not require user
// authentication. VAPID public keys are safe to expose.
func (h *Handler) RegisterPublicRoutes(r chi.Router) {
	if h.push != nil {
		r.Get("/notifications/push/public-key", h.PushPublicKey)
	}
}

// PushPublicKey handles GET /notifications/push/public-key.
func (h *Handler) PushPublicKey(w http.ResponseWriter, _ *http.Request) {
	if h.push == nil {
		writeJSON(w, http.StatusOK, response.PushPublicKey{Enabled: false})
		return
	}
	result := h.push.PublicKey()
	writeJSON(w, http.StatusOK, response.PushPublicKey{
		Enabled:   result.Enabled,
		PublicKey: result.PublicKey,
	})
}

// SubscribePush handles POST /notifications/push/subscribe.
func (h *Handler) SubscribePush(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if h.push == nil {
		writeJSON(w, http.StatusOK, response.PushSubscribe{Subscribed: false, Enabled: false})
		return
	}

	var req request.SubscribePush
	if !decodeJSON(w, r, &req) {
		return
	}
	subscribed, err := h.push.Subscribe(r.Context(), push.SubscribeInput{
		UserID:      user.ID,
		Endpoint:    strings.TrimSpace(req.Subscription.Endpoint),
		P256DH:      strings.TrimSpace(req.Subscription.Keys.P256DH),
		Auth:        strings.TrimSpace(req.Subscription.Keys.Auth),
		UserAgent:   request.NormalizeOptionalString(req.UserAgent),
		Browser:     request.NormalizeOptionalString(req.Browser),
		DeviceLabel: request.NormalizeOptionalString(req.DeviceLabel),
	})
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, response.PushSubscribe{Subscribed: subscribed, Enabled: subscribed})
}

// UnsubscribePush handles DELETE /notifications/push/unsubscribe.
func (h *Handler) UnsubscribePush(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if h.push == nil {
		writeJSON(w, http.StatusOK, response.PushUnsubscribe{Unsubscribed: true})
		return
	}

	var req request.UnsubscribePush
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := h.push.Unsubscribe(r.Context(), user.ID, strings.TrimSpace(req.Endpoint)); err != nil {
		writeServiceError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, response.PushUnsubscribe{Unsubscribed: true})
}

// PushStatus handles GET /notifications/push/status.
func (h *Handler) PushStatus(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if h.push == nil {
		writeJSON(w, http.StatusOK, response.PushStatus{Enabled: false})
		return
	}
	status, err := h.push.Status(r.Context(), user.ID)
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, response.PushStatus{
		Enabled:             status.Enabled,
		ActiveSubscriptions: status.ActiveSubscriptions,
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
			recordNotificationSSEEventDropped("connect", "max_connections")
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
	recordNotificationSSEConnection("active", 1)
	defer func() {
		h.streams.Unregister(user.ID, client.ID)
		recordNotificationSSEConnection("active", -1)
	}()

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
			recordNotificationSSEEventDropped(event.Name, "write_failed")
			h.log.Debug("write notification stream event failed",
				zap.String("user_id", user.ID.String()),
				zap.String("client_id", client.ID),
				zap.String("event", event.Name),
				zap.Error(err),
			)
			return false
		}
		flusher.Flush()
		recordNotificationSSEEventSent(event.Name)
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
	if h.controls != nil {
		settings, settingsErr := h.controls.GetSettings(r.Context(), user.ID)
		if settingsErr != nil {
			writeServiceError(w, h.log, settingsErr)
			return
		}
		result.Settings = settingsResult(*settings)
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
	if req.Items == nil && req.Settings == nil {
		writeError(w, http.StatusBadRequest, "items or settings is required")
		return
	}

	var result *preferences.PreferencesResult
	if req.Items != nil {
		inputs, inputErr := req.ToInputs()
		if inputErr != nil {
			writeServiceError(w, h.log, inputErr)
			return
		}
		result, err = h.preferences.UpdatePreferences(r.Context(), user.ID, inputs)
	} else {
		result, err = h.preferences.GetPreferences(r.Context(), user.ID)
	}
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}
	if h.controls != nil && req.Settings != nil {
		settings, settingsErr := h.controls.UpdateSettings(r.Context(), controls.SettingsInput{
			UserID: user.ID, QuietHoursEnabled: req.Settings.QuietHoursEnabled,
			QuietHoursStart: req.Settings.QuietHoursStart, QuietHoursEnd: req.Settings.QuietHoursEnd,
			QuietHoursTimezone:       req.Settings.QuietHoursTimezone,
			UrgentBypassesQuietHours: req.Settings.UrgentBypassesQuietHours,
			DailyDigestTime:          req.Settings.DailyDigestTime, WeeklyDigestDay: req.Settings.WeeklyDigestDay,
			WeeklyDigestTime: req.Settings.WeeklyDigestTime,
		})
		if settingsErr != nil {
			writeServiceError(w, h.log, settingsErr)
			return
		}
		result.Settings = settingsResult(*settings)
	}

	writeJSON(w, http.StatusOK, response.NewNotificationPreferences(result))
}

func (h *Handler) ListTripMutes(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID, err := uuid.Parse(strings.TrimSpace(r.URL.Query().Get("tripId")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "tripId must be a valid uuid")
		return
	}
	items, err := h.controls.ListTripMutes(r.Context(), user.ID, tripID)
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripMutes(items))
}

func (h *Handler) UpsertTripMute(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req request.UpsertTripMute
	if !decodeJSON(w, r, &req) {
		return
	}
	tripID, err := uuid.Parse(strings.TrimSpace(req.TripID))
	if err != nil {
		writeError(w, http.StatusBadRequest, "tripId must be a valid uuid")
		return
	}
	item, err := h.controls.UpsertTripMute(r.Context(), controls.TripMuteInput{UserID: user.ID, TripID: tripID, Category: req.Category, MutedUntil: req.MutedUntil})
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewTripMutes([]entity.NotificationTripMute{*item}).Items[0])
}

func (h *Handler) DeleteTripMute(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "muteId")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid mute id")
		return
	}
	if err := h.controls.DeleteTripMute(r.Context(), id, user.ID); err != nil {
		writeServiceError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, response.Success{Success: true})
}

func (h *Handler) ListPendingDigests(w http.ResponseWriter, r *http.Request) {
	h.listDigests(w, r, true)
}
func (h *Handler) ListDigestHistory(w http.ResponseWriter, r *http.Request) {
	h.listDigests(w, r, false)
}
func (h *Handler) listDigests(w http.ResponseWriter, r *http.Request, pending bool) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	limit, ok := parseLimit(w, r.URL.Query().Get("limit"))
	if !ok {
		return
	}
	if limit == 0 {
		limit = 20
	}
	var items []entity.NotificationDigestBatch
	if pending {
		items, err = h.digests.ListPending(r.Context(), user.ID, limit)
	} else {
		items, err = h.digests.ListHistory(r.Context(), user.ID, limit)
	}
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewDigestList(items))
}
func (h *Handler) GetDigest(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := uuid.Parse(strings.TrimSpace(chi.URLParam(r, "digestId")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid digest id")
		return
	}
	batch, err := h.digests.Get(r.Context(), id, user.ID)
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, response.NewDigestBatch(*batch))
}

func settingsResult(value entity.NotificationSettings) preferences.NotificationSettings {
	return preferences.NotificationSettings{
		QuietHoursEnabled: value.QuietHoursEnabled, QuietHoursStart: value.QuietHoursStart,
		QuietHoursEnd: value.QuietHoursEnd, QuietHoursTimezone: value.QuietHoursTimezone,
		UrgentBypassesQuietHours: value.UrgentBypassesQuietHours, DailyDigestTime: value.DailyDigestTime,
		WeeklyDigestDay: value.WeeklyDigestDay, WeeklyDigestTime: value.WeeklyDigestTime,
	}
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

func (h *Handler) MarkTripRead(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tripID, err := uuid.Parse(strings.TrimSpace(r.URL.Query().Get("tripId")))
	if err != nil {
		writeError(w, http.StatusBadRequest, "tripId must be a valid uuid")
		return
	}
	service, ok := h.svc.(interface {
		MarkTripRead(context.Context, uuid.UUID, uuid.UUID) (int, error)
	})
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if _, err := service.MarkTripRead(r.Context(), user.ID, tripID); err != nil {
		writeServiceError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, response.Success{Success: true})
}

// Cleanup handles POST /notifications/cleanup. The endpoint makes hard
// deletion clear to callers and protects unread notifications unless the user
// explicitly opts out.
func (h *Handler) Cleanup(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req request.CleanupNotifications
	if !decodeJSON(w, r, &req) {
		return
	}
	onlyRead := true
	if req.OnlyRead != nil {
		onlyRead = *req.OnlyRead
	}
	service, ok := h.svc.(interface {
		Cleanup(context.Context, notifications.CleanupInput) (int, error)
	})
	if !ok {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	count, err := service.Cleanup(r.Context(), notifications.CleanupInput{UserID: user.ID, OlderThanDays: req.OlderThanDays, OnlyRead: onlyRead, Categories: req.Categories})
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}
	recordNotificationCleanupDeleted(count)
	writeJSON(w, http.StatusOK, map[string]int{"deletedOrArchivedCount": count})
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
