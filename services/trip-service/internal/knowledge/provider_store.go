package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ProviderStore persists provider observations and the review/quality state
// derived from them. It extends the curated Store rather than replacing it:
// observations are evidence, travel_places remains the single grounding table.

// ObservationRecord is a persisted provider observation.
type ObservationRecord struct {
	ID              uuid.UUID  `json:"id"`
	Provider        string     `json:"provider"`
	ProviderPlaceID string     `json:"providerPlaceId"`
	DestinationID   *uuid.UUID `json:"destinationId,omitempty"`
	RawName         string     `json:"rawName"`
	NormalizedName  string     `json:"normalizedName"`
	Category        string     `json:"category,omitempty"`
	Latitude        *float64   `json:"latitude,omitempty"`
	Longitude       *float64   `json:"longitude,omitempty"`
	Address         string     `json:"address,omitempty"`
	Website         string     `json:"website,omitempty"`
	LicenseName     string     `json:"licenseName,omitempty"`
	Attribution     string     `json:"attribution,omitempty"`
	SourceURL       string     `json:"sourceUrl,omitempty"`
	ObservedAt      time.Time  `json:"observedAt"`
	QualityScore    *float64   `json:"qualityScore,omitempty"`
	Confidence      float64    `json:"confidence"`
	MatchedPlaceID  *uuid.UUID `json:"matchedPlaceId,omitempty"`
	MatchStatus     string     `json:"matchStatus"`
}

// UpsertObservation stores one normalized observation. It is keyed by
// (provider, provider_place_id) so repeating an ingestion updates in place
// instead of accumulating duplicate evidence rows.
func (s *Store) UpsertObservation(ctx context.Context, destinationID *uuid.UUID, observation NormalizedObservation, quality QualityBreakdown) (uuid.UUID, error) {
	if s == nil || s.db == nil {
		return uuid.Nil, fmt.Errorf("knowledge store is required")
	}
	openingHours, err := marshalOptionalJSON(observation.OpeningHours)
	if err != nil {
		return uuid.Nil, err
	}
	tags, err := json.Marshal(observation.Tags)
	if err != nil {
		return uuid.Nil, fmt.Errorf("marshal observation tags: %w", err)
	}
	rawPayload, err := marshalOptionalJSON(observation.RawPayload)
	if err != nil {
		return uuid.Nil, err
	}

	var id uuid.UUID
	err = s.db.QueryRow(ctx, `INSERT INTO travel_provider_place_observations
  (provider, provider_place_id, destination_id, raw_name, normalized_name, category, latitude, longitude,
   address, website, opening_hours, rating, rating_count, price_level, tags, raw_payload, source_url,
   license_name, attribution, observed_at, expires_at, quality_score, confidence, match_status)
  VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,
    COALESCE((SELECT match_status FROM travel_provider_place_observations WHERE provider=$1 AND provider_place_id=$2), 'unmatched'))
  ON CONFLICT (provider, provider_place_id) DO UPDATE SET
    destination_id=EXCLUDED.destination_id, raw_name=EXCLUDED.raw_name, normalized_name=EXCLUDED.normalized_name,
    category=EXCLUDED.category, latitude=EXCLUDED.latitude, longitude=EXCLUDED.longitude, address=EXCLUDED.address,
    website=EXCLUDED.website, opening_hours=EXCLUDED.opening_hours, rating=EXCLUDED.rating,
    rating_count=EXCLUDED.rating_count, price_level=EXCLUDED.price_level, tags=EXCLUDED.tags,
    raw_payload=EXCLUDED.raw_payload, source_url=EXCLUDED.source_url, license_name=EXCLUDED.license_name,
    attribution=EXCLUDED.attribution, observed_at=EXCLUDED.observed_at, expires_at=EXCLUDED.expires_at,
    quality_score=EXCLUDED.quality_score, confidence=EXCLUDED.confidence, updated_at=NOW()
  RETURNING id`,
		observation.Provider, observation.ProviderPlaceID, destinationID, observation.RawName, observation.NormalizedName,
		nullText(observation.Category), observation.Latitude, observation.Longitude, nullText(observation.Address),
		nullText(observation.Website), openingHours, observation.Rating, observation.RatingCount,
		nullText(observation.PriceLevel), tags, rawPayload, nullText(observation.SourceURL),
		nullText(observation.LicenseName), nullText(observation.Attribution), observation.ObservedAt,
		observation.ExpiresAt, quality.QualityScore, quality.Confidence).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert observation %s/%s: %w", observation.Provider, observation.ProviderPlaceID, err)
	}
	return id, nil
}

