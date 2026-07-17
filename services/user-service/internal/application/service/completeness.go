package service

import (
	"strings"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/entity"
)

// calculatePreferenceCompleteness uses stable, documented weights. Defaults
// created by the service (EUR/en/balanced) are not considered answers.
func calculatePreferenceCompleteness(profile *entity.Profile, preferences *entity.Preferences) appdto.PreferenceCompleteness {
	result := appdto.PreferenceCompleteness{MissingFields: []appdto.PreferenceMissingField{}, RecommendedActions: []appdto.PreferenceRecommendedAction{}}
	add := func(done bool, score int, field, label, reason string) {
		if done {
			result.Score += score
			return
		}
		result.MissingFields = append(result.MissingFields, appdto.PreferenceMissingField{Field: field, Label: label, Reason: reason})
	}
	add(profile != nil && (nonEmpty(profile.HomeCity) || nonEmpty(profile.HomeCountry)), 10, "homeLocation", "Home city or country", "Helps us suggest practical starting points.")
	add(profile != nil && strings.TrimSpace(profile.PreferredCurrency) != "", 10, "preferredCurrency", "Preferred currency", "Helps us compare costs in a useful currency.")
	add(profile != nil && strings.TrimSpace(profile.PreferredLanguage) != "", 10, "preferredLanguage", "Preferred language", "Helps us present suggestions in your language.")
	add(preferences != nil && len(preferences.TravelStyles) > 0, 15, "travelStyles", "Travel styles", "Helps us find trips that match your interests.")
	add(preferences != nil && strings.TrimSpace(preferences.Pace) != "", 10, "pace", "Trip pace", "Helps us avoid plans that feel too packed or too slow.")
	add(preferences != nil && preferences.MaxWalkingKmPerDay != nil, 10, "maxWalkingKmPerDay", "Walking tolerance", "Helps us keep daily movement comfortable.")
	add(preferences != nil && len(preferences.PreferredTransport) > 0, 15, "preferredTransport", "Preferred transport", "Helps us suggest better routes.")
	add(preferences != nil && (len(preferences.FoodPreferences) > 0 || len(preferences.DietaryRestrictions) > 0), 10, "foodAndDietaryPreferences", "Food and dietary preferences", "Helps us make food suggestions that suit you.")
	add(preferences != nil && len(preferences.AccommodationStyle) > 0, 5, "accommodationStyle", "Accommodation style", "Helps us recommend the right places to stay.")
	add(preferences != nil && len(preferences.Avoid) > 0, 5, "avoid", "Avoid list", "Helps us avoid activities and places that do not suit you.")

	switch {
	case result.Score >= 90:
		result.Level = "excellent"
	case result.Score >= 70:
		result.Level = "good"
	case result.Score >= 40:
		result.Level = "partial"
	default:
		result.Level = "poor"
	}
	if len(result.MissingFields) > 0 {
		result.RecommendedActions = append(result.RecommendedActions, appdto.PreferenceRecommendedAction{Label: "Review travel preferences", Href: "/settings?section=preferences"})
	}
	return result
}

func nonEmpty(value *string) bool { return value != nil && strings.TrimSpace(*value) != "" }
