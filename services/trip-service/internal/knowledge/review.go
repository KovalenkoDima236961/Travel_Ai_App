package knowledge

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Review queries back the Ops AI Knowledge Quality panel. They return
// knowledge records, provenance, and scores. They never return provider
// credentials, raw provider payloads, or private user/trip content.

// PlaceReviewFilters drive the Ops place list.
type PlaceReviewFilters struct {
	DestinationID *uuid.UUID
	ReviewStatus  string
	// Filter is a named preset: low_quality, stale, missing_coordinates,
	// needs_review, duplicates, rejected.
	Filter string
	Limit  int
}

// PlaceReviewRecord is one row in the Ops place list.
type PlaceReviewRecord struct {
	ID                    uuid.UUID  `json:"id"`
	DestinationID         uuid.UUID  `json:"destinationId"`
	DestinationName       string     `json:"destinationName"`
	CanonicalName         string     `json:"canonicalName"`
	Category              string     `json:"category"`
	ReviewStatus          string     `json:"reviewStatus"`
	QualityScore          *float64   `json:"qualityScore,omitempty"`
	FreshnessScore        *float64   `json:"freshnessScore,omitempty"`
	SourceTrustScore      *float64   `json:"sourceTrustScore,omitempty"`
	Confidence            float64    `json:"confidence"`
	GroundingStrength     string     `json:"groundingStrength"`
	HasCoordinates        bool       `json:"hasCoordinates"`
	HasOpeningHours       bool       `json:"hasOpeningHours"`
	SourceKey             string     `json:"sourceKey,omitempty"`
	TrustLevel            string     `json:"trustLevel,omitempty"`
	LicenseName           string     `json:"licenseName,omitempty"`
	Attribution           string     `json:"attribution,omitempty"`
	DuplicateGroupID      *uuid.UUID `json:"duplicateGroupId,omitempty"`
	RejectedReason        string     `json:"rejectedReason,omitempty"`
	LastProviderRefreshAt *time.Time `json:"lastProviderRefreshAt,omitempty"`
	Status                string     `json:"status"`
}

// PlaceDetail adds provenance, observations, and the quality breakdown for the
// Ops detail drawer.
type PlaceDetail struct {
	PlaceReviewRecord
	Aliases             []string            `json:"aliases"`
	Tags                []string            `json:"tags"`
	Latitude            *float64            `json:"latitude,omitempty"`
	Longitude           *float64            `json:"longitude,omitempty"`
	Address             string              `json:"address,omitempty"`
	Website             string              `json:"website,omitempty"`
	SourceURL           string              `json:"sourceUrl,omitempty"`
	OpeningHoursSummary string              `json:"openingHoursSummary,omitempty"`
	ProviderRefs        []string            `json:"providerRefs"`
	Observations        []ObservationRecord `json:"observations"`
}

// ListPlacesForReview returns knowledge records matching an Ops filter.
func (s *Store) ListPlacesForReview(ctx context.Context, filters PlaceReviewFilters) ([]PlaceReviewRecord, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("knowledge store is required")
	}
	limit := filters.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	thresholds := DefaultThresholds()

	rows, err := s.db.Query(ctx, `SELECT p.id, p.destination_id, d.canonical_name, p.canonical_name, p.category,
      p.review_status, p.quality_score, p.freshness_score, p.source_trust_score, p.confidence,
      (p.latitude IS NOT NULL AND p.longitude IS NOT NULL), (p.opening_hours IS NOT NULL),
      COALESCE(s.source_key,''), COALESCE(s.trust_level,''), COALESCE(p.license_name,''),
      COALESCE(s.attribution,''), p.duplicate_group_id, COALESCE(p.rejected_reason,''),
      p.last_provider_refresh_at, p.status
    FROM travel_places p
    JOIN travel_destinations d ON d.id = p.destination_id
    LEFT JOIN travel_knowledge_sources s ON s.id = p.source_id
    WHERE ($1::uuid IS NULL OR p.destination_id = $1)
      AND ($2::text IS NULL OR p.review_status = $2)
      AND CASE $3::text
        WHEN 'low_quality' THEN COALESCE(p.quality_score, 0) < $4
        WHEN 'stale' THEN p.last_provider_refresh_at IS NULL
          OR p.last_provider_refresh_at < NOW() - make_interval(days => $5)
        WHEN 'missing_coordinates' THEN p.latitude IS NULL OR p.longitude IS NULL
        WHEN 'needs_review' THEN p.review_status = 'needs_review'
        WHEN 'duplicates' THEN p.duplicate_group_id IS NOT NULL
        WHEN 'rejected' THEN p.review_status = 'rejected'
        ELSE TRUE
      END
    ORDER BY p.quality_score ASC NULLS FIRST, p.canonical_name, p.id
    LIMIT $6`,
		filters.DestinationID, nullText(filters.ReviewStatus), nullText(filters.Filter),
		thresholds.StrongMinQuality, thresholds.StaleAfterDays, limit)
	if err != nil {
		return nil, fmt.Errorf("list places for review: %w", err)
	}
	defer rows.Close()

	places := make([]PlaceReviewRecord, 0, limit)
	for rows.Next() {
		record, scanErr := scanPlaceReviewRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		places = append(places, record)
	}
	return places, rows.Err()
}