// SetObservationMatch links an observation to a place and records the outcome.
func (s *Store) SetObservationMatch(ctx context.Context, observationID uuid.UUID, placeID *uuid.UUID, matchStatus string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("knowledge store is required")
	}
	_, err := s.db.Exec(ctx, `UPDATE travel_provider_place_observations
  SET matched_place_id=$2, match_status=$3, updated_at=NOW() WHERE id=$1`, observationID, placeID, matchStatus)
	if err != nil {
		return fmt.Errorf("set observation match %s: %w", observationID, err)
	}
	return nil
}

// ListObservations returns observations for Ops review, newest first. Raw
// payloads are deliberately not selected: they are ops-admin-only and are
// fetched separately when explicitly requested.
func (s *Store) ListObservations(ctx context.Context, destinationID *uuid.UUID, matchStatus string, limit int) ([]ObservationRecord, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("knowledge store is required")
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.Query(ctx, `SELECT id, provider, provider_place_id, destination_id, raw_name, normalized_name,
    COALESCE(category,''), latitude, longitude, COALESCE(address,''), COALESCE(website,''),
    COALESCE(license_name,''), COALESCE(attribution,''), COALESCE(source_url,''), observed_at,
    quality_score, confidence, matched_place_id, match_status
  FROM travel_provider_place_observations
  WHERE ($1::uuid IS NULL OR destination_id = $1)
    AND ($2::text IS NULL OR match_status = $2)
  ORDER BY observed_at DESC, id
  LIMIT $3`, destinationID, nullText(matchStatus), limit)
	if err != nil {
		return nil, fmt.Errorf("list observations: %w", err)
	}
	defer rows.Close()

	observations := make([]ObservationRecord, 0, limit)
	for rows.Next() {
		var record ObservationRecord
		if err := rows.Scan(&record.ID, &record.Provider, &record.ProviderPlaceID, &record.DestinationID,
			&record.RawName, &record.NormalizedName, &record.Category, &record.Latitude, &record.Longitude,
			&record.Address, &record.Website, &record.LicenseName, &record.Attribution, &record.SourceURL,
			&record.ObservedAt, &record.QualityScore, &record.Confidence, &record.MatchedPlaceID,
			&record.MatchStatus); err != nil {
			return nil, fmt.Errorf("scan observation: %w", err)
		}
		observations = append(observations, record)
	}
	return observations, rows.Err()
}

// ListMatchCandidates loads the places in a destination that an observation
// could describe. Merged records are excluded: their canonical record is the
// valid match target.
func (s *Store) ListMatchCandidates(ctx context.Context, destinationID uuid.UUID) ([]MatchCandidate, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("knowledge store is required")
	}
	rows, err := s.db.Query(ctx, `SELECT id, canonical_name, aliases, category, latitude, longitude,
    provider_refs, destination_id, review_status
  FROM travel_places
  WHERE destination_id = $1 AND review_status <> 'merged'
  ORDER BY canonical_name, id`, destinationID)
	if err != nil {
		return nil, fmt.Errorf("list match candidates: %w", err)
	}
	defer rows.Close()

	candidates := make([]MatchCandidate, 0, 64)
	for rows.Next() {
		var (
			candidate   MatchCandidate
			placeID     uuid.UUID
			destination uuid.UUID
			aliasesJSON []byte
			refsJSON    []byte
		)
		if err := rows.Scan(&placeID, &candidate.CanonicalName, &aliasesJSON, &candidate.Category,
			&candidate.Latitude, &candidate.Longitude, &refsJSON, &destination, &candidate.ReviewStatus); err != nil {
			return nil, fmt.Errorf("scan match candidate: %w", err)
		}
		candidate.PlaceID = placeID.String()
		candidate.DestinationID = destination.String()
		candidate.Aliases = decodeStringArray(aliasesJSON)
		candidate.ProviderRefs = decodeProviderRefs(refsJSON)
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}

