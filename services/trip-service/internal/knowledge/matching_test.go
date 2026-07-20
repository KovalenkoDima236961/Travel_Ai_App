package knowledge

import (
	"testing"
)

func coords(latitude, longitude float64) (*float64, *float64) {
	return &latitude, &longitude
}

func colosseumObservation() NormalizedObservation {
	latitude, longitude := coords(41.8902, 12.4922)
	return NormalizedObservation{
		Provider:        "mock",
		ProviderPlaceID: "mock-rome-001",
		NormalizedName:  NormalizeMatchName("Colosseum"),
		Aliases:         []string{"Colosseo"},
		Category:        "landmark",
		Latitude:        latitude,
		Longitude:       longitude,
	}
}

func colosseumCandidate() MatchCandidate {
	latitude, longitude := coords(41.8902, 12.4922)
	return MatchCandidate{
		PlaceID: "place-colosseum", CanonicalName: "Colosseum", Category: "landmark",
		Latitude: latitude, Longitude: longitude, DestinationID: "rome",
		ReviewStatus: ReviewStatusAuto,
	}
}

func TestMatchScoreProviderReferenceIsConclusive(t *testing.T) {
	candidate := colosseumCandidate()
	candidate.CanonicalName = "Something Entirely Different"
	candidate.ProviderRefs = []string{"mock:mock-rome-001"}
	score, reasons := MatchScore(colosseumObservation(), candidate)
	if score != 1.0 {
		t.Fatalf("provider reference match must score 1.0, got %.4f", score)
	}
	if len(reasons) != 1 || reasons[0] != "provider reference match" {
		t.Fatalf("unexpected reasons: %v", reasons)
	}
}

func TestMatchScoreExactNameAndProximityAutoMatches(t *testing.T) {
	score, _ := MatchScore(colosseumObservation(), colosseumCandidate())
	thresholds := DefaultThresholds()
	if score < thresholds.AutoMatchConfidence {
		t.Fatalf("identical name and coordinates must auto-match, got %.4f", score)
	}
}

// "Colosseo (Anfiteatro Flavio)" must match "Colosseum" via its alias.
func TestMatchScoreAliasMatchesLocalLanguageName(t *testing.T) {
	latitude, longitude := coords(41.8905, 12.4924)
	observation := NormalizedObservation{
		Provider: "mock", ProviderPlaceID: "mock-rome-002",
		NormalizedName: NormalizeMatchName("Colosseo (Anfiteatro Flavio)"),
		Aliases:        []string{"Colosseum"},
		Category:       "landmark", Latitude: latitude, Longitude: longitude,
	}
	score, _ := MatchScore(observation, colosseumCandidate())
	if score < DefaultThresholds().ReviewMatchConfidence {
		t.Fatalf("alias match with nearby coordinates must be at least review-worthy, got %.4f", score)
	}
}

// Two places sharing a generic name in different cities must not match.
func TestMatchScoreDistanceVetoBlocksNameCollisions(t *testing.T) {
	romeLat, romeLng := coords(41.8902, 12.4922)
	parisLat, parisLng := coords(48.8584, 2.2945)
	observation := NormalizedObservation{
		Provider: "mock", ProviderPlaceID: "x", NormalizedName: "old town",
		Category: "neighborhood", Latitude: romeLat, Longitude: romeLng,
	}
	candidate := MatchCandidate{
		PlaceID: "other-city", CanonicalName: "Old Town", Category: "neighborhood",
		Latitude: parisLat, Longitude: parisLng, DestinationID: "paris",
	}
	score, reasons := MatchScore(observation, candidate)
	if score >= DefaultThresholds().ReviewMatchConfidence {
		t.Fatalf("identical names far apart must not match, got %.4f (%v)", score, reasons)
	}
}

