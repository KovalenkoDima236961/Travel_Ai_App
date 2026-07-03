package preferences

import (
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
)

// EffectiveSet is a snapshot of several users' effective preferences (defaults
// merged with their stored overrides). It is built once per internal batch so
// the create path and the email fan-out can both consult preferences without
// per-notification database round-trips.
//
// It satisfies the channel-gate interfaces expected by the notifications and
// emailnotifications packages (AllowInApp / AllowEmail), wiring preferences into
// those flows without creating an import cycle.
type EffectiveSet struct {
	// byUser holds a full merged matrix per requested user:
	// user -> channel -> category -> enabled.
	byUser map[uuid.UUID]map[string]map[string]bool
}

// BuildEffectiveSet constructs an EffectiveSet for the given users by starting
// from the default matrix for each user and applying their stored overrides.
// Only rows for the requested users are applied; rows for other users are
// ignored. The returned set is never nil.
func BuildEffectiveSet(userIDs []uuid.UUID, rows []entity.NotificationPreference) *EffectiveSet {
	byUser := make(map[uuid.UUID]map[string]map[string]bool, len(userIDs))
	for _, id := range userIDs {
		if id == uuid.Nil {
			continue
		}
		byUser[id] = defaultMatrixCopy()
	}

	for i := range rows {
		row := rows[i]
		matrix, ok := byUser[row.UserID]
		if !ok {
			continue
		}
		if !IsKnownChannel(row.Channel) || !IsKnownCategory(row.Category) {
			continue
		}
		matrix[row.Channel][row.Category] = row.Enabled
	}

	return &EffectiveSet{byUser: byUser}
}

// AllowInApp reports whether an in-app notification of the given type should be
// created for the user. An unknown type is allowed in-app so a future,
// not-yet-categorised type is never silently dropped from a user's inbox.
func (e *EffectiveSet) AllowInApp(userID uuid.UUID, notificationType string) bool {
	return e.allow(ChannelInApp, userID, notificationType)
}

// AllowEmail reports whether an email for the given type should be sent to the
// user. An unknown type is never emailed (email is opt-in per category and there
// is no template for an uncategorised type).
func (e *EffectiveSet) AllowEmail(userID uuid.UUID, notificationType string) bool {
	return e.allow(ChannelEmail, userID, notificationType)
}

// AllowPush reports whether a browser push notification for the given type
// should be sent to the user. Unknown types are not pushed so new event types do
// not unexpectedly appear on a user's lock screen before they have an explicit
// payload policy.
func (e *EffectiveSet) AllowPush(userID uuid.UUID, notificationType string) bool {
	category, ok := CategoryForNotificationType(notificationType)
	if !ok {
		return false
	}
	if e != nil {
		if matrix, ok := e.byUser[userID]; ok {
			if byCategory, ok := matrix[ChannelPush]; ok {
				if enabled, ok := byCategory[category]; ok {
					return enabled
				}
			}
		}
	}
	return defaultEnabled(ChannelPush, category)
}

// Matrix returns the full merged channel/category matrix for a single user. It
// falls back to defaults for a user not present in the set so callers always get
// a complete matrix.
func (e *EffectiveSet) Matrix(userID uuid.UUID) map[string]map[string]bool {
	if e != nil {
		if matrix, ok := e.byUser[userID]; ok {
			return matrix
		}
	}
	return defaultMatrixCopy()
}

func (e *EffectiveSet) allow(channel string, userID uuid.UUID, notificationType string) bool {
	category, ok := CategoryForNotificationType(notificationType)
	if !ok {
		// Unknown type: allowed in-app, not emailed. (Channels independent.)
		return channel == ChannelInApp
	}
	if e != nil {
		if matrix, ok := e.byUser[userID]; ok {
			if byCategory, ok := matrix[channel]; ok {
				if enabled, ok := byCategory[category]; ok {
					return enabled
				}
			}
		}
	}
	return defaultEnabled(channel, category)
}

// defaultMatrixCopy returns a fresh, independent copy of the default matrix so a
// per-user override never mutates the shared default.
func defaultMatrixCopy() map[string]map[string]bool {
	out := make(map[string]map[string]bool, len(defaultMatrix))
	for channel, byCategory := range defaultMatrix {
		copied := make(map[string]bool, len(byCategory))
		for category, enabled := range byCategory {
			copied[category] = enabled
		}
		out[channel] = copied
	}
	return out
}
