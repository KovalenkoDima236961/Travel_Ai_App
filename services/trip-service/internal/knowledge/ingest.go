package knowledge

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge/provider"
)

// Ingestion is the pipeline that turns provider results into reviewable
// knowledge. It is shared by the Worker job and the Ops "run ingestion" action
// so both follow identical rules; there is no second path into travel_places
// that skips scoring or review.

// Job type names, used by the Worker and reported in job status.
const (
	JobIngestDestination     = "knowledge_provider_ingest_destination"
	JobRefreshStalePlaces    = "knowledge_provider_refresh_stale_places"
	JobMatchObservations     = "knowledge_provider_match_observations"
	JobQualityScoreRecompute = "knowledge_quality_score_recompute"
	JobDuplicateDetection    = "knowledge_duplicate_detection"
	JobReindexAfterMerge     = "knowledge_reindex_after_merge"
)

// trustLevelForProvider maps a provider name onto a trust level. Unknown
// providers are untrusted by default: a provider must be added deliberately,
// with its license and terms documented, before its data can score well.
func trustLevelForProvider(providerName string) string {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case provider.ProviderMock:
		return "mock"
	case provider.ProviderFoursquare:
		return "trusted_provider"
	case provider.ProviderOpenTripMap, provider.ProviderWikidata:
		return "public_open_data"
	default:
		return "unknown"
	}
}

// IngestRequest describes one destination ingestion run.
type IngestRequest struct {
	DestinationID   *uuid.UUID `json:"destinationId,omitempty"`
	DestinationName string     `json:"destinationName"`
	CountryCode     string     `json:"countryCode,omitempty"`
	Categories      []string   `json:"categories,omitempty"`
	Provider        string     `json:"provider,omitempty"`
	Limit           int        `json:"limit,omitempty"`
	DryRun          bool       `json:"dryRun,omitempty"`
	Language        string     `json:"language,omitempty"`
}

// IngestResult is the job outcome. It is safe to log: counts and identifiers
// only, never provider payloads.
type IngestResult struct {
	JobType              string   `json:"jobType"`
	Provider             string   `json:"provider"`
	DestinationID        string   `json:"destinationId,omitempty"`
	Destination          string   `json:"destination"`
	RecordsFetched       int      `json:"recordsFetched"`
	ObservationsStored   int      `json:"observationsStored"`
	PlacesCreated        int      `json:"placesCreated"`
	PlacesUpdated        int      `json:"placesUpdated"`
	NeedsReviewCount     int      `json:"needsReviewCount"`
	RejectedCount        int      `json:"rejectedCount"`
	LowQualityCount      int      `json:"lowQualityCount"`
	DuplicateGroupsFound int      `json:"duplicatesFound"`
	NormalizationErrors  int      `json:"normalizationErrors"`
	ReindexRequired      bool     `json:"reindexRequired"`
	DryRun               bool     `json:"dryRun"`
	DurationMs           int64    `json:"durationMs"`
	Warnings             []string `json:"warnings"`
}

// Ingestor runs provider ingestion against the knowledge store.
type Ingestor struct {
	store      *Store
	provider   provider.TravelKnowledgeProvider
	thresholds Thresholds
	policy     provider.SourcePolicy
	now        func() time.Time
}

// NewIngestor wires a provider adapter to the store. The clock is injectable so
// tests and CI runs are reproducible.
func NewIngestor(store *Store, knowledgeProvider provider.TravelKnowledgeProvider, thresholds Thresholds, policy provider.SourcePolicy) *Ingestor {
	return &Ingestor{
		store:      store,
		provider:   knowledgeProvider,
		thresholds: thresholds,
		policy:     policy,
		now:        time.Now,
	}
}

// WithClock overrides the ingestion clock, which controls freshness scoring.
func (i *Ingestor) WithClock(now func() time.Time) *Ingestor {
	if now != nil {
		i.now = now
	}
	return i
}

