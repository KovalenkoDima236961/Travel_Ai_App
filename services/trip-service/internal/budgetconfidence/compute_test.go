package budgetconfidence

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func TestComputeRewardsConfirmedMajorCosts(t *testing.T) {
	tripID := uuid.New()
	userID := uuid.New()
	budgetAmount := 1000.0
	trip := &entity.Trip{
		ID:             tripID,
		UserID:         &userID,
		Days:           3,
		BudgetAmount:   &budgetAmount,
		BudgetCurrency: "EUR",
		Status:         entity.StatusCompleted,
		Accommodation: &aggregate.Accommodation{
			Name: "Hotel",
			EstimatedCost: &aggregate.EstimatedCost{
				Amount:     floatPtr(300),
				Currency:   "EUR",
				Category:   budget.CategoryAccommodation,
				Confidence: budget.ConfidenceHigh,
				Source:     budget.SourceProvider,
			},
		},
		Route: &aggregate.TripRoute{
			Legs: []aggregate.RouteLeg{
				{
					ID:   "leg-1",
					Mode: aggregate.TransportModeTrain,
					SelectedTransportOption: &aggregate.SelectedTransportOption{
						Provider:   "rail-provider",
						Mode:       aggregate.TransportModeTrain,
						Confidence: budget.ConfidenceHigh,
						EstimatedPrice: &aggregate.TransportMoney{
							Amount:   100,
							Currency: "EUR",
						},
					},
				},
			},
		},
	}
	itinerary := aggregate.Itinerary{
		Currency: "EUR",
		Days: []aggregate.ItineraryDay{
			{Day: 1, Items: []aggregate.ItineraryItem{
				costedItem("09:00", "museum", "Museum", 25, budget.CategoryTicket, budget.SourceProvider, budget.ConfidenceHigh),
				costedItem("12:00", "food", "Lunch", 35, budget.CategoryFood, budget.SourceManual, budget.ConfidenceMedium),
			}},
			{Day: 2, Items: []aggregate.ItineraryItem{
				costedItem("09:00", "activity", "Tour", 45, budget.CategoryActivity, budget.SourceProvider, budget.ConfidenceHigh),
				costedItem("12:00", "food", "Dinner", 40, budget.CategoryFood, budget.SourceManual, budget.ConfidenceMedium),
			}},
			{Day: 3, Items: []aggregate.ItineraryItem{
				costedItem("12:00", "food", "Brunch", 30, budget.CategoryFood, budget.SourceManual, budget.ConfidenceMedium),
			}},
		},
	}

	got := Compute(context.Background(), Input{
		Trip:      trip,
		Itinerary: itinerary,
		Currency:  "EUR",
		Now:       time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC),
		Config:    DefaultConfig(),
	})

	if got.Score < 75 {
		t.Fatalf("expected high confidence score, got %d with issues %+v", got.Score, got.Issues)
	}
	if got.Level != LevelHigh && got.Level != LevelVeryHigh {
		t.Fatalf("expected high confidence level, got %s", got.Level)
	}
	if got.RiskLevel != RiskLow {
		t.Fatalf("expected low risk, got %s with issues %+v", got.RiskLevel, got.Issues)
	}
	if got.EstimatedTotal.Amount != 575 {
		t.Fatalf("expected estimated total 575, got %+v", got.EstimatedTotal)
	}
	if len(got.Issues) != 0 {
		t.Fatalf("expected no issues for complete confirmed costs, got %+v", got.Issues)
	}
}

func TestComputeFlagsMissingMajorCostsAndActualGap(t *testing.T) {
	tripID := uuid.New()
	userID := uuid.New()
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	budgetAmount := 500.0
	trip := &entity.Trip{
		ID:             tripID,
		UserID:         &userID,
		Days:           3,
		BudgetAmount:   &budgetAmount,
		BudgetCurrency: "EUR",
		StartDate:      &start,
		Status:         entity.StatusCompleted,
		Route: &aggregate.TripRoute{
			Legs: []aggregate.RouteLeg{
				{ID: "leg-1", Mode: aggregate.TransportModeTrain},
				{ID: "leg-2", Mode: aggregate.TransportModeBus},
			},
		},
	}
	itinerary := aggregate.Itinerary{
		Currency: "EUR",
		Days: []aggregate.ItineraryDay{
			{Day: 1, Items: []aggregate.ItineraryItem{
				{Time: "10:00", Type: "museum", Name: "Museum"},
			}},
		},
	}
	expenseID := uuid.New()

	got := Compute(context.Background(), Input{
		Trip:      trip,
		Itinerary: itinerary,
		Expenses: []entity.TripExpense{
			{
				ID:           expenseID,
				TripID:       tripID,
				Title:        "Meals so far",
				Amount:       420,
				Currency:     "EUR",
				Category:     entity.ExpenseCategoryFood,
				Status:       entity.ExpenseStatusActive,
				ExpenseDate:  start,
				PaidByUserID: userID,
			},
		},
		Currency: "EUR",
		Now:      time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC),
		Config:   DefaultConfig(),
	})

	for _, id := range []string{
		"missing_accommodation_cost",
		"missing_transport_prices",
		"missing_activity_prices",
		"planned_actual_gap:food",
		"actual_spend_high_before_trip_end",
	} {
		if !hasIssue(got.Issues, id) {
			t.Fatalf("expected issue %q, got %+v", id, got.Issues)
		}
	}
	if got.RiskLevel != RiskHigh && got.RiskLevel != RiskCritical {
		t.Fatalf("expected elevated risk, got %s with issues %+v", got.RiskLevel, got.Issues)
	}
	if got.Coverage.Accommodation == nil || *got.Coverage.Accommodation != 0 {
		t.Fatalf("expected accommodation coverage 0, got %+v", got.Coverage.Accommodation)
	}
	if got.Score >= 55 {
		t.Fatalf("expected low confidence score, got %d", got.Score)
	}
}

func costedItem(timeOfDay, itemType, name string, amount float64, category, source, confidence string) aggregate.ItineraryItem {
	return aggregate.ItineraryItem{
		Time: timeOfDay,
		Type: itemType,
		Name: name,
		EstimatedCost: &aggregate.EstimatedCost{
			Amount:     floatPtr(amount),
			Currency:   "EUR",
			Category:   category,
			Source:     source,
			Confidence: confidence,
		},
	}
}

func floatPtr(value float64) *float64 {
	return &value
}

func hasIssue(issues []Issue, id string) bool {
	for _, issue := range issues {
		if issue.ID == id {
			return true
		}
	}
	return false
}
