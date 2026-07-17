package search

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestScoreResultBoostsCurrentTrip(t *testing.T) {
	tripID := uuid.New()
	otherTripID := uuid.New()
	now := time.Now()

	current := Result{
		ID:        "route_leg:current",
		Type:      ResultTypeRouteLeg,
		Title:     "Rome to Florence",
		TripID:    &tripID,
		UpdatedAt: now,
	}
	other := Result{
		ID:        "route_leg:other",
		Type:      ResultTypeRouteLeg,
		Title:     "Rome to Florence",
		TripID:    &otherTripID,
		UpdatedAt: now,
	}

	if scoreResult("rome", []string{"rome"}, current, &tripID, now) <= scoreResult("rome", []string{"rome"}, other, &tripID, now) {
		t.Fatalf("expected current trip result to outrank otherwise equal result")
	}
}

func TestBuildResponseAppliesLimitAndPerCategoryLimit(t *testing.T) {
	results := []Result{
		{ID: "trip:1", Type: ResultTypeTrip, Title: "Rome", Category: categoryTrips, Score: 1},
		{ID: "trip:2", Type: ResultTypeTrip, Title: "Rome again", Category: categoryTrips, Score: 0.9},
		{ID: "expense:1", Type: ResultTypeExpense, Title: "Dinner", Category: categoryMoney, Score: 0.8},
	}

	response := buildResponse("rome", results, 3, 1)

	if len(response.Items) != 2 {
		t.Fatalf("expected 2 items after per-category limiting, got %d", len(response.Items))
	}
	if !response.HasMore {
		t.Fatalf("expected hasMore when a category result is skipped")
	}
	if len(response.Groups) != 2 {
		t.Fatalf("expected grouped response, got %d groups", len(response.Groups))
	}
	if response.Groups[0].Title != categoryTrips || response.Groups[1].Title != categoryMoney {
		t.Fatalf("unexpected group ordering: %#v", response.Groups)
	}
}
