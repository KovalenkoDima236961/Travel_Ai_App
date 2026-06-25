package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/emailnotifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/preferences"
)

// internalService is the create port used by trusted internal callers. It now
// returns the created notifications so the handler can fan out email.
type internalService interface {
	CreateBatchWithPreferences(ctx context.Context, inputs []notifications.CreateInput, gate notifications.InAppPreferenceGate) (*notifications.BatchCreateResult, error)
}

// emailDispatcher sends email for selected notification types after the in-app
// rows are created. The emailnotifications.Service satisfies it; tests
// substitute a noop/fake.
type emailDispatcher interface {
	SendEmailsForNotifications(ctx context.Context, notifications []entity.Notification, gates ...emailnotifications.EmailPreferenceGate) (emailnotifications.EmailSendResult, error)
}

// internalPreferenceService loads effective recipient preferences for an
// internal batch. The preferences.Service satisfies it.
type internalPreferenceService interface {
	EffectiveForUsers(ctx context.Context, userIDs []uuid.UUID) (*preferences.EffectiveSet, error)
}

// InternalHandler serves service-to-service endpoints. Its routes must be
// mounted behind the internal service-token middleware — never exposed to
// browsers and never requiring a user JWT.
type InternalHandler struct {
	svc         internalService
	emails      emailDispatcher
	preferences internalPreferenceService
	log         *zap.Logger
}

// NewInternal constructs the internal notification HTTP handler. A nil email
// dispatcher disables email fan-out (a noop) so callers/tests that do not wire
// email keep working.
func NewInternal(svc internalService, emails emailDispatcher, log *zap.Logger, preferenceSvc ...internalPreferenceService) *InternalHandler {
	if log == nil {
		log = zap.NewNop()
	}
	if emails == nil {
		emails = noopEmailDispatcher{}
	}
	var prefs internalPreferenceService
	if len(preferenceSvc) > 0 {
		prefs = preferenceSvc[0]
	}
	return &InternalHandler{svc: svc, emails: emails, preferences: prefs, log: log}
}

// RegisterRoutes mounts the internal routes. The caller wraps these in the
// internal service-token middleware.
func (h *InternalHandler) RegisterRoutes(r chi.Router) {
	r.Route("/internal/notifications", func(r chi.Router) {
		r.Post("/batch", h.CreateBatch)
	})
}

// batchResponse is the body of POST /internal/notifications/batch. It reports
// how many in-app notifications were created plus an email fan-out summary.
type batchResponse struct {
	Requested           int                                `json:"requested"`
	Created             int                                `json:"created"`
	Skipped             int                                `json:"skipped"`
	SkippedByPreference int                                `json:"skippedByPreference"`
	Email               emailnotifications.EmailSendResult `json:"email"`
}

// CreateBatch handles POST /internal/notifications/batch. It trusts the caller
// to provide recipient user ids; it skips self-notifications, creates the in-app
// rows, then fans out email for selected types.
//
// In-app notification creation always happens first and is never rolled back
// because of an email failure. When email is fail-open (or disabled), a send
// failure is reported in the response's email.failed count with HTTP 201. When
// email is fail-closed and a send fails, the rows still exist but the endpoint
// returns 502 so the caller can observe the degraded delivery.
func (h *InternalHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var req request.CreateNotificationsBatch
	if !decodeJSON(w, r, &req) {
		return
	}

	inputs, err := req.ToInputs()
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}

	var preferenceSet *preferences.EffectiveSet
	if h.preferences != nil {
		preferenceSet, err = h.preferences.EffectiveForUsers(r.Context(), recipientIDs(inputs))
		if err != nil {
			writeServiceError(w, h.log, err)
			return
		}
	}

	batchResult, err := h.svc.CreateBatchWithPreferences(r.Context(), inputs, preferenceSet)
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}

	emailResult, emailErr := h.emails.SendEmailsForNotifications(r.Context(), batchResult.EmailCandidates, preferenceSet)
	if emailErr != nil {
		// Fail-closed: rows are committed but email delivery failed. Surface a
		// 502 so the caller can observe degraded delivery; details are logged.
		h.log.Warn("email delivery failed for created notifications (fail-closed)",
			zap.Int("created", len(batchResult.Created)),
			zap.Int("email_failed", emailResult.Failed),
			zap.Error(emailErr),
		)
		writeError(w, http.StatusBadGateway, "notifications created but email delivery failed")
		return
	}

	writeJSON(w, http.StatusCreated, batchResponse{
		Requested:           batchResult.Requested,
		Created:             len(batchResult.Created),
		Skipped:             batchResult.Skipped,
		SkippedByPreference: batchResult.SkippedByPreference,
		Email:               emailResult,
	})
}

// noopEmailDispatcher reports every created notification as skipped without
// sending. It is used when email is not wired (e.g. in unit tests of the HTTP
// layer that do not exercise email).
type noopEmailDispatcher struct{}

func (noopEmailDispatcher) SendEmailsForNotifications(_ context.Context, notifications []entity.Notification, _ ...emailnotifications.EmailPreferenceGate) (emailnotifications.EmailSendResult, error) {
	return emailnotifications.EmailSendResult{Skipped: len(notifications)}, nil
}

func recipientIDs(inputs []notifications.CreateInput) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(inputs))
	for _, input := range inputs {
		ids = append(ids, input.UserID)
	}
	return ids
}