// IngestDestination fetches, normalizes, scores, and stores provider records
// for one destination.
//
// A record only reaches travel_places when it scores above the reject
// threshold; below that it stays an observation, which keeps bad data out of
// grounding while preserving the evidence for review.
func (i *Ingestor) IngestDestination(ctx context.Context, request IngestRequest) (IngestResult, error) {
	started := i.now()
	if i == nil || i.store == nil || i.provider == nil {
		return IngestResult{}, fmt.Errorf("ingestor requires a store and a provider")
	}
	destinationName := strings.TrimSpace(request.DestinationName)
	if destinationName == "" {
		return IngestResult{}, fmt.Errorf("destinationName is required")
	}

	result := IngestResult{
		JobType:     JobIngestDestination,
		Provider:    i.provider.ProviderName(),
		Destination: destinationName,
		DryRun:      request.DryRun,
		Warnings:    []string{},
	}

	license := i.provider.LicenseInfo()
	if i.policy.RequireLicense && !license.Valid() {
		return result, fmt.Errorf("%w: provider %s", ErrLicenseMissing, i.provider.ProviderName())
	}

	records, metadata, err := i.provider.SearchPlaces(ctx, provider.SearchRequest{
		DestinationName: destinationName,
		CountryCode:     request.CountryCode,
		Categories:      request.Categories,
		Limit:           request.Limit,
		Language:        request.Language,
		SourcePolicy:    i.policy,
	})
	if err != nil {
		return result, classifyProviderError(err)
	}
	result.RecordsFetched = len(records)
	if metadata.Truncated {
		result.Warnings = append(result.Warnings, "provider results were truncated by the configured limit")
	}
	if metadata.FallbackUsed {
		result.Warnings = append(result.Warnings, "provider fallback was used")
	}

	// Normalize first so a malformed record fails before anything is written.
	observations := make([]NormalizedObservation, 0, len(records))
	for _, record := range records {
		observation, normalizeErr := NormalizeProviderRecord(record, i.policy)
		if normalizeErr != nil {
			result.NormalizationErrors++
			result.Warnings = append(result.Warnings, normalizeErr.Error())
			continue
		}
		observations = append(observations, observation)
	}

	agreement := providerAgreementByName(observations)
	trustLevel := trustLevelForProvider(i.provider.ProviderName())

	if request.DryRun {
		// A dry run reports what would happen without writing, which is how the
		// Ops "preview" action and CI validation stay side-effect free.
		for _, observation := range observations {
			quality := i.scoreObservation(observation, trustLevel, agreement[observation.NormalizedName], true)
			i.tallyQuality(&result, quality)
		}
		result.DurationMs = i.now().Sub(started).Milliseconds()
		return result, nil
	}

	destinationID, err := i.store.ResolveDestination(ctx, destinationName, request.CountryCode, request.DestinationID)
	if err != nil {
		return result, err
	}
	result.DestinationID = destinationID.String()

	sourceID, err := i.store.EnsureProviderSource(ctx, i.provider.ProviderName(), trustLevel, license, i.provider.SupportsRefresh())
	if err != nil {
		return result, err
	}

	for _, observation := range observations {
		quality := i.scoreObservation(observation, trustLevel, agreement[observation.NormalizedName], true)
		i.tallyQuality(&result, quality)

		observationID, storeErr := i.store.UpsertObservation(ctx, &destinationID, observation, quality)
		if storeErr != nil {
			return result, storeErr
		}
		result.ObservationsStored++

		// Below the reject threshold the record stays evidence only. This is
		// the gate that keeps low-quality provider data out of grounding.
		if quality.QualityScore < i.thresholds.RejectBelow {
			if err := i.store.SetObservationMatch(ctx, observationID, nil, MatchStatusRejected); err != nil {
				return result, err
			}
			continue
		}

		candidates, candidateErr := i.store.ListMatchCandidates(ctx, destinationID)
		if candidateErr != nil {
			return result, candidateErr
		}
		decision := DecideMatch(observation, candidates, i.thresholds)

		reviewStatus := ReviewStatusForScore(ReviewStatusAuto, quality.QualityScore, FeedbackCounts{}, i.thresholds)
		if decision.Action == MatchActionNeedsReview {
			// An ambiguous match is a human decision: record the evidence and
			// leave the knowledge record flagged rather than guessing.
			reviewStatus = ReviewStatusNeedsReview
			if err := i.store.SetObservationMatch(ctx, observationID, nil, MatchStatusNeedsReview); err != nil {
				return result, err
			}
		}

		placeID, inserted, upsertErr := i.store.UpsertPlaceFromObservation(ctx, destinationID, observation, quality, reviewStatus, &sourceID)
		if upsertErr != nil {
			return result, upsertErr
		}
		if inserted {
			result.PlacesCreated++
		} else {
			result.PlacesUpdated++
		}
		result.ReindexRequired = true

		if decision.Action != MatchActionNeedsReview {
			if err := i.store.SetObservationMatch(ctx, observationID, &placeID, MatchStatusMatched); err != nil {
				return result, err
			}
		}
	}

	groups, err := i.DetectAndRecordDuplicates(ctx, destinationID)
	if err != nil {
		return result, err
	}
	result.DuplicateGroupsFound = groups
	result.DurationMs = i.now().Sub(started).Milliseconds()
	return result, nil
}

