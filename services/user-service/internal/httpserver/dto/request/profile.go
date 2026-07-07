// Package request holds inbound HTTP payloads for user endpoints.
package request

import (
	"bytes"
	"encoding/json"
	"fmt"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/dto"
)

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
	TravelStyles        *[]string       `json:"travelStyles" validate:"omitempty,dive,max=100"`
	Pace                *string         `json:"pace" validate:"omitempty,oneof=relaxed balanced intensive"`
	MaxWalkingKmPerDay  NullableFloat64 `json:"maxWalkingKmPerDay" validate:"-"`
	FoodPreferences     *[]string       `json:"foodPreferences" validate:"omitempty,dive,max=100"`
	Avoid               *[]string       `json:"avoid" validate:"omitempty,dive,max=100"`
	PreferredTransport  *[]string       `json:"preferredTransport" validate:"omitempty,dive,max=100"`
	AccommodationStyle  *[]string       `json:"accommodationStyle" validate:"omitempty,dive,max=100"`
	DietaryRestrictions *[]string       `json:"dietaryRestrictions" validate:"omitempty,dive,max=100"`
}

// ToInput maps the transport request to the application-level input.
func (r PatchPreferences) ToInput() appdto.PatchPreferencesInput {
	var maxWalking *appdto.OptionalFloat64
	if r.MaxWalkingKmPerDay.Set {
		maxWalking = &appdto.OptionalFloat64{Value: r.MaxWalkingKmPerDay.Value}
	}

	return appdto.PatchPreferencesInput{
		TravelStyles:        r.TravelStyles,
		Pace:                r.Pace,
		MaxWalkingKmPerDay:  maxWalking,
		FoodPreferences:     r.FoodPreferences,
		Avoid:               r.Avoid,
		PreferredTransport:  r.PreferredTransport,
		AccommodationStyle:  r.AccommodationStyle,
		DietaryRestrictions: r.DietaryRestrictions,
	}
}

// NullableFloat64 distinguishes an omitted PATCH field from an explicit null.
type NullableFloat64 struct {
	Set   bool
	Value *float64
}

// UnmarshalJSON marks the field as present and decodes either a number or null.
func (n *NullableFloat64) UnmarshalJSON(data []byte) error {
	n.Set = true

	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		n.Value = nil
		return nil
	}

	var value float64
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("maxWalkingKmPerDay must be a number or null: %w", err)
	}
	n.Value = &value
	return nil
}
