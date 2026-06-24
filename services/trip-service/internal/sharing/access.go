package sharing

import (
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func IsShareActive(share *entity.TripShare, now time.Time) bool {
	if share == nil {
		return false
	}
	return share.Enabled && share.DisabledAt == nil && !IsShareExpired(share, now)
}

func IsShareExpired(share *entity.TripShare, now time.Time) bool {
	if share == nil || share.ExpiresAt == nil {
		return false
	}
	return !share.ExpiresAt.After(now)
}

func RequiresPassword(share *entity.TripShare) bool {
	return share != nil && share.PasswordRequired
}
