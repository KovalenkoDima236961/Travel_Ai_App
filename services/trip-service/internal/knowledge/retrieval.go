package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge/provider"
)

// Retrieval is the boundary where data quality becomes AI behaviour. Every
// record that reaches a prompt passes through here, so the exclusion rules for
// rejected, merged, and low-quality records are enforced in one place rather
// than trusted to each caller.

// GroundingPlace is one place as sent to the AI Planning Service. Field names
// match the existing GroundingPlace pydantic schema; the quality fields are
// additive so older consumers keep working.
type GroundingPlace struct {
	ID                     string   `json:"id"`
	CanonicalName          string   `json:"canonicalName"`
	Category               string   `json:"category"`
	Subcategory            string   `json:"subcategory,omitempty"`
	Tags                   []string `json:"tags"`
	Neighborhood           string   `json:"neighborhood,omitempty"`
	TypicalDurationMinutes *int     `json:"typicalDurationMinutes,omitempty"`
	Latitude               *float64 `json:"lat,omitempty"`
	Longitude              *float64 `json:"lng,omitempty"`
	Outdoor                *bool    `json:"outdoor,omitempty"`
	RainFriendly           *bool    `json:"rainFriendly,omitempty"`
	FamilyFriendly         *bool    `json:"familyFriendly,omitempty"`
	PriceLevel             string   `json:"priceLevel,omitempty"`
	OpeningHoursSummary    string   `json:"openingHoursSummary,omitempty"`
	BestTimeOfDay          []string `json:"bestTimeOfDay,omitempty"`
	QualityScore           float64  `json:"qualityScore"`
	FreshnessScore         float64  `json:"freshnessScore"`
	Confidence             float64  `json:"confidence"`
	ReviewStatus           string   `json:"reviewStatus"`
	GroundingStrength      string   `json:"groundingStrength"`
	SourceKey              string   `json:"sourceKey,omitempty"`
	SourceTrustLevel       string   `json:"sourceTrustLevel,omitempty"`
	SourceURL              string   `json:"sourceUrl,omitempty"`
	Attribution            string   `json:"attribution,omitempty"`
	Warnings               []string `json:"warnings,omitempty"`
}

// GroundingResult is the retrieval payload for one destination.
type GroundingResult struct {
	Status            string              `json:"status"`
	DestinationID     string              `json:"destinationId,omitempty"`
	DestinationName   string              `json:"destinationName,omitempty"`
	Places            []GroundingPlace    `json:"places"`
	StrongCount       int                 `json:"strongCount"`
	WeakCount         int                 `json:"weakCount"`
	ExcludedCount     int                 `json:"excludedCount"`
	Coverage          DestinationCoverage `json:"coverage"`
	RetrievalWarnings []string            `json:"retrievalWarnings"`
	Attributions      []string            `json:"attributions"`
}

// GroundingQuery parameterises retrieval.
type GroundingQuery struct {
	DestinationName string
	CountryCode     string
	Categories      []string
	Limit           int
	// IncludeWeak controls whether medium-quality records are offered at all.
	// They are always marked so the prompt can require review of items built
	// from them.
	IncludeWeak bool
	Thresholds  Thresholds
}

const defaultGroundingLimit = 20

