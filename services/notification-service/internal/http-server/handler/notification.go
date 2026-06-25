package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server/dto/response"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
)

// notificationService is the user-facing port. The concrete notifications.Service
// satisfies it; tests substitute a fake.
type notificationService interface {
	List(ctx context.Context, in notifications.ListInput) (*notifications.ListResult, error)
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
	MarkRead(ctx context.Context, id, userID uuid.UUID) (*entity.Notification, error)
	MarkAllRead(ctx context.Context, userID uuid.UUID) (int, error)
}

// Handler serves a user's own notifications. All routes it registers must be
// mounted behind JWT auth so user_id always comes from a validated token.
type Handler struct {
	svc notificationService
	log *zap.Logger
}

// New constructs the user-facing notification HTTP handler.
func New(svc notificationService, log *zap.Logger) *Handler {
	if log == nil {
		log = zap.NewNop()
	}
	return &Handler{svc: svc, log: log}
}

// RegisterRoutes mounts the user-facing notification routes. The caller is
// responsible for wrapping these in JWT auth middleware.
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/notifications", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/unread-count", h.UnreadCount)
		r.Patch("/read-all", h.MarkAllRead)
		r.Patch("/{id}/read", h.MarkRead)
	})
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
