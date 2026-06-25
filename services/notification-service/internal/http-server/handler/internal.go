package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/emailnotifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/http-server/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
)

// internalService is the create port used by trusted internal callers. It now
// returns the created notifications so the handler can fan out email.
type internalService interface {
	CreateBatch(ctx context.Context, inputs []notifications.CreateInput) ([]entity.Notification, error)
}

// emailDispatcher sends email for selected notification types after the in-app
// rows are created. The emailnotifications.Service satisfies it; tests
// substitute a noop/fake.
type emailDispatcher interface {
	SendEmailsForNotifications(ctx context.Context, notifications []entity.Notification) (emailnotifications.EmailSendResult, error)
}

// InternalHandler serves service-to-service endpoints. Its routes must be
// mounted behind the internal service-token middleware — never exposed to
// browsers and never requiring a user JWT.
type InternalHandler struct {
	svc    internalService
	emails emailDispatcher
	log    *zap.Logger
}

// NewInternal constructs the internal notification HTTP handler. A nil email
// dispatcher disables email fan-out (a noop) so callers/tests that do not wire
// email keep working.
func NewInternal(svc internalService, emails emailDispatcher, log *zap.Logger) *InternalHandler {
	if log == nil {
		log = zap.NewNop()
	}
	if emails == nil {
		emails = noopEmailDispatcher{}
	}
	return &InternalHandler{svc: svc, emails: emails, log: log}
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
	Created int                                `json:"created"`
	Email   emailnotifications.EmailSendResult `json:"email"`
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

	created, err := h.svc.CreateBatch(r.Context(), inputs)
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}

	emailResult, emailErr := h.emails.SendEmailsForNotifications(r.Context(), created)
	if emailErr != nil {
		// Fail-closed: rows are committed but email delivery failed. Surface a
		// 502 so the caller can observe degraded delivery; details are logged.
		h.log.Warn("email delivery failed for created notifications (fail-closed)",
			zap.Int("created", len(created)),
			zap.Int("email_failed", emailResult.Failed),
			zap.Error(emailErr),
		)
		writeError(w, http.StatusBadGateway, "notifications created but email delivery failed")
		return
	}

	writeJSON(w, http.StatusCreated, batchResponse{
		Created: len(created),
		Email:   emailResult,
	})
}

// noopEmailDispatcher reports every created notification as skipped without
// sending. It is used when email is not wired (e.g. in unit tests of the HTTP
// layer that do not exercise email).
type noopEmailDispatcher struct{}

func (noopEmailDispatcher) SendEmailsForNotifications(_ context.Context, notifications []entity.Notification) (emailnotifications.EmailSendResult, error) {
	return emailnotifications.EmailSendResult{Skipped: len(notifications)}, nil
}
