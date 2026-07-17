package push

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/preferences"
	pushdelivery "github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/pkg/push"
)

const (
	disableReasonUnsubscribed = "user_unsubscribed"
	disableReasonGone         = "push_service_gone"
	disableReasonInvalid      = "push_service_invalid"
)

// Repository is the persistence port for browser push subscriptions.
type Repository interface {
	UpsertPushSubscription(ctx context.Context, subscription entity.PushSubscription) (*entity.PushSubscription, error)
	GetPushSubscriptionByEndpoint(ctx context.Context, endpoint string) (*entity.PushSubscription, error)
	ListActivePushSubscriptionsByUserID(ctx context.Context, userID uuid.UUID) ([]entity.PushSubscription, error)
	DisablePushSubscriptionByEndpoint(ctx context.Context, userID uuid.UUID, endpoint, reason string) error
	DisablePushSubscriptionByID(ctx context.Context, id uuid.UUID, reason string) error
	CountActivePushSubscriptionsByUserID(ctx context.Context, userID uuid.UUID) (int, error)
	UpdatePushSubscriptionLastUsed(ctx context.Context, id uuid.UUID) error
}

// PreferenceGate reports whether push is enabled for a recipient/type pair.
// preferences.EffectiveSet implements it without creating an import cycle.
type PreferenceGate interface {
	AllowPush(userID uuid.UUID, notificationType string) bool
}

// Service validates subscriptions, exposes public-key/status data, and fans
// out push notifications for internal batches.
type Service struct {
	cfg    Config
	repo   Repository
	sender pushdelivery.PushSender
	log    *zap.Logger
}

// New constructs the push service.
func New(cfg Config, repo Repository, sender pushdelivery.PushSender, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	if sender == nil {
		sender = pushdelivery.NewMockSender(log)
	}
	cfg.Enabled = cfg.Enabled && strings.TrimSpace(cfg.VAPIDPublicKey) != "" && strings.TrimSpace(cfg.VAPIDPrivateKey) != ""
	return &Service{cfg: cfg, repo: repo, sender: sender, log: log}
}

// PublicKey returns the browser-visible VAPID public key when push is enabled.
func (s *Service) PublicKey() PublicKeyResult {
	if !s.cfg.Enabled {
		return PublicKeyResult{Enabled: false}
	}
	key := s.cfg.VAPIDPublicKey
	return PublicKeyResult{Enabled: true, PublicKey: &key}
}

// Subscribe stores or refreshes one browser push subscription for the user.
func (s *Service) Subscribe(ctx context.Context, input SubscribeInput) (bool, error) {
	if !s.cfg.Enabled {
		return false, nil
	}
	if err := validateSubscribeInput(input); err != nil {
		return false, err
	}
	subscription := entity.PushSubscription{
		ID:          uuid.New(),
		UserID:      input.UserID,
		Endpoint:    input.Endpoint,
		P256DH:      input.P256DH,
		Auth:        input.Auth,
		UserAgent:   input.UserAgent,
		Browser:     input.Browser,
		DeviceLabel: input.DeviceLabel,
	}
	stored, err := s.repo.UpsertPushSubscription(ctx, subscription)
	if err != nil {
		return false, err
	}
	fields := []zap.Field{
		zap.String("userId", input.UserID.String()),
		zap.String("endpointHash", pushdelivery.EndpointHash(input.Endpoint)),
	}
	if stored != nil {
		fields = append(fields, zap.String("subscriptionId", stored.ID.String()))
	}
	s.log.Info("push_subscribed", fields...)
	return true, nil
}

