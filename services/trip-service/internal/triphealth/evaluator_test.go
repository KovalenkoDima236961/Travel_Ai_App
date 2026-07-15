package triphealth

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func TestEvaluateReturnsReadyForConsistentSingleDayTrip(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	tripID := uuid.New()
	budgetAmount := 500.0
	start := now.AddDate(0, 2, 0)
	trip := &entity.Trip{
		ID:                tripID,
		TripType:          entity.TripTypeSingleDestination,
		Destination:       "Lisbon",
		StartDate:         &start,
		Days:              1,
		BudgetAmount:      &budgetAmount,
		BudgetCurrency:    "EUR",
		ItineraryRevision: 2,
		UpdatedAt:         now,
	}

	resp := Evaluate(Snapshot{
		Trip: trip,
		Itinerary: aggregate.Itinerary{Days: []aggregate.ItineraryDay{
			{
				Day:   1,
				Title: "Arrival",
				Items: []aggregate.ItineraryItem{
					{Name: "Museum", Time: "10:00", EndTime: "12:00"},
				},
			},
		}},
		Now: now,
	}, Options{})

	if resp.TripID != tripID {
		t.Fatalf("trip ID = %s, want %s", resp.TripID, tripID)
	}
	if resp.Score != 100 {
		t.Fatalf("score = %d, want 100", resp.Score)
	}
	if resp.Level != LevelReady {
		t.Fatalf("level = %s, want %s", resp.Level, LevelReady)
	}
	if len(resp.Issues) != 0 {
		t.Fatalf("issues = %#v, want none", resp.Issues)
	}
}

func TestEvaluateDetectsRouteTransportBudgetAndChecklistIssues(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	tripID := uuid.New()
	start := now.AddDate(0, 0, 7)
	budgetAmount := 1000.0
	overBudgetBy := 250.0
	previousRevision := 1
	overdue := now.AddDate(0, 0, -1)
	checklistItemID := uuid.New()
	trip := &entity.Trip{
		ID:                tripID,
		TripType:          entity.TripTypeMultiDestination,
		Destination:       "Portugal",
		StartDate:         &start,
		Days:              2,
		BudgetAmount:      &budgetAmount,
		BudgetCurrency:    "EUR",
		ItineraryRevision: 3,
		UpdatedAt:         now,
		Route: &aggregate.TripRoute{
			Stops: []aggregate.RouteStop{
				{ID: "lisbon", Destination: "Lisbon", ArrivalDate: "2026-07-22", DepartureDate: "2026-07-22"},
				{ID: "porto", Destination: "Porto", ArrivalDate: "2026-07-23", DepartureDate: "2026-07-23"},
			},
			Legs: []aggregate.RouteLeg{
				{ID: "leg1", FromStopID: "lisbon", ToStopID: "porto", Mode: aggregate.TransportModeTrain},
			},
		},
	}

	resp := Evaluate(Snapshot{
		Trip: trip,
		Itinerary: aggregate.Itinerary{Days: []aggregate.ItineraryDay{
			{
				Day:           1,
				PrimaryStopID: "lisbon",
				Items:         []aggregate.ItineraryItem{{Name: "Dinner", Time: "19:00"}},
			},
			{
				Day:           2,
				PrimaryStopID: "porto",
				Items:         []aggregate.ItineraryItem{{Name: "Market", Time: "10:00"}},
			},
		}},
		Budget: &budget.Summary{
			Currency:             "EUR",
			EstimatedTotal:       1250,
			OverBudgetBy:         &overBudgetBy,
			MissingEstimateCount: 2,
			EstimatedItemCount:   2,
		},
		Checklist: &entity.TripChecklist{
			ID:                             uuid.New(),
			TripID:                         tripID,
			Status:                         entity.ChecklistStatusActive,
			GeneratedFromItineraryRevision: &previousRevision,
			UpdatedAt:                      now.Add(-time.Hour),
			Items: []entity.TripChecklistItem{
				{
					ID:       checklistItemID,
					TripID:   tripID,
					Title:    "Book train tickets",
					Priority: entity.ChecklistPriorityHigh,
					DueDate:  &overdue,
				},
			},
		},
		Now: now,
	}, Options{})

	assertIssue(t, resp, "transport_missing_option:leg1", CategoryTransport, SeverityHigh)
	assertIssue(t, resp, "estimated_budget_exceeded", CategoryBudget, SeverityHigh)
	assertIssue(t, resp, "missing_cost_estimates", CategoryBudget, SeverityHigh)
	assertIssue(t, resp, "checklist_stale", CategoryChecklist, SeverityWarning)
	assertIssue(t, resp, "high_priority_checklist_incomplete:"+checklistItemID.String(), CategoryChecklist, SeverityWarning)
	assertIssue(t, resp, "checklist_item_overdue:"+checklistItemID.String(), CategoryChecklist, SeverityHigh)

	if resp.Score >= 100 {
		t.Fatalf("score = %d, want penalized score", resp.Score)
	}
	if resp.Level == LevelReady {
		t.Fatalf("level = %s, want not ready/almost-ready state", resp.Level)
	}
	if len(resp.TopFixes) == 0 {
		t.Fatalf("top fixes are empty")
	}
}

func TestEvaluateHandlesSubsystemFailuresWithoutFailingResponse(t *testing.T) {
	now := time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	trip := &entity.Trip{
		ID:             uuid.New(),
		TripType:       entity.TripTypeSingleDestination,
		Destination:    "Paris",
		Days:           1,
		BudgetCurrency: "EUR",
		UpdatedAt:      now,
	}

	resp := Evaluate(Snapshot{
		Trip:              trip,
		Itinerary:         aggregate.Itinerary{Days: []aggregate.ItineraryDay{{Day: 1, Items: []aggregate.ItineraryItem{{Name: "Walk", Time: "09:00"}}}}},
		BudgetLoadFailed:  true,
		SubsystemFailures: []string{"receipts"},
		Now:               now,
	}, Options{IncludeDebug: true})

	assertIssue(t, resp, "health_subsystem_unavailable:budget", CategoryDataQuality, SeverityWarning)
	assertIssue(t, resp, "health_subsystem_unavailable:receipts", CategoryDataQuality, SeverityWarning)
	if resp.Debug == nil {
		t.Fatalf("debug is nil, want subsystem failure details")
	}
	if resp.GeneratedAt != now {
		t.Fatalf("generated at = %s, want %s", resp.GeneratedAt, now)
	}
}

func assertIssue(t *testing.T, resp Response, id string, category Category, severity Severity) {
	t.Helper()
	for _, issue := range resp.Issues {
		if issue.ID != id {
			continue
		}
		if issue.Category != category {
			t.Fatalf("%s category = %s, want %s", id, issue.Category, category)
		}
		if issue.Severity != severity {
			t.Fatalf("%s severity = %s, want %s", id, issue.Severity, severity)
		}
		return
	}
	t.Fatalf("issue %q not found in %#v", id, resp.Issues)
}
