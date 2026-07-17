package deliverypolicy

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/preferences"
)

func TestDeliveryPolicyUrgentSecurityIsInstant(t *testing.T) {
	n := policyNotification(notifications.TypeShareSecurityChanged, notifications.CategorySecurity, notifications.PriorityUrgent)
	policy := testPolicy(n.UserID, nil, nil, time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC))
	decision := policy.Evaluate(n, preferences.ChannelEmail)
	if decision.Decision != DecisionSendInstant {
		t.Fatalf("expected instant, got %+v", decision)
	}
}

func TestDeliveryPolicyNormalTripUpdateUsesDailyEmailDigest(t *testing.T) {
	n := policyNotification(notifications.TypeItineraryUpdated, notifications.CategoryTripUpdates, notifications.PriorityNormal)
	policy := testPolicy(n.UserID, nil, nil, time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC))
	decision := policy.Evaluate(n, preferences.ChannelEmail)
	if decision.Decision != DecisionDigest || decision.Mode != preferences.ModeDailyDigest {
		t.Fatalf("expected daily digest, got %+v", decision)
	}
}

func TestDeliveryPolicyMutedCategorySuppresses(t *testing.T) {
	n := policyNotification(notifications.TypeCommentCreated, notifications.CategoryComments, notifications.PriorityNormal)
	rows := []entity.NotificationPreference{{UserID: n.UserID, Channel: preferences.ChannelEmail, Category: preferences.CategoryComments, Enabled: false, DeliveryMode: preferences.ModeMuted}}
	policy := testPolicy(n.UserID, rows, nil, time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC))
	if decision := policy.Evaluate(n, preferences.ChannelEmail); decision.Decision != DecisionMute {
		t.Fatalf("expected mute, got %+v", decision)
	}
}

func TestDeliveryPolicyPreservesLegacyReminderPreference(t *testing.T) {
	n := policyNotification(notifications.TypePreTripReminderDue, notifications.CategoryReminders, notifications.PriorityHigh)
	rows := []entity.NotificationPreference{{
		UserID: n.UserID, Channel: preferences.ChannelEmail,
		Category: preferences.CategoryPreTripReminders, Enabled: false, DeliveryMode: preferences.ModeMuted,
	}}
	policy := testPolicy(n.UserID, rows, nil, time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC))
	if decision := policy.Evaluate(n, preferences.ChannelEmail); decision.Decision != DecisionMute {
		t.Fatalf("expected migrated pre-trip reminder mute to remain effective, got %+v", decision)
	}
}

func TestDeliveryPolicyQuietHoursDelayAndUrgentBypass(t *testing.T) {
	userID := uuid.New()
	now := time.Date(2026, 7, 17, 23, 0, 0, 0, time.UTC)
	settings := map[uuid.UUID]entity.NotificationSettings{userID: {UserID: userID, QuietHoursEnabled: true, QuietHoursStart: "22:00", QuietHoursEnd: "08:00", QuietHoursTimezone: "UTC", UrgentBypassesQuietHours: true, DailyDigestTime: "08:00", WeeklyDigestDay: 1, WeeklyDigestTime: "08:00"}}
	policy := NewBatchPolicy(preferences.BuildEffectiveSet([]uuid.UUID{userID}, nil), settings, nil, now)
	normal := policyNotification(notifications.TypeCommentCreated, notifications.CategoryComments, notifications.PriorityNormal)
	normal.UserID = userID
	if decision := policy.Evaluate(normal, preferences.ChannelEmail); decision.Decision != DecisionDelayQuietHours || decision.ScheduledFor == nil {
		t.Fatalf("expected quiet-hours delay, got %+v", decision)
	}
	urgent := policyNotification(notifications.TypeGenerationJobFailed, notifications.CategoryAIGeneration, notifications.PriorityUrgent)
	urgent.UserID = userID
	if decision := policy.Evaluate(urgent, preferences.ChannelPush); decision.Decision != DecisionSendInstant {
		t.Fatalf("expected urgent bypass, got %+v", decision)
	}
}

func TestDeliveryPolicyTripMuteDoesNotSuppressProtectedEvents(t *testing.T) {
	userID := uuid.New()
	tripID := uuid.New()
	mutes := []entity.NotificationTripMute{{ID: uuid.New(), UserID: userID, TripID: tripID}}
	policy := testPolicy(userID, nil, mutes, time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC))
	comment := policyNotification(notifications.TypeCommentCreated, notifications.CategoryComments, notifications.PriorityNormal)
	comment.UserID = userID
	comment.TripID = &tripID
	if decision := policy.Evaluate(comment, preferences.ChannelInApp); decision.Decision != DecisionMute {
		t.Fatalf("expected trip mute, got %+v", decision)
	}
	security := policyNotification(notifications.TypeShareSecurityChanged, notifications.CategorySecurity, notifications.PriorityUrgent)
	security.UserID = userID
	security.TripID = &tripID
	if decision := policy.Evaluate(security, preferences.ChannelInApp); decision.Decision != DecisionSendInstant {
		t.Fatalf("expected protected security delivery, got %+v", decision)
	}
}

func policyNotification(typ, category, priority string) entity.Notification {
	return entity.Notification{ID: uuid.New(), UserID: uuid.New(), Type: typ, Category: category, Priority: priority, Title: "Title", Message: "Message"}
}
func testPolicy(userID uuid.UUID, rows []entity.NotificationPreference, mutes []entity.NotificationTripMute, now time.Time) *BatchPolicy {
	return NewBatchPolicy(preferences.BuildEffectiveSet([]uuid.UUID{userID}, rows), map[uuid.UUID]entity.NotificationSettings{}, mutes, now)
}
