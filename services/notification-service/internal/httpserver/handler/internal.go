package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/deliverypolicy"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/digests"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/emailnotifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/httpserver/dto/request"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/httpserver/dto/response"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/preferences"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/push"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/stream"
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

// pushDispatcher sends browser push for selected notification types after the
// batch has been validated and in-app rows have been created/skipped.
type pushDispatcher interface {
	SendPushForNotifications(ctx context.Context, notifications []entity.Notification, gates ...push.PreferenceGate) (push.BatchResult, error)
}

// internalPreferenceService loads effective recipient preferences for an
// internal batch. The preferences.Service satisfies it.
type internalPreferenceService interface {
	EffectiveForUsers(ctx context.Context, userIDs []uuid.UUID) (*preferences.EffectiveSet, error)
}

type internalControlsService interface {
	SettingsForUsers(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]entity.NotificationSettings, error)
	ActiveTripMutesForUsers(ctx context.Context, userIDs []uuid.UUID, now time.Time) ([]entity.NotificationTripMute, error)
}

type internalDigestService interface {
	Queue(ctx context.Context, input digests.QueueInput) (bool, error)
	ProcessDue(ctx context.Context, input digests.ProcessInput) (*digests.ProcessResult, error)
}

// InternalHandler serves service-to-service endpoints. Its routes must be
// mounted behind the internal service-token middleware — never exposed to
// browsers and never requiring a user JWT.
type InternalHandler struct {
	svc         internalService
	emails      emailDispatcher
	pushes      pushDispatcher
	preferences internalPreferenceService
	controls    internalControlsService
	digests     internalDigestService
	streams     stream.Manager
	log         *zap.Logger
}

