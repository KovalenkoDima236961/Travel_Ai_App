package deliverypolicy

import (
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/preferences"
)

type BatchPolicy struct {
	preferences *preferences.EffectiveSet
	settings    map[uuid.UUID]entity.NotificationSettings
	mutes       []entity.NotificationTripMute
	now         time.Time
}

func NewBatchPolicy(set *preferences.EffectiveSet, settings map[uuid.UUID]entity.NotificationSettings, mutes []entity.NotificationTripMute, now time.Time) *BatchPolicy {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return &BatchPolicy{preferences: set, settings: settings, mutes: mutes, now: now.UTC()}
}

func (p *BatchPolicy) AllowInApp(userID uuid.UUID, notificationType string) bool {
	if p == nil || p.preferences == nil {
		return true
	}
	return p.preferences.AllowInApp(userID, notificationType)
}

func (p *BatchPolicy) AllowInAppNotification(notification entity.Notification) bool {
	decision := p.Evaluate(notification, preferences.ChannelInApp)
	return decision.Decision != DecisionMute && decision.Decision != DecisionDropDuplicate
}

func (p *BatchPolicy) ImmediateInApp(notification entity.Notification) bool {
	return p.Evaluate(notification, preferences.ChannelInApp).Decision == DecisionSendInstant
}

func (p *BatchPolicy) InAppDelivery(notification entity.Notification) (string, string) {
	decision := p.Evaluate(notification, preferences.ChannelInApp)
	switch decision.Decision {
	case DecisionSendInstant:
		return decision.Mode, "sent"
	case DecisionCreateInAppOnly:
		if decision.ScheduledFor != nil && decision.Mode != preferences.ModeInstant {
			return decision.Mode, "pending"
		}
		return decision.Mode, "created_silently"
	default:
		return decision.Mode, "muted"
	}
}

func (p *BatchPolicy) Evaluate(notification entity.Notification, channel string) Decision {
	category := notification.Category
	if category == "" {
		category = notifications.DefaultCategory(notification.Type)
	}
	priority := notification.Priority
	if !notifications.IsPriority(priority) {
		priority = notifications.DefaultPriority(notification.Type)
	}

	protected := notifications.IsProtectedFromTripMute(notification.Type, category)
	if !protected && p.isTripMuted(notification.UserID, notification.TripID, category) {
		return Decision{Decision: DecisionMute, Mode: preferences.ModeMuted, Reason: "trip_mute"}
	}

	mode := preferences.ModeInstant
	if p != nil && p.preferences != nil {
		preferenceCategory := category
		// Preserve the two fine-grained categories that existed before the v2
		// category expansion. Users who had reminder preferences must not have
		// those choices silently replaced by the broader checklist/reminders
		// defaults after migration.
		if legacyCategory, ok := preferences.CategoryForNotificationType(notification.Type); ok &&
			(legacyCategory == preferences.CategoryChecklistReminders || legacyCategory == preferences.CategoryPreTripReminders) {
			preferenceCategory = legacyCategory
		}
		mode = p.preferences.DeliveryMode(notification.UserID, channel, preferenceCategory)
	}
	if mode == preferences.ModeMuted {
		if category != notifications.CategorySecurity {
			return Decision{Decision: DecisionMute, Mode: mode, Reason: "user_preference"}
		}
		mode = preferences.ModeInstant
	}

	settings := p.userSettings(notification.UserID)
	quiet, quietEnd := quietHours(settings, p.now)
	if quiet {
		if priority == notifications.PriorityUrgent && settings.UrgentBypassesQuietHours {
			return Decision{Decision: DecisionSendInstant, Mode: preferences.ModeInstant, Reason: "urgent_quiet_hours_bypass"}
		}
		if channel == preferences.ChannelInApp {
			return Decision{Decision: DecisionCreateInAppOnly, Mode: mode, Reason: "quiet_hours"}
		}
		return Decision{Decision: DecisionDelayQuietHours, Mode: digestMode(mode), Reason: "quiet_hours", ScheduledFor: &quietEnd}
	}

	if priority == notifications.PriorityUrgent && mode != preferences.ModeMuted {
		return Decision{Decision: DecisionSendInstant, Mode: preferences.ModeInstant, Reason: "urgent_priority"}
	}
	if mode == preferences.ModeInstant {
		return Decision{Decision: DecisionSendInstant, Mode: mode, Reason: "user_preference"}
	}
	if channel == preferences.ChannelInApp {
		scheduled := NextDigestAt(mode, settings, p.now)
		return Decision{Decision: DecisionCreateInAppOnly, Mode: mode, Reason: "user_preference", ScheduledFor: &scheduled}
	}
	scheduled := NextDigestAt(mode, settings, p.now)
	return Decision{Decision: DecisionDigest, Mode: mode, Reason: "user_preference", ScheduledFor: &scheduled}
}

