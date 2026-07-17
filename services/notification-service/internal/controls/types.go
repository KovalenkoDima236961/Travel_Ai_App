package controls

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/preferences"
)

type TripMuteInput struct {
	UserID     uuid.UUID
	TripID     uuid.UUID
	Category   *string
	MutedUntil *time.Time
}

type SettingsInput struct {
	UserID                   uuid.UUID
	QuietHoursEnabled        bool
	QuietHoursStart          string
	QuietHoursEnd            string
	QuietHoursTimezone       string
	UrgentBypassesQuietHours bool
	DailyDigestTime          string
	WeeklyDigestDay          int
	WeeklyDigestTime         string
}

func DefaultSettings(userID uuid.UUID) SettingsInput {
	defaults := preferences.DefaultSettings()
	return SettingsInput{
		UserID:                   userID,
		QuietHoursStart:          defaults.QuietHoursStart,
		QuietHoursEnd:            defaults.QuietHoursEnd,
		QuietHoursTimezone:       defaults.QuietHoursTimezone,
		UrgentBypassesQuietHours: defaults.UrgentBypassesQuietHours,
		DailyDigestTime:          defaults.DailyDigestTime,
		WeeklyDigestDay:          defaults.WeeklyDigestDay,
		WeeklyDigestTime:         defaults.WeeklyDigestTime,
	}
}
