package knowledge

import (
	"math"
	"sort"
	"strings"
)

// Matching decides whether a provider observation describes a place already in
// the store. It is deterministic scoring rather than a similarity model,
// because an incorrect automatic merge is expensive to undo and the thresholds
// need to be explainable to a reviewer.

// MatchDecision describes the outcome for one observation/place pair.
type MatchDecision struct {
	PlaceID    string   `json:"placeId"`
	Confidence float64  `json:"confidence"`
	Action     string   `json:"action"`
	Reasons    []string `json:"reasons"`
}

// Match actions returned by DecideMatch.
const (
	MatchActionAutoMatch   = "auto_match"
	MatchActionNeedsReview = "needs_review"
	MatchActionNoMatch     = "no_match"
)

// MatchCandidate is an existing knowledge record an observation might describe.
type MatchCandidate struct {
	PlaceID       string
	CanonicalName string
	Aliases       []string
	Category      string
	Latitude      *float64
	Longitude     *float64
	ProviderRefs  []string
	DestinationID string
	ReviewStatus  string
}

// MatchScore is the deterministic 0..1 similarity between an observation and a
// candidate place, with the reasons that produced it.
func MatchScore(observation NormalizedObservation, candidate MatchCandidate) (float64, []string) {
	reasons := make([]string, 0, 5)

	// A provider reference is an identity claim by the provider itself and is
	// treated as conclusive.
	providerRef := observation.Provider + ":" + observation.ProviderPlaceID
	for _, ref := range candidate.ProviderRefs {
		if strings.EqualFold(strings.TrimSpace(ref), providerRef) {
			return 1.0, []string{"provider reference match"}
		}
	}

	observationNames := append([]string{observation.NormalizedName}, normalizeAll(observation.Aliases)...)
	candidateNames := append([]string{NormalizeMatchName(candidate.CanonicalName)}, normalizeAll(candidate.Aliases)...)

	nameScore := 0.0
	switch {
	case anyExactMatch(observationNames, candidateNames):
		nameScore = 1.0
		reasons = append(reasons, "exact normalized name or alias match")
	default:
		nameScore = bestTokenOverlap(observationNames, candidateNames)
		if nameScore > 0 {
			reasons = append(reasons, "partial name token overlap")
		}
	}

	// Coordinate proximity. Unknown coordinates are neutral (0.5) rather than
	// disqualifying, because many valid records lack them.
	distanceScore := 0.5
	var distanceKm float64
	haveDistance := false
	if observation.Latitude != nil && observation.Longitude != nil &&
		candidate.Latitude != nil && candidate.Longitude != nil {
		haveDistance = true
		distanceKm = HaversineKm(*observation.Latitude, *observation.Longitude, *candidate.Latitude, *candidate.Longitude)
		switch {
		case distanceKm <= 0.05:
			distanceScore = 1.0
			reasons = append(reasons, "coordinates within 50m")
		case distanceKm <= 0.25:
			distanceScore = 0.9
			reasons = append(reasons, "coordinates within 250m")
		case distanceKm <= 1.0:
			distanceScore = 0.6
			reasons = append(reasons, "coordinates within 1km")
		case distanceKm <= 5.0:
			distanceScore = 0.2
		default:
			distanceScore = 0.0
			reasons = append(reasons, "coordinates far apart")
		}
	}

	categoryScore := 0.5
	if observation.Category != "" && candidate.Category != "" {
		if observation.Category == candidate.Category {
			categoryScore = 1.0
			reasons = append(reasons, "category match")
		} else if compatibleCategories(observation.Category, candidate.Category) {
			categoryScore = 0.7
			reasons = append(reasons, "compatible category")
		} else {
			categoryScore = 0.1
		}
	}

	score := 0.55*nameScore + 0.25*distanceScore + 0.20*categoryScore

	// Same name but demonstrably different location is a name collision
	// ("Old Town" in two cities), not a match.
	if haveDistance && distanceKm > 5.0 {
		score = math.Min(score, 0.45)
		reasons = append(reasons, "distance veto applied")
	}
	// Weak name evidence cannot be rescued by proximity alone: two different
	// cafes on one street are not the same place.
	if nameScore < 0.4 {
		score = math.Min(score, 0.55)
	}
	return roundTo(clamp01(score), 4), reasons
}

