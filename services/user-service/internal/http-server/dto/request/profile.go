// Package request holds inbound HTTP payloads for user endpoints.
package request

import appdto "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/dto"

// UpdateProfile is the JSON body accepted by PUT /users/me/profile.
type UpdateProfile struct {
	DisplayName       string `json:"displayName" validate:"omitempty,max=100"`
	HomeCity          string `json:"homeCity" validate:"omitempty,max=100"`
	HomeCountry       string `json:"homeCountry" validate:"omitempty,max=100"`
	PreferredCurrency string `json:"preferredCurrency" validate:"required,len=3,uppercase"`
	PreferredLanguage string `json:"preferredLanguage" validate:"required,min=2,max=10"`
}

// ToInput maps the transport request to the application-level input.
func (r UpdateProfile) ToInput() appdto.UpdateProfileInput {
	return appdto.UpdateProfileInput{
		DisplayName:       r.DisplayName,
		HomeCity:          r.HomeCity,
		HomeCountry:       r.HomeCountry,
		PreferredCurrency: r.PreferredCurrency,
		PreferredLanguage: r.PreferredLanguage,
	}
}

// PatchPreferences is the JSON body accepted by PATCH /users/me/preferences.
type PatchPreferences struct {
	TravelStyles        *[]string `json:"travelStyles" validate:"omitempty,dive,max=100"`
	Pace                *string   `json:"pace" validate:"omitempty,oneof=relaxed balanced intensive"`
	MaxWalkingKmPerDay  *float64  `json:"maxWalkingKmPerDay" validate:"omitempty,gte=0,lte=50"`
	FoodPreferences     *[]string `json:"foodPreferences" validate:"omitempty,dive,max=100"`
	Avoid               *[]string `json:"avoid" validate:"omitempty,dive,max=100"`
	PreferredTransport  *[]string `json:"preferredTransport" validate:"omitempty,dive,max=100"`
	AccommodationStyle  *[]string `json:"accommodationStyle" validate:"omitempty,dive,max=100"`
	DietaryRestrictions *[]string `json:"dietaryRestrictions" validate:"omitempty,dive,max=100"`
}

// ToInput maps the transport request to the application-level input.
func (r PatchPreferences) ToInput() appdto.PatchPreferencesInput {
	return appdto.PatchPreferencesInput{
		TravelStyles:        r.TravelStyles,
		Pace:                r.Pace,
		MaxWalkingKmPerDay:  r.MaxWalkingKmPerDay,
		FoodPreferences:     r.FoodPreferences,
		Avoid:               r.Avoid,
		PreferredTransport:  r.PreferredTransport,
		AccommodationStyle:  r.AccommodationStyle,
		DietaryRestrictions: r.DietaryRestrictions,
	}
}
