// Package dto holds application-level input types.
package dto

// UpdateProfileInput is the application-level representation of a profile PUT.
type UpdateProfileInput struct {
	DisplayName       string
	HomeCity          string
	HomeCountry       string
	PreferredCurrency string
	PreferredLanguage string
}

// PatchPreferencesInput is the application-level representation of a
// preferences PATCH. Nil fields were omitted by the caller.
type PatchPreferencesInput struct {
	TravelStyles        *[]string
	Pace                *string
	MaxWalkingKmPerDay  *float64
	FoodPreferences     *[]string
	Avoid               *[]string
	PreferredTransport  *[]string
	AccommodationStyle  *[]string
	DietaryRestrictions *[]string
}