// DecideMatch applies the documented thresholds: >=0.90 auto-match,
// 0.70-0.89 review, <0.70 no match. When several candidates are plausible the
// observation goes to review rather than picking the top score, since ambiguity
// is exactly the case a human should resolve.
func DecideMatch(observation NormalizedObservation, candidates []MatchCandidate, thresholds Thresholds) MatchDecision {
	if len(candidates) == 0 {
		return MatchDecision{Action: MatchActionNoMatch, Reasons: []string{"no candidate places in destination"}}
	}

	scored := make([]MatchDecision, 0, len(candidates))
	for _, candidate := range candidates {
		score, reasons := MatchScore(observation, candidate)
		scored = append(scored, MatchDecision{PlaceID: candidate.PlaceID, Confidence: score, Reasons: reasons})
	}
	// Deterministic ordering: highest score first, then place ID for stability.
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Confidence != scored[j].Confidence {
			return scored[i].Confidence > scored[j].Confidence
		}
		return scored[i].PlaceID < scored[j].PlaceID
	})

	best := scored[0]
	if best.Confidence < thresholds.ReviewMatchConfidence {
		return MatchDecision{Action: MatchActionNoMatch, Confidence: best.Confidence, Reasons: []string{"no candidate above review threshold"}}
	}

	if len(scored) > 1 && scored[1].Confidence >= thresholds.ReviewMatchConfidence {
		best.Action = MatchActionNeedsReview
		best.Reasons = append(best.Reasons, "multiple plausible candidates")
		return best
	}
	if best.Confidence >= thresholds.AutoMatchConfidence {
		best.Action = MatchActionAutoMatch
		return best
	}
	best.Action = MatchActionNeedsReview
	return best
}

// DuplicatePair is a suggested duplicate relationship between two stored
// places.
type DuplicatePair struct {
	PlaceID      string   `json:"placeId"`
	OtherPlaceID string   `json:"otherPlaceId"`
	Confidence   float64  `json:"confidence"`
	Reasons      []string `json:"reasons"`
}

// DetectDuplicates finds probable duplicate pairs within one destination.
// Results are sorted for deterministic group creation. Only pairs at or above
// the review threshold are returned; merging still requires Ops confirmation.
func DetectDuplicates(places []MatchCandidate, thresholds Thresholds) []DuplicatePair {
	pairs := make([]DuplicatePair, 0)
	for i := 0; i < len(places); i++ {
		for j := i + 1; j < len(places); j++ {
			left, right := places[i], places[j]
			if left.DestinationID != right.DestinationID {
				continue
			}
			// A merged or rejected record is already resolved.
			if isResolvedStatus(left.ReviewStatus) || isResolvedStatus(right.ReviewStatus) {
				continue
			}
			observation := NormalizedObservation{
				NormalizedName: NormalizeMatchName(left.CanonicalName),
				Aliases:        left.Aliases,
				Category:       left.Category,
				Latitude:       left.Latitude,
				Longitude:      left.Longitude,
			}
			score, reasons := MatchScore(observation, right)
			if score < thresholds.ReviewMatchConfidence {
				continue
			}
			pairID, otherID := left.PlaceID, right.PlaceID
			if pairID > otherID {
				pairID, otherID = otherID, pairID
			}
			pairs = append(pairs, DuplicatePair{
				PlaceID:      pairID,
				OtherPlaceID: otherID,
				Confidence:   score,
				Reasons:      reasons,
			})
		}
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].Confidence != pairs[j].Confidence {
			return pairs[i].Confidence > pairs[j].Confidence
		}
		if pairs[i].PlaceID != pairs[j].PlaceID {
			return pairs[i].PlaceID < pairs[j].PlaceID
		}
		return pairs[i].OtherPlaceID < pairs[j].OtherPlaceID
	})
	return pairs
}

