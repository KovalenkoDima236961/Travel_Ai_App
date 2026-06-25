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
	ID        uuid.UUID
	UserID    uuid.UUID
	Channel   string
	Category  string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}