func TestDecideMatchAppliesDocumentedThresholds(t *testing.T) {
	thresholds := DefaultThresholds()

	decision := DecideMatch(colosseumObservation(), []MatchCandidate{colosseumCandidate()}, thresholds)
	if decision.Action != MatchActionAutoMatch {
		t.Fatalf("expected auto match, got %q at %.4f", decision.Action, decision.Confidence)
	}

	if decision := DecideMatch(colosseumObservation(), nil, thresholds); decision.Action != MatchActionNoMatch {
		t.Fatalf("no candidates must yield no match, got %q", decision.Action)
	}

	unrelated := MatchCandidate{
		PlaceID: "unrelated", CanonicalName: "Villa Borghese", Category: "park", DestinationID: "rome",
	}
	if decision := DecideMatch(colosseumObservation(), []MatchCandidate{unrelated}, thresholds); decision.Action != MatchActionNoMatch {
		t.Fatalf("an unrelated candidate must yield no match, got %q at %.4f", decision.Action, decision.Confidence)
	}
}

// Ambiguity is a human decision: two plausible candidates must go to review
// rather than silently picking the higher score.
func TestDecideMatchAmbiguousCandidatesNeedReview(t *testing.T) {
	second := colosseumCandidate()
	second.PlaceID = "place-colosseum-duplicate"
	decision := DecideMatch(colosseumObservation(), []MatchCandidate{colosseumCandidate(), second}, DefaultThresholds())
	if decision.Action != MatchActionNeedsReview {
		t.Fatalf("multiple plausible candidates must need review, got %q", decision.Action)
	}
}

func TestDecideMatchIsDeterministic(t *testing.T) {
	candidates := []MatchCandidate{colosseumCandidate(), {
		PlaceID: "place-forum", CanonicalName: "Roman Forum", Category: "landmark", DestinationID: "rome",
	}}
	first := DecideMatch(colosseumObservation(), candidates, DefaultThresholds())
	reversed := []MatchCandidate{candidates[1], candidates[0]}
	second := DecideMatch(colosseumObservation(), reversed, DefaultThresholds())
	if first.PlaceID != second.PlaceID || first.Confidence != second.Confidence || first.Action != second.Action {
		t.Fatalf("matching must not depend on candidate order: %+v vs %+v", first, second)
	}
}

func TestDetectDuplicatesFindsNearIdenticalPlaces(t *testing.T) {
	latitude, longitude := coords(41.8902, 12.4922)
	nearLatitude, nearLongitude := coords(41.8905, 12.4924)
	places := []MatchCandidate{
		{PlaceID: "b-place", CanonicalName: "Colosseum", Category: "landmark", Latitude: latitude, Longitude: longitude, DestinationID: "rome", ReviewStatus: ReviewStatusAuto},
		{PlaceID: "a-place", CanonicalName: "Colosseo", Aliases: []string{"Colosseum"}, Category: "landmark", Latitude: nearLatitude, Longitude: nearLongitude, DestinationID: "rome", ReviewStatus: ReviewStatusAuto},
		{PlaceID: "c-place", CanonicalName: "Villa Borghese", Category: "park", DestinationID: "rome", ReviewStatus: ReviewStatusAuto},
	}
	pairs := DetectDuplicates(places, DefaultThresholds())
	if len(pairs) != 1 {
		t.Fatalf("expected exactly one duplicate pair, got %d: %+v", len(pairs), pairs)
	}
	// Pair IDs are ordered so group creation is stable across runs.
	if pairs[0].PlaceID != "a-place" || pairs[0].OtherPlaceID != "b-place" {
		t.Fatalf("duplicate pair IDs must be canonically ordered, got %+v", pairs[0])
	}
}

func TestDetectDuplicatesSkipsResolvedRecords(t *testing.T) {
	latitude, longitude := coords(41.8902, 12.4922)
	places := []MatchCandidate{
		{PlaceID: "a", CanonicalName: "Colosseum", Category: "landmark", Latitude: latitude, Longitude: longitude, DestinationID: "rome", ReviewStatus: ReviewStatusMerged},
		{PlaceID: "b", CanonicalName: "Colosseum", Category: "landmark", Latitude: latitude, Longitude: longitude, DestinationID: "rome", ReviewStatus: ReviewStatusAuto},
	}
	if pairs := DetectDuplicates(places, DefaultThresholds()); len(pairs) != 0 {
		t.Fatalf("already-merged records must not be re-detected, got %+v", pairs)
	}
}

