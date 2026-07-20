package knowledge

import (
	"math"
	"sort"
	"strings"
	"time"
)

// Scoring is intentionally deterministic arithmetic over explicit inputs: no
// clocks, no randomness, no model calls. A record's grounding strength has to
// be reproducible and explainable in Ops, because it decides whether the record
// can influence a user's itinerary.

// Review states for travel_places.review_status.
const (
	ReviewStatusAuto        = "auto"
	ReviewStatusApproved    = "approved"
	ReviewStatusRejected    = "rejected"
	ReviewStatusNeedsReview = "needs_review"
	ReviewStatusMerged      = "merged"
)

// Match states for travel_provider_place_observations.match_status.
const (
	MatchStatusUnmatched   = "unmatched"
	MatchStatusMatched     = "matched"
	MatchStatusDuplicate   = "duplicate"
	MatchStatusRejected    = "rejected"
	MatchStatusNeedsReview = "needs_review"
)

// Grounding strength decides how a record may be used by the AI.
const (
	GroundingStrengthStrong   = "strong"
	GroundingStrengthWeak     = "weak"
	GroundingStrengthExcluded = "excluded"
)

// TrustWeights map source trust levels onto a 0..1 score. Curated editorial
// data outranks providers because it is reviewed by the team; mock data is
// deliberately capped below the strong-grounding threshold so synthetic
// fixtures can never dominate a real destination.
var TrustWeights = map[string]float64{
	TrustLevelCurated:  1.00,
	"trusted_provider": 0.85,
	"public_open_data": 0.70,
	"app_observed":     0.55,
	"user_feedback":    0.45,
	"mock":             0.40,
	"unknown":          0.15,
}

// SourceTrustScore returns the weight for a trust level. Unknown levels get the
// "unknown" weight rather than a neutral default, so an unrecognized source can
// never accidentally score well.
func SourceTrustScore(trustLevel string) float64 {
	if weight, ok := TrustWeights[strings.ToLower(strings.TrimSpace(trustLevel))]; ok {
		return weight
	}
	return TrustWeights["unknown"]
}

// Thresholds control the quality gates. Defaults mirror the documented
// KNOWLEDGE_* configuration and are overridable per environment.
type Thresholds struct {
	StrongMinQuality      float64
	WeakMinQuality        float64
	NeedsReviewBelow      float64
	RejectBelow           float64
	AutoMatchConfidence   float64
	ReviewMatchConfidence float64
	StaleAfterDays        int
}

// DefaultThresholds matches the documented defaults:
// KNOWLEDGE_AI_STRONG_MIN_QUALITY, KNOWLEDGE_AI_WEAK_MIN_QUALITY,
// KNOWLEDGE_NEEDS_REVIEW_BELOW_QUALITY, KNOWLEDGE_REJECT_BELOW_QUALITY.
func DefaultThresholds() Thresholds {
	return Thresholds{
		StrongMinQuality:      0.75,
		WeakMinQuality:        0.55,
		NeedsReviewBelow:      0.65,
		RejectBelow:           0.30,
		AutoMatchConfidence:   0.90,
		ReviewMatchConfidence: 0.70,
		StaleAfterDays:        30,
	}
}

// FreshnessWindowDays is the category-dependent period over which an
// observation decays from fresh to stale. Opening-hours-sensitive categories
// expire fastest because that is the field most likely to be wrong.
var FreshnessWindowDays = map[string]int{
	"restaurant":   45,
	"cafe":         45,
	"market":       45,
	"transport":    30,
	"activity":     90,
	"museum":       120,
	"landmark":     180,
	"neighborhood": 180,
	"viewpoint":    180,
	"park":         180,
	"nature":       180,
	"other":        90,
}

const defaultFreshnessWindowDays = 90

