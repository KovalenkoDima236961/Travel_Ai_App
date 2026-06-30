package budget

import (
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

func cost(amount float64, currency, category string) *aggregate.EstimatedCost {
	a := amount
	return &aggregate.EstimatedCost{Amount: &a, Currency: currency, Category: category}
}

func sampleItinerary() aggregate.Itinerary {
	return aggregate.Itinerary{
		Currency: "EUR",
		Days: []aggregate.ItineraryDay{
			{
				Day:   1,
				Title: "Day 1",
				Items: []aggregate.ItineraryItem{
					{Time: "09:00", Type: "ticket", Name: "Museum", EstimatedCost: cost(20, "EUR", "ticket")},
					{Time: "13:00", Type: "food", Name: "Lunch", EstimatedCost: cost(15, "EUR", "food")},
					{Time: "16:00", Type: "transport", Name: "Metro"},     // missing estimate (paid type)
					{Time: "18:00", Type: "walk", Name: "Evening stroll"}, // free type, not flagged
				},
			},
			{
				Day:   2,
				Title: "Day 2",
				Items: []aggregate.ItineraryItem{
					{Time: "10:00", Type: "food", Name: "Brunch", EstimatedCost: cost(25, "EUR", "food")},
					{Time: "15:00", Type: "viewpoint", Name: "Free viewpoint", EstimatedCost: cost(0, "EUR", "other")},
				},
			},
		},
	}
}

func ptr(v float64) *float64 { return &v }

func TestCalculateBudgetSummary_SumsAndGroups(t *testing.T) {
	trip := TripBudget{Amount: ptr(100), Currency: "EUR", Days: 2}
	summary := CalculateBudgetSummary(trip, sampleItinerary())

	if summary.Currency != "EUR" {
		t.Fatalf("expected currency EUR, got %s", summary.Currency)
	}
	if summary.EstimatedTotal != 60 {
		t.Fatalf("expected estimatedTotal 60, got %v", summary.EstimatedTotal)
	}
	if summary.EstimatedItemCount != 4 {
		t.Fatalf("expected 4 estimated items, got %d", summary.EstimatedItemCount)
	}
	if summary.MissingEstimateCount != 1 {
		t.Fatalf("expected 1 missing estimate, got %d", summary.MissingEstimateCount)
	}

	// byDay grouping and ordering.
	if len(summary.ByDay) != 2 {
		t.Fatalf("expected 2 day rollups, got %d", len(summary.ByDay))
	}
	if summary.ByDay[0].DayNumber != 1 || summary.ByDay[0].EstimatedTotal != 35 {
		t.Fatalf("unexpected day 1 rollup: %+v", summary.ByDay[0])
	}
	if summary.ByDay[0].MissingEstimateCount != 1 {
		t.Fatalf("expected day 1 missing 1, got %d", summary.ByDay[0].MissingEstimateCount)
	}
	if summary.ByDay[1].DayNumber != 2 || summary.ByDay[1].EstimatedTotal != 25 {
		t.Fatalf("unexpected day 2 rollup: %+v", summary.ByDay[1])
	}

	// byCategory grouping: food 40 (2 items), ticket 20 (1), other 0 (1).
	categories := map[string]CategorySummary{}
	for _, c := range summary.ByCategory {
		categories[c.Category] = c
	}
	if categories["food"].EstimatedTotal != 40 || categories["food"].ItemCount != 2 {
		t.Fatalf("unexpected food category: %+v", categories["food"])
	}
	if categories["ticket"].EstimatedTotal != 20 || categories["ticket"].ItemCount != 1 {
		t.Fatalf("unexpected ticket category: %+v", categories["ticket"])
	}
}

func TestCalculateBudgetSummary_IncludesAccommodationCost(t *testing.T) {
	trip := TripBudget{
		Amount:   ptr(500),
		Currency: "EUR",
		Days:     1,
		Accommodation: &aggregate.Accommodation{
			Name:          "Hotel Roma",
			Type:          aggregate.AccommodationTypeHotel,
			EstimatedCost: cost(120, "EUR", "other"),
		},
	}

	summary := CalculateBudgetSummary(trip, aggregate.Itinerary{})

	if summary.EstimatedTotal != 120 {
		t.Fatalf("expected estimatedTotal 120, got %v", summary.EstimatedTotal)
	}
	if summary.AccommodationTotal == nil || *summary.AccommodationTotal != 120 {
		t.Fatalf("expected accommodationTotal 120, got %v", summary.AccommodationTotal)
	}
	if summary.EstimatedItemCount != 1 {
		t.Fatalf("expected 1 estimated item, got %d", summary.EstimatedItemCount)
	}
	if len(summary.ByCategory) != 1 || summary.ByCategory[0].Category != "accommodation" {
		t.Fatalf("expected accommodation category, got %+v", summary.ByCategory)
	}
}

func TestCalculateBudgetSummary_RemainingAndOverBudget(t *testing.T) {
	withinBudget := CalculateBudgetSummary(TripBudget{Amount: ptr(100), Currency: "EUR", Days: 2}, sampleItinerary())
	if withinBudget.Remaining == nil || *withinBudget.Remaining != 40 {
		t.Fatalf("expected remaining 40, got %v", withinBudget.Remaining)
	}
	if withinBudget.OverBudgetBy == nil || *withinBudget.OverBudgetBy != 0 {
		t.Fatalf("expected overBudgetBy 0, got %v", withinBudget.OverBudgetBy)
	}

	overBudget := CalculateBudgetSummary(TripBudget{Amount: ptr(50), Currency: "EUR", Days: 2}, sampleItinerary())
	if overBudget.Remaining == nil || *overBudget.Remaining != -10 {
		t.Fatalf("expected remaining -10, got %v", overBudget.Remaining)
	}
	if overBudget.OverBudgetBy == nil || *overBudget.OverBudgetBy != 10 {
		t.Fatalf("expected overBudgetBy 10, got %v", overBudget.OverBudgetBy)
	}

	// Daily budget share = 50 / 2 = 25. Day 1 total 35 is 10 over.
	if overBudget.ByDay[0].DailyBudgetShare == nil || *overBudget.ByDay[0].DailyBudgetShare != 25 {
		t.Fatalf("expected daily share 25, got %v", overBudget.ByDay[0].DailyBudgetShare)
	}
	if overBudget.ByDay[0].OverDailyBudgetBy == nil || *overBudget.ByDay[0].OverDailyBudgetBy != 10 {
		t.Fatalf("expected day 1 over daily by 10, got %v", overBudget.ByDay[0].OverDailyBudgetBy)
	}
}

func TestCalculateBudgetSummary_NoTripBudget(t *testing.T) {
	summary := CalculateBudgetSummary(TripBudget{Currency: "EUR", Days: 2}, sampleItinerary())
	if summary.TripBudget != nil || summary.Remaining != nil || summary.OverBudgetBy != nil {
		t.Fatalf("expected nil budget fields, got %+v", summary)
	}
	if summary.EstimatedTotal != 60 {
		t.Fatalf("expected estimatedTotal 60 without budget, got %v", summary.EstimatedTotal)
	}
	if summary.ByDay[0].DailyBudgetShare != nil {
		t.Fatalf("expected no daily share without budget")
	}
}

func TestCalculateBudgetSummary_UnsupportedCurrency(t *testing.T) {
	itinerary := aggregate.Itinerary{
		Days: []aggregate.ItineraryDay{{
			Day:   1,
			Title: "Day 1",
			Items: []aggregate.ItineraryItem{
				{Time: "09:00", Type: "food", Name: "Lunch", EstimatedCost: cost(10, "EUR", "food")},
				{Time: "12:00", Type: "ticket", Name: "Show", EstimatedCost: cost(99, "USD", "ticket")},
			},
		}},
	}
	summary := CalculateBudgetSummary(TripBudget{Amount: ptr(100), Currency: "EUR", Days: 1}, itinerary)
	if summary.UnsupportedCurrencyCount != 1 {
		t.Fatalf("expected 1 unsupported currency, got %d", summary.UnsupportedCurrencyCount)
	}
	if summary.EstimatedTotal != 10 {
		t.Fatalf("expected USD item excluded, estimatedTotal 10, got %v", summary.EstimatedTotal)
	}
	if summary.EstimatedItemCount != 1 {
		t.Fatalf("expected 1 estimated item, got %d", summary.EstimatedItemCount)
	}
}

func TestCalculateBudgetSummary_IgnoresInvalidEstimateSafely(t *testing.T) {
	negative := -5.0
	itinerary := aggregate.Itinerary{
		Days: []aggregate.ItineraryDay{{
			Day:   1,
			Title: "Day 1",
			Items: []aggregate.ItineraryItem{
				{Time: "09:00", Type: "food", Name: "Lunch", EstimatedCost: cost(10, "EUR", "food")},
				// Negative amount: ignored, not summed; food type so flagged missing.
				{Time: "12:00", Type: "food", Name: "Snack", EstimatedCost: &aggregate.EstimatedCost{Amount: &negative, Currency: "EUR"}},
				// nil amount object: treated as missing.
				{Time: "14:00", Type: "ticket", Name: "Museum", EstimatedCost: &aggregate.EstimatedCost{Currency: "EUR"}},
			},
		}},
	}
	summary := CalculateBudgetSummary(TripBudget{Currency: "EUR", Days: 1}, itinerary)
	if summary.EstimatedTotal != 10 {
		t.Fatalf("expected estimatedTotal 10, got %v", summary.EstimatedTotal)
	}
	if summary.MissingEstimateCount != 2 {
		t.Fatalf("expected 2 missing estimates, got %d", summary.MissingEstimateCount)
	}
}

func TestCalculateBudgetSummary_CurrencyFallbacks(t *testing.T) {
	// No trip currency: fall back to the first item estimate currency.
	itinerary := aggregate.Itinerary{
		Days: []aggregate.ItineraryDay{{
			Day:   1,
			Title: "Day 1",
			Items: []aggregate.ItineraryItem{
				{Time: "09:00", Type: "food", Name: "Lunch", EstimatedCost: cost(10, "GBP", "food")},
			},
		}},
	}
	summary := CalculateBudgetSummary(TripBudget{Days: 1}, itinerary)
	if summary.Currency != "GBP" {
		t.Fatalf("expected currency GBP from item estimate, got %s", summary.Currency)
	}

	// No currency anywhere: default.
	empty := CalculateBudgetSummary(TripBudget{Days: 1}, aggregate.Itinerary{
		Days: []aggregate.ItineraryDay{{Day: 1, Title: "Day 1", Items: []aggregate.ItineraryItem{{Time: "09:00", Type: "walk", Name: "Walk"}}}},
	})
	if empty.Currency != DefaultCurrency {
		t.Fatalf("expected default currency, got %s", empty.Currency)
	}
}

func TestCalculateBudgetSummary_RoundsToTwoDecimals(t *testing.T) {
	itinerary := aggregate.Itinerary{
		Days: []aggregate.ItineraryDay{{
			Day:   1,
			Title: "Day 1",
			Items: []aggregate.ItineraryItem{
				{Time: "09:00", Type: "food", Name: "A", EstimatedCost: cost(10.10, "EUR", "food")},
				{Time: "10:00", Type: "food", Name: "B", EstimatedCost: cost(20.20, "EUR", "food")},
				{Time: "11:00", Type: "food", Name: "C", EstimatedCost: cost(0.05, "EUR", "food")},
			},
		}},
	}
	summary := CalculateBudgetSummary(TripBudget{Amount: ptr(100), Currency: "EUR", Days: 1}, itinerary)
	if summary.EstimatedTotal != 30.35 {
		t.Fatalf("expected estimatedTotal 30.35, got %v", summary.EstimatedTotal)
	}
	if summary.Remaining == nil || *summary.Remaining != 69.65 {
		t.Fatalf("expected remaining 69.65, got %v", summary.Remaining)
	}
}
