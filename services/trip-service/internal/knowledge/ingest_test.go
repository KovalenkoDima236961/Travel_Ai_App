package knowledge

import (
	"context"
	"testing"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge/provider"
)

// These tests run the ingestion pipeline stages (fetch, normalize, score,
// match, deduplicate) against the real mock fixtures without a database. They
// are the CI guarantee that provider ingestion is deterministic and that the
// quality gates behave as documented.

func mockObservations(t *testing.T, destination string) []NormalizedObservation {
	t.Helper()
	records, _, err := provider.NewMockKnowledgeProvider().SearchPlaces(context.Background(), provider.SearchRequest{
		DestinationName: destination,
		SourcePolicy:    provider.DefaultSourcePolicy(),
	})
	if err != nil {
		t.Fatalf("mock SearchPlaces(%s) error = %v", destination, err)
	}
	observations := make([]NormalizedObservation, 0, len(records))
	for _, record := range records {
		observation, normalizeErr := NormalizeProviderRecord(record, provider.DefaultSourcePolicy())
		if normalizeErr != nil {
			t.Fatalf("NormalizeProviderRecord(%s) error = %v", record.ProviderPlaceID, normalizeErr)
		}
		observations = append(observations, observation)
	}
	return observations
}

func TestPipelineNormalizesEveryMockFixture(t *testing.T) {
	for _, destination := range []string{"Rome", "Paris", "Vienna", "Bratislava"} {
		observations := mockObservations(t, destination)
		if len(observations) == 0 {
			t.Fatalf("%s produced no observations", destination)
		}
		for _, observation := range observations {
			if _, ok := allowedCategories[observation.Category]; !ok {
				t.Errorf("%s: %q normalized to unstorable category %q",
					destination, observation.DisplayName, observation.Category)
			}
			if observation.NormalizedName == "" {
				t.Errorf("%s: %q produced an empty match name", destination, observation.DisplayName)
			}
			if observation.LicenseName == "" {
				t.Errorf("%s: %q has no license name", destination, observation.DisplayName)
			}
		}
	}
}

// Scoring must be stable across runs, otherwise a record could drift across a
// grounding threshold with no data change.
func TestPipelineScoringIsReproducible(t *testing.T) {
	now := provider.MockReferenceTime
	first := scoreAll(mockObservations(t, "Rome"), now)
	second := scoreAll(mockObservations(t, "Rome"), now)
	for name, score := range first {
		if second[name] != score {
			t.Fatalf("score for %q changed between runs: %.6f vs %.6f", name, score, second[name])
		}
	}
}

func scoreAll(observations []NormalizedObservation, now time.Time) map[string]float64 {
	agreement := providerAgreementByName(observations)
	scores := make(map[string]float64, len(observations))
	for _, observation := range observations {
		quality := ComputeQuality(QualityInput{
			TrustLevel:         "mock",
			Category:           observation.Category,
			DestinationMatched: true,
			HasCoordinates:     observation.Latitude != nil && observation.Longitude != nil,
			CategoryConfident:  observation.Category != "other",
			NameQuality:        NameQuality(observation.DisplayName),
			ObservedAt:         observation.ObservedAt,
			Now:                now,
			ProviderAgreement:  agreement[observation.NormalizedName],
			HasOpeningHours:    len(observation.OpeningHours) > 0,
			HasAddress:         observation.Address != "",
			HasWebsite:         observation.Website != "",
			ReviewStatus:       ReviewStatusAuto,
			LicensePresent:     observation.LicenseName != "",
		})
		scores[observation.ProviderPlaceID] = quality.QualityScore
	}
	return scores
}

// The incomplete Rome fixtures must score below the complete ones. This is the
// behaviour that keeps thin provider records out of strong grounding.
func TestPipelineIncompleteRecordsScoreLower(t *testing.T) {
	scores := scoreAll(mockObservations(t, "Rome"), provider.MockReferenceTime)

	complete := scores["mock-rome-001"] // Colosseum: coordinates, hours, address, website
	noCoordinates := scores["mock-rome-004"]
	stale := scores["mock-rome-005"]

	if complete <= noCoordinates {
		t.Errorf("a complete record (%.4f) must outscore one without coordinates (%.4f)", complete, noCoordinates)
	}
	if complete <= stale {
		t.Errorf("a fresh record (%.4f) must outscore a stale one (%.4f)", complete, stale)
	}
}

// Ingestion must find the intentional Colosseum duplicate in the Rome fixtures.
func TestPipelineDetectsIntentionalMockDuplicates(t *testing.T) {
	observations := mockObservations(t, "Rome")

	// Model the fixtures as stored places, which is what dedup runs against.
	candidates := make([]MatchCandidate, 0, len(observations))
	for _, observation := range observations {
		candidates = append(candidates, MatchCandidate{
			PlaceID:       observation.ProviderPlaceID,
			CanonicalName: observation.DisplayName,
			Aliases:       observation.Aliases,
			Category:      observation.Category,
			Latitude:      observation.Latitude,
			Longitude:     observation.Longitude,
			DestinationID: "rome",
			ReviewStatus:  ReviewStatusAuto,
		})
	}

	pairs := DetectDuplicates(candidates, DefaultThresholds())
	if len(pairs) == 0 {
		t.Fatal("the Rome fixtures contain a deliberate duplicate that was not detected")
	}
	found := false
	for _, pair := range pairs {
		if pair.PlaceID == "mock-rome-001" && pair.OtherPlaceID == "mock-rome-002" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the Colosseum duplicate pair, got %+v", pairs)
	}
}