func (h *InternalHandler) EnableNoiseControl(controls internalControlsService, digestService internalDigestService) *InternalHandler {
	h.controls = controls
	h.digests = digestService
	return h
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

// EnableStream wires optional in-memory fanout after batch creation.
func (h *InternalHandler) EnableStream(manager stream.Manager) *InternalHandler {
	h.streams = manager
	return h
}

// EnablePush wires optional browser push fanout after batch creation.
func (h *InternalHandler) EnablePush(pushes pushDispatcher) *InternalHandler {
	h.pushes = pushes
	return h
}

// RegisterRoutes mounts the internal routes. The caller wraps these in the
// internal service-token middleware.
func (h *InternalHandler) RegisterRoutes(r chi.Router) {
	r.Route("/internal/notifications", func(r chi.Router) {
		r.Post("/batch", h.CreateBatch)
		if h.digests != nil {
			r.Post("/process-digests", h.ProcessDigests)
		}
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
	Push                push.BatchResult                   `json:"push"`
	Digested            int                                `json:"digested"`
	Muted               int                                `json:"muted"`
	Delayed             int                                `json:"delayed"`
	Grouped             int                                `json:"grouped"`
	DuplicatesDropped   int                                `json:"duplicatesDropped"`
}

// CreateBatch handles POST /internal/notifications/batch. It trusts the caller
// to provide recipient user ids; it skips self-notifications, creates the in-app
// rows, then fans out email and push for selected types.
//
// In-app notification creation always happens first and is never rolled back
// because of an email or push failure. When a channel is fail-open (or
// disabled), send failures are reported in response stats with HTTP 201. When a
// channel is fail-closed and a send fails, the rows still exist but the endpoint
// returns 502 so the caller can observe the degraded delivery.
func (h *InternalHandler) CreateBatch(w http.ResponseWriter, r *http.Request) {
	var req request.CreateNotificationsBatch
	if !decodeJSON(w, r, &req) {
		return
	}

	inputs, err := req.ToInputs()
	if err != nil {
		for _, item := range req.Notifications {
			recordNotificationFailed(item.Type, "in_app", "invalid_request")
		}
		writeServiceError(w, h.log, err)
		return
	}

	var preferenceSet *preferences.EffectiveSet
	var policySet *deliverypolicy.BatchPolicy
	if h.preferences != nil {
		preferenceSet, err = h.preferences.EffectiveForUsers(r.Context(), recipientIDs(inputs))
		if err != nil {
			for _, input := range inputs {
				recordNotificationFailed(input.Type, "in_app", "preferences_error")
			}
			writeServiceError(w, h.log, err)
			return
		}
	}
	if h.controls != nil {
		ids := recipientIDs(inputs)
		settings, settingsErr := h.controls.SettingsForUsers(r.Context(), ids)
		if settingsErr != nil {
			writeServiceError(w, h.log, settingsErr)
			return
		}
		mutes, mutesErr := h.controls.ActiveTripMutesForUsers(r.Context(), ids, time.Now().UTC())
		if mutesErr != nil {
			writeServiceError(w, h.log, mutesErr)
			return
		}
		policySet = deliverypolicy.NewBatchPolicy(preferenceSet, settings, mutes, time.Now().UTC())
	}
	var inAppGate notifications.InAppPreferenceGate = preferenceSet
	if policySet != nil {
		inAppGate = policySet
	}

	batchResult, err := h.svc.CreateBatchWithPreferences(r.Context(), inputs, inAppGate)
	if err != nil {
		for _, input := range inputs {
			recordNotificationFailed(input.Type, "in_app", "create_failed")
		}
		writeServiceError(w, h.log, err)
		return
	}
	for i := range batchResult.Created {
		recordNotificationCreated(batchResult.Created[i].Type, "in_app")
	}
	recordDedupeDropped(batchResult.DuplicatesDropped)

	createdToPublish := batchResult.Created
	digested, muted, delayed, emailMutedByPreference := 0, 0, 0, 0
	if policySet != nil {
		createdToPublish = createdToPublish[:0]
		for _, notification := range batchResult.Created {
			decision := policySet.Evaluate(notification, preferences.ChannelInApp)
			recordDeliveryDecision(preferences.ChannelInApp, notification.Category, notification.Priority, decision.Mode, decision.Decision, decision.Reason)
			if policySet.ImmediateInApp(notification) {
				createdToPublish = append(createdToPublish, notification)
			}
			if decision.ScheduledFor != nil && decision.Mode != preferences.ModeInstant && h.digests != nil {
				if _, queueErr := h.digests.Queue(r.Context(), digests.QueueInput{Notification: notification, Channel: preferences.ChannelInApp, Mode: decision.Mode, ScheduledFor: *decision.ScheduledFor}); queueErr != nil {
					writeServiceError(w, h.log, queueErr)
					return
				}
				digested++
			}
		}
		// A related low/normal event can update an existing in-app row instead
		// of creating a second card. It still contributes one occurrence to a
		// scheduled in-app digest, but it does not emit another live-create event.
		for _, notification := range batchResult.GroupedInApp {
			decision := policySet.Evaluate(notification, preferences.ChannelInApp)
			recordDeliveryDecision(preferences.ChannelInApp, notification.Category, notification.Priority, decision.Mode, decision.Decision, decision.Reason)
			if decision.ScheduledFor != nil && decision.Mode != preferences.ModeInstant && h.digests != nil {
				if _, queueErr := h.digests.Queue(r.Context(), digests.QueueInput{Notification: notification, Channel: preferences.ChannelInApp, Mode: decision.Mode, ScheduledFor: *decision.ScheduledFor}); queueErr != nil {
					writeServiceError(w, h.log, queueErr)
					return
				}
				digested++
			}
		}
	}
	h.publishCreated(r.Context(), createdToPublish)

	emailCandidates := batchResult.EmailCandidates
	pushCandidates := batchResult.EmailCandidates
	if policySet != nil {
		emailCandidates = make([]entity.Notification, 0, len(batchResult.EmailCandidates))
		pushCandidates = make([]entity.Notification, 0, len(batchResult.EmailCandidates))
		for _, notification := range batchResult.EmailCandidates {
			for _, channel := range []string{preferences.ChannelEmail, preferences.ChannelPush} {
				decision := policySet.Evaluate(notification, channel)
				recordDeliveryDecision(channel, notification.Category, notification.Priority, decision.Mode, decision.Decision, decision.Reason)
				switch decision.Decision {
				case deliverypolicy.DecisionSendInstant:
					if channel == preferences.ChannelEmail {
						emailCandidates = append(emailCandidates, notification)
					} else {
						pushCandidates = append(pushCandidates, notification)
					}
				case deliverypolicy.DecisionDigest, deliverypolicy.DecisionDelayQuietHours:
					if h.digests != nil && decision.ScheduledFor != nil {
						if _, queueErr := h.digests.Queue(r.Context(), digests.QueueInput{Notification: notification, Channel: channel, Mode: decision.Mode, ScheduledFor: *decision.ScheduledFor}); queueErr != nil {
							writeServiceError(w, h.log, queueErr)
							return
						}
					}
					if decision.Decision == deliverypolicy.DecisionDelayQuietHours {
						delayed++
						recordQuietHoursDelayed(channel, notification.Category)
					} else {
						digested++
					}
				case deliverypolicy.DecisionMute:
					muted++
					if channel == preferences.ChannelEmail {
						emailMutedByPreference++
					}
				}
			}
		}
	}

	emailResult, emailErr := h.emails.SendEmailsForNotifications(r.Context(), emailCandidates, preferenceSet)
	emailResult.Skipped += emailMutedByPreference
	emailResult.SkippedByPreference += emailMutedByPreference
	recordNotificationEmail("batch", "sent", emailResult.Sent)
	recordNotificationEmail("batch", "failed", emailResult.Failed)
	recordNotificationEmail("batch", "skipped", emailResult.Skipped)

	pushResult := push.BatchResult{Skipped: len(pushCandidates)}
	var pushErr error
	if h.pushes != nil {
		pushResult, pushErr = h.pushes.SendPushForNotifications(r.Context(), pushCandidates, preferenceSet)
		recordNotificationPush("batch", "batch", "sent", pushResult.Sent)
		recordNotificationPush("batch", "batch", "failed", pushResult.Failed)
		recordNotificationPush("batch", "batch", "skipped", pushResult.Skipped)
		recordNotificationPush("batch", "batch", "disabled", pushResult.SubscriptionsDisabled)
	}
	h.log.Info("notification delivery decision summary",
		zap.Int("requested", batchResult.Requested),
		zap.Int("in_app_created", len(batchResult.Created)),
		zap.Int("digested", digested),
		zap.Int("muted", muted),
		zap.Int("quiet_hours_delayed", delayed),
		zap.Int("grouped", batchResult.Grouped),
		zap.Int("duplicates_dropped", batchResult.DuplicatesDropped),
		zap.Int("email_sent", emailResult.Sent),
		zap.Int("push_sent", pushResult.Sent),
	)
	if emailErr != nil {
		recordNotificationFailed("batch", "email", "send_failed")
		h.log.Warn("email delivery failed for created notifications (fail-closed)",
			zap.Int("created", len(batchResult.Created)),
			zap.Int("email_failed", emailResult.Failed),
			zap.Error(emailErr),
		)
		writeError(w, http.StatusBadGateway, "notifications created but email delivery failed")
		return
	}
	if pushErr != nil {
		recordNotificationFailed("batch", "push", "send_failed")
		h.log.Warn("push delivery failed for created notifications (fail-closed)",
			zap.Int("created", len(batchResult.Created)),
			zap.Int("push_failed", pushResult.Failed),
			zap.Error(pushErr),
		)
		writeError(w, http.StatusBadGateway, "notifications created but push delivery failed")
		return
	}

	writeJSON(w, http.StatusCreated, batchResponse{
		Requested:           batchResult.Requested,
		Created:             len(batchResult.Created),
		Skipped:             batchResult.Skipped,
		SkippedByPreference: batchResult.SkippedByPreference,
		Email:               emailResult,
		Push:                pushResult,
		Digested:            digested,
		Muted:               muted,
		Delayed:             delayed,
		Grouped:             batchResult.Grouped,
		DuplicatesDropped:   batchResult.DuplicatesDropped,
	})
}

func (h *InternalHandler) ProcessDigests(w http.ResponseWriter, r *http.Request) {
	var req request.ProcessDigests
	if !decodeJSON(w, r, &req) {
		return
	}
	now := time.Now().UTC()
	if req.Now != nil {
		now = req.Now.UTC()
	}
	result, err := h.digests.ProcessDue(r.Context(), digests.ProcessInput{Now: now, Limit: req.Limit})
	if err != nil {
		writeServiceError(w, h.log, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

type notificationCreatedPayload struct {
	Notification response.Notification `json:"notification"`
}

func (h *InternalHandler) publishCreated(ctx context.Context, created []entity.Notification) {
	if h.streams == nil || len(created) == 0 {
		return
	}
	for i := range created {
		notification := created[i]
		h.streams.PublishToUser(ctx, notification.UserID, stream.StreamEvent{
			Name: stream.EventNotificationCreated,
			Data: notificationCreatedPayload{
				Notification: response.NewNotification(notification),
			},
		})
	}
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
