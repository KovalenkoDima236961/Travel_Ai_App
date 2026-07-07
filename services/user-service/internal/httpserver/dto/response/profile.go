// Package response holds outbound HTTP payloads for user endpoints.
package response

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/entity"
)

// Profile is the JSON representation of a travel profile.
type Profile struct {
	UserID            uuid.UUID `json:"userId"`
	DisplayName       *string   `json:"displayName"`
	HomeCity          *string   `json:"homeCity"`
	HomeCountry       *string   `json:"homeCountry"`
	PreferredCurrency string    `json:"preferredCurrency"`
	PreferredLanguage string    `json:"preferredLanguage"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

// NewProfile maps a domain Profile to its API representation.
func NewProfile(p *entity.Profile) Profile {
	return Profile{
		UserID:            p.UserID,
		DisplayName:       p.DisplayName,
		HomeCity:          p.HomeCity,
		HomeCountry:       p.HomeCountry,
		PreferredCurrency: p.PreferredCurrency,
		PreferredLanguage: p.PreferredLanguage,
		CreatedAt:         p.CreatedAt,
		UpdatedAt:         p.UpdatedAt,
	}
}

// Preferences is the JSON representation of travel preferences.
type Preferences struct {
	UserID              uuid.UUID `json:"userId"`
	TravelStyles        []string  `json:"travelStyles"`
	Pace                string    `json:"pace"`
	MaxWalkingKmPerDay  *float64  `json:"maxWalkingKmPerDay"`
	FoodPreferences     []string  `json:"foodPreferences"`
	Avoid               []string  `json:"avoid"`
	PreferredTransport  []string  `json:"preferredTransport"`
	AccommodationStyle  []string  `json:"accommodationStyle"`
	DietaryRestrictions []string  `json:"dietaryRestrictions"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

// NewPreferences maps domain Preferences to their API representation.
func NewPreferences(p *entity.Preferences) Preferences {
	return Preferences{
		UserID:              p.UserID,
		TravelStyles:        nonNil(p.TravelStyles),
		Pace:                p.Pace,
		MaxWalkingKmPerDay:  p.MaxWalkingKmPerDay,
		FoodPreferences:     nonNil(p.FoodPreferences),
		Avoid:               nonNil(p.Avoid),
		PreferredTransport:  nonNil(p.PreferredTransport),
		AccommodationStyle:  nonNil(p.AccommodationStyle),
		DietaryRestrictions: nonNil(p.DietaryRestrictions),
		CreatedAt:           p.CreatedAt,
		UpdatedAt:           p.UpdatedAt,
	}
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
