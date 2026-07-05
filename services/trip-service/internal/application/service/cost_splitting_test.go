package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func TestCalculateCostSplittingSummaryDefaultAllEqual(t *testing.T) {
	trip := testCostSplitTrip("EUR")
	travelers := testTripTravelers(3)
	amount := 90.0
	itinerary := aggregate.Itinerary{
		Currency: "EUR",
		Days: []aggregate.ItineraryDay{{
			Day:   1,
			Title: "Arrival",
			Items: []aggregate.ItineraryItem{{
				Time: "10:00",
				Type: "ticket",
				Name: "Museum",
				EstimatedCost: &aggregate.EstimatedCost{
					Amount:   &amount,
					Currency: "EUR",
					Category: budget.CategoryTicket,
				},
			}},
		}},
	}

	result, err := (&Service{}).calculateCostSplittingSummary(
		context.Background(),
		trip,
		itinerary,
		travelers,
		"EUR",
		time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("summary error: %v", err)
	}

	if result.Summary.DefaultSplitCount != 1 {
		t.Fatalf("default split count = %d, want 1", result.Summary.DefaultSplitCount)
	}
	if result.Summary.AllocatedTotal != 90 {
		t.Fatalf("allocated total = %.2f, want 90", result.Summary.AllocatedTotal)
	}
	for _, traveler := range result.Travelers {
		if traveler.AllocatedTotal != 30 {
			t.Fatalf("%s allocated %.2f, want 30", traveler.Name, traveler.AllocatedTotal)
		}
	}
}

func TestCalculateCostSplittingSummarySelectedAndCustomSplits(t *testing.T) {
	trip := testCostSplitTrip("EUR")
	travelers := testTripTravelers(2)
	selectedAmount := 60.0
	accommodationAmount := 200.0
	itinerary := aggregate.Itinerary{
		Currency: "EUR",
		Days: []aggregate.ItineraryDay{{
			Day:   2,
			Title: "Tour",
			Items: []aggregate.ItineraryItem{{
				Time: "09:00",
				Type: "activity",
				Name: "Private tour",
				EstimatedCost: &aggregate.EstimatedCost{
					Amount:   &selectedAmount,
					Currency: "EUR",
					Category: budget.CategoryActivity,
					Split: &aggregate.CostSplitRule{
						Type:        splitTypeSelectedEqual,
						TravelerIDs: []string{travelers[0].ID.String()},
					},
				},
			}},
		}},
	}
	trip.Accommodation = &aggregate.Accommodation{
		Name: "Hotel",
		Type: aggregate.AccommodationTypeHotel,
		EstimatedCost: &aggregate.EstimatedCost{
			Amount:   &accommodationAmount,
			Currency: "EUR",
			Category: budget.CategoryAccommodation,
			Split: &aggregate.CostSplitRule{
				Type: splitTypeCustomPercentages,
				Percentages: map[string]float64{
					travelers[0].ID.String(): 25,
					travelers[1].ID.String(): 75,
				},
			},
		},
	}

	result, err := (&Service{}).calculateCostSplittingSummary(
		context.Background(),
		trip,
		itinerary,
		travelers,
		"EUR",
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("summary error: %v", err)
	}

	if got := result.Travelers[0].AllocatedTotal; got != 110 {
		t.Fatalf("traveler 1 allocated %.2f, want 110", got)
	}
	if got := result.Travelers[1].AllocatedTotal; got != 150 {
		t.Fatalf("traveler 2 allocated %.2f, want 150", got)
	}
	if result.Summary.UnassignedTotal != 0 {
		t.Fatalf("unassigned total = %.2f, want 0", result.Summary.UnassignedTotal)
	}
}

