package aivalidation

import (
	"context"
	"strings"
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func TestValidatorFlagsDayCountAndSelectedTransportConflict(t *testing.T) {
	cfg := DefaultConfig()
	validator := NewValidator(cfg)
	trip := entity.Trip{
		Destination:    "Vienna",
		Days:           2,
		BudgetCurrency: "EUR",
		Route: &aggregate.TripRoute{
			Legs: []aggregate.RouteLeg{
				{
					ID:         "leg_1",
					FromStopID: "origin",
					ToStopID:   "stop_1",
					FromName:   "Bratislava",
					ToName:     "Vienna",
					Mode:       aggregate.TransportModeTrain,
					SelectedTransportOption: &aggregate.SelectedTransportOption{
						ID:            "opt_1",
						Mode:          aggregate.TransportModeTrain,
						Provider:      "mock",
						DepartureDate: "2026-09-10",
						DepartureTime: "08:00",
						ArrivalDate:   "2026-09-10",
						ArrivalTime:   "11:00",
					},
				},
			},
		},
	}
	itinerary := aggregate.Itinerary{
		Destination: "Vienna",
		Currency:    "EUR",
		Days: []aggregate.ItineraryDay{
			{
				Day:           1,
				Date:          "2026-09-10",
				Title:         "Arrival",
				PrimaryStopID: "stop_1",
				Items: []aggregate.ItineraryItem{
					{
						Time:    "09:00",
						EndTime: "10:00",
						Type:    "activity",
						Name:    "Old town walk",
					},
				},
			},
		},
	}

	result, err := validator.Validate(context.Background(), ValidationInput{
		GenerationType: GenerationTypeFullItinerary,
		Trip:           trip,
		Itinerary:      itinerary,
		Context: ValidationContext{
			ExpectedDayCount: 2,
			RepairAllowed:    true,
		},
	})
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if result.SaveAllowed {
		t.Fatalf("expected save to be blocked")
	}
	if !hasIssuePrefix(result.Issues, "itinerary_day_count_mismatch") {
		t.Fatalf("expected day count issue, got %#v", result.Issues)
	}
	if !hasIssuePrefix(result.Issues, "activity_before_transport_arrival") {
		t.Fatalf("expected selected transport arrival issue, got %#v", result.Issues)
	}
}

func TestValidatorAllowsOpeningHoursUnknownWarning(t *testing.T) {
	cfg := DefaultConfig()
	validator := NewValidator(cfg)
	trip := entity.Trip{Destination: "Vienna", Days: 1, BudgetCurrency: "EUR"}
	itinerary := aggregate.Itinerary{
		Destination: "Vienna",
		Currency:    "EUR",
		Days: []aggregate.ItineraryDay{
			{
				Day:   1,
				Date:  "2026-09-10",
				Title: "Museums",
				Items: []aggregate.ItineraryItem{
					{
						Time: "10:00",
						Type: "place",
						Name: "Museum visit",
						Place: &aggregate.PlaceRef{
							Provider:        "mock",
							ProviderPlaceID: "museum-1",
							Name:            "Museum visit",
							Address:         "Vienna",
						},
					},
				},
			},
		},
	}

	result, err := validator.Validate(context.Background(), ValidationInput{
		GenerationType: GenerationTypeFullItinerary,
		Trip:           trip,
		Itinerary:      itinerary,
		Context: ValidationContext{
			ExpectedDayCount: 1,
			RepairAllowed:    true,
		},
	})
	if err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}
	if !result.SaveAllowed {
		t.Fatalf("expected warning-only output to be saveable, got %#v", result.Issues)
	}
	if result.QualityStatus != StatusValidatedWithWarnings {
		t.Fatalf("expected warnings status, got %s", result.QualityStatus)
	}
	if !hasIssuePrefix(result.Issues, "opening_hours_unknown") {
		t.Fatalf("expected opening-hours warning, got %#v", result.Issues)
	}
}

func hasIssuePrefix(issues []ValidationIssue, prefix string) bool {
	for _, issue := range issues {
		if strings.HasPrefix(issue.ID, prefix) {
			return true
		}
	}
	return false
}
