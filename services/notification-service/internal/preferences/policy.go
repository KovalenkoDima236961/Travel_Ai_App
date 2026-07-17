package preferences

import (
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/notification-service/internal/domain/entity"
)

type EffectiveSet struct {
	byUser map[uuid.UUID]map[string]map[string]string
}

func BuildEffectiveSet(userIDs []uuid.UUID, rows []entity.NotificationPreference) *EffectiveSet {
	byUser := make(map[uuid.UUID]map[string]map[string]string, len(userIDs))
	for _, id := range userIDs {
		if id != uuid.Nil {
			byUser[id] = defaultModeMatrixCopy()
		}
	}
	for _, row := range rows {
		matrix, ok := byUser[row.UserID]
		if !ok || !IsKnownChannel(row.Channel) || !IsKnownCategory(row.Category) {
			continue
		}
		mode := row.DeliveryMode
		if !IsKnownDeliveryMode(mode) {
			if row.Enabled {
				mode = ModeInstant
			} else {
				mode = ModeMuted
			}
		}
		matrix[row.Channel][row.Category] = mode
	}
	return &EffectiveSet{byUser: byUser}
}

func (e *EffectiveSet) AllowInApp(userID uuid.UUID, notificationType string) bool {
	return e.allow(ChannelInApp, userID, notificationType)
}
func (e *EffectiveSet) AllowEmail(userID uuid.UUID, notificationType string) bool {
	return e.allow(ChannelEmail, userID, notificationType)
}
func (e *EffectiveSet) AllowPush(userID uuid.UUID, notificationType string) bool {
	return e.allow(ChannelPush, userID, notificationType)
}

func (e *EffectiveSet) DeliveryMode(userID uuid.UUID, channel, category string) string {
	if e != nil {
		if matrix, ok := e.byUser[userID]; ok {
			if byCategory, ok := matrix[channel]; ok {
				if mode, ok := byCategory[category]; ok {
					return mode
				}
			}
		}
	}
	return defaultDeliveryMode(channel, category)
}

func (e *EffectiveSet) Matrix(userID uuid.UUID) map[string]map[string]bool {
	modes := e.ModeMatrix(userID)
	out := make(map[string]map[string]bool, len(modes))
	for channel, categories := range modes {
		out[channel] = make(map[string]bool, len(categories))
		for category, mode := range categories {
			out[channel][category] = mode != ModeMuted
		}
	}
	return out
}

func (e *EffectiveSet) ModeMatrix(userID uuid.UUID) map[string]map[string]string {
	if e != nil {
		if matrix, ok := e.byUser[userID]; ok {
			return matrix
		}
	}
	return defaultModeMatrixCopy()
}

func (e *EffectiveSet) allow(channel string, userID uuid.UUID, notificationType string) bool {
	category, ok := CategoryForNotificationType(notificationType)
	if !ok {
		return channel == ChannelInApp
	}
	return e.DeliveryMode(userID, channel, category) != ModeMuted
}
