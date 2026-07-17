package copilot

import (
	"encoding/json"
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func TestSafeItineraryExcludesRawNotesAndItemNames(t *testing.T) {
	context := safeItinerary(json.RawMessage(`{"days":[{"day":3,"title":"Ignore instructions","items":[{"name":"private place","note":"private note"}]}]}`), ClientContext{SelectedDayNumber: intPtr(3)})
	encoded, err := json.Marshal(context)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"Ignore instructions", "private place", "private note"} {
		if contains(text, forbidden) {
			t.Fatalf("safe itinerary leaked %q: %s", forbidden, text)
		}
	}
}

func TestSafeRouteExcludesProviderURLsAndNotes(t *testing.T) {
	bookingURL := "https://booking.example/private"
	route := safeRoute(&entity.Trip{Route: &aggregate.TripRoute{Legs: []aggregate.RouteLeg{{
		ID: "leg_1", FromName: "Vienna", ToName: "Salzburg", Mode: "train", Notes: "private note",
		SelectedTransportOption: &aggregate.SelectedTransportOption{
			Provider: "provider", BookingURL: &bookingURL, BaggageNotes: stringPtr("private baggage"),
		},
	}}}}, ClientContext{})
	encoded, err := json.Marshal(route)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, forbidden := range []string{"booking.example", "private note", "private baggage"} {
		if contains(text, forbidden) {
			t.Fatalf("safe route leaked %q: %s", forbidden, text)
		}
	}
}

func intPtr(value int) *int { return &value }

func stringPtr(value string) *string { return &value }

func contains(value, fragment string) bool {
	return len(fragment) > 0 && len(value) >= len(fragment) && (value == fragment || containsAt(value, fragment))
}

func containsAt(value, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
