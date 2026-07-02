package routes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/cache"
	extobs "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/observability"
)

// routeCacheMaxEntries bounds the in-memory route cache.
const routeCacheMaxEntries = 2048

// cachingRouteProvider memoises successful estimates from the wrapped provider.
// It caches only real, non-fallback results so a transient outage does not pin a
// degraded mock answer for the (long) route TTL. Errors are never cached.
type cachingRouteProvider struct {
	providerName string
	next         service.RouteProvider
	cache        *cache.TTLCache
	ttl          time.Duration
	log          *zap.Logger
}

func newCachingRouteProvider(
	providerName string,
	next service.RouteProvider,
	store *cache.TTLCache,
	ttl time.Duration,
	log *zap.Logger,
) service.RouteProvider {
	if log == nil {
		log = zap.NewNop()
	}
	return &cachingRouteProvider{
		providerName: strings.ToLower(strings.TrimSpace(providerName)),
		next:         next,
		cache:        store,
		ttl:          ttl,
		log:          log,
	}
}

func (p *cachingRouteProvider) EstimateRoute(ctx context.Context, req entity.RouteEstimateRequest) (*entity.RouteEstimate, error) {
	key := routeCacheKey(p.providerName, req)

	if cached, ok := p.cache.Get(key); ok {
		if estimate, ok := cached.(entity.RouteEstimate); ok {
			extobs.RecordProviderCacheHit(p.providerName, "route_estimate")
			p.log.Info("route cache lookup",
				zap.String("endpoint", "route"),
				zap.String("provider", p.providerName),
				zap.Bool("cacheHit", true),
			)
			result := estimate
			return &result, nil
		}
	}

	estimate, err := p.next.EstimateRoute(ctx, req)
	if err != nil {
		return nil, err
	}
	extobs.RecordProviderCacheMiss(p.providerName, "route_estimate")

	p.log.Info("route cache lookup",
		zap.String("endpoint", "route"),
		zap.String("provider", p.providerName),
		zap.Bool("cacheHit", false),
	)

	if !estimate.FallbackUsed {
		p.cache.Set(key, *estimate, p.ttl)
	}
	return estimate, nil
}

// routeCacheKey is provider + mode + 5-decimal-rounded coordinates so nearby
// requests share an entry without large accuracy loss.
func routeCacheKey(provider string, req entity.RouteEstimateRequest) string {
	mode := strings.ToLower(strings.TrimSpace(req.Mode))
	points := make([]string, 0, len(req.Stops))
	for _, stop := range req.Stops {
		points = append(points, fmt.Sprintf("%.5f,%.5f", stop.Latitude, stop.Longitude))
	}
	return fmt.Sprintf("route:%s:%s:%s", provider, mode, strings.Join(points, "|"))
}
