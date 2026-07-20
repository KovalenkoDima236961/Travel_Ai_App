package knowledge

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge/provider"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

// These tests exercise the real SQL in provider_store.go, duplicates.go,
// retrieval.go, and review.go. Query strings are not validated by the compiler,
// so without them the store is unverified.
//
// They connect to the database described by TRIP_TEST_POSTGRES_* and skip when
// it is unset, keeping the default suite hermetic. The schema must already be
// migrated (services/trip-service/migrations).

func newIntegrationStore(t *testing.T) (*Store, *storage.DB) {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("TRIP_TEST_POSTGRES_HOST"))
	if host == "" {
		t.Skip("TRIP_TEST_POSTGRES_HOST not set; skipping Postgres integration test")
	}
	port, err := strconv.Atoi(strings.TrimSpace(os.Getenv("TRIP_TEST_POSTGRES_PORT")))
	if err != nil {
		t.Fatalf("TRIP_TEST_POSTGRES_PORT must be numeric: %v", err)
	}
	cfg := storage.Config{
		Database: envOrDefault("TRIP_TEST_POSTGRES_DB", "trip_service"),
		Username: envOrDefault("TRIP_TEST_POSTGRES_USER", "postgres"),
		Password: envOrDefault("TRIP_TEST_POSTGRES_PASSWORD", "postgres"),
		Host:     host,
		Port:     port,
		MinConns: 1,
		MaxConns: 4,
		// storage.New applies migrations, so the test database is brought up to
		// the current schema (including 000043) rather than assuming it.
		MigPath:             envOrDefault("TRIP_TEST_POSTGRES_MIG_PATH", "../../migrations"),
		QueryTimeoutSeconds: 10,
	}
	db, err := storage.New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(db.Close)

	// Each test starts from a clean knowledge slate so ordering never matters.
	for _, table := range []string{
		"travel_knowledge_review_events",
		"travel_place_duplicate_group_members",
		"travel_place_duplicate_groups",
		"travel_provider_place_observations",
		"travel_places",
		"travel_destinations",
		"travel_knowledge_sources",
	} {
		if _, err := db.Exec(context.Background(), "TRUNCATE TABLE "+table+" CASCADE"); err != nil {
			t.Fatalf("truncate %s: %v", table, err)
		}
	}
	return NewStore(db), db
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func newTestIngestor(t *testing.T, store *Store) *Ingestor {
	t.Helper()
	return NewIngestor(store, provider.NewMockKnowledgeProvider(), DefaultThresholds(), provider.DefaultSourcePolicy()).
		WithClock(func() time.Time { return provider.MockReferenceTime })
}

func TestIntegrationIngestDestinationStoresObservationsAndPlaces(t *testing.T) {
	store, _ := newIntegrationStore(t)
	ingestor := newTestIngestor(t, store)
	ctx := context.Background()

	result, err := ingestor.IngestDestination(ctx, IngestRequest{DestinationName: "Rome", CountryCode: "IT"})
	if err != nil {
		t.Fatalf("IngestDestination() error = %v", err)
	}
	if result.RecordsFetched == 0 {
		t.Fatal("expected the mock provider to return Rome fixtures")
	}
	if result.ObservationsStored == 0 {
		t.Fatal("expected observations to be persisted")
	}
	if result.PlacesCreated == 0 {
		t.Fatalf("expected places to be created: %+v", result)
	}

	observations, err := store.ListObservations(ctx, nil, "", 100)
	if err != nil {
		t.Fatalf("ListObservations() error = %v", err)
	}
	if len(observations) != result.ObservationsStored {
		t.Fatalf("stored %d observations but listed %d", result.ObservationsStored, len(observations))
	}
	// NUMERIC columns must round-trip into float64 without a scan error.
	for _, observation := range observations {
		if observation.Confidence < 0 || observation.Confidence > 1 {
			t.Fatalf("confidence out of range for %s: %v", observation.ProviderPlaceID, observation.Confidence)
		}
	}
}