// MergeCandidate is one record participating in a merge.
type MergeCandidate struct {
	PlaceID      string
	TrustLevel   string
	QualityScore float64
	ObservedAt   int64 // unix seconds; freshest wins field-level ties
	Category     string
	Latitude     *float64
	Longitude    *float64
	Address      string
	Website      string
	OpeningHours []byte
	Aliases      []string
	Tags         []string
	ProviderRefs []string
}

// MergeResolution is the field-by-field outcome of merging duplicates into a
// canonical record.
type MergeResolution struct {
	CanonicalPlaceID string            `json:"canonicalPlaceId"`
	MergedPlaceIDs   []string          `json:"mergedPlaceIds"`
	Category         string            `json:"category"`
	Latitude         *float64          `json:"latitude,omitempty"`
	Longitude        *float64          `json:"longitude,omitempty"`
	Address          string            `json:"address,omitempty"`
	Website          string            `json:"website,omitempty"`
	OpeningHours     []byte            `json:"-"`
	Aliases          []string          `json:"aliases"`
	Tags             []string          `json:"tags"`
	ProviderRefs     []string          `json:"providerRefs"`
	FieldSources     map[string]string `json:"fieldSources"`
}

// ResolveMerge picks the canonical record and the best value for each field.
// Coordinates and category follow the highest-trust record; opening hours
// follow the freshest one, since hours change more often than location.
func ResolveMerge(canonicalPlaceID string, candidates []MergeCandidate) MergeResolution {
	resolution := MergeResolution{
		CanonicalPlaceID: canonicalPlaceID,
		MergedPlaceIDs:   []string{},
		Aliases:          []string{},
		Tags:             []string{},
		ProviderRefs:     []string{},
		FieldSources:     map[string]string{},
	}
	if len(candidates) == 0 {
		return resolution
	}

	ordered := make([]MergeCandidate, len(candidates))
	copy(ordered, candidates)
	// Highest trust, then highest quality, then freshest, then stable by ID.
	sort.SliceStable(ordered, func(i, j int) bool {
		leftTrust, rightTrust := SourceTrustScore(ordered[i].TrustLevel), SourceTrustScore(ordered[j].TrustLevel)
		if leftTrust != rightTrust {
			return leftTrust > rightTrust
		}
		if ordered[i].QualityScore != ordered[j].QualityScore {
			return ordered[i].QualityScore > ordered[j].QualityScore
		}
		if ordered[i].ObservedAt != ordered[j].ObservedAt {
			return ordered[i].ObservedAt > ordered[j].ObservedAt
		}
		return ordered[i].PlaceID < ordered[j].PlaceID
	})

	primary := ordered[0]
	resolution.Category = primary.Category
	resolution.FieldSources["category"] = primary.PlaceID

	for _, candidate := range ordered {
		if candidate.PlaceID != canonicalPlaceID {
			resolution.MergedPlaceIDs = append(resolution.MergedPlaceIDs, candidate.PlaceID)
		}
		resolution.Aliases = appendUnique(resolution.Aliases, candidate.Aliases...)
		resolution.Tags = appendUnique(resolution.Tags, candidate.Tags...)
		resolution.ProviderRefs = appendUnique(resolution.ProviderRefs, candidate.ProviderRefs...)

		if resolution.Latitude == nil && candidate.Latitude != nil && candidate.Longitude != nil {
			resolution.Latitude, resolution.Longitude = candidate.Latitude, candidate.Longitude
			resolution.FieldSources["coordinates"] = candidate.PlaceID
		}
		if resolution.Address == "" && strings.TrimSpace(candidate.Address) != "" {
			resolution.Address = candidate.Address
			resolution.FieldSources["address"] = candidate.PlaceID
		}
		if resolution.Website == "" && strings.TrimSpace(candidate.Website) != "" {
			resolution.Website = candidate.Website
			resolution.FieldSources["website"] = candidate.PlaceID
		}
	}

	// Opening hours are the one field where freshness beats trust.
	freshest := ordered[0]
	for _, candidate := range ordered {
		if len(candidate.OpeningHours) == 0 {
			continue
		}
		if len(freshest.OpeningHours) == 0 || candidate.ObservedAt > freshest.ObservedAt {
			freshest = candidate
		}
	}
	if len(freshest.OpeningHours) > 0 {
		resolution.OpeningHours = freshest.OpeningHours
		resolution.FieldSources["openingHours"] = freshest.PlaceID
	}

	sort.Strings(resolution.MergedPlaceIDs)
	return resolution
}

