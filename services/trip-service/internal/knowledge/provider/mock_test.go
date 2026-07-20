package provider

import (
	"context"
	"errors"
	"testing"
)

func TestMockProviderIsDeterministic(t *testing.T) {
	first, firstMeta, err := NewMockKnowledgeProvider().SearchPlaces(context.Background(), SearchRequest{
		DestinationName: "Rome", SourcePolicy: DefaultSourcePolicy(),
	})
	if err != nil {
		t.Fatalf("SearchPlaces() error = %v", err)
	}
	second, secondMeta, err := NewMockKnowledgeProvider().SearchPlaces(context.Background(), SearchRequest{
		DestinationName: "Rome", SourcePolicy: DefaultSourcePolicy(),
	})
	if err != nil {
		t.Fatalf("SearchPlaces() error = %v", err)
	}
	if len(first) != len(second) || firstMeta.ResultCount != secondMeta.ResultCount {
		t.Fatalf("mock provider must return identical results across runs: %d vs %d", len(first), len(second))
	}
	for index := range first {
		if first[index].ProviderPlaceID != second[index].ProviderPlaceID {
			t.Fatalf("result order must be stable at index %d: %q vs %q",
				index, first[index].ProviderPlaceID, second[index].ProviderPlaceID)
		}
		if !first[index].ObservedAt.Equal(second[index].ObservedAt) {
			t.Fatalf("observedAt must be anchored to MockReferenceTime for %q", first[index].ProviderPlaceID)
		}
	}
}

func TestMockProviderCoversRequiredDestinations(t *testing.T) {
	provider := NewMockKnowledgeProvider()
	for _, destination := range []string{"Rome", "Paris", "Vienna", "Bratislava"} {
		records, _, err := provider.SearchPlaces(context.Background(), SearchRequest{
			DestinationName: destination, SourcePolicy: DefaultSourcePolicy(),
		})
		if err != nil {
			t.Fatalf("SearchPlaces(%s) error = %v", destination, err)
		}
		if len(records) == 0 {
			t.Fatalf("destination %s must have deterministic fixtures", destination)
		}
	}
	records, _, err := provider.SearchPlaces(context.Background(), SearchRequest{
		DestinationName: "Atlantis", SourcePolicy: DefaultSourcePolicy(),
	})
	if err != nil {
		t.Fatalf("an unknown destination must not error: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("an unknown destination must return no records, got %d", len(records))
	}
}

// The fixtures exist to exercise dedup, review, and refresh paths; if these
// cases disappear the integration tests silently stop testing anything.
func TestMockFixturesContainTestableEdgeCases(t *testing.T) {
	records, _, err := NewMockKnowledgeProvider().SearchPlaces(context.Background(), SearchRequest{
		DestinationName: "Rome", SourcePolicy: DefaultSourcePolicy(),
	})
	if err != nil {
		t.Fatalf("SearchPlaces() error = %v", err)
	}

	missingCoordinates := false
	withOpeningHours := false
	for _, record := range records {
		if record.Latitude == nil || record.Longitude == nil {
			missingCoordinates = true
		}
		if len(record.OpeningHours) > 0 {
			withOpeningHours = true
		}
	}
	if !missingCoordinates {
		t.Error("Rome fixtures must include a record without coordinates")
	}
	if !withOpeningHours {
		t.Error("Rome fixtures must include a record with opening hours")
	}

	// A stale record must exist for the refresh-stale-places job.
	staleFound := false
	for _, record := range records {
		if MockReferenceTime.Sub(record.ObservedAt).Hours()/24 > 180 {
			staleFound = true
		}
	}
	if !staleFound {
		t.Error("Rome fixtures must include a stale observation")
	}
}

func TestMockProviderRespectsLimitAndCategoryFilters(t *testing.T) {
	provider := NewMockKnowledgeProvider()
	records, metadata, err := provider.SearchPlaces(context.Background(), SearchRequest{
		DestinationName: "Rome", Limit: 2, SourcePolicy: DefaultSourcePolicy(),
	})
	if err != nil {
		t.Fatalf("SearchPlaces() error = %v", err)
	}
	if len(records) != 2 || !metadata.Truncated {
		t.Fatalf("limit must truncate results, got %d (truncated=%v)", len(records), metadata.Truncated)
	}

	landmarks, _, err := provider.SearchPlaces(context.Background(), SearchRequest{
		DestinationName: "Rome", Categories: []string{"landmark"}, SourcePolicy: DefaultSourcePolicy(),
	})
	if err != nil {
		t.Fatalf("SearchPlaces() error = %v", err)
	}
	for _, record := range landmarks {
		if record.Category != "landmark" {
			t.Fatalf("category filter leaked %q", record.Category)
		}
	}
}

func TestMockProviderRequireCoordsFiltersIncompleteRecords(t *testing.T) {
	policy := DefaultSourcePolicy()
	policy.RequireCoords = true
	records, _, err := NewMockKnowledgeProvider().SearchPlaces(context.Background(), SearchRequest{
		DestinationName: "Rome", SourcePolicy: policy,
	})
	if err != nil {
		t.Fatalf("SearchPlaces() error = %v", err)
	}
	for _, record := range records {
		if record.Latitude == nil || record.Longitude == nil {
			t.Fatalf("RequireCoords must exclude %q", record.ProviderPlaceID)
		}
	}
}

func TestMockProviderWithholdsRawPayloadUnlessPolicyAllows(t *testing.T) {
	records, _, err := NewMockKnowledgeProvider().SearchPlaces(context.Background(), SearchRequest{
		DestinationName: "Vienna", SourcePolicy: DefaultSourcePolicy(),
	})
	if err != nil {
		t.Fatalf("SearchPlaces() error = %v", err)
	}
	for _, record := range records {
		if record.RawPayload != nil {
			t.Fatalf("raw payload must be withheld by default for %q", record.ProviderPlaceID)
		}
	}
}

func TestMockProviderGetPlaceDetails(t *testing.T) {
	provider := NewMockKnowledgeProvider()
	record, _, err := provider.GetPlaceDetails(context.Background(), "mock-paris-001")
	if err != nil {
		t.Fatalf("GetPlaceDetails() error = %v", err)
	}
	if record.Name != "Louvre Museum" {
		t.Fatalf("unexpected record %q", record.Name)
	}
	if _, _, err := provider.GetPlaceDetails(context.Background(), "does-not-exist"); !errors.Is(err, ErrPlaceNotFound) {
		t.Fatalf("expected ErrPlaceNotFound, got %v", err)
	}
}

func TestMockProviderLicenseIsValidAndLabelled(t *testing.T) {
	license := NewMockKnowledgeProvider().LicenseInfo()
	if !license.Valid() {
		t.Fatal("the mock license must be valid so ingestion paths are exercised")
	}
	if license.Attribution == "" {
		t.Fatal("attribution must be present even for synthetic data")
	}
}

func TestMockProviderHonoursContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := NewMockKnowledgeProvider().SearchPlaces(ctx, SearchRequest{DestinationName: "Rome"}); err == nil {
		t.Fatal("a cancelled context must be honoured")
	}
}
