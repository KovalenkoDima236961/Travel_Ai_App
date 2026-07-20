package knowledge

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	tripknowledge "github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge"
	tripprovider "github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge/provider"
)

// Provider ingestion reuses Trip Service's knowledge module for the same reason
// curated ingestion does: Trip Service owns the normalized store, and a second
// implementation of scoring or review rules would inevitably diverge from it.
//
// The adapter is selected from configuration and defaults to the deterministic
// mock, so local development and CI never depend on a real provider API.

// ProviderRequest is one provider job invocation.
type ProviderRequest struct {
	JobType         string   `json:"jobType"`
	DestinationID   string   `json:"destinationId,omitempty"`
	DestinationName string   `json:"destinationName,omitempty"`
	CountryCode     string   `json:"countryCode,omitempty"`
	Categories      []string `json:"categories,omitempty"`
	Provider        string   `json:"provider,omitempty"`
	Limit           int      `json:"limit,omitempty"`
	BatchSize       int      `json:"batchSize,omitempty"`
	DryRun          bool     `json:"dryRun,omitempty"`
}

// ProviderConfig mirrors the KNOWLEDGE_PROVIDER_* environment contract.
type ProviderConfig struct {
	Provider          string
	FallbackToMock    bool
	MaxResults        int
	RefreshEnabled    bool
	StaleAfterDays    int
	RefreshBatchSize  int
	AllowRawPayload   bool
	StrongMinQuality  float64
	WeakMinQuality    float64
	NeedsReviewBelow  float64
	RejectBelowQualty float64
}

// DefaultProviderConfig matches the documented defaults.
func DefaultProviderConfig() ProviderConfig {
	thresholds := tripknowledge.DefaultThresholds()
	return ProviderConfig{
		Provider:          tripprovider.ProviderMock,
		FallbackToMock:    true,
		MaxResults:        100,
		RefreshEnabled:    true,
		StaleAfterDays:    thresholds.StaleAfterDays,
		RefreshBatchSize:  100,
		AllowRawPayload:   false,
		StrongMinQuality:  thresholds.StrongMinQuality,
		WeakMinQuality:    thresholds.WeakMinQuality,
		NeedsReviewBelow:  thresholds.NeedsReviewBelow,
		RejectBelowQualty: thresholds.RejectBelow,
	}
}

func (c ProviderConfig) thresholds() tripknowledge.Thresholds {
	thresholds := tripknowledge.DefaultThresholds()
	if c.StrongMinQuality > 0 {
		thresholds.StrongMinQuality = c.StrongMinQuality
	}
	if c.WeakMinQuality > 0 {
		thresholds.WeakMinQuality = c.WeakMinQuality
	}
	if c.NeedsReviewBelow > 0 {
		thresholds.NeedsReviewBelow = c.NeedsReviewBelow
	}
	if c.RejectBelowQualty > 0 {
		thresholds.RejectBelow = c.RejectBelowQualty
	}
	if c.StaleAfterDays > 0 {
		thresholds.StaleAfterDays = c.StaleAfterDays
	}
	return thresholds
}

func (c ProviderConfig) sourcePolicy() tripprovider.SourcePolicy {
	policy := tripprovider.DefaultSourcePolicy()
	policy.AllowRawPayload = c.AllowRawPayload
	return policy
}

// SelectProvider resolves the configured adapter. Real network-backed adapters
// belong in External Integrations Service behind its quota and cache guards;
// until one is wired here, any non-mock selection falls back to mock when
// configured and otherwise fails loudly rather than silently doing nothing.
func SelectProvider(cfg ProviderConfig) (tripprovider.TravelKnowledgeProvider, error) {
	name := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if name == "" {
		name = tripprovider.ProviderMock
	}
	switch name {
	case tripprovider.ProviderMock:
		return tripprovider.NewMockKnowledgeProvider(), nil
	case tripprovider.ProviderFoursquare, tripprovider.ProviderOpenTripMap, tripprovider.ProviderWikidata:
		if cfg.FallbackToMock {
			return tripprovider.NewMockKnowledgeProvider(), nil
		}
		return nil, fmt.Errorf("knowledge provider %q is not configured in this deployment: "+
			"add the adapter in External Integrations Service or set KNOWLEDGE_PROVIDER_FALLBACK_TO_MOCK=true", name)
	default:
		return nil, fmt.Errorf("unsupported KNOWLEDGE_PROVIDER %q: supported values are mock, foursquare, opentripmap, wikidata", cfg.Provider)
	}
}

// ProviderRunner executes the knowledge provider job types.
type ProviderRunner struct {
	store    *tripknowledge.Store
	ingestor *tripknowledge.Ingestor
	cfg      ProviderConfig
}