// Unsubscribe soft-disables the current user's endpoint. Missing endpoints are
// treated as success for idempotent browser cleanup.
func (s *Service) Unsubscribe(ctx context.Context, userID uuid.UUID, endpoint string) error {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return apperrs.NewInvalidInput("endpoint is required")
	}
	if len(endpoint) > MaxEndpointLength {
		return apperrs.NewInvalidInput("endpoint must be at most %d characters", MaxEndpointLength)
	}
	if err := s.repo.DisablePushSubscriptionByEndpoint(ctx, userID, endpoint, disableReasonUnsubscribed); err != nil {
		return err
	}
	s.log.Info("push_unsubscribed",
		zap.String("userId", userID.String()),
		zap.String("endpointHash", pushdelivery.EndpointHash(endpoint)),
	)
	return nil
}

// Status reports whether push is globally enabled and how many active
// subscriptions the current user has.
func (s *Service) Status(ctx context.Context, userID uuid.UUID) (*StatusResult, error) {
	result := &StatusResult{Enabled: s.cfg.Enabled}
	if !s.cfg.Enabled {
		return result, nil
	}
	count, err := s.repo.CountActivePushSubscriptionsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	result.ActiveSubscriptions = count
	return result, nil
}

// SendPushForNotifications sends browser push notifications for eligible
// batch candidates. It never includes secrets or full private objects in the
// payload, and it disables subscriptions that push services report as gone or
// invalid.
func (s *Service) SendPushForNotifications(ctx context.Context, notifications []entity.Notification, gates ...PreferenceGate) (BatchResult, error) {
	var result BatchResult
	gate := firstGate(gates)
	if !s.cfg.Enabled {
		result.Skipped = len(notifications)
		return result, nil
	}

	var firstErr error
	for i := range notifications {
		notification := notifications[i]
		if !shouldConsiderPush(notification) {
			result.Skipped++
			continue
		}
		if gate != nil && !gate.AllowPush(notification.UserID, notification.Type) {
			result.Skipped++
			result.SkippedByPreference++
			continue
		}
		category, _ := preferences.CategoryForNotificationType(notification.Type)
		payload := buildPayload(notification, category)
		subscriptions, err := s.repo.ListActivePushSubscriptionsByUserID(ctx, notification.UserID)
		if err != nil {
			result.Failed++
			if firstErr == nil {
				firstErr = err
			}
			s.log.Warn("push_send_failed",
				zap.String("userId", notification.UserID.String()),
				zap.String("notificationType", notification.Type),
				zap.String("category", category),
				zap.String("errorCode", "list_subscriptions_failed"),
				zap.Error(err),
			)
			continue
		}
		if len(subscriptions) == 0 {
			result.Skipped++
			continue
		}

		for j := range subscriptions {
			subscription := subscriptions[j]
			result.Attempted++
			sendResult, err := s.sender.Send(ctx, pushdelivery.PushSubscription{
				ID:       subscription.ID,
				UserID:   subscription.UserID,
				Endpoint: subscription.Endpoint,
				P256DH:   subscription.P256DH,
				Auth:     subscription.Auth,
			}, payload)
			statusCode := 0
			if sendResult != nil {
				statusCode = sendResult.StatusCode
			}
			if err != nil {
				result.Failed++
				if firstErr == nil {
					firstErr = err
				}
				s.log.Warn("push_send_failed",
					zap.String("userId", notification.UserID.String()),
					zap.String("notificationType", notification.Type),
					zap.String("category", category),
					zap.String("subscriptionId", subscription.ID.String()),
					zap.String("endpointHash", pushdelivery.EndpointHash(subscription.Endpoint)),
					zap.Int("statusCode", statusCode),
					zap.String("errorCode", "send_failed"),
					zap.Error(err),
				)
				continue
			}
			if sendResult != nil && sendResult.SubscriptionGone {
				result.SubscriptionsDisabled++
				if isGoneStatus(sendResult.StatusCode) {
					result.SubscriptionsDisabledAsGone++
				} else {
					result.SubscriptionsDisabledAsInvalid++
				}
				reason := disableReasonInvalid
				if isGoneStatus(sendResult.StatusCode) {
					reason = disableReasonGone
				}
				if err := s.repo.DisablePushSubscriptionByID(ctx, subscription.ID, reason); err != nil {
					result.Failed++
					if firstErr == nil {
						firstErr = err
					}
					s.log.Warn("push_subscription_disabled_failed",
						zap.String("userId", notification.UserID.String()),
						zap.String("subscriptionId", subscription.ID.String()),
						zap.String("endpointHash", pushdelivery.EndpointHash(subscription.Endpoint)),
						zap.Int("statusCode", sendResult.StatusCode),
						zap.Error(err),
					)
					continue
				}
				s.log.Info("push_subscription_disabled",
					zap.String("userId", notification.UserID.String()),
					zap.String("notificationType", notification.Type),
					zap.String("category", category),
					zap.String("subscriptionId", subscription.ID.String()),
					zap.String("endpointHash", pushdelivery.EndpointHash(subscription.Endpoint)),
					zap.Int("statusCode", sendResult.StatusCode),
					zap.String("errorCode", reason),
				)
				continue
			}
			result.Sent++
			if err := s.repo.UpdatePushSubscriptionLastUsed(ctx, subscription.ID); err != nil {
				s.log.Warn("update push subscription last_used_at failed",
					zap.String("subscriptionId", subscription.ID.String()),
					zap.String("endpointHash", pushdelivery.EndpointHash(subscription.Endpoint)),
					zap.Error(err),
				)
			}
			s.log.Info("push_send_success",
				zap.String("userId", notification.UserID.String()),
				zap.String("notificationType", notification.Type),
				zap.String("category", category),
				zap.String("subscriptionId", subscription.ID.String()),
				zap.String("endpointHash", pushdelivery.EndpointHash(subscription.Endpoint)),
				zap.Int("statusCode", statusCode),
			)
		}
	}

	if firstErr != nil && !s.cfg.FailOpen {
		return result, fmt.Errorf("send push notifications: %w", firstErr)
	}
	return result, nil
}