// RetrieveGrounding returns quality-filtered grounding context for one
// destination.
//
// Rejected and merged records are excluded in SQL, not in Go, so no caller can
// bypass the rule by using a different code path. Strong records are ordered
// first and the result is capped, because prompt space is finite and the
// highest-quality evidence should occupy it.
func (s *Store) RetrieveGrounding(ctx context.Context, query GroundingQuery) (GroundingResult, error) {
	if s == nil || s.db == nil {
		return GroundingResult{}, fmt.Errorf("knowledge store is required")
	}
	thresholds := query.Thresholds
	if thresholds.StrongMinQuality == 0 && thresholds.WeakMinQuality == 0 {
		thresholds = DefaultThresholds()
	}
	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = defaultGroundingLimit
	}

	result := GroundingResult{
		Status:            "unavailable",
		Places:            []GroundingPlace{},
		RetrievalWarnings: []string{},
		Attributions:      []string{},
	}

	var (
		destinationID   uuid.UUID
		destinationName string
	)
	err := s.db.QueryRow(ctx, `SELECT id, canonical_name FROM travel_destinations
    WHERE (lower(canonical_name) = lower($1) OR aliases @> to_jsonb($1::text))
      AND ($2::text IS NULL OR country_code = $2)
    ORDER BY confidence DESC, canonical_name
    LIMIT 1`, strings.TrimSpace(query.DestinationName), nullText(query.CountryCode)).Scan(&destinationID, &destinationName)
	if err != nil {
		// An unknown destination is not an error: generation continues with
		// generic activities rather than failing or inventing places.
		result.RetrievalWarnings = append(result.RetrievalWarnings, "No knowledge record for this destination.")
		result.Coverage = ComputeCoverage(CoverageInput{DestinationName: query.DestinationName})
		return result, nil
	}
	result.DestinationID = destinationID.String()
	result.DestinationName = destinationName

	// The minimum quality accepted at all. Approved records are admitted at the
	// weak floor because a human has vouched for them.
	minQuality := thresholds.WeakMinQuality
	if !query.IncludeWeak {
		minQuality = thresholds.StrongMinQuality
	}

	rows, err := s.db.Query(ctx, `SELECT p.id, p.canonical_name, p.category, COALESCE(p.subcategory,''), p.tags,
      COALESCE(p.neighborhood,''), p.typical_duration_minutes, p.latitude, p.longitude,
      p.outdoor, p.rain_friendly, p.family_friendly, COALESCE(p.price_level,''), p.opening_hours,
      p.best_time_of_day, COALESCE(p.quality_score, 0), COALESCE(p.freshness_score, 0), p.confidence,
      p.review_status, COALESCE(s.source_key,''), COALESCE(s.trust_level,'unknown'),
      COALESCE(p.source_url,''), COALESCE(s.attribution,'')
    FROM travel_places p
    LEFT JOIN travel_knowledge_sources s ON s.id = p.source_id
    WHERE p.destination_id = $1
      AND p.status = 'active'
      -- Rejected and merged records never reach a prompt.
      AND p.review_status NOT IN ('rejected', 'merged')
      AND p.merged_into_place_id IS NULL
      AND (p.review_status = 'approved' OR COALESCE(p.quality_score, 0) >= $2)
      AND ($3::text[] IS NULL OR p.category = ANY($3))
    ORDER BY (p.review_status = 'approved') DESC, p.quality_score DESC NULLS LAST,
      p.confidence DESC, p.canonical_name
    LIMIT $4`, destinationID, minQuality, categoryFilter(query.Categories), limit)
	if err != nil {
		return GroundingResult{}, fmt.Errorf("retrieve grounding places: %w", err)
	}
	defer rows.Close()

	attributions := map[string]struct{}{}
	for rows.Next() {
		var (
			place            GroundingPlace
			placeID          uuid.UUID
			tagsJSON         []byte
			openingHoursJSON []byte
			bestTimeJSON     []byte
			attribution      string
		)
		if err := rows.Scan(&placeID, &place.CanonicalName, &place.Category, &place.Subcategory, &tagsJSON,
			&place.Neighborhood, &place.TypicalDurationMinutes, &place.Latitude, &place.Longitude,
			&place.Outdoor, &place.RainFriendly, &place.FamilyFriendly, &place.PriceLevel, &openingHoursJSON,
			&bestTimeJSON, &place.QualityScore, &place.FreshnessScore, &place.Confidence, &place.ReviewStatus,
			&place.SourceKey, &place.SourceTrustLevel, &place.SourceURL, &attribution); err != nil {
			return GroundingResult{}, fmt.Errorf("scan grounding place: %w", err)
		}
		place.ID = placeID.String()
		place.Tags = decodeStringArray(tagsJSON)
		place.BestTimeOfDay = decodeStringArray(bestTimeJSON)
		place.OpeningHoursSummary = SummarizeOpeningHours(openingHoursJSON)
		place.GroundingStrength = GroundingStrength(place.QualityScore, place.ReviewStatus, thresholds)
		place.Warnings = groundingWarnings(place)
		place.Attribution = attribution

		if place.GroundingStrength == GroundingStrengthExcluded {
			result.ExcludedCount++
			continue
		}
		if place.GroundingStrength == GroundingStrengthWeak {
			if !query.IncludeWeak {
				result.ExcludedCount++
				continue
			}
			result.WeakCount++
		} else {
			result.StrongCount++
		}
		if attribution != "" {
			attributions[attribution] = struct{}{}
		}
		result.Places = append(result.Places, place)
	}
	if err := rows.Err(); err != nil {
		return GroundingResult{}, err
	}

	for attribution := range attributions {
		result.Attributions = append(result.Attributions, attribution)
	}
	sort.Strings(result.Attributions)

	coverage, err := s.DestinationCoverage(ctx, destinationID, thresholds)
	if err != nil {
		return GroundingResult{}, err
	}
	coverage.DestinationName = destinationName
	result.Coverage = coverage

	switch {
	case result.StrongCount == 0 && result.WeakCount == 0:
		result.Status = "unavailable"
		result.RetrievalWarnings = append(result.RetrievalWarnings, "No verified places passed the quality threshold.")
	case result.StrongCount == 0 || coverage.Status != "available":
		result.Status = "partial"
		result.RetrievalWarnings = append(result.RetrievalWarnings, "Limited verified place data for this destination.")
	default:
		result.Status = "available"
	}
	if result.WeakCount > 0 {
		result.RetrievalWarnings = append(result.RetrievalWarnings,
			fmt.Sprintf("%d place record(s) are weak grounding and need review.", result.WeakCount))
	}
	return result, nil
}

