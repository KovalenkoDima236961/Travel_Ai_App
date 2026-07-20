package knowledge

import (
	"testing"
	"time"
)

var scoringNow = time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)

// highQualityInput is a well-formed curated record: the baseline that must
// reach strong grounding.
func highQualityInput() QualityInput {
	return QualityInput{
		TrustLevel:         "trusted_provider",
		Category:           "landmark",
		DestinationMatched: true,
		HasCoordinates:     true,
		CategoryConfident:  true,
		NameQuality:        NameQuality("Colosseum"),
		ObservedAt:         scoringNow.AddDate(0, 0, -2),
		Now:                scoringNow,
		ProviderAgreement:  0.9,
		HasOpeningHours:    true,
		HasAddress:         true,
		HasWebsite:         true,
		ReviewStatus:       ReviewStatusAuto,
		LicensePresent:     true,
	}
}

func TestComputeQualityIsDeterministic(t *testing.T) {
	first := ComputeQuality(highQualityInput())
	second := ComputeQuality(highQualityInput())
	if first != second {
		t.Fatalf("quality scoring must be deterministic: %+v vs %+v", first, second)
	}
}

func TestComputeQualityHighQualityRecordReachesStrongGrounding(t *testing.T) {
	breakdown := ComputeQuality(highQualityInput())
	thresholds := DefaultThresholds()
	if breakdown.QualityScore < thresholds.StrongMinQuality {
		t.Fatalf("expected strong-grounding score, got %.4f (%+v)", breakdown.QualityScore, breakdown)
	}
	if got := GroundingStrength(breakdown.QualityScore, ReviewStatusAuto, thresholds); got != GroundingStrengthStrong {
		t.Fatalf("GroundingStrength() = %q, want strong", got)
	}
}

func TestComputeQualityPenalizesMissingCoordinates(t *testing.T) {
	withCoordinates := ComputeQuality(highQualityInput())
	input := highQualityInput()
	input.HasCoordinates = false
	withoutCoordinates := ComputeQuality(input)
	if withoutCoordinates.QualityScore >= withCoordinates.QualityScore {
		t.Fatal("missing coordinates must lower the quality score")
	}
	// 0.15 weight, scaled by the name-quality multiplier.
	if delta := withCoordinates.QualityScore - withoutCoordinates.QualityScore; delta < 0.12 || delta > 0.15 {
		t.Fatalf("unexpected coordinate weight delta %.4f", delta)
	}
}

func TestComputeQualityMissingLicenseIsCapped(t *testing.T) {
	input := highQualityInput()
	input.LicensePresent = false
	breakdown := ComputeQuality(input)
	if breakdown.QualityScore > 0.40 {
		t.Fatalf("unlicensed record must be capped at 0.40, got %.4f", breakdown.QualityScore)
	}
	if got := GroundingStrength(breakdown.QualityScore, ReviewStatusAuto, DefaultThresholds()); got != GroundingStrengthExcluded {
		t.Fatalf("unlicensed record must be excluded from grounding, got %q", got)
	}
}

func TestComputeQualityRejectedAndMergedScoreZero(t *testing.T) {
	for _, status := range []string{ReviewStatusRejected, ReviewStatusMerged} {
		input := highQualityInput()
		input.ReviewStatus = status
		if breakdown := ComputeQuality(input); breakdown.QualityScore != 0 {
			t.Fatalf("%s record must score 0, got %.4f", status, breakdown.QualityScore)
		}
	}
}

func TestComputeQualityGenericNameCannotReachStrongGrounding(t *testing.T) {
	input := highQualityInput()
	input.TrustLevel = "mock"
	input.NameQuality = NameQuality("Cafe")
	input.Category = "cafe"
	input.HasOpeningHours = false
	input.HasWebsite = false
	input.ObservedAt = scoringNow.AddDate(0, 0, -120)
	breakdown := ComputeQuality(input)
	if got := GroundingStrength(breakdown.QualityScore, ReviewStatusAuto, DefaultThresholds()); got == GroundingStrengthStrong {
		t.Fatalf("generic stale mock record must not be strong grounding (score %.4f)", breakdown.QualityScore)
	}
}

