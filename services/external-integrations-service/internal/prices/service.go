package prices

import (
	"context"
	"time"

	"go.uber.org/zap"

	extobs "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/observability"
)

type Service struct {
	provider PriceProvider
	log      *zap.Logger
}

func NewService(provider PriceProvider, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{provider: provider, log: log}
}

func (s *Service) EstimatePrice(ctx context.Context, input PriceEstimateInput) (*PriceEstimateResult, error) {
	start := time.Now()
	result, err := s.provider.EstimatePrice(ctx, input)
	if err != nil {
		extobs.RecordProviderRequest("unknown", "price_estimate", "error", time.Since(start))
		extobs.RecordProviderFailure("unknown", "price_estimate", "provider_error")
		s.log.Warn("price_estimate",
			zap.String("destination", input.Destination),
			zap.String("place", safePlaceName(input.Place)),
			zap.Bool("success", false),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err),
		)
		return nil, err
	}
	if result == nil {
		result = noMatch("No likely paid ticket price found", 0.2)
	}
	extobs.RecordProviderRequest(result.Provider, "price_estimate", "success", time.Since(start))
	if result.FallbackUsed {
		extobs.RecordProviderFallback(result.Provider, "price_estimate", "mock")
	}
	s.log.Info("price_estimate",
		zap.String("destination", input.Destination),
		zap.String("place", safePlaceName(input.Place)),
		zap.String("provider", result.Provider),
		zap.Bool("matched", result.Matched),
		zap.Float64("matchConfidence", result.MatchConfidence),
		zap.Bool("success", true),
		zap.Duration("duration", time.Since(start)),
	)
	return result, nil
}

func safePlaceName(place *PricePlace) string {
	if place == nil {
		return ""
	}
	return place.Name
}