// The Paris fixtures disagree about the Eiffel Tower's category. Cross-provider
// disagreement must be visible to scoring rather than silently resolved.
func TestPipelineSurfacesProviderDisagreement(t *testing.T) {
	observations := mockObservations(t, "Paris")

	var eiffel, tourEiffel NormalizedObservation
	for _, observation := range observations {
		switch observation.ProviderPlaceID {
		case "mock-paris-002":
			eiffel = observation
		case "mock-paris-003":
			tourEiffel = observation
		}
	}
	if eiffel.ProviderPlaceID == "" || tourEiffel.ProviderPlaceID == "" {
		t.Fatal("the Paris disagreement fixtures are missing")
	}
	if eiffel.Category == tourEiffel.Category {
		t.Fatal("the fixtures are meant to disagree about category")
	}

	// They still describe the same place, so matching must connect them.
	score, _ := MatchScore(tourEiffel, MatchCandidate{
		PlaceID: "place-eiffel", CanonicalName: eiffel.DisplayName, Aliases: eiffel.Aliases,
		Category: eiffel.Category, Latitude: eiffel.Latitude, Longitude: eiffel.Longitude,
		DestinationID: "paris", ReviewStatus: ReviewStatusAuto,
	})
	if score < DefaultThresholds().ReviewMatchConfidence {
		t.Fatalf("conflicting observations of the same place must still match, got %.4f", score)
	}
}

// A generic, stale, low-signal record must not reach strong grounding.
func TestPipelineWeakRecordDoesNotReachStrongGrounding(t *testing.T) {
	scores := scoreAll(mockObservations(t, "Bratislava"), provider.MockReferenceTime)
	strength := GroundingStrength(scores["mock-bratislava-003"], ReviewStatusAuto, DefaultThresholds())
	if strength == GroundingStrengthStrong {
		t.Fatalf("the generic 'Cafe' fixture must not be strong grounding (score %.4f)", scores["mock-bratislava-003"])
	}
}

func TestClassifyProviderErrorMapsToKnowledgeVocabulary(t *testing.T) {
	if got := ErrorCode(classifyProviderError(provider.ErrProviderRateLimited)); got != "knowledge_provider_rate_limited" {
		t.Fatalf("rate limit must map to knowledge_provider_rate_limited, got %q", got)
	}
	if got := ErrorCode(classifyProviderError(provider.ErrProviderUnavailable)); got != "knowledge_provider_unavailable" {
		t.Fatalf("unavailability must map to knowledge_provider_unavailable, got %q", got)
	}
	if got := ErrorCode(classifyProviderError(provider.ErrLicenseMissing)); got != "knowledge_license_missing" {
		t.Fatalf("missing license must map to knowledge_license_missing, got %q", got)
	}
}

func TestTrustLevelForProviderDefaultsToUnknown(t *testing.T) {
	if got := trustLevelForProvider("some-new-provider"); got != "unknown" {
		t.Fatalf("an unregistered provider must be untrusted, got %q", got)
	}
	if got := trustLevelForProvider(provider.ProviderMock); got != "mock" {
		t.Fatalf("mock provider trust level = %q", got)
	}
}

// Mock data must never outrank real curated or provider data.
func TestMockDataCannotOutrankRealSources(t *testing.T) {
	if SourceTrustScore(trustLevelForProvider(provider.ProviderMock)) >=
		SourceTrustScore(trustLevelForProvider(provider.ProviderFoursquare)) {
		t.Fatal("mock data must rank below a trusted provider")
	}
}

func TestSummarizeOpeningHoursGroupsWeekdays(t *testing.T) {
	daily := []byte(`[{"weekday":0,"opens":"09:00","closes":"19:00"},{"weekday":1,"opens":"09:00","closes":"19:00"},
    {"weekday":2,"opens":"09:00","closes":"19:00"},{"weekday":3,"opens":"09:00","closes":"19:00"},
    {"weekday":4,"opens":"09:00","closes":"19:00"},{"weekday":5,"opens":"09:00","closes":"19:00"},
    {"weekday":6,"opens":"09:00","closes":"19:00"}]`)
	if got := SummarizeOpeningHours(daily); got != "Daily 09:00-19:00" {
		t.Fatalf("SummarizeOpeningHours() = %q, want %q", got, "Daily 09:00-19:00")
	}
	if got := SummarizeOpeningHours(nil); got != "" {
		t.Fatalf("absent opening hours must summarize to empty, got %q", got)
	}
	if got := SummarizeOpeningHours([]byte(`not json`)); got != "" {
		t.Fatalf("malformed opening hours must summarize to empty, got %q", got)
	}
}
