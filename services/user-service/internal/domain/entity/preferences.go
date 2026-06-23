package entity

import (
	"time"

	"github.com/google/uuid"
)

// Preferences stores travel planning preferences for one authenticated user.
type Preferences struct {
	UserID              uuid.UUID
	TravelStyles        []string
	Pace                string
	MaxWalkingKmPerDay  *float64
	FoodPreferences     []string
	Avoid               []string
	PreferredTransport  []string
	AccommodationStyle  []string
	DietaryRestrictions []string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
