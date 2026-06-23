// Package generator holds itinerary-generator adapters that satisfy the
// application.ItineraryGenerator port.
package generator

import (
	"context"
	"fmt"
	"time"
	"unicode"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

// MockItineraryGenerator produces a deterministic, interest-aware sample plan
// locally.
type MockItineraryGenerator struct {
	logger *zap.Logger
}

// NewMockItineraryGenerator constructs the mock generator.
func NewMockItineraryGenerator(logger *zap.Logger) *MockItineraryGenerator {
	return &MockItineraryGenerator{logger: logger}
}

// Generate builds a sample itinerary derived from the trip's destination,
// interests, pace and duration.
func (g *MockItineraryGenerator) Generate(_ context.Context, input application.GenerateItineraryInput) (*aggregate.Itinerary, error) {
	trip := input.Trip
	g.logger.Info("generating mock itinerary",
		zap.String("trip_id", trip.ID.String()),
		zap.String("destination", trip.Destination),
		zap.Int32("days", trip.Days),
		zap.Bool("user_context_loaded", input.UserProfile != nil || input.UserPreferences != nil),
	)

	interests := trip.Interests
	if len(interests) == 0 {
		interests = []string{"sightseeing"}
	}

	days := make([]aggregate.ItineraryDay, 0, trip.Days)
	for i := int32(0); i < trip.Days; i++ {
		focus := interests[int(i)%len(interests)]
		days = append(days, aggregate.ItineraryDay{
			Day:   int(i) + 1,
			Title: fmt.Sprintf("Day %d in %s — %s", i+1, trip.Destination, titleCase(focus)),
			Items: []aggregate.ItineraryItem{
				{
					Time: "09:00",
					Type: "activity",
					Name: fmt.Sprintf("Explore %s highlights", trip.Destination),
					Note: fmt.Sprintf("focused on %s", focus),
				},
				{
					Time: "13:00",
					Type: "meal",
					Name: "Lunch at a local spot",
				},
				{
					Time: "15:00",
					Type: "activity",
					Name: fmt.Sprintf("A %s-paced %s experience", trip.Pace, focus),
				},
				{
					Time: "19:30",
					Type: "meal",
					Name: "Dinner recommendation",
				},
			},
		})
	}

	summary := fmt.Sprintf("A %d-day %s trip to %s for %d traveler(s).",
		trip.Days, trip.Pace, trip.Destination, trip.Travelers)
	if input.UserProfile != nil || input.UserPreferences != nil {
		summary += " Personalized with saved traveler context."
	}

	return &aggregate.Itinerary{
		Destination: trip.Destination,
		Summary:     summary,
		Travelers:   trip.Travelers,
		Pace:        trip.Pace,
		Currency:    trip.BudgetCurrency,
		TotalBudget: trip.BudgetAmount,
		Days:        days,
		GeneratedAt: time.Now().UTC(),
		Source:      "mock-local-generator",
	}, nil
}

// titleCase upper-cases the first rune of s.
func titleCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