func validateSubscribeInput(input SubscribeInput) error {
	if input.UserID == uuid.Nil {
		return apperrs.NewInvalidInput("user id is required")
	}
	endpoint := strings.TrimSpace(input.Endpoint)
	if endpoint == "" {
		return apperrs.NewInvalidInput("endpoint is required")
	}
	if len(endpoint) > MaxEndpointLength {
		return apperrs.NewInvalidInput("endpoint must be at most %d characters", MaxEndpointLength)
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return apperrs.NewInvalidInput("endpoint must be a valid https URL")
	}
	if strings.TrimSpace(input.P256DH) == "" {
		return apperrs.NewInvalidInput("subscription.keys.p256dh is required")
	}
	if len(input.P256DH) > MaxKeyLength {
		return apperrs.NewInvalidInput("subscription.keys.p256dh must be at most %d characters", MaxKeyLength)
	}
	if strings.TrimSpace(input.Auth) == "" {
		return apperrs.NewInvalidInput("subscription.keys.auth is required")
	}
	if len(input.Auth) > MaxAuthLength {
		return apperrs.NewInvalidInput("subscription.keys.auth must be at most %d characters", MaxAuthLength)
	}
	if err := validateOptional(input.UserAgent, "userAgent", MaxUserAgentLength); err != nil {
		return err
	}
	if err := validateOptional(input.Browser, "browser", MaxBrowserLength); err != nil {
		return err
	}
	if err := validateOptional(input.DeviceLabel, "deviceLabel", MaxDeviceLabelLength); err != nil {
		return err
	}
	return nil
}

func validateOptional(value *string, field string, max int) error {
	if value == nil {
		return nil
	}
	if len(*value) > max {
		return apperrs.NewInvalidInput("%s must be at most %d characters", field, max)
	}
	return nil
}

func firstGate(gates []PreferenceGate) PreferenceGate {
	if len(gates) == 0 {
		return nil
	}
	return gates[0]
}

