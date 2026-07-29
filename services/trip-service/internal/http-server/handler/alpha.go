package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/alpha"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/observability"
)

func (h *Handler) EnableAlpha(service *alpha.Service) *Handler {
	h.alpha = service
	return h
}

func (h *Handler) registerOpsAlphaRoutes(r chi.Router) {
	r.Route("/alpha", func(r chi.Router) {
		r.Get("/invites", h.OpsListAlphaInvites)
		r.Post("/invites", h.OpsCreateAlphaInvite)
		r.Patch("/invites/{inviteId}", h.OpsUpdateAlphaInvite)
		r.Delete("/invites/{inviteId}", h.OpsDisableAlphaInvite)
		r.Get("/waitlist", h.OpsListAlphaWaitlist)
		r.Patch("/waitlist/{waitlistId}", h.OpsUpdateAlphaWaitlist)
		r.Post("/invite-from-waitlist", h.OpsInviteFromWaitlist)
		r.Get("/dashboard", h.OpsAlphaDashboard)
		r.Get("/feedback", h.OpsListAlphaFeedback)
		r.Get("/feedback/{feedbackId}", h.OpsGetAlphaFeedback)
		r.Patch("/feedback/{feedbackId}", h.OpsUpdateAlphaFeedback)
		r.Get("/reports/weekly", h.OpsListWeeklyAlphaReports)
		r.Post("/reports/weekly/generate", h.OpsGenerateWeeklyAlphaReport)
	})
}

func (h *Handler) JoinAlphaWaitlist(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) || !h.allowAlphaRate(w, r, h.alphaWaitlistLimiter, "waitlist") {
		return
	}
	var req struct {
		Email  string `json:"email"`
		Source string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	entry, err := h.alpha.JoinWaitlist(r.Context(), alpha.JoinWaitlistInput{Email: req.Email, Source: firstAlpha(req.Source, "web")})
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"waitlist": entry})
}

func (h *Handler) ActivateAlphaInvite(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) || !h.allowAlphaRate(w, r, h.alphaInviteLimiter, "invite") {
		return
	}
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	participant, err := h.alpha.ActivateInvite(r.Context(), alpha.ActivateInviteInput{
		UserID:    user.ID,
		Code:      req.Code,
		RequestID: observability.RequestIDFromContext(r.Context()),
	})
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"participant": participant})
}

func (h *Handler) GetAlphaParticipant(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) {
		return
	}
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	participant, err := h.alpha.GetParticipant(r.Context(), user.ID)
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"participant": participant})
}