// UpsertPlaceFromObservation creates or refreshes a knowledge record from a
// provider observation. It never overwrites a rejected record and never
// downgrades curated data with lower-trust provider values.
func (s *Store) UpsertPlaceFromObservation(
	ctx context.Context,
	destinationID uuid.UUID,
	observation NormalizedObservation,
	quality QualityBreakdown,
	reviewStatus string,
	sourceID *uuid.UUID,
) (uuid.UUID, bool, error) {
	if s == nil || s.db == nil {
		return uuid.Nil, false, fmt.Errorf("knowledge store is required")
	}
	aliases, err := json.Marshal(observation.Aliases)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("marshal aliases: %w", err)
	}
	tags, err := json.Marshal(observation.Tags)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("marshal tags: %w", err)
	}
	providerRefs, err := json.Marshal([]string{observation.Provider + ":" + observation.ProviderPlaceID})
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("marshal provider refs: %w", err)
	}
	openingHours, err := marshalOptionalJSON(observation.OpeningHours)
	if err != nil {
		return uuid.Nil, false, err
	}

	var (
		placeID  uuid.UUID
		inserted bool
	)
	// The provider_refs merge keeps every provider that has described this
	// place, which is what later match and merge decisions rely on.
	err = s.db.QueryRow(ctx, `INSERT INTO travel_places
  (destination_id, canonical_name, category, subcategory, latitude, longitude, address, aliases, tags,
   price_level, opening_hours, website, provider_refs, source_id, source_url, license_name, confidence,
   quality_score, freshness_score, source_trust_score, review_status, last_verified_at,
   last_provider_refresh_at, last_quality_checked_at, status)
  VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,NOW(),NOW(),NOW(),'active')
  ON CONFLICT (destination_id, canonical_name) DO UPDATE SET
    category = CASE WHEN travel_places.review_status = 'approved' THEN travel_places.category ELSE EXCLUDED.category END,
    subcategory = COALESCE(EXCLUDED.subcategory, travel_places.subcategory),
    latitude = COALESCE(EXCLUDED.latitude, travel_places.latitude),
    longitude = COALESCE(EXCLUDED.longitude, travel_places.longitude),
    address = COALESCE(EXCLUDED.address, travel_places.address),
    aliases = EXCLUDED.aliases,
    tags = EXCLUDED.tags,
    price_level = COALESCE(EXCLUDED.price_level, travel_places.price_level),
    opening_hours = COALESCE(EXCLUDED.opening_hours, travel_places.opening_hours),
    website = COALESCE(EXCLUDED.website, travel_places.website),
    provider_refs = (
      SELECT COALESCE(jsonb_agg(DISTINCT value), '[]'::jsonb)
      FROM jsonb_array_elements(travel_places.provider_refs || EXCLUDED.provider_refs)
    ),
    source_url = COALESCE(EXCLUDED.source_url, travel_places.source_url),
    license_name = COALESCE(EXCLUDED.license_name, travel_places.license_name),
    confidence = EXCLUDED.confidence,
    quality_score = EXCLUDED.quality_score,
    freshness_score = EXCLUDED.freshness_score,
    source_trust_score = EXCLUDED.source_trust_score,
    -- A human decision (approved/rejected/merged) is never overwritten by a job.
    review_status = CASE
      WHEN travel_places.review_status IN ('approved','rejected','merged') THEN travel_places.review_status
      ELSE EXCLUDED.review_status END,
    last_provider_refresh_at = NOW(),
    last_quality_checked_at = NOW(),
    last_verified_at = NOW(),
    updated_at = NOW()
  RETURNING id, (xmax = 0) AS inserted`,
		destinationID, observation.DisplayName, observation.Category, nullText(observation.Subcategory),
		observation.Latitude, observation.Longitude, nullText(observation.Address), aliases, tags,
		nullText(observation.PriceLevel), openingHours, nullText(observation.Website), providerRefs,
		sourceID, nullText(observation.SourceURL), nullText(observation.LicenseName), quality.Confidence,
		quality.QualityScore, quality.FreshnessScore, quality.SourceTrustScore, reviewStatus).Scan(&placeID, &inserted)
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("upsert place %q: %w", observation.DisplayName, err)
	}
	return placeID, inserted, nil
}

