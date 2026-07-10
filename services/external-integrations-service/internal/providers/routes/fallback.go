package routes

import (
	"context"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

// fallbackRouteProvider tries the primary (real) provider and, on failure, falls
// back to the mock provider so local development and transient outages keep
// working. It mirrors the place-provider fallback model.
type fallbackRouteProvider struct {
	providerName         string
	fallbackProviderName string
	primary              service.RouteProvider
	fallback             service.RouteProvider
	log                  *zap.Logger
}

func newFallbackRouteProvider(
	providerName string,
	primary service.RouteProvider,
	fallback service.RouteProvider,
	log *zap.Logger,
) service.RouteProvider {
	if log == nil {
		log = zap.NewNop()
	}
	return &fallbackRouteProvider{
		providerName:         providerName,
		fallbackProviderName: "mock",
		primary:              primary,
		fallback:             fallback,
		log:                  log,
	}
}

func (p *fallbackRouteProvider) EstimateRoute(ctx context.Context, req entity.RouteEstimateRequest) (*entity.RouteEstimate, error) {
	estimate, err := p.primary.EstimateRoute(ctx, req)
	if err == nil {
		return estimate, nil
	}

	p.log.Warn("route provider fallback used",
		zap.String("action", "route_estimate"),
		zap.String("provider", p.providerName),
		zap.String("fallbackProvider", p.fallbackProviderName),
		zap.Bool("fallbackUsed", true),
		zap.String("mode", req.Mode),
		zap.Int("stopCount", len(req.Stops)),
		zap.String("errorType", providerErrorKind(err)),
		zap.Error(err),
	)

	fallbackEstimate, fallbackErr := p.fallback.EstimateRoute(ctx, req)
	if fallbackErr != nil {
		p.log.Warn("route provider fallback failed",
			zap.String("action", "route_estimate"),
			zap.String("provider", p.providerName),
			zap.String("fallbackProvider", p.fallbackProviderName),
			zap.String("errorType", providerErrorKind(fallbackErr)),
			zap.Error(fallbackErr),
		)
		// Return the original provider error so the handler reports a safe
		// provider-unavailable response.
		return nil, err
	}

	// The response provider reflects the actual source that answered.
	fallbackEstimate.Provider = p.fallbackProviderName
	fallbackEstimate.FallbackUsed = true
	fallbackEstimate.Warnings = append([]string{
		"Live routing was unavailable or unsupported for this mode; returned a deterministic estimate.",
	}, fallbackEstimate.Warnings...)
	return fallbackEstimate, nil
}