func (h *Handler) RecordAnalyticsEvent(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) || !h.allowAlphaRate(w, r, h.alphaEventLimiter, "event") {
		return
	}
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req struct {
		SessionID     string          `json:"sessionId"`
		EventName     string          `json:"eventName"`
		Feature       string          `json:"feature"`
		EntityType    string          `json:"entityType"`
		EntityID      string          `json:"entityId"`
		Metadata      json.RawMessage `json:"metadata"`
		OccurredAt    *time.Time      `json:"occurredAt"`
		CorrelationID string          `json:"correlationId"`
		AppVersion    string          `json:"appVersion"`
		BrowserFamily string          `json:"browserFamily"`
		OSFamily      string          `json:"osFamily"`
		DeviceType    string          `json:"deviceType"`
		Source        string          `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	event, err := h.alpha.RecordEvent(r.Context(), alpha.EventInput{
		UserID:        &user.ID,
		SessionID:     req.SessionID,
		EventName:     req.EventName,
		Feature:       req.Feature,
		EntityType:    req.EntityType,
		EntityID:      req.EntityID,
		Metadata:      req.Metadata,
		OccurredAt:    req.OccurredAt,
		RequestID:     observability.RequestIDFromContext(r.Context()),
		CorrelationID: req.CorrelationID,
		AppVersion:    req.AppVersion,
		BrowserFamily: req.BrowserFamily,
		OSFamily:      req.OSFamily,
		DeviceType:    req.DeviceType,
		Source:        req.Source,
	})
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"event": event})
}

func (h *Handler) SubmitAlphaFeedback(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) || !h.allowAlphaRate(w, r, h.alphaFeedbackLimiter, "feedback") {
		return
	}
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	input, ok := h.decodeFeedbackInput(w, r, user.ID)
	if !ok {
		return
	}
	detail, err := h.alpha.SubmitFeedback(r.Context(), input)
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, detail)
}

func (h *Handler) GetMyAlphaFeedback(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) {
		return
	}
	user, err := auth.MustUserFromContext(r.Context())
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, ok := parseUUIDParam(w, r, "feedbackId", "invalid feedback id")
	if !ok {
		return
	}
	detail, err := h.alpha.GetFeedback(r.Context(), id)
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	if detail.Feedback.UserID != user.ID {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) OpsListAlphaInvites(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) {
		return
	}
	limit, offset, ok := parseAlphaLimitOffset(w, r)
	if !ok {
		return
	}
	invites, err := h.alpha.ListInvites(r.Context(), limit, offset)
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invites": invites, "limit": limit, "offset": offset})
}

func (h *Handler) OpsCreateAlphaInvite(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) {
		return
	}
	user, _ := auth.UserFromContext(r.Context())
	var req struct {
		Code           string     `json:"code"`
		ExpiresAt      *time.Time `json:"expiresAt"`
		MaxActivations int        `json:"maxActivations"`
		Notes          string     `json:"notes"`
		TesterGroup    string     `json:"testerGroup"`
		Enabled        *bool      `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	invite, err := h.alpha.CreateInvite(r.Context(), alpha.CreateInviteInput{
		Code:           req.Code,
		ExpiresAt:      req.ExpiresAt,
		MaxActivations: req.MaxActivations,
		CreatorUserID:  user.ID,
		Notes:          req.Notes,
		TesterGroup:    req.TesterGroup,
		Enabled:        enabled,
	})
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"invite": invite})
}

func (h *Handler) OpsUpdateAlphaInvite(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) {
		return
	}
	id, ok := parseUUIDParam(w, r, "inviteId", "invalid invite id")
	if !ok {
		return
	}
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input := alpha.UpdateInviteInput{ID: id}
	if value, exists := raw["expiresAt"]; exists {
		expiresAt, ok := parseNullableTime(w, value)
		if !ok {
			return
		}
		input.ExpiresAt = &expiresAt
	}
	if value, exists := raw["maxActivations"]; exists {
		var parsed int
		if err := json.Unmarshal(value, &parsed); err != nil {
			writeError(w, http.StatusBadRequest, "invalid maxActivations")
			return
		}
		input.MaxActivations = &parsed
	}
	if value, exists := raw["notes"]; exists {
		var parsed string
		if err := json.Unmarshal(value, &parsed); err != nil {
			writeError(w, http.StatusBadRequest, "invalid notes")
			return
		}
		input.Notes = &parsed
	}
	if value, exists := raw["testerGroup"]; exists {
		var parsed string
		if err := json.Unmarshal(value, &parsed); err != nil {
			writeError(w, http.StatusBadRequest, "invalid testerGroup")
			return
		}
		input.TesterGroup = &parsed
	}
	if value, exists := raw["enabled"]; exists {
		var parsed bool
		if err := json.Unmarshal(value, &parsed); err != nil {
			writeError(w, http.StatusBadRequest, "invalid enabled")
			return
		}
		input.Enabled = &parsed
	}
	invite, err := h.alpha.UpdateInvite(r.Context(), input)
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invite": invite})
}

func (h *Handler) OpsDisableAlphaInvite(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) {
		return
	}
	id, ok := parseUUIDParam(w, r, "inviteId", "invalid invite id")
	if !ok {
		return
	}
	invite, err := h.alpha.DisableInvite(r.Context(), id)
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"invite": invite})
}

