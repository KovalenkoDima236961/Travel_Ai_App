package entity

import (
	"encoding/json"
	"testing"
	"time"
)

func TestDeriveLifecycle(t *testing.T) {
	today := time.Date(2026, time.July, 18, 12, 0, 0, 0, time.UTC)
	itinerary := json.RawMessage(`{"days":[{"day":1,"items":[]}]}`)

	tests := []struct {
		name string
		trip Trip
		want TripLifecycle
	}{
		{name: "archive overrides every other state", trip: Trip{ArchivedAt: ptrTime(today), StartDate: ptrTime(today), Days: 2, Itinerary: itinerary}, want: TripLifecycleArchived},
		{name: "date range is active inclusive", trip: Trip{StartDate: ptrTime(today.AddDate(0, 0, -1)), Days: 2, Itinerary: itinerary}, want: TripLifecycleActive},
		{name: "ended trip is completed", trip: Trip{StartDate: ptrTime(today.AddDate(0, 0, -3)), Days: 2, Itinerary: itinerary}, want: TripLifecycleCompleted},
		{name: "upcoming trip becomes ready with both score snapshots", trip: Trip{StartDate: ptrTime(today.AddDate(0, 0, 2)), Days: 2, Itinerary: itinerary, CreationMetadata: map[string]any{"tripHealthScore": 80.0, "verificationScore": 75.0}}, want: TripLifecycleReady},
		{name: "upcoming itinerary without readiness is planning", trip: Trip{StartDate: ptrTime(today.AddDate(0, 0, 2)), Days: 2, Itinerary: itinerary}, want: TripLifecyclePlanning},
		{name: "missing dates and itinerary is draft", trip: Trip{}, want: TripLifecycleDraft},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DeriveLifecycle(&test.trip, LifecycleOptions{Now: today, ReadyHealthScoreThreshold: 80, ReadyVerificationThreshold: 75}); got != test.want {
				t.Fatalf("DeriveLifecycle() = %q, want %q", got, test.want)
			}
		})
	}
}

func ptrTime(value time.Time) *time.Time { return &value }
