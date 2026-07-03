package providerlimits

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// fakeStore is a thread-safe in-memory QuotaStore that mirrors the atomic
// reservation contract of the Postgres store: reservations are serialized so
// concurrent reservations can never exceed the quota.
type fakeStore struct {
	mu            sync.Mutex
	used          map[string]int64
	blocked       map[string]int64
	fallback      map[string]int64
	reserveErr    error
	blockedCalls  int
	fallbackCalls int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		used:     map[string]int64{},
		blocked:  map[string]int64{},
		fallback: map[string]int64{},
	}
}

func key(provider string, date time.Time) string {
	return provider + "|" + date.UTC().Format("2006-01-02")
}

func (s *fakeStore) Reserve(_ context.Context, provider, _ string, date time.Time, cost, quota int64) (Reservation, error) {
	if s.reserveErr != nil {
		return Reservation{}, s.reserveErr
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k := key(provider, date)
	used := s.used[k]
	if quota > 0 && used+cost > quota {
		s.blocked[k] += cost
		return Reservation{Allowed: false, QuotaExceeded: true, DailyQuota: quota, DailyUsed: used, DailyRemaining: remaining(quota, used)}, nil
	}
	s.used[k] = used + cost
	return Reservation{Allowed: true, DailyQuota: quota, DailyUsed: used + cost, DailyRemaining: remaining(quota, used+cost)}, nil
}

func (s *fakeStore) IncrementBlocked(_ context.Context, provider, _ string, date time.Time, amount int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blocked[key(provider, date)] += amount
	s.blockedCalls++
	return nil
}

func (s *fakeStore) IncrementFallback(_ context.Context, provider, _ string, date time.Time, amount int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.fallback[key(provider, date)] += amount
	s.fallbackCalls++
	return nil
}

func (s *fakeStore) ListUsageByDate(context.Context, time.Time) ([]OperationUsage, error) {
	return nil, nil
}
func (s *fakeStore) ListUsageByProvider(context.Context, string, time.Time, time.Time) ([]OperationUsage, error) {
	return nil, nil
}
func (s *fakeStore) ResetProviderForDate(context.Context, string, time.Time) error { return nil }

func newTestGuard(store QuotaStore, enabled, failOpen bool, limit ProviderLimit) *Guard {
	return NewGuard(GuardParams{
		Enabled:  enabled,
		FailOpen: failOpen,
		Store:    store,
		Limits:   []ProviderLimit{limit},
	})
}

func routeLimit(rate int, quota int64) ProviderLimit {
	return ProviderLimit{Category: CategoryRoutes, Provider: "ors", RatePerMinute: rate, Burst: 100, DailyQuota: quota}
}

func TestGuardDisabledAllowsWithoutStore(t *testing.T) {
	store := newFakeStore()
	g := newTestGuard(store, false, false, routeLimit(0, 1))
	d, err := g.CheckAndReserve(context.Background(), ProviderCall{Provider: "ors", Operation: OpRouteEstimate, Cost: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !d.Allowed || d.Reason != ReasonDisabled {
		t.Fatalf("disabled guard should allow, got %+v", d)
	}
	if len(store.used) != 0 {
		t.Fatal("disabled guard must not touch the store")
	}
}

func TestGuardAllowedConsumesQuota(t *testing.T) {
	store := newFakeStore()
	g := newTestGuard(store, true, false, routeLimit(0, 10))
	d, _ := g.CheckAndReserve(context.Background(), ProviderCall{Provider: "ors", Operation: OpRouteEstimate, Cost: 1})
	if !d.Allowed {
		t.Fatalf("expected allowed, got %+v", d)
	}
	if d.DailyUsed != 1 || d.DailyRemaining != 9 {
		t.Fatalf("expected used=1 remaining=9, got %+v", d)
	}
}

func TestGuardQuotaExceededReturnsControlledDecision(t *testing.T) {
	store := newFakeStore()
	g := newTestGuard(store, true, false, routeLimit(0, 1))
	_, _ = g.CheckAndReserve(context.Background(), ProviderCall{Provider: "ors", Operation: OpRouteEstimate, Cost: 1})
	d, _ := g.CheckAndReserve(context.Background(), ProviderCall{Provider: "ors", Operation: OpRouteEstimate, Cost: 1})
	if d.Allowed || !d.QuotaExceeded {
		t.Fatalf("expected quota exceeded, got %+v", d)
	}
	if le := LimitErrorFrom(d); le == nil || le.Code != CodeQuotaExceeded {
		t.Fatalf("expected provider_quota_exceeded error, got %+v", le)
	}
}

func TestGuardRateLimitedReturnsControlledDecision(t *testing.T) {
	store := newFakeStore()
	g := newTestGuard(store, true, false, ProviderLimit{Category: CategoryRoutes, Provider: "ors", RatePerMinute: 1, Burst: 1, DailyQuota: 0})
	if d, _ := g.CheckAndReserve(context.Background(), ProviderCall{Provider: "ors", Operation: OpRouteEstimate, Cost: 1}); !d.Allowed {
		t.Fatalf("first call should be allowed, got %+v", d)
	}
	d, _ := g.CheckAndReserve(context.Background(), ProviderCall{Provider: "ors", Operation: OpRouteEstimate, Cost: 1})
	if d.Allowed || !d.Limited {
		t.Fatalf("expected rate limited, got %+v", d)
	}
	if d.RetryAfterSeconds < 1 {
		t.Fatalf("expected positive retry-after, got %d", d.RetryAfterSeconds)
	}
	if le := LimitErrorFrom(d); le == nil || le.Code != CodeRateLimited {
		t.Fatalf("expected provider_rate_limited error, got %+v", le)
	}
	if store.blockedCalls == 0 {
		t.Fatal("rate-limit denial should record a blocked count")
	}
}

func TestGuardStoreUnavailableFailOpen(t *testing.T) {
	store := newFakeStore()
	store.reserveErr = errors.New("db down")
	g := newTestGuard(store, true, true, routeLimit(0, 10))
	d, _ := g.CheckAndReserve(context.Background(), ProviderCall{Provider: "ors", Operation: OpRouteEstimate, Cost: 1})
	if !d.Allowed || d.Reason != ReasonFailOpen {
		t.Fatalf("fail-open should allow on store error, got %+v", d)
	}
}

func TestGuardStoreUnavailableFailClosed(t *testing.T) {
	store := newFakeStore()
	store.reserveErr = errors.New("db down")
	g := newTestGuard(store, true, false, routeLimit(0, 10))
	d, _ := g.CheckAndReserve(context.Background(), ProviderCall{Provider: "ors", Operation: OpRouteEstimate, Cost: 1})
	if d.Allowed || !d.Unavailable {
		t.Fatalf("fail-closed should block on store error, got %+v", d)
	}
	if le := LimitErrorFrom(d); le == nil || le.Code != CodeLimitsUnavailable {
		t.Fatalf("expected provider_limits_unavailable error, got %+v", le)
	}
}

func TestGuardRecordFallback(t *testing.T) {
	store := newFakeStore()
	g := newTestGuard(store, true, false, routeLimit(0, 10))
	g.RecordFallback(context.Background(), "ors", OpRouteEstimate, ReasonQuotaExceeded)
	if store.fallbackCalls != 1 {
		t.Fatalf("expected fallback recorded once, got %d", store.fallbackCalls)
	}
}

func TestGuardConcurrentReservationsDoNotExceedQuota(t *testing.T) {
	store := newFakeStore()
	const quota = 20
	g := newTestGuard(store, true, false, routeLimit(0, quota))

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			d, _ := g.CheckAndReserve(context.Background(), ProviderCall{Provider: "ors", Operation: OpRouteEstimate, Cost: 1})
			if d.Allowed {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if allowed != quota {
		t.Fatalf("expected exactly %d allowed reservations, got %d", quota, allowed)
	}
}
