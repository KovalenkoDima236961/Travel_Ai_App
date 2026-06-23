package placeenrichment

import (
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

func TestScorePlace_ExactNameMatchHighConfidence(t *testing.T) {
	place := testPlace("Colosseum", "landmark", withCoordinates(), withRating())

	score := ScorePlace(aggregate.ItineraryItem{Type: "place", Name: "Colosseum"}, "Rome", place)

	if score.Confidence < 0.75 {
		t.Fatalf("expected high confidence, got %+v", score)
	}
	if score.Reason != "exact_name_match" {
		t.Fatalf("expected exact_name_match reason, got %q", score.Reason)
	}
}

func TestScorePlace_ContainsMatchMediumConfidence(t *testing.T) {
	place := testPlace("Trastevere Local Trattoria", "restaurant", withCoordinates(), withRating())

	score := ScorePlace(aggregate.ItineraryItem{Type: "food", Name: "Trastevere"}, "Rome", place)

	if score.Confidence < 0.65 || score.Confidence > 1 {
		t.Fatalf("expected medium confidence contains score, got %+v", score)
	}
	if score.Reason != "name_contains_query" {
		t.Fatalf("expected name_contains_query reason, got %q", score.Reason)
	}
}

func TestScorePlace_TokenOverlapWorks(t *testing.T) {
	place := testPlace("Roman Forum Archaeological Area", "historic site", withCoordinates())

	score := ScorePlace(aggregate.ItineraryItem{Type: "landmark", Name: "Roman Forum"}, "Rome", place)

	if score.Confidence <= 0 {
		t.Fatalf("expected positive token overlap score, got %+v", score)
	}
}

func TestScorePlace_CoordinatesAddBonus(t *testing.T) {
	item := aggregate.ItineraryItem{Type: "place", Name: "Trevi Fountain"}
	withoutCoordinates := testPlace("Trevi Fountain", "landmark")
	withCoordinates := testPlace("Trevi Fountain", "landmark", withCoordinates())

	withoutScore := ScorePlace(item, "Rome", withoutCoordinates)
	withScore := ScorePlace(item, "Rome", withCoordinates)

	if withScore.Confidence <= withoutScore.Confidence {
		t.Fatalf("expected coordinates to increase score, without=%+v with=%+v", withoutScore, withScore)
	}
}

func TestScorePlace_LowMatchBelowThreshold(t *testing.T) {
	score := ScorePlace(
		aggregate.ItineraryItem{Type: "activity", Name: "Cooking class"},
		"Rome",
		testPlace("Colosseum", "landmark", withCoordinates(), withRating()),
	)

	if score.Confidence >= 0.75 {
		t.Fatalf("expected low confidence below threshold, got %+v", score)
	}
}

func TestIsCandidateType(t *testing.T) {
	for _, itemType := range []string{"place", "food", "activity", "museum", "cafe", "market"} {
		if !isCandidateType(itemType) {
			t.Fatalf("expected %q to be a candidate type", itemType)
		}
	}
	for _, itemType := range []string{"transport", "rest", "free_time", "note", "accommodation"} {
		if isCandidateType(itemType) {
			t.Fatalf("expected %q to be skipped", itemType)
		}
	}
}

type placeOption func(*aggregate.PlaceRef)

func testPlace(name, category string, opts ...placeOption) aggregate.PlaceRef {
	place := aggregate.PlaceRef{
		Provider:        "mock",
		ProviderPlaceID: "mock-" + normalizeText(name) + "-rome",
		Name:            name,
		Address:         "Rome, Italy",
		Category:        category,
	}
	for _, opt := range opts {
		opt(&place)
	}
	return place
}

func withCoordinates() placeOption {
	return func(place *aggregate.PlaceRef) {
		lat := 41.8902
		lng := 12.4922
		place.Latitude = &lat
		place.Longitude = &lng
	}
}

func withRating() placeOption {
	return func(place *aggregate.PlaceRef) {
		rating := 4.7
		place.Rating = &rating
	}
}
