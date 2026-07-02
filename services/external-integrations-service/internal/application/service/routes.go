package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
	extobs "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/observability"
)

// RouteProvider is implemented by each route-estimation provider. v1 ships only
// a deterministic mock; real providers (OSRM, Mapbox, Google) can be added later
// behind this port without touching the handler or service.
type RouteProvider interface {
	EstimateRoute(ctx context.Context, req entity.RouteEstimateRequest) (*entity.RouteEstimate, error)
}

// RoutesService contains route-estimation use cases over the configured
// provider. The HTTP handler is responsible for request validation; this layer
// only delegates and emits the structured estimate log line.
type RoutesService struct {
	provider RouteProvider
	log      *zap.Logger
}

func NewRoutesService(provider RouteProvider, log *zap.Logger) *RoutesService {
	if log == nil {
		log = zap.NewNop()
	}
	return &RoutesService{provider: provider, log: log}
}

// EstimateRoute delegates to the provider and logs a single structured line per
// estimate. Stop names are considered low-sensitivity; the full coordinate
// payload is intentionally never logged.
func (s *RoutesService) EstimateRoute(ctx context.Context, req entity.RouteEstimateRequest) (*entity.RouteEstimate, error) {
	start := time.Now()

	estimate, err := s.provider.EstimateRoute(ctx, req)
	if err != nil {
		extobs.RecordProviderRequest("unknown", "route_estimate", "error", time.Since(start))
		extobs.RecordProviderFailure("unknown", "route_estimate", "provider_error")
		s.log.Warn("route_estimate",
			zap.String("mode", req.Mode),
			zap.Int("stop_count", len(req.Stops)),
			zap.Duration("duration_ms", time.Since(start)),
			zap.Bool("success", false),
			zap.Error(err),
		)
		return nil, err
	}
	extobs.RecordProviderRequest(estimate.Provider, "route_estimate", "success", time.Since(start))
	if estimate.FallbackUsed {
		extobs.RecordProviderFallback(estimate.Provider, "route_estimate", "mock")
	}

	s.log.Info("route_estimate",
		zap.String("action", "route_estimate"),
		zap.String("provider", estimate.Provider),
		zap.String("mode", estimate.Mode),
		zap.Int("stop_count", len(req.Stops)),
		zap.Float64("distance_km", estimate.DistanceKm),
		zap.Int("duration_minutes", estimate.DurationMinutes),
		zap.Duration("duration_ms", time.Since(start)),
		zap.Bool("success", true),
	)

	return estimate, nil
}