func TestCalculateCostSplittingSummaryInvalidRemovedTravelerReference(t *testing.T) {
	trip := testCostSplitTrip("EUR")
	travelers := testTripTravelers(1)
	removedTravelerID := uuid.New()
	amount := 40.0
	itinerary := aggregate.Itinerary{
		Currency: "EUR",
		Days: []aggregate.ItineraryDay{{
			Day:   1,
			Title: "Transfer",
			Items: []aggregate.ItineraryItem{{
				Time: "11:00",
				Type: "transport",
				Name: "Transfer",
				EstimatedCost: &aggregate.EstimatedCost{
					Amount:   &amount,
					Currency: "EUR",
					Category: budget.CategoryTransport,
					Split: &aggregate.CostSplitRule{
						Type:        splitTypeSelectedEqual,
						TravelerIDs: []string{removedTravelerID.String()},
					},
				},
			}},
		}},
	}

	result, err := (&Service{}).calculateCostSplittingSummary(
		context.Background(),
		trip,
		itinerary,
		travelers,
		"EUR",
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("summary error: %v", err)
	}

	if result.Summary.InvalidSplitCount != 1 {
		t.Fatalf("invalid split count = %d, want 1", result.Summary.InvalidSplitCount)
	}
	if result.Summary.UnassignedTotal != 40 {
		t.Fatalf("unassigned total = %.2f, want 40", result.Summary.UnassignedTotal)
	}
	if len(result.UnassignedCosts) != 1 || result.UnassignedCosts[0].Reason != "invalid_split_rule" {
		t.Fatalf("unexpected unassigned costs: %#v", result.UnassignedCosts)
	}
}

func TestCalculateCostSplittingSummaryConversion(t *testing.T) {
	trip := testCostSplitTrip("EUR")
	travelers := testTripTravelers(2)
	amount := 100.0
	itinerary := aggregate.Itinerary{
		Currency: "EUR",
		Days: []aggregate.ItineraryDay{{
			Day:   1,
			Title: "Tickets",
			Items: []aggregate.ItineraryItem{{
				Time: "12:00",
				Type: "ticket",
				Name: "Show",
				EstimatedCost: &aggregate.EstimatedCost{
					Amount:   &amount,
					Currency: "USD",
					Category: budget.CategoryTicket,
				},
			}},
		}},
	}
	svc := &Service{
		budgetConversionProvider: fakeCostSplitConverter{},
		budgetConversionEnabled:  true,
		budgetConversionFailOpen: true,
	}

	result, err := svc.calculateCostSplittingSummary(
		context.Background(),
		trip,
		itinerary,
		travelers,
		"EUR",
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("summary error: %v", err)
	}

	if result.Summary.EstimatedTotal != 200 || result.Summary.ConvertedItemCount != 1 {
		t.Fatalf("summary totals = %.2f converted=%d, want 200 converted=1", result.Summary.EstimatedTotal, result.Summary.ConvertedItemCount)
	}
	if result.Travelers[0].AllocatedTotal != 100 || result.Travelers[1].AllocatedTotal != 100 {
		t.Fatalf("unexpected allocations: %#v", result.Travelers)
	}
	if result.ExchangeRateInfo == nil || result.ExchangeRateInfo.Provider != "test" {
		t.Fatalf("missing exchange rate info: %#v", result.ExchangeRateInfo)
	}
}

func testCostSplitTrip(currency string) *entity.Trip {
	userID := uuid.New()
	return &entity.Trip{
		ID:             uuid.New(),
		UserID:         &userID,
		Destination:    "Rome",
		Days:           2,
		BudgetCurrency: currency,
		Status:         entity.StatusCompleted,
	}
}

func testTripTravelers(count int) []entity.TripTraveler {
	tripID := uuid.New()
	out := make([]entity.TripTraveler, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, entity.TripTraveler{
			ID:     uuid.New(),
			TripID: tripID,
			Name:   string(rune('A' + i)),
			Role:   entity.TripTravelerRoleTraveler,
			Status: entity.TripTravelerStatusActive,
		})
	}
	return out
}

type fakeCostSplitConverter struct{}

func (fakeCostSplitConverter) Convert(_ context.Context, amount float64, from string, to string) (*budget.CurrencyConversionResult, error) {
	return &budget.CurrencyConversionResult{
		Provider:        "test",
		From:            from,
		To:              to,
		Amount:          amount,
		ConvertedAmount: amount * 2,
		Rate:            2,
		AsOf:            time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC),
	}, nil
}