func TestDetectDuplicatesIgnoresOtherDestinations(t *testing.T) {
	places := []MatchCandidate{
		{PlaceID: "a", CanonicalName: "Old Town", Category: "neighborhood", DestinationID: "rome", ReviewStatus: ReviewStatusAuto},
		{PlaceID: "b", CanonicalName: "Old Town", Category: "neighborhood", DestinationID: "bratislava", ReviewStatus: ReviewStatusAuto},
	}
	if pairs := DetectDuplicates(places, DefaultThresholds()); len(pairs) != 0 {
		t.Fatalf("duplicates must be scoped to one destination, got %+v", pairs)
	}
}

func TestResolveMergeKeepsBestFieldsFromHighestTrustRecord(t *testing.T) {
	trustedLat, trustedLng := coords(41.8902, 12.4922)
	candidates := []MergeCandidate{
		{
			PlaceID: "low-trust", TrustLevel: "mock", QualityScore: 0.5, ObservedAt: 3000,
			Category: "viewpoint", Address: "Fallback address",
			OpeningHours: []byte(`[{"weekday":1,"opens":"10:00","closes":"18:00"}]`),
			Aliases:      []string{"Anfiteatro Flavio"}, Tags: []string{"ancient"},
			ProviderRefs: []string{"mock:mock-rome-002"},
		},
		{
			PlaceID: "high-trust", TrustLevel: TrustLevelCurated, QualityScore: 0.95, ObservedAt: 1000,
			Category: "landmark", Latitude: trustedLat, Longitude: trustedLng,
			Website: "https://example.invalid/colosseum",
			Aliases: []string{"Colosseo"}, Tags: []string{"unesco"},
			ProviderRefs: []string{"mock:mock-rome-001"},
		},
	}
	resolution := ResolveMerge("high-trust", candidates)

	if resolution.Category != "landmark" {
		t.Fatalf("category must come from the highest-trust record, got %q", resolution.Category)
	}
	if resolution.FieldSources["category"] != "high-trust" {
		t.Fatalf("field provenance must be recorded, got %+v", resolution.FieldSources)
	}
	if resolution.Latitude == nil || *resolution.Latitude != 41.8902 {
		t.Fatal("coordinates must come from the highest-trust record that has them")
	}
	// Opening hours are the one field where freshness beats trust.
	if resolution.FieldSources["openingHours"] != "low-trust" {
		t.Fatalf("opening hours must come from the freshest record, got %+v", resolution.FieldSources)
	}
	if resolution.Address != "Fallback address" {
		t.Fatal("a field absent from the primary record must be filled from another member")
	}
	if len(resolution.Aliases) != 2 || len(resolution.Tags) != 2 || len(resolution.ProviderRefs) != 2 {
		t.Fatalf("aliases, tags, and provider refs must be combined: %+v", resolution)
	}
	if len(resolution.MergedPlaceIDs) != 1 || resolution.MergedPlaceIDs[0] != "low-trust" {
		t.Fatalf("non-canonical members must be listed as merged, got %v", resolution.MergedPlaceIDs)
	}
}

func TestResolveMergeIsOrderIndependent(t *testing.T) {
	candidates := []MergeCandidate{
		{PlaceID: "a", TrustLevel: "mock", QualityScore: 0.4, Category: "cafe"},
		{PlaceID: "b", TrustLevel: TrustLevelCurated, QualityScore: 0.9, Category: "landmark"},
	}
	first := ResolveMerge("b", candidates)
	second := ResolveMerge("b", []MergeCandidate{candidates[1], candidates[0]})
	if first.Category != second.Category || first.FieldSources["category"] != second.FieldSources["category"] {
		t.Fatalf("merge resolution must not depend on input order: %+v vs %+v", first, second)
	}
}

func TestHaversineKmKnownDistance(t *testing.T) {
	// Rome Colosseum to Paris Eiffel Tower is roughly 1100 km.
	distance := HaversineKm(41.8902, 12.4922, 48.8584, 2.2945)
	if distance < 1080 || distance > 1130 {
		t.Fatalf("unexpected distance %.2f km", distance)
	}
	if got := HaversineKm(41.8902, 12.4922, 41.8902, 12.4922); got != 0 {
		t.Fatalf("identical coordinates must be 0 km apart, got %.6f", got)
	}
}