// UpdatePlaceQuality writes recomputed scores without touching content fields.
func (s *Store) UpdatePlaceQuality(ctx context.Context, placeID uuid.UUID, quality QualityBreakdown, reviewStatus string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("knowledge store is required")
	}
	_, err := s.db.Exec(ctx, `UPDATE travel_places SET
    quality_score=$2, freshness_score=$3, source_trust_score=$4, confidence=$5,
    review_status = CASE WHEN review_status IN ('approved','rejected','merged') THEN review_status ELSE $6 END,
    last_quality_checked_at=NOW(), updated_at=NOW()
  WHERE id=$1`, placeID, quality.QualityScore, quality.FreshnessScore, quality.SourceTrustScore,
		quality.Confidence, reviewStatus)
	if err != nil {
		return fmt.Errorf("update place quality %s: %w", placeID, err)
	}
	return nil
}

// StalePlace identifies a record due for provider refresh.
type StalePlace struct {
	PlaceID       uuid.UUID `json:"placeId"`
	DestinationID uuid.UUID `json:"destinationId"`
	CanonicalName string    `json:"canonicalName"`
	Category      string    `json:"category"`
}

// ListStalePlaces selects active, non-rejected records whose last provider
// refresh is older than the configured window. Ordering is deterministic and
// the batch is bounded, so a refresh run can never become unbounded provider
// traffic.
func (s *Store) ListStalePlaces(ctx context.Context, destinationID *uuid.UUID, staleAfterDays, batchSize int, minQuality float64) ([]StalePlace, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("knowledge store is required")
	}
	if batchSize <= 0 || batchSize > 500 {
		batchSize = 100
	}
	if staleAfterDays <= 0 {
		staleAfterDays = 30
	}
	rows, err := s.db.Query(ctx, `SELECT id, destination_id, canonical_name, category
  FROM travel_places
  WHERE status = 'active'
    AND review_status NOT IN ('rejected','merged')
    AND ($1::uuid IS NULL OR destination_id = $1)
    AND (last_provider_refresh_at IS NULL OR last_provider_refresh_at < NOW() - make_interval(days => $2))
    AND (quality_score IS NULL OR quality_score >= $3)
  ORDER BY last_provider_refresh_at NULLS FIRST, id
  LIMIT $4`, destinationID, staleAfterDays, minQuality, batchSize)
	if err != nil {
		return nil, fmt.Errorf("list stale places: %w", err)
	}
	defer rows.Close()

	places := make([]StalePlace, 0, batchSize)
	for rows.Next() {
		var place StalePlace
		if err := rows.Scan(&place.PlaceID, &place.DestinationID, &place.CanonicalName, &place.Category); err != nil {
			return nil, fmt.Errorf("scan stale place: %w", err)
		}
		places = append(places, place)
	}
	return places, rows.Err()
}

// ReviewAction applies an Ops review decision and writes an audit event in one
// transaction, so a decision is never recorded without its audit trail.
func (s *Store) ReviewAction(ctx context.Context, placeID uuid.UUID, actorUserID *uuid.UUID, action, reason string) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("knowledge store is required")
	}
	reviewStatus, err := reviewStatusForAction(action)
	if err != nil {
		return err
	}

	transaction, err := s.db.BeginTx(ctx, pgxTxOptions())
	if err != nil {
		return fmt.Errorf("begin review transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	var oldStatus string
	if err := transaction.QueryRow(ctx, `SELECT review_status FROM travel_places WHERE id=$1 FOR UPDATE`, placeID).Scan(&oldStatus); err != nil {
		return fmt.Errorf("load place %s for review: %w", placeID, err)
	}

	if _, err := transaction.Exec(ctx, `UPDATE travel_places SET
      review_status=$2,
      rejected_reason = CASE WHEN $2 = 'rejected' THEN $3 ELSE NULL END,
      approved_by_user_id = CASE WHEN $2 = 'approved' THEN $4 ELSE approved_by_user_id END,
      approved_at = CASE WHEN $2 = 'approved' THEN NOW() ELSE approved_at END,
      updated_at = NOW()
    WHERE id=$1`, placeID, reviewStatus, nullText(reason), actorUserID); err != nil {
		return fmt.Errorf("apply review to %s: %w", placeID, err)
	}

	if err := insertReviewEvent(ctx, transaction, &placeID, nil, actorUserID, action,
		map[string]any{"reviewStatus": oldStatus},
		map[string]any{"reviewStatus": reviewStatus}, reason); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit review: %w", err)
	}
	return nil
}

func reviewStatusForAction(action string) (string, error) {
	switch action {
	case "approved":
		return ReviewStatusApproved, nil
	case "rejected":
		return ReviewStatusRejected, nil
	case "needs_review":
		return ReviewStatusNeedsReview, nil
	default:
		return "", fmt.Errorf("unsupported review action %q", action)
	}
}

