package routes

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/cache"
)

// countingRouteProvider records how many times the underlying provider is hit
// and returns a configurable estimate.
type countingRouteProvider struct {
	calls    int
	estimate entity.RouteEstimate
}

func (p *countingRouteProvider) EstimateRoute(context.Context, entity.RouteEstimateRequest) (*entity.RouteEstimate, error) {
	p.calls++
	result := p.estimate
	return &result, nil
}

func TestCachingRouteProviderHitAvoidsSecondCall(t *testing.T) {
	counter := &countingRouteProvider{estimate: entity.RouteEstimate{Provider: "ors", Mode: "walking", DistanceKm: 1.2, DurationMinutes: 15}}
	provider := newCachingRouteProvider("ors", counter, cache.New(0), time.Minute, zap.NewNop())

	req := orsTwoStopRequest()
	if _, err := provider.EstimateRoute(context.Background(), req); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := provider.EstimateRoute(context.Background(), req); err != nil {
		t.Fatalf("second call: %v", err)
	}

	if counter.calls != 1 {
		t.Fatalf("expected underlying provider called once, got %d", counter.calls)
	}
}

func TestCachingRouteProviderDifferentRequestMisses(t *testing.T) {
	counter := &countingRouteProvider{estimate: entity.RouteEstimate{Provider: "ors", Mode: "walking", DistanceKm: 1.2, DurationMinutes: 15}}
	provider := newCachingRouteProvider("ors", counter, cache.New(0), time.Minute, zap.NewNop())

	first := orsTwoStopRequest()
	second := orsTwoStopRequest()
	second.Stops[1].Latitude = 41.95 // different coordinates -> different key

	_, _ = provider.EstimateRoute(context.Background(), first)
	_, _ = provider.EstimateRoute(context.Background(), second)

	if counter.calls != 2 {
		t.Fatalf("expected two provider calls for distinct requests, got %d", counter.calls)
	}
}

func TestCachingRouteProviderDoesNotCacheFallbackResults(t *testing.T) {
	counter := &countingRouteProvider{estimate: entity.RouteEstimate{Provider: "mock", Mode: "walking", DistanceKm: 1.2, DurationMinutes: 15, FallbackUsed: true}}
	provider := newCachingRouteProvider("ors", counter, cache.New(0), time.Minute, zap.NewNop())

	req := orsTwoStopRequest()
	_, _ = provider.EstimateRoute(context.Background(), req)
	_, _ = provider.EstimateRoute(context.Background(), req)

	if counter.calls != 2 {
		t.Fatalf("expected fallback results not to be cached, got %d calls", counter.calls)
	}
}

func TestNewWiresCacheOnlyWhenEnabled(t *testing.T) {
	// Cache disabled: the selector returns the raw provider, so every request
	// reaches the provider (no memoization layer).
	disabled, err := New(&config.Config{RouteProvider: config.RouteProviderConfig{
		Provider:     config.RouteProviderMock,
		CacheEnabled: false,
	}}, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("new (cache disabled): %v", err)
	}
	if _, ok := disabled.(*MockRouteProvider); !ok {
		t.Fatalf("expected raw mock provider when cache disabled, got %T", disabled)
	}

	enabled, err := New(&config.Config{RouteProvider: config.RouteProviderConfig{
		Provider:        config.RouteProviderMock,
		CacheEnabled:    true,
		CacheTTLSeconds: 60,
	}}, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("new (cache enabled): %v", err)
	}
	if _, ok := enabled.(*cachingRouteProvider); !ok {
		t.Fatalf("expected caching wrapper when cache enabled, got %T", enabled)
	}
}

func TestRouteCacheKeyStable(t *testing.T) {
	req := entity.RouteEstimateRequest{
		Mode: "walking",
		Stops: []entity.RouteStop{
			{Name: "Colosseum", Latitude: 41.8902, Longitude: 12.4922},
			{Name: "Trevi Fountain", Latitude: 41.8925, Longitude: 12.4853},
		},
	}
	want := "route:ors:walking:41.89020,12.49220|41.89250,12.48530"
	if got := routeCacheKey("ors", req); got != want {
		t.Fatalf("unexpected cache key:\n got %q\nwant %q", got, want)
	}
}