// GetPlaceDetail returns one record with its provenance and the provider
// observations that produced it.
func (s *Store) GetPlaceDetail(ctx context.Context, placeID uuid.UUID) (PlaceDetail, error) {
	if s == nil || s.db == nil {
		return PlaceDetail{}, fmt.Errorf("knowledge store is required")
	}
	var (
		detail           PlaceDetail
		aliasesJSON      []byte
		tagsJSON         []byte
		refsJSON         []byte
		openingHoursJSON []byte
	)
	err := s.db.QueryRow(ctx, `SELECT p.id, p.destination_id, d.canonical_name, p.canonical_name, p.category,
      p.review_status, p.quality_score, p.freshness_score, p.source_trust_score, p.confidence,
      (p.latitude IS NOT NULL AND p.longitude IS NOT NULL), (p.opening_hours IS NOT NULL),
      COALESCE(s.source_key,''), COALESCE(s.trust_level,''), COALESCE(p.license_name,''),
      COALESCE(s.attribution,''), p.duplicate_group_id, COALESCE(p.rejected_reason,''),
      p.last_provider_refresh_at, p.status,
      p.aliases, p.tags, p.provider_refs, p.opening_hours,
      p.latitude, p.longitude, COALESCE(p.address,''), COALESCE(p.website,''), COALESCE(p.source_url,'')
    FROM travel_places p
    JOIN travel_destinations d ON d.id = p.destination_id
    LEFT JOIN travel_knowledge_sources s ON s.id = p.source_id
    WHERE p.id = $1`, placeID).Scan(
		&detail.ID, &detail.DestinationID, &detail.DestinationName, &detail.CanonicalName, &detail.Category,
		&detail.ReviewStatus, &detail.QualityScore, &detail.FreshnessScore, &detail.SourceTrustScore,
		&detail.Confidence, &detail.HasCoordinates, &detail.HasOpeningHours, &detail.SourceKey,
		&detail.TrustLevel, &detail.LicenseName, &detail.Attribution, &detail.DuplicateGroupID,
		&detail.RejectedReason, &detail.LastProviderRefreshAt, &detail.Status,
		&aliasesJSON, &tagsJSON, &refsJSON, &openingHoursJSON,
		&detail.Latitude, &detail.Longitude, &detail.Address, &detail.Website, &detail.SourceURL)
	if err != nil {
		return PlaceDetail{}, fmt.Errorf("load place %s: %w", placeID, err)
	}

	detail.Aliases = decodeStringArray(aliasesJSON)
	detail.Tags = decodeStringArray(tagsJSON)
	detail.ProviderRefs = decodeProviderRefs(refsJSON)
	detail.OpeningHoursSummary = SummarizeOpeningHours(openingHoursJSON)
	detail.GroundingStrength = GroundingStrength(derefFloat(detail.QualityScore), detail.ReviewStatus, DefaultThresholds())

	observations, err := s.observationsForPlace(ctx, placeID)
	if err != nil {
		return PlaceDetail{}, err
	}
	detail.Observations = observations
	return detail, nil
}

func (s *Store) observationsForPlace(ctx context.Context, placeID uuid.UUID) ([]ObservationRecord, error) {
	rows, err := s.db.Query(ctx, `SELECT id, provider, provider_place_id, destination_id, raw_name, normalized_name,
      COALESCE(category,''), latitude, longitude, COALESCE(address,''), COALESCE(website,''),
      COALESCE(license_name,''), COALESCE(attribution,''), COALESCE(source_url,''), observed_at,
      quality_score, confidence, matched_place_id, match_status
    FROM travel_provider_place_observations
    WHERE matched_place_id = $1
    ORDER BY observed_at DESC, id`, placeID)
	if err != nil {
		return nil, fmt.Errorf("list observations for place %s: %w", placeID, err)
	}
	defer rows.Close()

	observations := make([]ObservationRecord, 0, 4)
	for rows.Next() {
		var record ObservationRecord
		if err := rows.Scan(&record.ID, &record.Provider, &record.ProviderPlaceID, &record.DestinationID,
			&record.RawName, &record.NormalizedName, &record.Category, &record.Latitude, &record.Longitude,
			&record.Address, &record.Website, &record.LicenseName, &record.Attribution, &record.SourceURL,
			&record.ObservedAt, &record.QualityScore, &record.Confidence, &record.MatchedPlaceID,
			&record.MatchStatus); err != nil {
			return nil, fmt.Errorf("scan place observation: %w", err)
		}
		observations = append(observations, record)
	}
	return observations, rows.Err()
}

// rowScanner is satisfied by pgx.Rows and pgx.Row.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanPlaceReviewRecord(row rowScanner) (PlaceReviewRecord, error) {
	var record PlaceReviewRecord
	if err := row.Scan(&record.ID, &record.DestinationID, &record.DestinationName, &record.CanonicalName,
		&record.Category, &record.ReviewStatus, &record.QualityScore, &record.FreshnessScore,
		&record.SourceTrustScore, &record.Confidence, &record.HasCoordinates, &record.HasOpeningHours,
		&record.SourceKey, &record.TrustLevel, &record.LicenseName, &record.Attribution,
		&record.DuplicateGroupID, &record.RejectedReason, &record.LastProviderRefreshAt,
		&record.Status); err != nil {
		return PlaceReviewRecord{}, fmt.Errorf("scan place review record: %w", err)
	}
	record.GroundingStrength = GroundingStrength(derefFloat(record.QualityScore), record.ReviewStatus, DefaultThresholds())
	return record, nil
}

func derefFloat(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}