func shouldConsiderPush(notification entity.Notification) bool {
	if notification.UserID == uuid.Nil {
		return false
	}
	if notification.ActorUserID != nil && *notification.ActorUserID == notification.UserID {
		return false
	}
	_, ok := pushTypeAllowlist[notification.Type]
	return ok
}

var pushTypeAllowlist = map[string]struct{}{
	notifications.TypeCollaborationInvited:     {},
	notifications.TypeCollaborationAccepted:    {},
	notifications.TypeCollaboratorRoleChange:   {},
	notifications.TypeCollaboratorRemoved:      {},
	notifications.TypeCommentCreated:           {},
	notifications.TypeItineraryGenerated:       {},
	notifications.TypeGenerationJobFailed:      {},
	notifications.TypeBudgetOptimizationReady:  {},
	notifications.TypeBudgetOptimizationFailed: {},
	notifications.TypeNotificationDigest:       {},
}

func (s *Service) SendDigest(ctx context.Context, userID uuid.UUID, title, body string) error {
	_, err := s.SendPushForNotifications(ctx, []entity.Notification{{
		ID: uuid.New(), UserID: userID, Type: notifications.TypeNotificationDigest,
		Title: title, Message: body, Category: notifications.CategorySystem,
		Priority: notifications.PriorityNormal,
	}})
	return err
}

func buildPayload(notification entity.Notification, category string) pushdelivery.PushPayload {
	title, body := safeTitleBody(notification)
	return pushdelivery.PushPayload{
		Title:          title,
		Body:           body,
		URL:            safeNotificationURL(notification),
		NotificationID: persistedNotificationID(notification),
		Type:           notification.Type,
		Category:       category,
	}
}

func safeTitleBody(notification entity.Notification) (string, string) {
	switch notification.Type {
	case notifications.TypeCollaborationInvited:
		return "Trip invitation", "You were invited to collaborate on a trip."
	case notifications.TypeCollaborationAccepted:
		return "Invitation accepted", "A collaborator accepted your trip invitation."
	case notifications.TypeCollaboratorRoleChange:
		return "Trip role changed", "Your role changed on a trip."
	case notifications.TypeCollaboratorRemoved:
		return "Trip access changed", "You were removed from a trip."
	case notifications.TypeCommentCreated:
		return "New comment", "Someone commented on your trip."
	case notifications.TypeItineraryGenerated:
		return "Itinerary ready", "Your itinerary has been generated."
	case notifications.TypeGenerationJobFailed:
		return "Generation failed", "Your itinerary generation failed."
	case notifications.TypeBudgetOptimizationReady:
		return "Budget proposal ready", "A budget optimization proposal is ready to review."
	case notifications.TypeBudgetOptimizationFailed:
		return "Budget optimization failed", "Your budget optimization request failed."
	case notifications.TypeNotificationDigest:
		title := strings.TrimSpace(notification.Title)
		if title == "" {
			title = "Trip update digest"
		}
		body := strings.TrimSpace(notification.Message)
		if body == "" {
			body = "Open the app to review your grouped updates."
		}
		return truncate(title, 80), truncate(body, 160)
	default:
		title := strings.TrimSpace(notification.Title)
		if title == "" {
			title = "Travel update"
		}
		return truncate(title, 80), "Open the app for details."
	}
}

func safeNotificationURL(notification entity.Notification) string {
	if notification.Type == notifications.TypeCollaborationInvited {
		return "/trips"
	}
	if notification.TripID != nil && *notification.TripID != uuid.Nil {
		return "/trips/" + notification.TripID.String()
	}
	return "/notifications"
}

func persistedNotificationID(notification entity.Notification) string {
	if notification.ID == uuid.Nil || notification.CreatedAt.IsZero() {
		return ""
	}
	return notification.ID.String()
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func isGoneStatus(status int) bool {
	return status == 404 || status == 410
}
