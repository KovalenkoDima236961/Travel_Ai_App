package entity

import (
	"time"

	"github.com/google/uuid"
)

// Profile is the authenticated user's travel profile.
type Profile struct {
	UserID            uuid.UUID
	DisplayName       *string
	HomeCity          *string
	HomeCountry       *string
	PreferredCurrency string
	PreferredLanguage string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
