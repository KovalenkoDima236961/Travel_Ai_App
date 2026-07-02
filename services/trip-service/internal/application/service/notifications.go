package service

import (
	"context"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
	tripobs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/observability"
)

// notifier is the Notification Service port. The concrete notifications.Client
// satisfies it; tests substitute a fake to assert which notifications are sent.
type notifier interface {
	CreateNotifications(ctx context.Context, notifications []notifications.NotificationCreateInput) error
}

// WithNotifications enables synchronous in-app notification fan-out after
// successful actions. When not configured (or disabled), sending is a no-op so
// older trips and tests keep working.
//
// failOpen controls behavior when the Notification Service call fails. In v1 a
// failure never breaks the originating action (it has already been committed and
// its activity event recorded); failOpen only controls log severity so a
// misconfiguration is still visible.
func WithNotifications(n notifier, enabled, failOpen bool) Option {
	return func(s *Service) {
		s.notifier = n
		s.notificationsEnabled = enabled
		s.notificationsFailOpen = failOpen
	}
}

// sendNotifications fans out a batch best-effort. It is only ever called after
// the originating action has succeeded, so a failure is logged and swallowed —
// a notification problem must never roll back a saved trip change.
func (s *Service) sendNotifications(ctx context.Context, inputs []notifications.NotificationCreateInput) {
	if !s.notificationsEnabled || s.notifier == nil || len(inputs) == 0 {
		return
	}
	if err := s.notifier.CreateNotifications(ctx, inputs); err != nil {
		recordNotificationRequestMetrics(inputs, "error")
		if s.notificationsFailOpen {
			s.log.Warn("failed to send notifications (fail-open)",
				zap.Int("count", len(inputs)),
				zap.Error(err),
			)
			return
		}
		s.log.Error("failed to send notifications",
			zap.Int("count", len(inputs)),
			zap.Error(err),
		)
		return
	}
	recordNotificationRequestMetrics(inputs, "success")
}

func recordNotificationRequestMetrics(inputs []notifications.NotificationCreateInput, result string) {
	for _, input := range inputs {
		tripobs.RecordNotificationsRequested(input.Type, result, 1)
	}
}

// broadcastRecipients returns the trip owner plus accepted collaborators,
// excluding the actor, de-duplicated. Pending/removed collaborators and the
// actor themselves never receive a notification. A nil/empty result means
// "nobody to notify" and sendNotifications will skip the HTTP call entirely.
func (s *Service) broadcastRecipients(ctx context.Context, trip *entity.Trip, actorID uuid.UUID) []uuid.UUID {
	seen := map[uuid.UUID]struct{}{actorID: {}}
	recipients := make([]uuid.UUID, 0)

	if trip != nil && trip.UserID != nil {
		if _, ok := seen[*trip.UserID]; !ok {
			seen[*trip.UserID] = struct{}{}
			recipients = append(recipients, *trip.UserID)
		}
	}

	collaborators, err := s.repo.ListTripCollaborators(ctx, trip.ID)
	if err != nil {
		// Recipient resolution is best-effort; degrade to whoever we already have
		// (typically the owner) rather than failing the surrounding action.
		s.log.Warn("failed to list collaborators for notification fan-out",
			zap.String("trip_id", trip.ID.String()),
			zap.Error(err),
		)
		return recipients
	}
	for _, collaborator := range collaborators {
		if collaborator.Status != entity.CollaboratorStatusAccepted {
			continue
		}
		if _, ok := seen[collaborator.UserID]; ok {
			continue
		}
		seen[collaborator.UserID] = struct{}{}
		recipients = append(recipients, collaborator.UserID)
	}
	return recipients
}

// --- per-event notification builders ---

// notifyTripBroadcast notifies the trip owner and accepted collaborators (except
// the actor) of an itinerary/comment change. The same title/message/metadata is
// delivered to each recipient.
func (s *Service) notifyTripBroadcast(
	ctx context.Context,
	trip *entity.Trip,
	actorID uuid.UUID,
	notificationType, title, message string,
	entityType string,
	entityID *uuid.UUID,
	metadata map[string]any,
) {
	if !s.notificationsEnabled || s.notifier == nil || trip == nil {
		return
	}
	recipients := s.broadcastRecipients(ctx, trip, actorID)
	if len(recipients) == 0 {
		return
	}

	tripID := trip.ID
	actor := actorID
	inputs := make([]notifications.NotificationCreateInput, 0, len(recipients))
	for _, recipient := range recipients {
		inputs = append(inputs, notifications.NotificationCreateInput{
			UserID:      recipient,
			TripID:      &tripID,
			ActorUserID: &actor,
			Type:        notificationType,
			Title:       title,
			Message:     message,
			EntityType:  activityEntityType(entityType),
			EntityID:    entityID,
			Metadata:    metadata,
		})
	}
	s.sendNotifications(ctx, inputs)
}

// notifyDirect notifies a single recipient (used for collaboration events that
// target one specific user). It skips self-notifications.
func (s *Service) notifyDirect(
	ctx context.Context,
	recipientID uuid.UUID,
	tripID uuid.UUID,
	actorID uuid.UUID,
	notificationType, title, message string,
	entityType string,
	entityID *uuid.UUID,
	metadata map[string]any,
) {
	if !s.notificationsEnabled || s.notifier == nil {
		return
	}
	if recipientID == uuid.Nil || recipientID == actorID {
		return
	}
	trip := tripID
	actor := actorID
	s.sendNotifications(ctx, []notifications.NotificationCreateInput{{
		UserID:      recipientID,
		TripID:      &trip,
		ActorUserID: &actor,
		Type:        notificationType,
		Title:       title,
		Message:     message,
		EntityType:  activityEntityType(entityType),
		EntityID:    entityID,
		Metadata:    metadata,
	}})
}

// tripDestination returns a trip's destination, or a neutral fallback so a
// notification message never reads "for ." when the destination is somehow blank.
func tripDestination(trip *entity.Trip) string {
	if trip == nil || trip.Destination == "" {
		return "your trip"
	}
	return trip.Destination
}
