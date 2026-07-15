package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/calendar"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

type CalendarHandler struct {
	svc *calendar.Service
	log *zap.Logger
}

func NewCalendarHandler(svc *calendar.Service, log *zap.Logger) *CalendarHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &CalendarHandler{svc: svc, log: log}
}

func (h *CalendarHandler) Status(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	status, err := h.svc.Status(r.Context(), user.ID)
	if err != nil {
		h.writeCalendarError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *CalendarHandler) Connect(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		ReturnURL string `json:"returnUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	authURL, err := h.svc.StartConnect(r.Context(), user.ID, req.ReturnURL)
	if err != nil {
		h.writeCalendarError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"authUrl": authURL})
}

func (h *CalendarHandler) Callback(w http.ResponseWriter, r *http.Request) {
	redirectURL, err := h.svc.HandleCallback(
		r.Context(),
		r.URL.Query().Get("code"),
		r.URL.Query().Get("state"),
		r.URL.Query().Get("error"),
	)
	if err != nil {
		h.log.Warn("calendar oauth callback failed", zap.Error(err))
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *CalendarHandler) Disconnect(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if err := h.svc.Disconnect(r.Context(), user.ID); err != nil {
		h.writeCalendarError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *CalendarHandler) FreeBusy(w http.ResponseWriter, r *http.Request) {
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req calendar.FreeBusyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := h.svc.FreeBusy(r.Context(), user.ID, req)
	if err != nil {
		h.writeCalendarError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *CalendarHandler) writeCalendarError(w http.ResponseWriter, err error) {
	var limitErr *providerlimits.LimitError
	if errors.As(err, &limitErr) {
		status := http.StatusServiceUnavailable
		if limitErr.Code == providerlimits.CodeRateLimited {
			status = http.StatusTooManyRequests
		}
		writeError(w, status, limitErr.Code)
		return
	}
	switch {
	case errors.Is(err, calendar.ErrCalendarDisabled):
		writeError(w, http.StatusServiceUnavailable, "calendar_disabled")
	case errors.Is(err, calendar.ErrCalendarFreeBusyDisabled):
		writeError(w, http.StatusServiceUnavailable, "calendar_free_busy_disabled")
	case errors.Is(err, calendar.ErrCalendarNotConnected):
		writeError(w, http.StatusNotFound, "calendar_not_connected")
	case errors.Is(err, calendar.ErrCalendarReauthRequired):
		writeError(w, http.StatusConflict, "calendar_connection_revoked")
	case errors.Is(err, calendar.ErrCalendarFreeBusyInvalidRange):
		writeError(w, http.StatusBadRequest, "invalid_date_range")
	case errors.Is(err, calendar.ErrCalendarFreeBusyRangeTooLarge):
		writeError(w, http.StatusBadRequest, "date_range_too_large")
	case errors.Is(err, calendar.ErrCalendarFreeBusyInvalidTimeZone):
		writeError(w, http.StatusBadRequest, "invalid_timezone")
	case errors.Is(err, calendar.ErrCalendarFreeBusyUnsupportedCalendar):
		writeError(w, http.StatusBadRequest, "unsupported_calendar_ids")
	case errors.Is(err, calendar.ErrCalendarFreeBusyMalformedResponse):
		writeError(w, http.StatusBadGateway, "calendar_free_busy_malformed_response")
	case errors.Is(err, calendar.ErrCalendarFreeBusyUnavailable):
		writeError(w, http.StatusBadGateway, "calendar_free_busy_unavailable")
	default:
		h.log.Warn("calendar request failed", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "calendar_request_failed")
	}
}

type InternalCalendarHandler struct {
	svc *calendar.Service
	log *zap.Logger
}

func NewInternalCalendarHandler(svc *calendar.Service, log *zap.Logger) *InternalCalendarHandler {
	if log == nil {
		log = zap.NewNop()
	}
	return &InternalCalendarHandler{svc: svc, log: log}
}

func (h *InternalCalendarHandler) SyncGoogleEvents(w http.ResponseWriter, r *http.Request) {
	var req calendar.SyncEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "userId is required")
		return
	}
	result, err := h.svc.SyncEvents(r.Context(), req)
	if err != nil {
		h.writeCalendarError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *InternalCalendarHandler) DeleteGoogleEvents(w http.ResponseWriter, r *http.Request) {
	var req calendar.DeleteEventsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.UserID == uuid.Nil {
		writeError(w, http.StatusBadRequest, "userId is required")
		return
	}
	result, err := h.svc.DeleteEvents(r.Context(), req)
	if err != nil {
		h.writeCalendarError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *InternalCalendarHandler) writeCalendarError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, calendar.ErrCalendarDisabled):
		writeError(w, http.StatusServiceUnavailable, "calendar_disabled")
	case errors.Is(err, calendar.ErrCalendarNotConnected):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "calendar_not_connected"})
	case errors.Is(err, calendar.ErrCalendarReauthRequired):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "calendar_reauth_required"})
	default:
		h.log.Warn("internal calendar request failed", zap.String("error", sanitizeCalendarError(err)))
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "provider_error"})
	}
}

func sanitizeCalendarError(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	for _, marker := range []string{"access_token", "refresh_token", "client_secret", "code="} {
		if strings.Contains(strings.ToLower(msg), marker) {
			return "redacted"
		}
	}
	return msg
}