func (h *Handler) OpsListAlphaWaitlist(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) {
		return
	}
	limit, offset, ok := parseAlphaLimitOffset(w, r)
	if !ok {
		return
	}
	entries, err := h.alpha.ListWaitlist(r.Context(), strings.TrimSpace(r.URL.Query().Get("status")), limit, offset)
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"waitlist": entries, "limit": limit, "offset": offset})
}

func (h *Handler) OpsUpdateAlphaWaitlist(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) {
		return
	}
	id, ok := parseUUIDParam(w, r, "waitlistId", "invalid waitlist id")
	if !ok {
		return
	}
	var req struct {
		Status *string `json:"status"`
		Notes  *string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	entry, err := h.alpha.UpdateWaitlist(r.Context(), alpha.UpdateWaitlistInput{ID: id, Status: req.Status, Notes: req.Notes})
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"waitlist": entry})
}

func (h *Handler) OpsInviteFromWaitlist(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) {
		return
	}
	user, _ := auth.UserFromContext(r.Context())
	var req struct {
		WaitlistID     string     `json:"waitlistId"`
		TesterGroup    string     `json:"testerGroup"`
		Notes          string     `json:"notes"`
		ExpiresAt      *time.Time `json:"expiresAt"`
		MaxActivations int        `json:"maxActivations"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	waitlistID, err := uuid.Parse(strings.TrimSpace(req.WaitlistID))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid waitlist id")
		return
	}
	invite, entry, err := h.alpha.InviteFromWaitlist(r.Context(), alpha.InviteFromWaitlistInput{
		WaitlistID:     waitlistID,
		CreatorUserID:  user.ID,
		TesterGroup:    req.TesterGroup,
		Notes:          req.Notes,
		ExpiresAt:      req.ExpiresAt,
		MaxActivations: req.MaxActivations,
	})
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"invite": invite, "waitlist": entry})
}

func (h *Handler) OpsAlphaDashboard(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) {
		return
	}
	dashboard, err := h.alpha.Dashboard(r.Context())
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dashboard)
}

func (h *Handler) OpsListAlphaFeedback(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) {
		return
	}
	limit, offset, ok := parseAlphaLimitOffset(w, r)
	if !ok {
		return
	}
	items, err := h.alpha.ListFeedback(r.Context(), r.URL.Query().Get("category"), r.URL.Query().Get("status"), limit, offset)
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"feedback": items, "limit": limit, "offset": offset})
}

func (h *Handler) OpsGetAlphaFeedback(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) {
		return
	}
	id, ok := parseUUIDParam(w, r, "feedbackId", "invalid feedback id")
	if !ok {
		return
	}
	detail, err := h.alpha.GetFeedback(r.Context(), id)
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handler) OpsUpdateAlphaFeedback(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) {
		return
	}
	id, ok := parseUUIDParam(w, r, "feedbackId", "invalid feedback id")
	if !ok {
		return
	}
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	input := alpha.UpdateFeedbackInput{ID: id}
	if value, exists := raw["status"]; exists {
		var parsed string
		if err := json.Unmarshal(value, &parsed); err != nil {
			writeError(w, http.StatusBadRequest, "invalid status")
			return
		}
		input.Status = &parsed
	}
	if value, exists := raw["priority"]; exists {
		var parsed string
		if err := json.Unmarshal(value, &parsed); err != nil {
			writeError(w, http.StatusBadRequest, "invalid priority")
			return
		}
		input.Priority = &parsed
	}
	if value, exists := raw["ownerUserId"]; exists {
		ownerID, ok := parseNullableUUID(w, value)
		if !ok {
			return
		}
		input.OwnerUserID = &ownerID
	}
	if value, exists := raw["internalNotes"]; exists {
		var parsed string
		if err := json.Unmarshal(value, &parsed); err != nil {
			writeError(w, http.StatusBadRequest, "invalid internalNotes")
			return
		}
		input.InternalNotes = &parsed
	}
	feedback, err := h.alpha.UpdateFeedback(r.Context(), input)
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"feedback": feedback})
}

func (h *Handler) OpsListWeeklyAlphaReports(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) {
		return
	}
	limit, offset, ok := parseAlphaLimitOffset(w, r)
	if !ok {
		return
	}
	reports, err := h.alpha.ListWeeklyReports(r.Context(), limit, offset)
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reports": reports, "limit": limit, "offset": offset})
}

func (h *Handler) OpsGenerateWeeklyAlphaReport(w http.ResponseWriter, r *http.Request) {
	if !h.alphaAvailable(w) {
		return
	}
	user, _ := auth.UserFromContext(r.Context())
	var req struct {
		WeekStart string `json:"weekStart"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var weekStart time.Time
	if strings.TrimSpace(req.WeekStart) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(req.WeekStart))
		if err != nil {
			writeError(w, http.StatusBadRequest, "weekStart must be YYYY-MM-DD")
			return
		}
		weekStart = parsed
	}
	report, err := h.alpha.GenerateWeeklyReport(r.Context(), weekStart, &user.ID)
	if err != nil {
		h.writeAlphaError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"report": report})
}