// Re-running ingestion must converge rather than duplicate rows: this is what
// makes the Worker job safe to retry.
func TestIntegrationIngestIsIdempotent(t *testing.T) {
	store, db := newIntegrationStore(t)
	ingestor := newTestIngestor(t, store)
	ctx := context.Background()

	first, err := ingestor.IngestDestination(ctx, IngestRequest{DestinationName: "Vienna", CountryCode: "AT"})
	if err != nil {
		t.Fatalf("first IngestDestination() error = %v", err)
	}
	second, err := ingestor.IngestDestination(ctx, IngestRequest{DestinationName: "Vienna", CountryCode: "AT"})
	if err != nil {
		t.Fatalf("second IngestDestination() error = %v", err)
	}

	if second.PlacesCreated != 0 {
		t.Fatalf("a repeated ingestion must create no new places, created %d", second.PlacesCreated)
	}
	if second.ObservationsStored != first.ObservationsStored {
		t.Fatalf("observation count changed between runs: %d then %d",
			first.ObservationsStored, second.ObservationsStored)
	}

	var places, observations int
	if err := db.QueryRow(ctx, `SELECT
      (SELECT count(*) FROM travel_places),
      (SELECT count(*) FROM travel_provider_place_observations)`).Scan(&places, &observations); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if places != first.PlacesCreated {
		t.Fatalf("expected %d places after two runs, found %d", first.PlacesCreated, places)
	}
	if observations != first.ObservationsStored {
		t.Fatalf("expected %d observations after two runs, found %d", first.ObservationsStored, observations)
	}
}

func TestIntegrationDuplicateDetectionAndMerge(t *testing.T) {
	store, _ := newIntegrationStore(t)
	ingestor := newTestIngestor(t, store)
	ctx := context.Background()

	// The Rome fixtures contain a deliberate Colosseum duplicate.
	if _, err := ingestor.IngestDestination(ctx, IngestRequest{DestinationName: "Rome", CountryCode: "IT"}); err != nil {
		t.Fatalf("IngestDestination() error = %v", err)
	}

	groups, err := store.ListDuplicateGroups(ctx, nil, DuplicateGroupOpen, 50)
	if err != nil {
		t.Fatalf("ListDuplicateGroups() error = %v", err)
	}
	if len(groups) == 0 {
		t.Fatal("expected the deliberate Colosseum duplicate to produce a group")
	}
	group := groups[0]
	if len(group.Members) < 2 {
		t.Fatalf("a duplicate group needs at least two members, got %d", len(group.Members))
	}

	canonical := group.Members[0].PlaceID
	other := group.Members[1].PlaceID
	resolution, err := store.MergeDuplicateGroup(ctx, group.ID, canonical, nil, "integration test merge")
	if err != nil {
		t.Fatalf("MergeDuplicateGroup() error = %v", err)
	}
	if len(resolution.MergedPlaceIDs) == 0 {
		t.Fatal("merge must report the records it absorbed")
	}

	merged, err := store.GetPlaceDetail(ctx, other)
	if err != nil {
		t.Fatalf("GetPlaceDetail() error = %v", err)
	}
	if merged.ReviewStatus != ReviewStatusMerged {
		t.Fatalf("the absorbed record must be marked merged, got %q", merged.ReviewStatus)
	}
	if merged.GroundingStrength != GroundingStrengthExcluded {
		t.Fatalf("a merged record must be excluded from grounding, got %q", merged.GroundingStrength)
	}

	// Merging twice must not silently succeed.
	if _, err := store.MergeDuplicateGroup(ctx, group.ID, canonical, nil, "second merge"); err == nil {
		t.Fatal("merging an already-merged group must conflict")
	}
}

// The exclusion rules are enforced in SQL, so this is the test that proves bad
// records cannot reach a prompt.
func TestIntegrationRetrievalExcludesRejectedAndMergedRecords(t *testing.T) {
	store, _ := newIntegrationStore(t)
	ingestor := newTestIngestor(t, store)
	ctx := context.Background()

	if _, err := ingestor.IngestDestination(ctx, IngestRequest{DestinationName: "Paris", CountryCode: "FR"}); err != nil {
		t.Fatalf("IngestDestination() error = %v", err)
	}

	query := GroundingQuery{DestinationName: "Paris", IncludeWeak: true, Thresholds: DefaultThresholds()}
	before, err := store.RetrieveGrounding(ctx, query)
	if err != nil {
		t.Fatalf("RetrieveGrounding() error = %v", err)
	}
	if len(before.Places) == 0 {
		t.Fatal("expected Paris grounding places after ingestion")
	}

	rejected := before.Places[0].ID
	rejectedID, err := uuid.Parse(rejected)
	if err != nil {
		t.Fatalf("parse place id: %v", err)
	}
	if err := store.ReviewAction(ctx, rejectedID, nil, "rejected", "integration test rejection"); err != nil {
		t.Fatalf("ReviewAction() error = %v", err)
	}

	after, err := store.RetrieveGrounding(ctx, query)
	if err != nil {
		t.Fatalf("RetrieveGrounding() error = %v", err)
	}
	for _, place := range after.Places {
		if place.ID == rejected {
			t.Fatal("a rejected record must never appear in grounding context")
		}
		if place.ReviewStatus == ReviewStatusRejected || place.ReviewStatus == ReviewStatusMerged {
			t.Fatalf("retrieval returned a %s record", place.ReviewStatus)
		}
		if place.GroundingStrength == GroundingStrengthExcluded {
			t.Fatal("retrieval returned a record marked excluded")
		}
	}
	if len(after.Places) != len(before.Places)-1 {
		t.Fatalf("expected exactly one record to drop out, had %d now %d", len(before.Places), len(after.Places))
	}
}

