package entity

import (
	"time"

	"github.com/google/uuid"
)

// NotificationPreference is one stored per-user override for a single
// (channel, category) pair. A user has at most one row per pair; the absence of
// a row means "use the default". Preferences gate the creation of future
// notifications only — they never affect existing notifications, the activity
// feed, or core collaboration data.
type NotificationPreference struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Channel      string
	Category     string
	Enabled      bool
	DeliveryMode string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NotificationSettings stores one user's digest schedule and quiet-hours policy.
type NotificationSettings struct {
	UserID                   uuid.UUID
	QuietHoursEnabled        bool
	QuietHoursStart          string
	QuietHoursEnd            string
	QuietHoursTimezone       string
	UrgentBypassesQuietHours bool
	DailyDigestTime          string
	WeeklyDigestDay          int
	WeeklyDigestTime         string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

// NotificationTripMute is a user-owned trip/category suppression rule. A nil
// Category means all mutably-safe categories for the trip.
type NotificationTripMute struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	TripID     uuid.UUID
	Category   *string
	MutedUntil *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