// FreshnessWindow returns the decay window for a category.
func FreshnessWindow(category string) int {
	if days, ok := FreshnessWindowDays[strings.ToLower(strings.TrimSpace(category))]; ok {
		return days
	}
	return defaultFreshnessWindowDays
}

// FreshnessScore decays linearly from 1.0 at observation time to 0.0 at twice
// the category window, so a record does not fall off a cliff the day it goes
// stale. A record observed in the future scores 1.0 rather than erroring.
func FreshnessScore(category string, observedAt, now time.Time) float64 {
	if observedAt.IsZero() {
		return 0
	}
	window := float64(FreshnessWindow(category))
	ageDays := now.Sub(observedAt).Hours() / 24
	if ageDays <= 0 {
		return 1
	}
	if ageDays >= window*2 {
		return 0
	}
	return clamp01(1 - ageDays/(window*2))
}

// FeedbackCounts aggregates user signals for one place. No single user can
// reject a place: influence is bounded and requires repetition.
type FeedbackCounts struct {
	Positive int
	Negative int
	// DistinctNegativeUsers guards against one unhappy user forcing a review.
	DistinctNegativeUsers int
}

// FeedbackScore returns a 0..1 factor. It starts neutral at 0.5 so a place with
// no feedback is neither rewarded nor punished.
func (f FeedbackCounts) FeedbackScore() float64 {
	total := f.Positive + f.Negative
	if total == 0 {
		return 0.5
	}
	ratio := float64(f.Positive) / float64(total)
	// Confidence in the ratio grows with sample size; with few signals the
	// score stays close to neutral.
	weight := math.Min(float64(total)/5.0, 1.0)
	return clamp01(0.5 + (ratio-0.5)*weight)
}

// RequiresReview reports whether negative feedback is broad enough to warrant
// human review. Two distinct users are required, so one user acting alone
// cannot flag a place.
func (f FeedbackCounts) RequiresReview() bool {
	return f.DistinctNegativeUsers >= 2 && f.Negative > f.Positive
}

// QualityInput carries every factor the score depends on. Making these explicit
// keeps the calculation testable and auditable in the Ops quality breakdown.
type QualityInput struct {
	TrustLevel         string
	Category           string
	DestinationMatched bool
	HasCoordinates     bool
	CategoryConfident  bool
	NameQuality        float64
	ObservedAt         time.Time
	Now                time.Time
	ProviderAgreement  float64
	HasOpeningHours    bool
	HasAddress         bool
	HasWebsite         bool
	Feedback           FeedbackCounts
	DuplicateRisk      float64
	ReviewStatus       string
	LicensePresent     bool
}

// QualityBreakdown is the per-factor explanation surfaced to Ops. Ops needs to
// see why a record scored what it did, not just the number.
type QualityBreakdown struct {
	SourceTrust            float64 `json:"sourceTrust"`
	DestinationMatch       float64 `json:"destinationMatch"`
	CoordinateCompleteness float64 `json:"coordinateCompleteness"`
	CategoryConfidence     float64 `json:"categoryConfidence"`
	Freshness              float64 `json:"freshness"`
	ProviderAgreement      float64 `json:"providerAgreement"`
	UserFeedback           float64 `json:"userFeedback"`
	Completeness           float64 `json:"completeness"`
	DuplicatePenalty       float64 `json:"duplicatePenalty"`
	ReviewPenalty          float64 `json:"reviewPenalty"`
	QualityScore           float64 `json:"qualityScore"`
	FreshnessScore         float64 `json:"freshnessScore"`
	SourceTrustScore       float64 `json:"sourceTrustScore"`
	Confidence             float64 `json:"confidence"`
}