func (h *Handler) decodeFeedbackInput(w http.ResponseWriter, r *http.Request, userID uuid.UUID) (alpha.SubmitFeedbackInput, bool) {
	contentType := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(contentType, "multipart/form-data") {
		return h.decodeMultipartFeedback(w, r, userID)
	}
	var req struct {
		Category      string                          `json:"category"`
		Title         string                          `json:"title"`
		Description   string                          `json:"description"`
		Metadata      json.RawMessage                 `json:"metadata"`
		AppVersion    string                          `json:"appVersion"`
		BrowserFamily string                          `json:"browserFamily"`
		OSFamily      string                          `json:"osFamily"`
		DeviceType    string                          `json:"deviceType"`
		CorrelationID string                          `json:"correlationId"`
		Provider      string                          `json:"provider"`
		ModelAlias    string                          `json:"modelAlias"`
		PromptVersion string                          `json:"promptVersion"`
		FeatureFlags  json.RawMessage                 `json:"featureFlags"`
		Attachments   []alpha.FeedbackAttachmentInput `json:"attachments"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return alpha.SubmitFeedbackInput{}, false
	}
	return alpha.SubmitFeedbackInput{
		UserID:        userID,
		Category:      req.Category,
		Title:         req.Title,
		Description:   req.Description,
		Metadata:      req.Metadata,
		AppVersion:    req.AppVersion,
		BrowserFamily: req.BrowserFamily,
		OSFamily:      req.OSFamily,
		DeviceType:    req.DeviceType,
		RequestID:     observability.RequestIDFromContext(r.Context()),
		CorrelationID: req.CorrelationID,
		Provider:      req.Provider,
		ModelAlias:    req.ModelAlias,
		PromptVersion: req.PromptVersion,
		FeatureFlags:  req.FeatureFlags,
		Attachments:   req.Attachments,
	}, true
}

func (h *Handler) decodeMultipartFeedback(w http.ResponseWriter, r *http.Request, userID uuid.UUID) (alpha.SubmitFeedbackInput, bool) {
	if err := r.ParseMultipartForm(6 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart feedback")
		return alpha.SubmitFeedbackInput{}, false
	}
	input := alpha.SubmitFeedbackInput{
		UserID:        userID,
		Category:      r.FormValue("category"),
		Title:         r.FormValue("title"),
		Description:   r.FormValue("description"),
		AppVersion:    r.FormValue("appVersion"),
		BrowserFamily: r.FormValue("browserFamily"),
		OSFamily:      r.FormValue("osFamily"),
		DeviceType:    r.FormValue("deviceType"),
		RequestID:     observability.RequestIDFromContext(r.Context()),
		CorrelationID: r.FormValue("correlationId"),
		Provider:      r.FormValue("provider"),
		ModelAlias:    r.FormValue("modelAlias"),
		PromptVersion: r.FormValue("promptVersion"),
		Metadata:      json.RawMessage(firstAlpha(r.FormValue("metadata"), "{}")),
		FeatureFlags:  json.RawMessage(firstAlpha(r.FormValue("featureFlags"), "{}")),
	}
	files := r.MultipartForm.File["screenshot"]
	if len(files) > 3 {
		writeError(w, http.StatusBadRequest, "at most 3 attachments are allowed")
		return alpha.SubmitFeedbackInput{}, false
	}
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid attachment")
			return alpha.SubmitFeedbackInput{}, false
		}
		hash := sha256.New()
		n, err := io.Copy(hash, io.LimitReader(file, 5*1024*1024+1))
		_ = file.Close()
		if err != nil || n <= 0 || n > 5*1024*1024 {
			writeError(w, http.StatusBadRequest, "attachment rejected")
			return alpha.SubmitFeedbackInput{}, false
		}
		input.Attachments = append(input.Attachments, alpha.FeedbackAttachmentInput{
			FileName:      fileHeader.Filename,
			MIMEType:      fileHeader.Header.Get("Content-Type"),
			SizeBytes:     int(n),
			ContentSHA256: hex.EncodeToString(hash.Sum(nil)),
		})
	}
	return input, true
}

func (h *Handler) alphaAvailable(w http.ResponseWriter) bool {
	if h.alpha == nil || !h.alpha.Available() {
		writeError(w, http.StatusServiceUnavailable, "alpha program is not configured")
		return false
	}
	return true
}

func (h *Handler) allowAlphaRate(w http.ResponseWriter, r *http.Request, limiter interface{ Allow(string) bool }, scope string) bool {
	if limiter == nil || limiter.Allow(alphaRateKey(r, scope)) {
		return true
	}
	writeJSON(w, http.StatusTooManyRequests, map[string]any{
		"error":   "rate_limited",
		"message": "Too many alpha requests. Please retry later.",
	})
	return false
}

func alphaRateKey(r *http.Request, scope string) string {
	user, ok := auth.UserFromContext(r.Context())
	if ok && user.ID != uuid.Nil {
		return scope + ":user:" + user.ID.String()
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return scope + ":ip:" + host
	}
	return scope + ":ip:" + r.RemoteAddr
}

func parseAlphaLimitOffset(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	limit, ok := parseQueryInt(w, r, "limit")
	if !ok {
		return 0, 0, false
	}
	offset, ok := parseQueryInt(w, r, "offset")
	if !ok {
		return 0, 0, false
	}
	return limit, offset, true
}

func parseNullableTime(w http.ResponseWriter, raw json.RawMessage) (*time.Time, bool) {
	if strings.TrimSpace(string(raw)) == "null" {
		return nil, true
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		writeError(w, http.StatusBadRequest, "invalid expiresAt")
		return nil, false
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		parsed, err = time.Parse("2006-01-02", strings.TrimSpace(value))
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "expiresAt must be RFC3339 or YYYY-MM-DD")
		return nil, false
	}
	return &parsed, true
}

func parseNullableUUID(w http.ResponseWriter, raw json.RawMessage) (*uuid.UUID, bool) {
	if strings.TrimSpace(string(raw)) == "null" {
		return nil, true
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		writeError(w, http.StatusBadRequest, "invalid ownerUserId")
		return nil, false
	}
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid ownerUserId")
		return nil, false
	}
	return &parsed, true
}

func firstAlpha(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (h *Handler) writeAlphaError(w http.ResponseWriter, err error) {
	var invalid *alpha.InputError
	switch {
	case errors.As(err, &invalid):
		writeError(w, http.StatusBadRequest, invalid.Error())
	case errors.Is(err, alpha.ErrInviteUnavailable):
		writeError(w, http.StatusForbidden, "invite code is invalid, expired, disabled, or fully used")
	case errors.Is(err, alpha.ErrAttachmentRejected):
		writeError(w, http.StatusBadRequest, "feedback attachment rejected")
	case errors.Is(err, alpha.ErrNotFound):
		writeError(w, http.StatusNotFound, "alpha record not found")
	default:
		h.log.Error("unhandled alpha service error", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