// Approval must promote a record, and the audit trail must record the change.
func TestIntegrationReviewActionWritesAuditEvent(t *testing.T) {
	store, db := newIntegrationStore(t)
	ingestor := newTestIngestor(t, store)
	ctx := context.Background()

	if _, err := ingestor.IngestDestination(ctx, IngestRequest{DestinationName: "Bratislava", CountryCode: "SK"}); err != nil {
		t.Fatalf("IngestDestination() error = %v", err)
	}
	places, err := store.ListPlacesForReview(ctx, PlaceReviewFilters{Limit: 10})
	if err != nil {
		t.Fatalf("ListPlacesForReview() error = %v", err)
	}
	if len(places) == 0 {
		t.Fatal("expected reviewable places after ingestion")
	}

	target := places[0].ID
	if err := store.ReviewAction(ctx, target, nil, "approved", "integration test approval"); err != nil {
		t.Fatalf("ReviewAction() error = %v", err)
	}

	detail, err := store.GetPlaceDetail(ctx, target)
	if err != nil {
		t.Fatalf("GetPlaceDetail() error = %v", err)
	}
	if detail.ReviewStatus != ReviewStatusApproved {
		t.Fatalf("expected approved status, got %q", detail.ReviewStatus)
	}

	var auditCount int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM travel_knowledge_review_events
      WHERE place_id = $1 AND action = 'approved'`, target).Scan(&auditCount); err != nil {
		t.Fatalf("count audit events: %v", err)
	}
	if auditCount != 1 {
		t.Fatalf("expected exactly one audit event, found %d", auditCount)
	}
}

// An ingestion job must not overwrite a human review decision.
func TestIntegrationIngestDoesNotOverrideHumanReview(t *testing.T) {
	store, _ := newIntegrationStore(t)
	ingestor := newTestIngestor(t, store)
	ctx := context.Background()

	if _, err := ingestor.IngestDestination(ctx, IngestRequest{DestinationName: "Rome", CountryCode: "IT"}); err != nil {
		t.Fatalf("IngestDestination() error = %v", err)
	}
	places, err := store.ListPlacesForReview(ctx, PlaceReviewFilters{Limit: 10})
	if err != nil {
		t.Fatalf("ListPlacesForReview() error = %v", err)
	}
	target := places[0].ID
	if err := store.ReviewAction(ctx, target, nil, "rejected", "integration test rejection"); err != nil {
		t.Fatalf("ReviewAction() error = %v", err)
	}

	if _, err := ingestor.IngestDestination(ctx, IngestRequest{DestinationName: "Rome", CountryCode: "IT"}); err != nil {
		t.Fatalf("second IngestDestination() error = %v", err)
	}

	detail, err := store.GetPlaceDetail(ctx, target)
	if err != nil {
		t.Fatalf("GetPlaceDetail() error = %v", err)
	}
	if detail.ReviewStatus != ReviewStatusRejected {
		t.Fatalf("re-ingestion overwrote a human rejection: status is now %q", detail.ReviewStatus)
	}
}

func TestIntegrationQualitySummaryAndCoverage(t *testing.T) {
	store, _ := newIntegrationStore(t)
	ingestor := newTestIngestor(t, store)
	ctx := context.Background()

	if _, err := ingestor.IngestDestination(ctx, IngestRequest{DestinationName: "Rome", CountryCode: "IT"}); err != nil {
		t.Fatalf("IngestDestination() error = %v", err)
	}

	thresholds := DefaultThresholds()
	summary, err := store.QualitySummary(ctx, thresholds.StrongMinQuality, thresholds.StaleAfterDays)
	if err != nil {
		t.Fatalf("QualitySummary() error = %v", err)
	}
	if summary.TotalPlaces == 0 || summary.ProviderObservations == 0 {
		t.Fatalf("summary did not reflect ingestion: %+v", summary)
	}
	if summary.DestinationsCovered != 1 {
		t.Fatalf("expected one covered destination, got %d", summary.DestinationsCovered)
	}

	// Mock data is low trust, so Rome must not report full coverage.
	result, err := store.RetrieveGrounding(ctx, GroundingQuery{
		DestinationName: "Rome", IncludeWeak: true, Thresholds: thresholds,
	})
	if err != nil {
		t.Fatalf("RetrieveGrounding() error = %v", err)
	}
	if result.Coverage.PlaceCount == 0 {
		t.Fatal("coverage must count the ingested places")
	}
	if result.Status == "available" {
		t.Fatalf("mock-only data must not report full grounding availability: %+v", result.Coverage)
	}
}

// An unknown destination must degrade gracefully rather than erroring, so
// generation can fall back to generic activities.
func TestIntegrationRetrievalUnknownDestinationDegradesGracefully(t *testing.T) {
	store, _ := newIntegrationStore(t)
	ctx := context.Background()

	result, err := store.RetrieveGrounding(ctx, GroundingQuery{DestinationName: "Atlantis"})
	if err != nil {
		t.Fatalf("an unknown destination must not error: %v", err)
	}
	if result.Status != "unavailable" || len(result.Places) != 0 {
		t.Fatalf("expected an empty unavailable result, got %+v", result)
	}
	if len(result.RetrievalWarnings) == 0 {
		t.Fatal("an unknown destination must produce a retrieval warning")
	}
}

func TestIntegrationStalePlaceSelection(t *testing.T) {
	store, _ := newIntegrationStore(t)
	ingestor := newTestIngestor(t, store)
	ctx := context.Background()

	if _, err := ingestor.IngestDestination(ctx, IngestRequest{DestinationName: "Vienna", CountryCode: "AT"}); err != nil {
		t.Fatalf("IngestDestination() error = %v", err)
	}
	// Freshly refreshed records are not stale.
	stale, err := store.ListStalePlaces(ctx, nil, 30, 100, 0)
	if err != nil {
		t.Fatalf("ListStalePlaces() error = %v", err)
	}
	if len(stale) != 0 {
		t.Fatalf("just-refreshed places must not be stale, got %d", len(stale))
	}

	// Age them past the window.
	if _, err := store.db.Exec(ctx, `UPDATE travel_places
      SET last_provider_refresh_at = NOW() - INTERVAL '90 days'`); err != nil {
		t.Fatalf("age places: %v", err)
	}
	stale, err = store.ListStalePlaces(ctx, nil, 30, 100, 0)
	if err != nil {
		t.Fatalf("ListStalePlaces() error = %v", err)
	}
	if len(stale) == 0 {
		t.Fatal("aged places must be selected for refresh")
	}
}

func TestIntegrationPlaceReviewFilters(t *testing.T) {
	store, _ := newIntegrationStore(t)
	ingestor := newTestIngestor(t, store)
	ctx := context.Background()

	if _, err := ingestor.IngestDestination(ctx, IngestRequest{DestinationName: "Rome", CountryCode: "IT"}); err != nil {
		t.Fatalf("IngestDestination() error = %v", err)
	}

	// The Rome fixtures include a record without coordinates.
	missing, err := store.ListPlacesForReview(ctx, PlaceReviewFilters{Filter: "missing_coordinates", Limit: 50})
	if err != nil {
		t.Fatalf("ListPlacesForReview(missing_coordinates) error = %v", err)
	}
	if len(missing) == 0 {
		t.Fatal("expected the coordinate-less Rome fixture to be listed")
	}
	for _, place := range missing {
		if place.HasCoordinates {
			t.Fatalf("filter leaked a record with coordinates: %s", place.CanonicalName)
		}
	}

	for _, filter := range []string{"low_quality", "stale", "needs_review", "duplicates", "rejected", ""} {
		if _, err := store.ListPlacesForReview(ctx, PlaceReviewFilters{Filter: filter, Limit: 10}); err != nil {
			t.Fatalf("ListPlacesForReview(%q) error = %v", filter, err)
		}
	}
}
