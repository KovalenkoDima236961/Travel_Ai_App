package usercontext

import "github.com/google/uuid"

// UserProfile mirrors the User Service /users/me/profile response fields that
// are useful for itinerary personalization.
type UserProfile struct {
	UserID            uuid.UUID `json:"userId"`
	DisplayName       *string   `json:"displayName,omitempty"`
	HomeCity          *string   `json:"homeCity,omitempty"`
	HomeCountry       *string   `json:"homeCountry,omitempty"`
	PreferredCurrency string    `json:"preferredCurrency,omitempty"`
	PreferredLanguage string    `json:"preferredLanguage,omitempty"`
}

// UserPreferences mirrors the User Service /users/me/preferences response
// fields that are useful for itinerary personalization.
type UserPreferences struct {
	UserID              uuid.UUID `json:"userId"`
	TravelStyles        []string  `json:"travelStyles"`
	Pace                string    `json:"pace,omitempty"`
	MaxWalkingKmPerDay  *float64  `json:"maxWalkingKmPerDay,omitempty"`
	FoodPreferences     []string  `json:"foodPreferences"`
	Avoid               []string  `json:"avoid"`
	PreferredTransport  []string  `json:"preferredTransport"`
	AccommodationStyle  []string  `json:"accommodationStyle"`
	DietaryRestrictions []string  `json:"dietaryRestrictions"`
}

// UserContext groups the trusted profile and preferences fetched by Trip
// Service. Each field is optional so generation can continue with partial data.
type UserContext struct {
	Profile     *UserProfile
	Preferences *UserPreferences
}
