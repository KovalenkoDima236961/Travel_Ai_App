package routes

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

// quotaStubStore is a minimal thread-safe QuotaStore honoring a per-provider
// daily quota, used to drive the guarded provider into a limited state.
type quotaStubStore struct {
	mu   sync.Mutex
	used map[string]int64
}

func newQuotaStubStore() *quotaStubStore { return &quotaStubStore{used: map[string]int64{}} }

func (s *quotaStubStore) Reserve(_ context.Context, provider, _ string, _ time.Time, cost, quota int64) (providerlimits.Reservation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	used := s.used[provider]
	if quota > 0 && used+cost > quota {
		return providerlimits.Reservation{Allowed: false, QuotaExceeded: true, DailyQuota: quota, DailyUsed: used}, nil
	}
	s.used[provider] = used + cost
	return providerlimits.Reservation{Allowed: true, DailyQuota: quota, DailyUsed: used + cost}, nil
}
func (s *quotaStubStore) IncrementBlocked(context.Context, string, string, time.Time, int64) error {
	return nil
}
func (s *quotaStubStore) IncrementFallback(context.Context, string, string, time.Time, int64) error {
	return nil
}
func (s *quotaStubStore) ListUsageByDate(context.Context, time.Time) ([]providerlimits.OperationUsage, error) {
	return nil, nil
}
func (s *quotaStubStore) ListUsageByProvider(context.Context, string, time.Time, time.Time) ([]providerlimits.OperationUsage, error) {
	return nil, nil
}
func (s *quotaStubStore) ResetProviderForDate(context.Context, string, time.Time) error { return nil }

type stubRealRouteProvider struct{}

func (stubRealRouteProvider) EstimateRoute(context.Context, entity.RouteEstimateRequest) (*entity.RouteEstimate, error) {
	return &entity.RouteEstimate{Mode: "walking", Provider: "ors", DistanceKm: 1, DurationMinutes: 1}, nil
}

func guardWithQuota(quota int64) *providerlimits.Guard {
	return providerlimits.NewGuard(providerlimits.GuardParams{
		Enabled: true,
		Store:   newQuotaStubStore(),
		Limits: []providerlimits.ProviderLimit{
			{Category: providerlimits.CategoryRoutes, Provider: "ors", RatePerMinute: 0, DailyQuota: quota},
		},
	})
}

func sampleRouteRequest() entity.RouteEstimateRequest {
	return entity.RouteEstimateRequest{Mode: "walking", Stops: []entity.RouteStop{
		{Name: "A", Latitude: 1, Longitude: 1},
		{Name: "B", Latitude: 2, Longitude: 2},
	}}
}

func TestGuardedRouteFallsBackToMockWhenLimited(t *testing.T) {
	guard := guardWithQuota(1)
	provider := newGuardedRouteProvider(guard, "ors", stubRealRouteProvider{}, NewMockRouteProvider(), true, zap.NewNop())

	first, err := provider.EstimateRoute(context.Background(), sampleRouteRequest())
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if first.FallbackUsed || first.Provider != "ors" {
		t.Fatalf("first call should hit the real provider, got %+v", first)
	}

	second, err := provider.EstimateRoute(context.Background(), sampleRouteRequest())
	if err != nil {
		t.Fatalf("second call should fall back, got error: %v", err)
	}
	if !second.FallbackUsed || second.Provider != "mock" {
		t.Fatalf("second call should use mock fallback, got %+v", second)
	}
}

func TestGuardedRouteReturnsControlledErrorWhenFallbackDisabled(t *testing.T) {
	guard := guardWithQuota(1)
	provider := newGuardedRouteProvider(guard, "ors", stubRealRouteProvider{}, NewMockRouteProvider(), false, zap.NewNop())

	if _, err := provider.EstimateRoute(context.Background(), sampleRouteRequest()); err != nil {
		t.Fatalf("first call: %v", err)
	}
	_, err := provider.EstimateRoute(context.Background(), sampleRouteRequest())
	var limitErr *providerlimits.LimitError
	if !errors.As(err, &limitErr) {
		t.Fatalf("expected a controlled LimitError, got %v", err)
	}
	if limitErr.Code != providerlimits.CodeQuotaExceeded {
		t.Fatalf("expected provider_quota_exceeded, got %s", limitErr.Code)
	}
}

func TestGuardedRouteNilGuardIsPassthrough(t *testing.T) {
	provider := newGuardedRouteProvider(nil, "ors", stubRealRouteProvider{}, NewMockRouteProvider(), true, zap.NewNop())
	estimate, err := provider.EstimateRoute(context.Background(), sampleRouteRequest())
	if err != nil {
		t.Fatalf("passthrough call: %v", err)
	}
	if estimate.Provider != "ors" {
		t.Fatalf("nil guard should pass through to the real provider, got %+v", estimate)
	}
}
