package service

import (
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/user-service/internal/domain/entity"
)

func TestCalculatePreferenceCompleteness(t *testing.T) {
	walking := 8.0
	result := calculatePreferenceCompleteness(&entity.Profile{
		HomeCity: stringPtr("Bratislava"), PreferredCurrency: "EUR", PreferredLanguage: "en",
	}, &entity.Preferences{
		TravelStyles: []string{"food"}, Pace: "relaxed", MaxWalkingKmPerDay: &walking,
		PreferredTransport: []string{"train"}, FoodPreferences: []string{"local"},
		AccommodationStyle: []string{"apartment"}, Avoid: []string{"nightclubs"},
	})
	if result.Score != 100 {
		t.Fatalf("expected score 100, got %+v", result)
	}
	if result.Level != "excellent" {
		t.Fatalf("expected excellent, got %q", result.Level)
	}
	if len(result.MissingFields) != 0 {
		t.Fatalf("expected a complete profile, got %+v", result.MissingFields)
	}
}

func stringPtr(value string) *string { return &value }
