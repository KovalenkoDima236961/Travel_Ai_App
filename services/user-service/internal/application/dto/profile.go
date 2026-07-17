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
	MaxWalkingKmPerDay  *OptionalFloat64
	FoodPreferences     *[]string
	Avoid               *[]string
	PreferredTransport  *[]string
	AccommodationStyle  *[]string
	DietaryRestrictions *[]string
}

// OptionalFloat64 represents a nullable numeric field that was explicitly
// provided by a PATCH request. Value nil means the caller wants to clear it.
type OptionalFloat64 struct {
	Value *float64
}

type PreferenceMissingField struct {
	Field  string `json:"field"`
	Label  string `json:"label"`
	Reason string `json:"reason"`
}

type PreferenceRecommendedAction struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

type PreferenceCompleteness struct {
	Score              int                           `json:"score"`
	Level              string                        `json:"level"`
	MissingFields      []PreferenceMissingField      `json:"missingFields"`
	RecommendedActions []PreferenceRecommendedAction `json:"recommendedActions"`
}