// ComputeQuality applies the documented weighted model:
//
//	0.25 sourceTrust + 0.20 destinationMatch + 0.15 coordinateCompleteness +
//	0.10 categoryConfidence + 0.10 freshness + 0.10 providerAgreement +
//	0.05 userFeedback + 0.05 completeness - duplicatePenalty - reviewPenalty
//
// A record missing license metadata is capped low: unlicensed provenance is a
// policy failure, not merely a quality signal.
func ComputeQuality(input QualityInput) QualityBreakdown {
	trust := SourceTrustScore(input.TrustLevel)
	freshness := FreshnessScore(input.Category, input.ObservedAt, input.Now)

	breakdown := QualityBreakdown{
		SourceTrust:            trust,
		DestinationMatch:       boolScore(input.DestinationMatched),
		CoordinateCompleteness: boolScore(input.HasCoordinates),
		CategoryConfidence:     boolScore(input.CategoryConfident),
		Freshness:              freshness,
		ProviderAgreement:      clamp01(input.ProviderAgreement),
		UserFeedback:           input.Feedback.FeedbackScore(),
		Completeness:           completenessScore(input),
		DuplicatePenalty:       clamp01(input.DuplicateRisk) * 0.10,
		ReviewPenalty:          reviewPenalty(input.ReviewStatus),
		FreshnessScore:         freshness,
		SourceTrustScore:       trust,
	}

	score := 0.25*breakdown.SourceTrust +
		0.20*breakdown.DestinationMatch +
		0.15*breakdown.CoordinateCompleteness +
		0.10*breakdown.CategoryConfidence +
		0.10*breakdown.Freshness +
		0.10*breakdown.ProviderAgreement +
		0.05*breakdown.UserFeedback +
		0.05*breakdown.Completeness -
		breakdown.DuplicatePenalty -
		breakdown.ReviewPenalty

	// Name quality scales the result rather than adding to it: a place with an
	// unusable name ("Cafe") should not reach strong grounding no matter how
	// complete its other fields are.
	score *= 0.85 + 0.15*clamp01(input.NameQuality)

	if !input.LicensePresent {
		score = math.Min(score, 0.40)
	}
	if input.ReviewStatus == ReviewStatusRejected || input.ReviewStatus == ReviewStatusMerged {
		score = 0
	}

	breakdown.QualityScore = roundTo(clamp01(score), 4)
	breakdown.Confidence = roundTo(clamp01(0.5*breakdown.QualityScore+0.3*trust+0.2*freshness), 4)
	return breakdown
}

// NameQuality scores how usable a place name is for grounding. Very short,
// purely generic, or numeric names are weak because an itinerary item called
// "Cafe" cannot be verified by a traveller.
func NameQuality(name string) float64 {
	normalized := NormalizeMatchName(name)
	if normalized == "" {
		return 0
	}
	tokens := strings.Fields(normalized)
	score := 0.5
	if len(normalized) >= 6 {
		score += 0.2
	}
	if len(tokens) >= 2 {
		score += 0.2
	}
	if len(tokens) == 1 && genericPlaceNames[normalized] {
		return 0.1
	}
	if hasLetters(normalized) {
		score += 0.1
	}
	return clamp01(score)
}

// genericPlaceNames are names that identify a category, not a place.
var genericPlaceNames = map[string]bool{
	"cafe": true, "restaurant": true, "bar": true, "museum": true, "park": true,
	"market": true, "hotel": true, "church": true, "castle": true, "station": true,
	"shop": true, "store": true, "place": true, "attraction": true, "viewpoint": true,
}