// DetectAndRecordDuplicates proposes duplicate groups for a destination. Groups
// are proposals: merging remains an explicit Ops action.
func (i *Ingestor) DetectAndRecordDuplicates(ctx context.Context, destinationID uuid.UUID) (int, error) {
	candidates, err := i.store.ListMatchCandidates(ctx, destinationID)
	if err != nil {
		return 0, err
	}
	pairs := DetectDuplicates(candidates, i.thresholds)
	created := 0
	for _, pair := range pairs {
		reason := fmt.Sprintf("confidence %.2f: %s", pair.Confidence, strings.Join(pair.Reasons, ", "))
		_, isNew, groupErr := i.store.CreateDuplicateGroup(ctx, destinationID, pair, reason)
		if groupErr != nil {
			return created, groupErr
		}
		if isNew {
			created++
		}
	}
	return created, nil
}

// RefreshStalePlaces re-observes records whose provider data has aged out.
// The batch is bounded by configuration so a refresh run cannot become
// unbounded provider traffic.
func (i *Ingestor) RefreshStalePlaces(ctx context.Context, destinationID *uuid.UUID, batchSize int, minQuality float64) (IngestResult, error) {
	started := i.now()
	result := IngestResult{
		JobType:  JobRefreshStalePlaces,
		Provider: i.provider.ProviderName(),
		Warnings: []string{},
	}
	if !i.provider.SupportsRefresh() {
		result.Warnings = append(result.Warnings, "provider does not support refresh")
		return result, nil
	}

	stale, err := i.store.ListStalePlaces(ctx, destinationID, i.thresholds.StaleAfterDays, batchSize, minQuality)
	if err != nil {
		return result, err
	}
	if len(stale) == 0 {
		result.DurationMs = i.now().Sub(started).Milliseconds()
		return result, nil
	}

	// Refresh is grouped by destination so each destination costs one provider
	// search rather than one request per stale place.
	byDestination := map[uuid.UUID][]StalePlace{}
	for _, place := range stale {
		byDestination[place.DestinationID] = append(byDestination[place.DestinationID], place)
	}
	destinationIDs := make([]uuid.UUID, 0, len(byDestination))
	for id := range byDestination {
		destinationIDs = append(destinationIDs, id)
	}
	sort.Slice(destinationIDs, func(a, b int) bool {
		return destinationIDs[a].String() < destinationIDs[b].String()
	})

	for _, id := range destinationIDs {
		name, countryCode, lookupErr := i.store.DestinationName(ctx, id)
		if lookupErr != nil {
			return result, lookupErr
		}
		destinationID := id
		runResult, runErr := i.IngestDestination(ctx, IngestRequest{
			DestinationID:   &destinationID,
			DestinationName: name,
			CountryCode:     countryCode,
		})
		if runErr != nil {
			return result, runErr
		}
		result.RecordsFetched += runResult.RecordsFetched
		result.ObservationsStored += runResult.ObservationsStored
		result.PlacesCreated += runResult.PlacesCreated
		result.PlacesUpdated += runResult.PlacesUpdated
		result.NeedsReviewCount += runResult.NeedsReviewCount
		result.LowQualityCount += runResult.LowQualityCount
		result.DuplicateGroupsFound += runResult.DuplicateGroupsFound
		result.ReindexRequired = result.ReindexRequired || runResult.ReindexRequired
	}
	result.Destination = fmt.Sprintf("%d destination(s)", len(destinationIDs))
	result.DurationMs = i.now().Sub(started).Milliseconds()
	return result, nil
}

