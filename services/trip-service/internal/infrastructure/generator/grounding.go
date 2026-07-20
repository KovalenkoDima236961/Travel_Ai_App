package generator

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge"
)

// Grounding attaches quality-filtered knowledge to the generation request.
// This is the point where data quality becomes model behaviour: without it the
// exclusion rules in the knowledge store would never run on a real request.
//
// Retrieval is fail-open. A knowledge store problem must degrade generation to
// ungrounded output — which is the pre-existing behaviour — rather than failing
// a user's itinerary request.

// GroundingRetriever is the subset of the knowledge store the generator needs.
type GroundingRetriever interface {
	RetrieveGrounding(ctx context.Context, query knowledge.GroundingQuery) (knowledge.GroundingResult, error)
}

// aiPlanningGroundingContext matches the GroundingContext schema in
// AI Planning Service. Place entries carry the quality metadata the prompt
// builder uses to decide how strongly an item may be asserted.
type aiPlanningGroundingContext struct {
	Status            string                         `json:"status"`
	Destination       *aiPlanningGroundingDest       `json:"destination,omitempty"`
	Places            []knowledge.GroundingPlace     `json:"places"`
	RetrievalWarnings []string                       `json:"retrievalWarnings,omitempty"`
	Coverage          *knowledge.DestinationCoverage `json:"coverage,omitempty"`
	Attributions      []string                       `json:"attributions,omitempty"`
	GeneratedAt       string                         `json:"generatedAt,omitempty"`
}

type aiPlanningGroundingDest struct {
	ID            string `json:"id,omitempty"`
	CanonicalName string `json:"canonicalName"`
	CountryCode   string `json:"countryCode,omitempty"`
}

// maxGroundingPlaces caps how much evidence is sent. Prompt space is finite and
// the highest-quality records are ordered first by retrieval.
const maxGroundingPlaces = 20

// buildGroundingContext retrieves grounding for a destination. It returns nil
// when no retriever is configured or retrieval yields nothing usable, which
// leaves the request exactly as it was before grounding existed.
func buildGroundingContext(
	ctx context.Context,
	retriever GroundingRetriever,
	logger *zap.Logger,
	destination string,
	countryCode string,
) *aiPlanningGroundingContext {
	if retriever == nil || destination == "" {
		return nil
	}

	result, err := retriever.RetrieveGrounding(ctx, knowledge.GroundingQuery{
		DestinationName: destination,
		CountryCode:     countryCode,
		Limit:           maxGroundingPlaces,
		// Weak records are included but clearly marked, so the model can use
		// them when coverage is thin while flagging the item for review.
		IncludeWeak: true,
		Thresholds:  knowledge.DefaultThresholds(),
	})
	if err != nil {
		logger.Warn("grounding retrieval failed; continuing without grounding context",
			zap.String("destination", destination),
			zap.Error(err),
		)
		return nil
	}
	if len(result.Places) == 0 {
		logger.Info("no grounding places passed the quality threshold",
			zap.String("destination", destination),
			zap.String("coverageStatus", result.Coverage.Status),
			zap.Int("excludedCount", result.ExcludedCount),
		)
		return nil
	}

	logger.Info("grounding context attached",
		zap.String("destination", destination),
		zap.Int("strongCount", result.StrongCount),
		zap.Int("weakCount", result.WeakCount),
		zap.Int("excludedCount", result.ExcludedCount),
		zap.String("coverageStatus", result.Coverage.Status),
	)

	coverage := result.Coverage
	context := &aiPlanningGroundingContext{
		Status:            result.Status,
		Places:            result.Places,
		RetrievalWarnings: result.RetrievalWarnings,
		Coverage:          &coverage,
		Attributions:      result.Attributions,
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
	}
	if result.DestinationName != "" {
		context.Destination = &aiPlanningGroundingDest{
			ID:            result.DestinationID,
			CanonicalName: result.DestinationName,
			CountryCode:   countryCode,
		}
	}
	return context
}