func TestNameQualityRanksSpecificNamesHigher(t *testing.T) {
	if NameQuality("Cafe") >= NameQuality("Kunsthistorisches Museum") {
		t.Fatal("a generic single-word name must score below a specific name")
	}
	if NameQuality("") != 0 {
		t.Fatal("an empty name must score 0")
	}
}

func TestFreshnessScoreDependsOnCategoryWindow(t *testing.T) {
	observedAt := scoringNow.AddDate(0, 0, -60)
	landmark := FreshnessScore("landmark", observedAt, scoringNow)
	restaurant := FreshnessScore("restaurant", observedAt, scoringNow)
	if landmark <= restaurant {
		t.Fatalf("landmarks decay slower than restaurants: landmark=%.4f restaurant=%.4f", landmark, restaurant)
	}
	if got := FreshnessScore("restaurant", scoringNow.AddDate(0, 0, -365), scoringNow); got != 0 {
		t.Fatalf("a year-old restaurant record must score 0, got %.4f", got)
	}
	if got := FreshnessScore("landmark", scoringNow, scoringNow); got != 1 {
		t.Fatalf("a just-observed record must score 1, got %.4f", got)
	}
}

func TestSourceTrustScoreOrdering(t *testing.T) {
	if SourceTrustScore(TrustLevelCurated) <= SourceTrustScore("trusted_provider") {
		t.Fatal("curated data must outrank provider data")
	}
	if SourceTrustScore("mock") >= SourceTrustScore("public_open_data") {
		t.Fatal("mock data must rank below open data")
	}
	if SourceTrustScore("not-a-level") != TrustWeights["unknown"] {
		t.Fatal("an unrecognized trust level must fall back to the unknown weight")
	}
}

func TestFeedbackScoreIsBoundedAndNeutralWithoutSignals(t *testing.T) {
	if got := (FeedbackCounts{}).FeedbackScore(); got != 0.5 {
		t.Fatalf("no feedback must be neutral, got %.4f", got)
	}
	single := FeedbackCounts{Negative: 1, DistinctNegativeUsers: 1}
	if single.FeedbackScore() < 0.3 {
		t.Fatalf("one negative signal must not collapse the score, got %.4f", single.FeedbackScore())
	}
	if single.RequiresReview() {
		t.Fatal("a single user must not be able to force review")
	}
	if !(FeedbackCounts{Negative: 3, DistinctNegativeUsers: 2}).RequiresReview() {
		t.Fatal("repeated negative feedback from distinct users must trigger review")
	}
}

func TestProviderAgreementRequiresIndependentProviders(t *testing.T) {
	single := []NormalizedObservation{{Provider: "mock", Category: "landmark"}}
	if got := ProviderAgreement(single); got != 0.5 {
		t.Fatalf("a lone observation must be neutral, got %.4f", got)
	}
	sameProvider := []NormalizedObservation{
		{Provider: "mock", Category: "landmark"},
		{Provider: "mock", Category: "landmark"},
	}
	if got := ProviderAgreement(sameProvider); got > 0.6 {
		t.Fatalf("repeat observations from one provider are not corroboration, got %.4f", got)
	}

	latitude, longitude := 41.8902, 12.4922
	latitude2, longitude2 := 41.8905, 12.4924
	agreeing := []NormalizedObservation{
		{Provider: "mock", Category: "landmark", Latitude: &latitude, Longitude: &longitude},
		{Provider: "osm", Category: "landmark", Latitude: &latitude2, Longitude: &longitude2},
	}
	if got := ProviderAgreement(agreeing); got < 0.9 {
		t.Fatalf("independent agreeing providers must score high, got %.4f", got)
	}

	farAway := 48.8584
	farLongitude := 2.2945
	disagreeing := []NormalizedObservation{
		{Provider: "mock", Category: "landmark", Latitude: &latitude, Longitude: &longitude},
		{Provider: "osm", Category: "cafe", Latitude: &farAway, Longitude: &farLongitude},
	}
	if got := ProviderAgreement(disagreeing); got > 0.4 {
		t.Fatalf("conflicting providers must score low, got %.4f", got)
	}
}