// scoreObservation computes the quality breakdown for one observation.
func (i *Ingestor) scoreObservation(observation NormalizedObservation, trustLevel string, agreement float64, destinationMatched bool) QualityBreakdown {
	categoryConfident := observation.Category != "other"
	for _, warning := range observation.Warnings {
		if strings.HasPrefix(warning, "unmapped provider category") || warning == "provider reported no category" {
			categoryConfident = false
		}
	}
	return ComputeQuality(QualityInput{
		TrustLevel:         trustLevel,
		Category:           observation.Category,
		DestinationMatched: destinationMatched,
		HasCoordinates:     observation.Latitude != nil && observation.Longitude != nil,
		CategoryConfident:  categoryConfident,
		NameQuality:        NameQuality(observation.DisplayName),
		ObservedAt:         observation.ObservedAt,
		Now:                i.now(),
		ProviderAgreement:  agreement,
		HasOpeningHours:    len(observation.OpeningHours) > 0,
		HasAddress:         observation.Address != "",
		HasWebsite:         observation.Website != "",
		ReviewStatus:       ReviewStatusAuto,
		LicensePresent:     observation.LicenseName != "",
	})
}

func (i *Ingestor) tallyQuality(result *IngestResult, quality QualityBreakdown) {
	switch {
	case quality.QualityScore < i.thresholds.RejectBelow:
		result.RejectedCount++
		result.LowQualityCount++
	case quality.QualityScore < i.thresholds.NeedsReviewBelow:
		result.NeedsReviewCount++
		result.LowQualityCount++
	case quality.QualityScore < i.thresholds.StrongMinQuality:
		result.LowQualityCount++
	}
}

// providerAgreementByName groups observations that describe the same place so
// cross-provider corroboration can be scored.
func providerAgreementByName(observations []NormalizedObservation) map[string]float64 {
	grouped := map[string][]NormalizedObservation{}
	for _, observation := range observations {
		grouped[observation.NormalizedName] = append(grouped[observation.NormalizedName], observation)
	}
	agreement := make(map[string]float64, len(grouped))
	for name, group := range grouped {
		agreement[name] = ProviderAgreement(group)
	}
	return agreement
}

// classifyProviderError maps adapter failures onto the knowledge error
// vocabulary so handlers can return stable error codes.
func classifyProviderError(err error) error {
	switch {
	case errors.Is(err, provider.ErrProviderRateLimited):
		return fmt.Errorf("%w: %s", ErrProviderRateLimited, err)
	case errors.Is(err, provider.ErrLicenseMissing):
		return fmt.Errorf("%w: %s", ErrLicenseMissing, err)
	case errors.Is(err, provider.ErrProviderUnavailable):
		return fmt.Errorf("%w: %s", ErrProviderUnavailable, err)
	default:
		return err
	}
}
