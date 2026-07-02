package priceenrichment

import (
	"context"
	"errors"
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/priceclient"
)

func TestClassifierCandidateTypes(t *testing.T) {
	cases := []struct {
		name string
		item aggregate.ItineraryItem
		want bool
	}{
		{"museum", aggregate.ItineraryItem{Name: "Museum", Type: "museum"}, true},
		{"attraction", aggregate.ItineraryItem{Name: "Castle", Type: "attraction"}, true},
		{"walk", aggregate.ItineraryItem{Name: "Walk", Type: "walk"}, false},
		{"food", aggregate.ItineraryItem{Name: "Lunch", Type: "food"}, false},
		{"transport", aggregate.ItineraryItem{Name: "Metro", Type: "transport"}, false},
		{"strong name no place", aggregate.ItineraryItem{Name: "City museum", Type: "activity"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsCandidateItem(tc.item); got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestEnrichItineraryAttachesProviderCostWhenMissing(t *testing.T) {
	estimator := &fakeEstimator{result: matchedPrice(18)}
	svc := New(estimator, Config{Enabled: true, FailOpen: true, MaxItems: 30})
	input := testPriceItinerary([]aggregate.ItineraryItem{{Time: "10:00", Type: "museum", Name: "City Museum"}})

	got, err := svc.EnrichItinerary(context.Background(), EnrichItineraryInput{
		Destination:    "Rome",
		BudgetCurrency: "EUR",
		Itinerary:      input,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item := got.Itinerary.Days[0].Items[0]
	if item.EstimatedCost == nil || item.EstimatedCost.Source != budget.SourceProvider {
		t.Fatalf("expected provider cost, got %+v", item.EstimatedCost)
	}
	if item.PriceEnrichment == nil || item.PriceEnrichment.Status != StatusMatched {
		t.Fatalf("expected matched metadata, got %+v", item.PriceEnrichment)
	}
	if got.Stats.Candidates != 1 || got.Stats.Matched != 1 {
		t.Fatalf("unexpected stats: %+v", got.Stats)
	}
}

func TestEnrichItineraryDoesNotOverwriteManualCostByDefault(t *testing.T) {
	manual := cost(12, budget.SourceManual)
	estimator := &fakeEstimator{result: matchedPrice(18)}
	svc := New(estimator, Config{Enabled: true, FailOpen: true, MaxItems: 30})
	input := testPriceItinerary([]aggregate.ItineraryItem{{Time: "10:00", Type: "museum", Name: "City Museum", EstimatedCost: manual}})

	got, err := svc.EnrichItinerary(context.Background(), EnrichItineraryInput{Destination: "Rome", Itinerary: input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	item := got.Itinerary.Days[0].Items[0]
	if item.EstimatedCost == nil || item.EstimatedCost.Source != budget.SourceManual || *item.EstimatedCost.Amount != 12 {
		t.Fatalf("expected manual cost preserved, got %+v", item.EstimatedCost)
	}
	if estimator.calls != 0 {
		t.Fatalf("expected no provider call, got %d", estimator.calls)
	}
	if got.Stats.NotOverwrittenExistingCost != 1 {
		t.Fatalf("expected preserved-cost stat, got %+v", got.Stats)
	}
}

func TestEnrichItineraryDoesNotOverwriteAICostByDefault(t *testing.T) {
	aiCost := cost(10, budget.SourceAI)
	estimator := &fakeEstimator{result: matchedPrice(18)}
	svc := New(estimator, Config{Enabled: true, FailOpen: true, MaxItems: 30})
	input := testPriceItinerary([]aggregate.ItineraryItem{{Time: "10:00", Type: "museum", Name: "City Museum", EstimatedCost: aiCost}})

	got, err := svc.EnrichItinerary(context.Background(), EnrichItineraryInput{Destination: "Rome", Itinerary: input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	item := got.Itinerary.Days[0].Items[0]
	if item.EstimatedCost == nil || item.EstimatedCost.Source != budget.SourceAI || *item.EstimatedCost.Amount != 10 {
		t.Fatalf("expected AI cost preserved, got %+v", item.EstimatedCost)
	}
	if estimator.calls != 0 {
		t.Fatalf("expected no provider call, got %d", estimator.calls)
	}
}

func TestEnrichItineraryOverwritesProviderCost(t *testing.T) {
	existing := cost(10, budget.SourceProvider)
	estimator := &fakeEstimator{result: matchedPrice(18)}
	svc := New(estimator, Config{Enabled: true, FailOpen: true, MaxItems: 30})
	input := testPriceItinerary([]aggregate.ItineraryItem{{Time: "10:00", Type: "museum", Name: "City Museum", EstimatedCost: existing}})

	got, err := svc.EnrichItinerary(context.Background(), EnrichItineraryInput{Destination: "Rome", Itinerary: input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	item := got.Itinerary.Days[0].Items[0]
	if item.EstimatedCost == nil || *item.EstimatedCost.Amount != 18 {
		t.Fatalf("expected provider cost overwritten, got %+v", item.EstimatedCost)
	}
	if got.Stats.Overwritten != 1 {
		t.Fatalf("expected overwritten stat, got %+v", got.Stats)
	}
}

func TestEnrichItineraryRespectsMinConfidence(t *testing.T) {
	estimator := &fakeEstimator{result: &priceclient.PriceEstimateResult{
		EstimatedCost:   cost(18, budget.SourceProvider),
		Provider:        "mock",
		Matched:         true,
		MatchConfidence: 0.3,
		PriceType:       stringPtr("ticket"),
	}}
	svc := New(estimator, Config{Enabled: true, FailOpen: true, MinMatchConfidence: 0.55, MaxItems: 30})
	input := testPriceItinerary([]aggregate.ItineraryItem{{Time: "10:00", Type: "museum", Name: "City Museum"}})

	got, err := svc.EnrichItinerary(context.Background(), EnrichItineraryInput{Destination: "Rome", Itinerary: input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	item := got.Itinerary.Days[0].Items[0]
	if item.EstimatedCost != nil || item.PriceEnrichment == nil || item.PriceEnrichment.Status != StatusNoMatch {
		t.Fatalf("expected no_match without cost, got item=%+v", item)
	}
}

func TestEnrichItineraryRespectsMaxItems(t *testing.T) {
	estimator := &fakeEstimator{result: matchedPrice(18)}
	svc := New(estimator, Config{Enabled: true, FailOpen: true, MaxItems: 1})
	input := testPriceItinerary([]aggregate.ItineraryItem{
		{Time: "10:00", Type: "museum", Name: "City Museum"},
		{Time: "12:00", Type: "castle", Name: "City Castle"},
	})

	got, err := svc.EnrichItinerary(context.Background(), EnrichItineraryInput{Destination: "Rome", Itinerary: input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if estimator.calls != 1 {
		t.Fatalf("expected one estimator call, got %d", estimator.calls)
	}
	if got.Stats.Skipped != 1 {
		t.Fatalf("expected one skipped stat, got %+v", got.Stats)
	}
}

func TestEnrichItineraryFailOpenKeepsItemOnProviderError(t *testing.T) {
	estimator := &fakeEstimator{err: errors.New("down")}
	svc := New(estimator, Config{Enabled: true, FailOpen: true, MaxItems: 30})
	input := testPriceItinerary([]aggregate.ItineraryItem{{Time: "10:00", Type: "museum", Name: "City Museum"}})

	got, err := svc.EnrichItinerary(context.Background(), EnrichItineraryInput{Destination: "Rome", Itinerary: input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	item := got.Itinerary.Days[0].Items[0]
	if item.EstimatedCost != nil || item.PriceEnrichment == nil || item.PriceEnrichment.Status != StatusFailed {
		t.Fatalf("expected failed metadata and no cost, got %+v", item)
	}
}

func TestEnrichItineraryFailClosedReturnsError(t *testing.T) {
	estimator := &fakeEstimator{err: errors.New("down")}
	svc := New(estimator, Config{Enabled: true, FailOpen: false, MaxItems: 30})
	input := testPriceItinerary([]aggregate.ItineraryItem{{Time: "10:00", Type: "museum", Name: "City Museum"}})

	_, err := svc.EnrichItinerary(context.Background(), EnrichItineraryInput{Destination: "Rome", Itinerary: input})
	if err == nil {
		t.Fatal("expected fail-closed error")
	}
}

type fakeEstimator struct {
	result *priceclient.PriceEstimateResult
	err    error
	calls  int
}

func (f *fakeEstimator) EstimatePrice(_ context.Context, input priceclient.PriceEstimateInput) (*priceclient.PriceEstimateResult, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	if input.Currency != "EUR" {
		return nil, errors.New("unexpected currency")
	}
	return f.result, nil
}

func testPriceItinerary(items []aggregate.ItineraryItem) aggregate.Itinerary {
	return aggregate.Itinerary{
		Destination: "Rome",
		Currency:    "EUR",
		Days: []aggregate.ItineraryDay{{
			Day:   1,
			Title: "Day 1",
			Items: items,
		}},
	}
}

func matchedPrice(amount float64) *priceclient.PriceEstimateResult {
	return &priceclient.PriceEstimateResult{
		EstimatedCost:   cost(amount, budget.SourceProvider),
		Provider:        "mock",
		Matched:         true,
		MatchConfidence: 0.82,
		PriceType:       stringPtr("ticket"),
		Metadata:        map[string]any{"reason": "test"},
	}
}

func cost(amount float64, source string) *aggregate.EstimatedCost {
	value := amount
	return &aggregate.EstimatedCost{
		Amount:     &value,
		Currency:   "EUR",
		Category:   budget.CategoryTicket,
		Confidence: budget.ConfidenceMedium,
		Source:     source,
	}
}

func stringPtr(value string) *string {
	return &value
}