func TestGroundingStrengthExcludesRejectedAndMerged(t *testing.T) {
	thresholds := DefaultThresholds()
	for _, status := range []string{ReviewStatusRejected, ReviewStatusMerged} {
		if got := GroundingStrength(0.99, status, thresholds); got != GroundingStrengthExcluded {
			t.Fatalf("%s must be excluded regardless of score, got %q", status, got)
		}
	}
	if got := GroundingStrength(0.60, ReviewStatusApproved, thresholds); got != GroundingStrengthStrong {
		t.Fatalf("human approval must promote a mid-scoring record, got %q", got)
	}
	if got := GroundingStrength(0.40, ReviewStatusApproved, thresholds); got != GroundingStrengthWeak {
		t.Fatalf("approval must not rescue a record below the weak floor, got %q", got)
	}
	if got := GroundingStrength(0.60, ReviewStatusAuto, thresholds); got != GroundingStrengthWeak {
		t.Fatalf("mid-scoring auto record must be weak, got %q", got)
	}
	if got := GroundingStrength(0.20, ReviewStatusAuto, thresholds); got != GroundingStrengthExcluded {
		t.Fatalf("low-scoring record must be excluded, got %q", got)
	}
}

func TestReviewStatusForScoreNeverOverridesHumanDecisions(t *testing.T) {
	thresholds := DefaultThresholds()
	for _, status := range []string{ReviewStatusApproved, ReviewStatusRejected, ReviewStatusMerged} {
		if got := ReviewStatusForScore(status, 0.01, FeedbackCounts{}, thresholds); got != status {
			t.Fatalf("human status %q must be preserved, got %q", status, got)
		}
	}
	if got := ReviewStatusForScore(ReviewStatusAuto, 0.20, FeedbackCounts{}, thresholds); got != ReviewStatusRejected {
		t.Fatalf("score below reject threshold must reject, got %q", got)
	}
	if got := ReviewStatusForScore(ReviewStatusAuto, 0.60, FeedbackCounts{}, thresholds); got != ReviewStatusNeedsReview {
		t.Fatalf("score below needs-review threshold must flag review, got %q", got)
	}
	if got := ReviewStatusForScore(ReviewStatusAuto, 0.90, FeedbackCounts{}, thresholds); got != ReviewStatusAuto {
		t.Fatalf("high score must stay auto, got %q", got)
	}
	if got := ReviewStatusForScore(ReviewStatusAuto, 0.90, FeedbackCounts{Negative: 3, DistinctNegativeUsers: 2}, thresholds); got != ReviewStatusNeedsReview {
		t.Fatalf("broad negative feedback must flag review even at a high score, got %q", got)
	}
}

func TestComputeCoverageFlagsLimitedDestinations(t *testing.T) {
	empty := ComputeCoverage(CoverageInput{DestinationName: "Nowhere"})
	if empty.Status != "unavailable" || len(empty.Warnings) == 0 {
		t.Fatalf("a destination without places must be unavailable: %+v", empty)
	}

	thin := ComputeCoverage(CoverageInput{
		DestinationName: "Thin", Places: 3, HighQualityPlaces: 1, FreshPlaces: 1,
		WithCoordinates: 1, WithOpeningHours: 0, CategoriesPresent: []string{"landmark"},
	})
	if thin.Status == "available" {
		t.Fatalf("a thin destination must not report full coverage: %+v", thin)
	}
	if !containsWarning(thin.Warnings, "Limited verified place data for this destination.") {
		t.Fatalf("expected the user-facing limited-coverage warning, got %v", thin.Warnings)
	}

	rich := ComputeCoverage(CoverageInput{
		DestinationName: "Rich", Places: 20, HighQualityPlaces: 18, FreshPlaces: 18,
		WithCoordinates: 20, WithOpeningHours: 15,
		CategoriesPresent: []string{"landmark", "museum", "park", "restaurant", "neighborhood"},
	})
	if rich.Status != "available" {
		t.Fatalf("a well-covered destination must be available: %+v", rich)
	}
	if rich.CategoryCoverage != 1 {
		t.Fatalf("all core categories present must give full category coverage, got %.4f", rich.CategoryCoverage)
	}
}

func containsWarning(warnings []string, want string) bool {
	for _, warning := range warnings {
		if warning == want {
			return true
		}
	}
	return false
}
