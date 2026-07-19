package knowledge

import "testing"

func TestPlaceKnowledgeNormalizeAndValidate(t *testing.T) {
	latitude, longitude := 41.8902, 12.4922
	duration := 120
	place := PlaceKnowledge{
		Name: "  Colosseum ", Category: "LANDMARK", Latitude: &latitude, Longitude: &longitude,
		Aliases: []string{"Colosseo", "colosseo", ""}, TypicalDurationMinutes: &duration,
		SourceKey: "manual_curated", Confidence: 0.95,
	}
	sources := map[string]Source{"manual_curated": {SourceKey: "manual_curated", SourceType: SourceTypeManualCurated, Enabled: true}}
	if err := place.NormalizeAndValidate(sources); err != nil {
		t.Fatalf("NormalizeAndValidate() error = %v", err)
	}
	if place.Name != "Colosseum" || place.Category != "landmark" || len(place.Aliases) != 1 {
		t.Fatalf("place was not normalized: %+v", place)
	}
}

func TestPlaceKnowledgeRejectsOutOfRangeConfidence(t *testing.T) {
	place := PlaceKnowledge{Name: "Example", Category: "museum", SourceKey: "manual_curated", Confidence: 1.1}
	sources := map[string]Source{"manual_curated": {SourceKey: "manual_curated", SourceType: SourceTypeManualCurated, Enabled: true}}
	if err := place.NormalizeAndValidate(sources); err == nil {
		t.Fatal("expected invalid confidence to be rejected")
	}
}

func TestNormalizeNameAndChecksumAreStable(t *testing.T) {
	if got, want := NormalizeName("  Musée-d'Orsay  "), "musée d orsay"; got != want {
		t.Fatalf("NormalizeName() = %q, want %q", got, want)
	}
	if Checksum(" note ") != Checksum("note") {
		t.Fatal("Checksum must ignore surrounding whitespace")
	}
}