func (p *BatchPolicy) isTripMuted(userID uuid.UUID, tripID *uuid.UUID, category string) bool {
	if p == nil || tripID == nil {
		return false
	}
	for _, mute := range p.mutes {
		if mute.UserID != userID || mute.TripID != *tripID {
			continue
		}
		if mute.MutedUntil != nil && !mute.MutedUntil.After(p.now) {
			continue
		}
		if mute.Category == nil || *mute.Category == category {
			return true
		}
	}
	return false
}

func (p *BatchPolicy) userSettings(userID uuid.UUID) entity.NotificationSettings {
	if p != nil {
		if settings, ok := p.settings[userID]; ok {
			return settings
		}
	}
	return entity.NotificationSettings{
		UserID: userID, QuietHoursStart: "22:00", QuietHoursEnd: "08:00",
		QuietHoursTimezone: "UTC", UrgentBypassesQuietHours: true,
		DailyDigestTime: "08:00", WeeklyDigestDay: 1, WeeklyDigestTime: "08:00",
	}
}

func digestMode(mode string) string {
	if mode == preferences.ModeHourlyDigest || mode == preferences.ModeDailyDigest || mode == preferences.ModeWeeklyDigest {
		return mode
	}
	return preferences.ModeHourlyDigest
}

func quietHours(settings entity.NotificationSettings, now time.Time) (bool, time.Time) {
	if !settings.QuietHoursEnabled {
		return false, time.Time{}
	}
	loc, err := time.LoadLocation(settings.QuietHoursTimezone)
	if err != nil {
		loc = time.UTC
	}
	localNow := now.In(loc)
	startHour, startMinute, okStart := parseClock(settings.QuietHoursStart)
	endHour, endMinute, okEnd := parseClock(settings.QuietHoursEnd)
	if !okStart || !okEnd {
		return false, time.Time{}
	}
	start := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), startHour, startMinute, 0, 0, loc)
	end := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), endHour, endMinute, 0, 0, loc)
	if start.Equal(end) {
		return true, end.Add(24 * time.Hour).UTC()
	}
	if end.Before(start) {
		if localNow.Before(end) {
			start = start.Add(-24 * time.Hour)
		} else {
			end = end.Add(24 * time.Hour)
		}
	}
	active := !localNow.Before(start) && localNow.Before(end)
	return active, end.UTC()
}

func NextDigestAt(mode string, settings entity.NotificationSettings, now time.Time) time.Time {
	loc, err := time.LoadLocation(settings.QuietHoursTimezone)
	if err != nil {
		loc = time.UTC
	}
	localNow := now.In(loc)
	switch mode {
	case preferences.ModeWeeklyDigest:
		hour, minute, ok := parseClock(settings.WeeklyDigestTime)
		if !ok {
			hour = 8
			minute = 0
		}
		targetDay := time.Weekday(settings.WeeklyDigestDay)
		days := (int(targetDay) - int(localNow.Weekday()) + 7) % 7
		target := time.Date(localNow.Year(), localNow.Month(), localNow.Day()+days, hour, minute, 0, 0, loc)
		if !target.After(localNow) {
			target = target.AddDate(0, 0, 7)
		}
		return target.UTC()
	case preferences.ModeDailyDigest:
		hour, minute, ok := parseClock(settings.DailyDigestTime)
		if !ok {
			hour = 8
			minute = 0
		}
		target := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), hour, minute, 0, 0, loc)
		if !target.After(localNow) {
			target = target.AddDate(0, 0, 1)
		}
		return target.UTC()
	default:
		target := localNow.Truncate(time.Hour).Add(time.Hour)
		return target.UTC()
	}
}

func parseClock(value string) (int, int, bool) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	hour, errH := strconv.Atoi(parts[0])
	minute, errM := strconv.Atoi(parts[1])
	if errH != nil || errM != nil || hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, 0, false
	}
	return hour, minute, true
}