// NewProviderRunner wires the configured adapter to the shared store.
func NewProviderRunner(store *tripknowledge.Store, cfg ProviderConfig) (*ProviderRunner, error) {
	if store == nil {
		return nil, fmt.Errorf("knowledge store is required")
	}
	knowledgeProvider, err := SelectProvider(cfg)
	if err != nil {
		return nil, err
	}
	return &ProviderRunner{
		store:    store,
		ingestor: tripknowledge.NewIngestor(store, knowledgeProvider, cfg.thresholds(), cfg.sourcePolicy()),
		cfg:      cfg,
	}, nil
}

// Run dispatches one provider job. Every job is idempotent: observations are
// keyed by (provider, provider_place_id) and places by (destination, name), so
// a retry after a partial failure converges rather than duplicating rows.
func (r *ProviderRunner) Run(ctx context.Context, request ProviderRequest) (tripknowledge.IngestResult, error) {
	if r == nil || r.ingestor == nil {
		return tripknowledge.IngestResult{}, fmt.Errorf("provider runner is not initialized")
	}

	jobType := strings.TrimSpace(request.JobType)
	if jobType == "" {
		jobType = tripknowledge.JobIngestDestination
	}

	switch jobType {
	case tripknowledge.JobIngestDestination:
		return r.ingestDestination(ctx, request)
	case tripknowledge.JobRefreshStalePlaces:
		return r.refreshStale(ctx, request)
	case tripknowledge.JobDuplicateDetection:
		return r.detectDuplicates(ctx, request)
	case tripknowledge.JobMatchObservations, tripknowledge.JobQualityScoreRecompute, tripknowledge.JobReindexAfterMerge:
		// These three re-run the ingestion pass, which is what actually
		// performs them: ingestion matches observations, recomputes every
		// score, and rewrites the affected places.
		//
		// There is deliberately no vector-index step here. Provider-backed
		// places reach the model through RetrieveGrounding, which reads
		// travel_places in SQL and excludes rejected and merged records in the
		// query itself. They are never written to the Chroma collection, which
		// holds curated *documents*. So a merge takes effect the moment the
		// rows change, and there is no embedding to invalidate. If provider
		// places are ever added to the vector index, this is the branch that
		// must gain a reindex call.
		return r.ingestDestination(ctx, request)
	default:
		return tripknowledge.IngestResult{}, fmt.Errorf("unsupported knowledge job type %q", jobType)
	}
}

func (r *ProviderRunner) ingestDestination(ctx context.Context, request ProviderRequest) (tripknowledge.IngestResult, error) {
	ingestRequest := tripknowledge.IngestRequest{
		DestinationName: request.DestinationName,
		CountryCode:     request.CountryCode,
		Categories:      request.Categories,
		Provider:        request.Provider,
		Limit:           request.Limit,
		DryRun:          request.DryRun,
	}
	if ingestRequest.Limit <= 0 {
		ingestRequest.Limit = r.cfg.MaxResults
	}
	if request.DestinationID != "" {
		destinationID, err := uuid.Parse(request.DestinationID)
		if err != nil {
			return tripknowledge.IngestResult{}, fmt.Errorf("parse destinationId: %w", err)
		}
		ingestRequest.DestinationID = &destinationID
	}
	return r.ingestor.IngestDestination(ctx, ingestRequest)
}

func (r *ProviderRunner) refreshStale(ctx context.Context, request ProviderRequest) (tripknowledge.IngestResult, error) {
	if !r.cfg.RefreshEnabled {
		return tripknowledge.IngestResult{
			JobType:  tripknowledge.JobRefreshStalePlaces,
			Warnings: []string{"refresh is disabled by KNOWLEDGE_PROVIDER_REFRESH_ENABLED"},
		}, nil
	}
	batchSize := request.BatchSize
	if batchSize <= 0 {
		batchSize = r.cfg.RefreshBatchSize
	}
	var destinationID *uuid.UUID
	if request.DestinationID != "" {
		parsed, err := uuid.Parse(request.DestinationID)
		if err != nil {
			return tripknowledge.IngestResult{}, fmt.Errorf("parse destinationId: %w", err)
		}
		destinationID = &parsed
	}
	return r.ingestor.RefreshStalePlaces(ctx, destinationID, batchSize, r.cfg.WeakMinQuality)
}

func (r *ProviderRunner) detectDuplicates(ctx context.Context, request ProviderRequest) (tripknowledge.IngestResult, error) {
	result := tripknowledge.IngestResult{JobType: tripknowledge.JobDuplicateDetection, Warnings: []string{}}
	if request.DestinationID == "" {
		return result, fmt.Errorf("destinationId is required for %s", tripknowledge.JobDuplicateDetection)
	}
	destinationID, err := uuid.Parse(request.DestinationID)
	if err != nil {
		return result, fmt.Errorf("parse destinationId: %w", err)
	}
	groups, err := r.ingestor.DetectAndRecordDuplicates(ctx, destinationID)
	if err != nil {
		return result, err
	}
	result.DestinationID = destinationID.String()
	result.DuplicateGroupsFound = groups
	return result, nil
}