// HaversineKm returns great-circle distance in kilometres.
func HaversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0
	dLat := degreesToRadians(lat2 - lat1)
	dLng := degreesToRadians(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degreesToRadians(lat1))*math.Cos(degreesToRadians(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	return earthRadiusKm * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}

func degreesToRadians(degrees float64) float64 { return degrees * math.Pi / 180 }

// compatibleCategories lists category pairs that commonly describe the same
// physical place across providers (a palace tagged both landmark and museum).
func compatibleCategories(left, right string) bool {
	groups := [][]string{
		{"landmark", "museum", "viewpoint"},
		{"park", "nature", "viewpoint"},
		{"restaurant", "cafe", "market"},
		{"neighborhood", "market"},
		{"activity", "nature"},
	}
	for _, group := range groups {
		leftFound, rightFound := false, false
		for _, category := range group {
			if category == left {
				leftFound = true
			}
			if category == right {
				rightFound = true
			}
		}
		if leftFound && rightFound {
			return true
		}
	}
	return false
}

// bestTokenOverlap returns the highest Jaccard-style token overlap across all
// name/alias combinations.
func bestTokenOverlap(left, right []string) float64 {
	best := 0.0
	for _, leftName := range left {
		for _, rightName := range right {
			if score := tokenOverlap(leftName, rightName); score > best {
				best = score
			}
		}
	}
	return best
}

func tokenOverlap(left, right string) float64 {
	leftTokens, rightTokens := strings.Fields(left), strings.Fields(right)
	if len(leftTokens) == 0 || len(rightTokens) == 0 {
		return 0
	}
	leftSet := make(map[string]struct{}, len(leftTokens))
	for _, token := range leftTokens {
		leftSet[token] = struct{}{}
	}
	shared := 0
	rightSet := make(map[string]struct{}, len(rightTokens))
	for _, token := range rightTokens {
		if _, seen := rightSet[token]; seen {
			continue
		}
		rightSet[token] = struct{}{}
		if _, ok := leftSet[token]; ok {
			shared++
		}
	}
	union := len(leftSet) + len(rightSet) - shared
	if union == 0 {
		return 0
	}
	return float64(shared) / float64(union)
}

func anyExactMatch(left, right []string) bool {
	rightSet := make(map[string]struct{}, len(right))
	for _, value := range right {
		if value != "" {
			rightSet[value] = struct{}{}
		}
	}
	for _, value := range left {
		if value == "" {
			continue
		}
		if _, ok := rightSet[value]; ok {
			return true
		}
	}
	return false
}

func normalizeAll(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := NormalizeMatchName(value)
		if normalized != "" {
			result = append(result, normalized)
		}
	}
	return result
}

func isResolvedStatus(status string) bool {
	return status == ReviewStatusMerged || status == ReviewStatusRejected
}

func appendUnique(target []string, values ...string) []string {
	seen := make(map[string]struct{}, len(target)+len(values))
	for _, value := range target {
		seen[strings.ToLower(value)] = struct{}{}
	}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		target = append(target, trimmed)
	}
	return target
}
