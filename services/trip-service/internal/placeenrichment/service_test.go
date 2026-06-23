package placeenrichment

import (
	"context"
	"errors"
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

type mockSearcher struct {
	results []aggregate.PlaceRef
	err     error
	calls   []searchCall
}

type searchCall struct {
	query       string
	destination string
}

func (s *mockSearcher) SearchPlaces(_ context.Context, query string, destination string) ([]aggregate.PlaceRef, error) {
	s.calls = append(s.calls, searchCall{query: query, destination: destination})
	if s.err != nil {
		return nil, s.err
	}
	return s.results, nil
}

func TestEnrichItinerary_AttachesBestMatchAboveThreshold(t *testing.T) {
	searcher := &mockSearcher{results: []aggregate.PlaceRef{
		testPlace("Roman Forum", "historic site", withCoordinates(), withRating()),
		testPlace("Colosseum", "landmark", withCoordinates(), withRating()),
	}}
	svc := New(searcher, Config{MinConfidence: 0.75, MaxItems: 20, FailOpen: true})
	input := testItinerary([]aggregate.ItineraryItem{{Time: "09:00", Type: "place", Name: "Colosseum"}})

	got, err := svc.EnrichItinerary(context.Background(), EnrichItineraryInput{Destination: "Rome", Itinerary: input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item := got.Itinerary.Days[0].Items[0]
	if item.Place == nil || item.Place.Name != "Colosseum" {
		t.Fatalf("expected Colosseum place to be attached, got %+v", item.Place)
	}
	if item.PlaceEnrichment == nil || item.PlaceEnrichment.Status != StatusMatched {
		t.Fatalf("expected matched metadata, got %+v", item.PlaceEnrichment)
	}
	if got.Stats.Attempted != 1 || got.Stats.Matched != 1 || got.Stats.NoMatch != 0 {
		t.Fatalf("unexpected stats: %+v", got.Stats)
	}
}

func TestEnrichItinerary_DoesNotAttachBelowThreshold(t *testing.T) {
	searcher := &mockSearcher{results: []aggregate.PlaceRef{
		testPlace("Colosseum", "landmark", withCoordinates(), withRating()),
	}}
	svc := New(searcher, Config{MinConfidence: 0.75, MaxItems: 20, FailOpen: true})
	input := testItinerary([]aggregate.ItineraryItem{{Time: "11:00", Type: "activity", Name: "Cooking class"}})

	got, err := svc.EnrichItinerary(context.Background(), EnrichItineraryInput{Destination: "Rome", Itinerary: input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	item := got.Itinerary.Days[0].Items[0]
	if item.Place != nil {
		t.Fatalf("expected no attached place, got %+v", item.Place)
	}
	if item.PlaceEnrichment == nil || item.PlaceEnrichment.Status != StatusNoMatch {
		t.Fatalf("expected no_match metadata, got %+v", item.PlaceEnrichment)
	}
	if got.Stats.NoMatch != 1 {
		t.Fatalf("expected one no-match stat, got %+v", got.Stats)
	}
}

func TestEnrichItinerary_RespectsMaxItemsAndSkipsTypes(t *testing.T) {
	searcher := &mockSearcher{results: []aggregate.PlaceRef{
		testPlace("Colosseum", "landmark", withCoordinates(), withRating()),
	}}
	svc := New(searcher, Config{MinConfidence: 0.75, MaxItems: 1, FailOpen: true})
	input := testItinerary([]aggregate.ItineraryItem{
		{Time: "09:00", Type: "place", Name: "Colosseum"},
		{Time: "10:00", Type: "transport", Name: "Metro"},
		{Time: "11:00", Type: "place", Name: "Roman Forum"},
	})

	got, err := svc.EnrichItinerary(context.Background(), EnrichItineraryInput{Destination: "Rome", Itinerary: input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(searcher.calls) != 1 {
		t.Fatalf("expected one search call, got %+v", searcher.calls)
	}
	if got.Stats.Attempted != 1 || got.Stats.Skipped != 2 {
		t.Fatalf("unexpected stats: %+v", got.Stats)
	}
}

func TestEnrichItinerary_SkipsExistingPlaceWhenOverwriteFalse(t *testing.T) {
	existing := testPlace("Existing", "landmark", withCoordinates())
	searcher := &mockSearcher{results: []aggregate.PlaceRef{testPlace("Colosseum", "landmark", withCoordinates(), withRating())}}
	svc := New(searcher, Config{MinConfidence: 0.75, MaxItems: 20, OverwriteExisting: false, FailOpen: true})
	input := testItinerary([]aggregate.ItineraryItem{{Time: "09:00", Type: "place", Name: "Colosseum", Place: &existing}})

	got, err := svc.EnrichItinerary(context.Background(), EnrichItineraryInput{Destination: "Rome", Itinerary: input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(searcher.calls) != 0 {
		t.Fatalf("expected no search calls, got %+v", searcher.calls)
	}
	if got.Itinerary.Days[0].Items[0].Place.Name != "Existing" {
		t.Fatalf("expected existing place to be preserved, got %+v", got.Itinerary.Days[0].Items[0].Place)
	}
}

func TestEnrichItinerary_OverwritesExistingPlaceWhenConfigured(t *testing.T) {
	existing := testPlace("Existing", "landmark", withCoordinates())
	searcher := &mockSearcher{results: []aggregate.PlaceRef{testPlace("Colosseum", "landmark", withCoordinates(), withRating())}}
	svc := New(searcher, Config{MinConfidence: 0.75, MaxItems: 20, OverwriteExisting: true, FailOpen: true})
	input := testItinerary([]aggregate.ItineraryItem{{Time: "09:00", Type: "place", Name: "Colosseum", Place: &existing}})

	got, err := svc.EnrichItinerary(context.Background(), EnrichItineraryInput{Destination: "Rome", Itinerary: input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Itinerary.Days[0].Items[0].Place.Name != "Colosseum" {
		t.Fatalf("expected existing place to be overwritten, got %+v", got.Itinerary.Days[0].Items[0].Place)
	}
}

func TestEnrichItinerary_DoesNotMutateOriginalItinerary(t *testing.T) {
	searcher := &mockSearcher{results: []aggregate.PlaceRef{testPlace("Colosseum", "landmark", withCoordinates(), withRating())}}
	svc := New(searcher, Config{MinConfidence: 0.75, MaxItems: 20, FailOpen: true})
	input := testItinerary([]aggregate.ItineraryItem{{Time: "09:00", Type: "place", Name: "Colosseum"}})

	got, err := svc.EnrichItinerary(context.Background(), EnrichItineraryInput{Destination: "Rome", Itinerary: input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if input.Days[0].Items[0].Place != nil || input.Days[0].Items[0].PlaceEnrichment != nil {
		t.Fatalf("input itinerary was mutated: %+v", input.Days[0].Items[0])
	}
	if got.Itinerary.Days[0].Items[0].Place == nil {
		t.Fatal("expected output itinerary to be enriched")
	}
}

func TestEnrichItinerary_SearchErrorFailOpenReturnsStats(t *testing.T) {
	searcher := &mockSearcher{err: errors.New("service down")}
	svc := New(searcher, Config{MinConfidence: 0.75, MaxItems: 20, FailOpen: true})
	input := testItinerary([]aggregate.ItineraryItem{{Time: "09:00", Type: "place", Name: "Colosseum"}})

	got, err := svc.EnrichItinerary(context.Background(), EnrichItineraryInput{Destination: "Rome", Itinerary: input})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Stats.Failed != 1 {
		t.Fatalf("expected one failed stat, got %+v", got.Stats)
	}
	if got.Itinerary.Days[0].Items[0].PlaceEnrichment == nil ||
		got.Itinerary.Days[0].Items[0].PlaceEnrichment.Status != StatusFailed {
		t.Fatalf("expected failed metadata, got %+v", got.Itinerary.Days[0].Items[0].PlaceEnrichment)
	}
}

func TestEnrichItinerary_SearchErrorFailClosedReturnsError(t *testing.T) {
	searcher := &mockSearcher{err: errors.New("service down")}
	svc := New(searcher, Config{MinConfidence: 0.75, MaxItems: 20, FailOpen: false})
	input := testItinerary([]aggregate.ItineraryItem{{Time: "09:00", Type: "place", Name: "Colosseum"}})

	_, err := svc.EnrichItinerary(context.Background(), EnrichItineraryInput{Destination: "Rome", Itinerary: input})
	if err == nil {
		t.Fatal("expected fail-closed search error")
	}
}

func testItinerary(items []aggregate.ItineraryItem) aggregate.Itinerary {
	return aggregate.Itinerary{
		Destination: "Rome",
		Days: []aggregate.ItineraryDay{{
			Day:   1,
			Title: "Day 1",
			Items: items,
		}},
	}
}
