package generator

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge"
)

type stubRetriever struct {
	result knowledge.GroundingResult
	err    error
	calls  int
	query  knowledge.GroundingQuery
}

func (s *stubRetriever) RetrieveGrounding(_ context.Context, query knowledge.GroundingQuery) (knowledge.GroundingResult, error) {
	s.calls++
	s.query = query
	return s.result, s.err
}

func groundedResult() knowledge.GroundingResult {
	return knowledge.GroundingResult{
		Status:          "available",
		DestinationID:   "11111111-1111-1111-1111-111111111111",
		DestinationName: "Rome",
		StrongCount:     1,
		WeakCount:       1,
		Places: []knowledge.GroundingPlace{
			{
				ID: "place-1", CanonicalName: "Colosseum", Category: "landmark",
				QualityScore: 0.91, ReviewStatus: knowledge.ReviewStatusApproved,
				GroundingStrength: knowledge.GroundingStrengthStrong,
			},
			{
				ID: "place-2", CanonicalName: "Testaccio Market", Category: "market",
				QualityScore: 0.60, ReviewStatus: knowledge.ReviewStatusNeedsReview,
				GroundingStrength: knowledge.GroundingStrengthWeak,
			},
		},
		Coverage:          knowledge.DestinationCoverage{Status: "available", PlaceCount: 2},
		RetrievalWarnings: []string{"1 place record(s) are weak grounding and need review."},
		Attributions:      []string{"Test Attribution"},
	}
}

func TestBuildGroundingContextAttachesQualityMetadata(t *testing.T) {
	retriever := &stubRetriever{result: groundedResult()}
	context := buildGroundingContext(context.Background(), retriever, zap.NewNop(), "Rome", "IT")

	if context == nil {
		t.Fatal("expected a grounding context")
	}
	if context.Status != "available" || len(context.Places) != 2 {
		t.Fatalf("unexpected grounding context: %+v", context)
	}
	if context.Destination == nil || context.Destination.CanonicalName != "Rome" {
		t.Fatalf("destination was not attached: %+v", context.Destination)
	}
	// The quality fields are what let the prompt distinguish strong from weak.
	if context.Places[0].GroundingStrength != knowledge.GroundingStrengthStrong {
		t.Fatal("strong record lost its grounding strength")
	}
	if context.Places[1].GroundingStrength != knowledge.GroundingStrengthWeak {
		t.Fatal("weak record lost its grounding strength")
	}
	if context.Coverage == nil || context.Coverage.Status != "available" {
		t.Fatalf("coverage was not attached: %+v", context.Coverage)
	}
	if len(context.Attributions) != 1 {
		t.Fatalf("attributions must be forwarded, got %v", context.Attributions)
	}
}

// Weak records are requested deliberately so thin destinations still get help,
// but they arrive marked rather than silently promoted.
func TestBuildGroundingContextRequestsWeakRecordsMarked(t *testing.T) {
	retriever := &stubRetriever{result: groundedResult()}
	buildGroundingContext(context.Background(), retriever, zap.NewNop(), "Rome", "IT")

	if !retriever.query.IncludeWeak {
		t.Fatal("grounding retrieval must request weak records so they can be marked, not dropped silently")
	}
	if retriever.query.Limit != maxGroundingPlaces {
		t.Fatalf("grounding retrieval must cap results, got limit %d", retriever.query.Limit)
	}
	if retriever.query.CountryCode != "IT" {
		t.Fatalf("country code must be forwarded, got %q", retriever.query.CountryCode)
	}
}

// A knowledge store failure must not fail a user's generation request.
func TestBuildGroundingContextFailsOpen(t *testing.T) {
	retriever := &stubRetriever{err: errors.New("database unavailable")}
	if context := buildGroundingContext(context.Background(), retriever, zap.NewNop(), "Rome", ""); context != nil {
		t.Fatal("a retrieval failure must yield no grounding context, not a partial one")
	}
}

func TestBuildGroundingContextSkipsWhenNothingPassesQuality(t *testing.T) {
	retriever := &stubRetriever{result: knowledge.GroundingResult{
		Status: "unavailable", DestinationName: "Rome", ExcludedCount: 5,
	}}
	if context := buildGroundingContext(context.Background(), retriever, zap.NewNop(), "Rome", ""); context != nil {
		t.Fatal("no qualifying places must mean no grounding context")
	}
}

func TestBuildGroundingContextSkippedWithoutRetriever(t *testing.T) {
	if context := buildGroundingContext(context.Background(), nil, zap.NewNop(), "Rome", ""); context != nil {
		t.Fatal("without a retriever the request must be left ungrounded")
	}
}

func TestBuildGroundingContextRequiresDestination(t *testing.T) {
	retriever := &stubRetriever{result: groundedResult()}
	if context := buildGroundingContext(context.Background(), retriever, zap.NewNop(), "", ""); context != nil {
		t.Fatal("an empty destination must not trigger retrieval")
	}
	if retriever.calls != 0 {
		t.Fatal("an empty destination must not reach the knowledge store")
	}
}

// The generator must send grounding on the wire, otherwise the whole quality
// pipeline is invisible to the model.
func TestGenerateRequestCarriesGroundingContext(t *testing.T) {
	generator := &AIPlanningHTTPGenerator{logger: zap.NewNop()}
	if generator.grounding != nil {
		t.Fatal("grounding must be opt-in")
	}
	retriever := &stubRetriever{result: groundedResult()}
	generator = generator.WithGrounding(retriever)

	payload := aiPlanningGenerateRequest{Destination: "Rome"}
	payload.GroundingContext = buildGroundingContext(context.Background(), generator.grounding,
		generator.logger, payload.Destination, "")

	if payload.GroundingContext == nil {
		t.Fatal("the generate request must carry grounding context when a retriever is wired")
	}
	if len(payload.GroundingContext.Places) == 0 {
		t.Fatal("grounding context must contain places")
	}
}