// DestinationCoverage computes the coverage summary used by generation quality
// metadata and by the Ops dashboard.
func (s *Store) DestinationCoverage(ctx context.Context, destinationID uuid.UUID, thresholds Thresholds) (DestinationCoverage, error) {
	if s == nil || s.db == nil {
		return DestinationCoverage{}, fmt.Errorf("knowledge store is required")
	}
	if thresholds.StrongMinQuality == 0 {
		thresholds = DefaultThresholds()
	}
	var (
		input      CoverageInput
		categories []string
	)
	err := s.db.QueryRow(ctx, `SELECT
      count(*),
      count(*) FILTER (WHERE quality_score >= $2),
      count(*) FILTER (WHERE last_provider_refresh_at IS NOT NULL
        AND last_provider_refresh_at >= NOW() - make_interval(days => $3)),
      count(*) FILTER (WHERE latitude IS NOT NULL AND longitude IS NOT NULL),
      count(*) FILTER (WHERE opening_hours IS NOT NULL),
      COALESCE(array_agg(DISTINCT category), ARRAY[]::text[])
    FROM travel_places
    WHERE destination_id = $1 AND status = 'active' AND review_status NOT IN ('rejected','merged')`,
		destinationID, thresholds.StrongMinQuality, thresholds.StaleAfterDays).Scan(
		&input.Places, &input.HighQualityPlaces, &input.FreshPlaces, &input.WithCoordinates,
		&input.WithOpeningHours, &categories)
	if err != nil {
		return DestinationCoverage{}, fmt.Errorf("compute destination coverage: %w", err)
	}
	input.DestinationID = destinationID.String()
	input.CategoriesPresent = categories
	return ComputeCoverage(input), nil
}

// SummarizeOpeningHours renders stored opening hours as a short human-readable
// hint. It is a planning aid, never an availability guarantee, and it is capped
// so a prompt cannot be flooded by one place's schedule.
func SummarizeOpeningHours(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var periods []provider.OpeningHoursPeriod
	if err := json.Unmarshal(raw, &periods); err != nil || len(periods) == 0 {
		return ""
	}
	weekdayNames := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	// Group weekdays that share the same window, which is how most places are
	// actually described.
	byWindow := map[string][]int{}
	order := make([]string, 0, len(periods))
	for _, period := range periods {
		if period.Weekday < 0 || period.Weekday > 6 {
			continue
		}
		window := period.Opens + "-" + period.Closes
		if _, seen := byWindow[window]; !seen {
			order = append(order, window)
		}
		byWindow[window] = append(byWindow[window], period.Weekday)
	}
	if len(order) == 0 {
		return ""
	}
	sort.Strings(order)

	parts := make([]string, 0, len(order))
	for _, window := range order {
		days := byWindow[window]
		sort.Ints(days)
		names := make([]string, 0, len(days))
		for _, day := range days {
			names = append(names, weekdayNames[day])
		}
		if len(days) == 7 {
			parts = append(parts, "Daily "+window)
			continue
		}
		parts = append(parts, strings.Join(names, ",")+" "+window)
	}
	summary := strings.Join(parts, "; ")
	if len(summary) > 160 {
		summary = summary[:157] + "..."
	}
	return summary
}

// groundingWarnings tells the prompt builder what is uncertain about a record,
// so weak evidence produces a review flag rather than a confident claim.
func groundingWarnings(place GroundingPlace) []string {
	warnings := make([]string, 0, 3)
	if place.Latitude == nil || place.Longitude == nil {
		warnings = append(warnings, "Coordinates unknown; travel time is an estimate.")
	}
	if place.OpeningHoursSummary == "" {
		warnings = append(warnings, "Opening hours unknown; confirm before visiting.")
	}
	if place.ReviewStatus == ReviewStatusNeedsReview {
		warnings = append(warnings, "Record is pending review.")
	}
	if place.FreshnessScore < 0.35 {
		warnings = append(warnings, "Record has not been refreshed recently.")
	}
	return warnings
}

func categoryFilter(categories []string) any {
	if len(categories) == 0 {
		return nil
	}
	filtered := make([]string, 0, len(categories))
	for _, category := range categories {
		trimmed := strings.ToLower(strings.TrimSpace(category))
		if trimmed == "" {
			continue
		}
		if _, ok := allowedCategories[trimmed]; ok {
			filtered = append(filtered, trimmed)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}