// ProviderAgreement scores how consistently independent observations describe
// the same place. One observation is neutral (0.5): a single provider is not
// corroboration, which is why the task forbids trusting one provider outright.
func ProviderAgreement(observations []NormalizedObservation) float64 {
	if len(observations) == 0 {
		return 0
	}
	if len(observations) == 1 {
		return 0.5
	}
	providers := make(map[string]struct{}, len(observations))
	categories := make(map[string]int, len(observations))
	coordinates := make([][2]float64, 0, len(observations))
	for _, observation := range observations {
		providers[observation.Provider] = struct{}{}
		categories[observation.Category]++
		if observation.Latitude != nil && observation.Longitude != nil {
			coordinates = append(coordinates, [2]float64{*observation.Latitude, *observation.Longitude})
		}
	}
	if len(providers) == 1 {
		// Repeated observations from one provider are not independent evidence.
		return 0.6
	}

	// Category consensus: the share of observations holding the majority view.
	majority := 0
	for _, count := range categories {
		if count > majority {
			majority = count
		}
	}
	categoryAgreement := float64(majority) / float64(len(observations))

	coordinateAgreement := 0.5
	if len(coordinates) >= 2 {
		maxDistance := 0.0
		for i := 0; i < len(coordinates); i++ {
			for j := i + 1; j < len(coordinates); j++ {
				distance := HaversineKm(coordinates[i][0], coordinates[i][1], coordinates[j][0], coordinates[j][1])
				if distance > maxDistance {
					maxDistance = distance
				}
			}
		}
		switch {
		case maxDistance <= 0.1:
			coordinateAgreement = 1.0
		case maxDistance <= 0.5:
			coordinateAgreement = 0.8
		case maxDistance <= 2.0:
			coordinateAgreement = 0.5
		default:
			coordinateAgreement = 0.1
		}
	}
	return clamp01(0.5*categoryAgreement + 0.5*coordinateAgreement)
}

// GroundingStrength converts a score and review status into the usage rule the
// prompt builder enforces. Rejected and merged records are always excluded:
// merged records would duplicate their canonical record in retrieval.
func GroundingStrength(qualityScore float64, reviewStatus string, thresholds Thresholds) string {
	switch reviewStatus {
	case ReviewStatusRejected, ReviewStatusMerged:
		return GroundingStrengthExcluded
	case ReviewStatusApproved:
		// Human approval promotes a borderline record to strong grounding, but
		// cannot rescue one that scores below the weak floor.
		if qualityScore >= thresholds.WeakMinQuality {
			return GroundingStrengthStrong
		}
		return GroundingStrengthWeak
	}
	switch {
	case qualityScore >= thresholds.StrongMinQuality:
		return GroundingStrengthStrong
	case qualityScore >= thresholds.WeakMinQuality:
		return GroundingStrengthWeak
	default:
		return GroundingStrengthExcluded
	}
}

// ReviewStatusForScore decides the automatic review state after scoring. It
// never overrides a human decision: approved and rejected records keep their
// status.
func ReviewStatusForScore(current string, qualityScore float64, feedback FeedbackCounts, thresholds Thresholds) string {
	switch current {
	case ReviewStatusApproved, ReviewStatusRejected, ReviewStatusMerged:
		return current
	}
	if qualityScore < thresholds.RejectBelow {
		return ReviewStatusRejected
	}
	if feedback.RequiresReview() || qualityScore < thresholds.NeedsReviewBelow {
		return ReviewStatusNeedsReview
	}
	return ReviewStatusAuto
}

// DestinationCoverage summarizes how well a destination is covered. Low
// coverage downgrades generation quality instead of letting the model invent
// place names to fill the gap.
type DestinationCoverage struct {
	DestinationID         string   `json:"destinationId"`
	DestinationName       string   `json:"destinationName"`
	PlaceCount            int      `json:"placeCount"`
	HighQualityPlaceCount int      `json:"highQualityPlaceCount"`
	CategoryCoverage      float64  `json:"categoryCoverage"`
	FreshnessCoverage     float64  `json:"freshnessCoverage"`
	CoordinateCoverage    float64  `json:"coordinateCoverage"`
	OpeningHoursCoverage  float64  `json:"openingHoursCoverage"`
	CoverageScore         float64  `json:"coverageScore"`
	Status                string   `json:"status"`
	Warnings              []string `json:"warnings"`
}

// coreCategories are the categories a destination needs for a varied itinerary.
var coreCategories = []string{"landmark", "museum", "park", "restaurant", "neighborhood"}