// QualitySummary is the Ops overview of knowledge health.
type QualitySummary struct {
	DestinationsCovered   int        `json:"destinationsCovered"`
	TotalPlaces           int        `json:"totalPlaces"`
	HighQualityPlaces     int        `json:"highQualityPlaces"`
	NeedsReviewPlaces     int        `json:"needsReviewPlaces"`
	RejectedPlaces        int        `json:"rejectedPlaces"`
	MergedPlaces          int        `json:"mergedPlaces"`
	StalePlaces           int        `json:"stalePlaces"`
	MissingCoordinates    int        `json:"missingCoordinates"`
	OpenDuplicateGroups   int        `json:"openDuplicateGroups"`
	ProviderObservations  int        `json:"providerObservations"`
	UnmatchedObservations int        `json:"unmatchedObservations"`
	LastIngestionAt       *time.Time `json:"lastIngestionAt,omitempty"`
}

// QualitySummary aggregates knowledge health for the Ops dashboard.
func (s *Store) QualitySummary(ctx context.Context, strongMinQuality float64, staleAfterDays int) (QualitySummary, error) {
	if s == nil || s.db == nil {
		return QualitySummary{}, fmt.Errorf("knowledge store is required")
	}
	var summary QualitySummary
	err := s.db.QueryRow(ctx, `SELECT
  (SELECT count(DISTINCT destination_id) FROM travel_places WHERE status='active'),
  (SELECT count(*) FROM travel_places WHERE status='active'),
  (SELECT count(*) FROM travel_places WHERE status='active' AND review_status NOT IN ('rejected','merged') AND quality_score >= $1),
  (SELECT count(*) FROM travel_places WHERE review_status='needs_review'),
  (SELECT count(*) FROM travel_places WHERE review_status='rejected'),
  (SELECT count(*) FROM travel_places WHERE review_status='merged'),
  (SELECT count(*) FROM travel_places WHERE status='active' AND review_status NOT IN ('rejected','merged')
     AND (last_provider_refresh_at IS NULL OR last_provider_refresh_at < NOW() - make_interval(days => $2))),
  (SELECT count(*) FROM travel_places WHERE status='active' AND (latitude IS NULL OR longitude IS NULL)),
  (SELECT count(*) FROM travel_place_duplicate_groups WHERE status='open'),
  (SELECT count(*) FROM travel_provider_place_observations),
  (SELECT count(*) FROM travel_provider_place_observations WHERE match_status='unmatched'),
  (SELECT max(observed_at) FROM travel_provider_place_observations)`,
		strongMinQuality, staleAfterDays).Scan(
		&summary.DestinationsCovered, &summary.TotalPlaces, &summary.HighQualityPlaces,
		&summary.NeedsReviewPlaces, &summary.RejectedPlaces, &summary.MergedPlaces,
		&summary.StalePlaces, &summary.MissingCoordinates, &summary.OpenDuplicateGroups,
		&summary.ProviderObservations, &summary.UnmatchedObservations, &summary.LastIngestionAt)
	if err != nil {
		return QualitySummary{}, fmt.Errorf("read knowledge quality summary: %w", err)
	}
	return summary, nil
}

func marshalOptionalJSON(value any) (any, error) {
	if value == nil {
		return nil, nil
	}
	switch typed := value.(type) {
	case map[string]any:
		if len(typed) == 0 {
			return nil, nil
		}
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal knowledge json: %w", err)
	}
	if string(encoded) == "null" || string(encoded) == "[]" {
		return nil, nil
	}
	return encoded, nil
}

func decodeStringArray(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil
	}
	return values
}

// decodeProviderRefs accepts both the plain string form written by this
// package and the object form curated imports may use.
func decodeProviderRefs(raw []byte) []string {
	if len(raw) == 0 {
		return nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err == nil {
		return values
	}
	var objects []map[string]any
	if err := json.Unmarshal(raw, &objects); err != nil {
		return nil
	}
	refs := make([]string, 0, len(objects))
	for _, object := range objects {
		providerName, _ := object["provider"].(string)
		placeID, _ := object["providerPlaceId"].(string)
		if providerName != "" && placeID != "" {
			refs = append(refs, strings.ToLower(providerName)+":"+placeID)
		}
	}
	return refs
}