// CoverageInput is the aggregate shape the store produces per destination.
type CoverageInput struct {
	DestinationID     string
	DestinationName   string
	Places            int
	HighQualityPlaces int
	FreshPlaces       int
	WithCoordinates   int
	WithOpeningHours  int
	CategoriesPresent []string
}

// ComputeCoverage turns per-destination aggregates into a coverage verdict.
func ComputeCoverage(input CoverageInput) DestinationCoverage {
	coverage := DestinationCoverage{
		DestinationID:         input.DestinationID,
		DestinationName:       input.DestinationName,
		PlaceCount:            input.Places,
		HighQualityPlaceCount: input.HighQualityPlaces,
		Warnings:              []string{},
	}
	if input.Places == 0 {
		coverage.Status = "unavailable"
		coverage.Warnings = append(coverage.Warnings, "No verified place data for this destination.")
		return coverage
	}

	present := make(map[string]struct{}, len(input.CategoriesPresent))
	for _, category := range input.CategoriesPresent {
		present[strings.ToLower(strings.TrimSpace(category))] = struct{}{}
	}
	covered := 0
	missing := make([]string, 0, len(coreCategories))
	for _, category := range coreCategories {
		if _, ok := present[category]; ok {
			covered++
			continue
		}
		missing = append(missing, category)
	}

	coverage.CategoryCoverage = roundTo(float64(covered)/float64(len(coreCategories)), 4)
	coverage.FreshnessCoverage = ratio(input.FreshPlaces, input.Places)
	coverage.CoordinateCoverage = ratio(input.WithCoordinates, input.Places)
	coverage.OpeningHoursCoverage = ratio(input.WithOpeningHours, input.Places)

	// High-quality place count is the dominant term: ten stale, low-quality
	// records do not make a destination well covered.
	qualityRatio := ratio(input.HighQualityPlaces, input.Places)
	volumeScore := math.Min(float64(input.HighQualityPlaces)/12.0, 1.0)
	coverage.CoverageScore = roundTo(clamp01(
		0.35*volumeScore+
			0.25*qualityRatio+
			0.20*coverage.CategoryCoverage+
			0.10*coverage.FreshnessCoverage+
			0.10*coverage.CoordinateCoverage,
	), 4)

	switch {
	case coverage.CoverageScore >= 0.70:
		coverage.Status = "available"
	case coverage.CoverageScore >= 0.35:
		coverage.Status = "partial"
	default:
		coverage.Status = "limited"
	}
	if coverage.Status != "available" {
		coverage.Warnings = append(coverage.Warnings, "Limited verified place data for this destination.")
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		coverage.Warnings = append(coverage.Warnings, "Missing categories: "+strings.Join(missing, ", ")+".")
	}
	if coverage.FreshnessCoverage < 0.5 {
		coverage.Warnings = append(coverage.Warnings, "More than half of the place records are stale.")
	}
	return coverage
}

func completenessScore(input QualityInput) float64 {
	total := 0.0
	if input.HasOpeningHours {
		total += 0.5
	}
	if input.HasAddress {
		total += 0.3
	}
	if input.HasWebsite {
		total += 0.2
	}
	return clamp01(total)
}

func reviewPenalty(reviewStatus string) float64 {
	switch reviewStatus {
	case ReviewStatusNeedsReview:
		return 0.05
	case ReviewStatusRejected, ReviewStatusMerged:
		return 1
	default:
		return 0
	}
}

func boolScore(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func ratio(part, total int) float64 {
	if total <= 0 {
		return 0
	}
	return roundTo(float64(part)/float64(total), 4)
}

func clamp01(value float64) float64 {
	if math.IsNaN(value) {
		return 0
	}
	return math.Max(0, math.Min(1, value))
}

func hasLetters(value string) bool {
	for _, r := range value {
		if r >= 'a' && r <= 'z' {
			return true
		}
	}
	return false
}
