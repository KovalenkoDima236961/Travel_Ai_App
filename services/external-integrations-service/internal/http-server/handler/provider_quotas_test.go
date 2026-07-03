package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

type stubStore struct {
	rows      []providerlimits.OperationUsage
	resetHits int
}

func (s *stubStore) Reserve(context.Context, string, string, time.Time, int64, int64) (providerlimits.Reservation, error) {
	return providerlimits.Reservation{Allowed: true}, nil
}
func (s *stubStore) IncrementBlocked(context.Context, string, string, time.Time, int64) error {
	return nil
}
func (s *stubStore) IncrementFallback(context.Context, string, string, time.Time, int64) error {
	return nil
}
func (s *stubStore) ListUsageByDate(context.Context, time.Time) ([]providerlimits.OperationUsage, error) {
	return s.rows, nil
}
func (s *stubStore) ListUsageByProvider(context.Context, string, time.Time, time.Time) ([]providerlimits.OperationUsage, error) {
	return s.rows, nil
}
func (s *stubStore) ResetProviderForDate(context.Context, string, time.Time) error {
	s.resetHits++
	return nil
}

func newQuotaTestHandler(env string, store providerlimits.QuotaStore) *ProviderQuotaOpsHandler {
	cfg := &config.Config{Env: env}
	guard := providerlimits.NewGuard(providerlimits.GuardParams{
		Enabled: true,
		Store:   store,
		Limits: []providerlimits.ProviderLimit{
			{Category: providerlimits.CategoryRoutes, Provider: "ors", RatePerMinute: 30, DailyQuota: 1500},
		},
	})
	return NewProviderQuotaOpsHandler(cfg, guard, zap.NewNop())
}

func serve(h *ProviderQuotaOpsHandler, method, target string) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	req := httptest.NewRequest(method, target, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func TestProviderQuotasListAggregatesUsage(t *testing.T) {
	store := &stubStore{rows: []providerlimits.OperationUsage{
		{Provider: "ors", Operation: providerlimits.OpRouteEstimate, UsedCount: 5, BlockedCount: 1, FallbackCount: 2},
	}}
	h := newQuotaTestHandler("local", store)

	rec := serve(h, http.MethodGet, "/ops/providers/quotas")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp ProviderQuotasResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !resp.ResetAllowed {
		t.Fatal("reset should be allowed outside production")
	}
	var routes *ProviderQuotaSummary
	for i := range resp.Providers {
		if resp.Providers[i].Category == providerlimits.CategoryRoutes {
			routes = &resp.Providers[i]
		}
	}
	if routes == nil {
		t.Fatal("expected routes provider in response")
	}
	if routes.UsedToday != 5 || routes.RemainingToday != 1495 || routes.BlockedToday != 1 || routes.FallbackToday != 2 {
		t.Fatalf("unexpected aggregation: %+v", routes)
	}
	if routes.Provider != "ors" || routes.DailyQuota != 1500 {
		t.Fatalf("unexpected provider summary: %+v", routes)
	}
	if len(routes.Operations) != 1 || routes.Operations[0].Operation != providerlimits.OpRouteEstimate {
		t.Fatalf("expected operation breakdown, got %+v", routes.Operations)
	}
}

func TestProviderQuotasResetDevForbiddenInProduction(t *testing.T) {
	store := &stubStore{}
	h := newQuotaTestHandler("production", store)

	rec := serve(h, http.MethodPost, "/ops/providers/quotas/ors/reset-dev")
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403 in production, got %d", rec.Code)
	}
	if store.resetHits != 0 {
		t.Fatal("reset must not run in production")
	}
}

func TestProviderQuotasResetDevWorksLocally(t *testing.T) {
	store := &stubStore{}
	h := newQuotaTestHandler("local", store)

	rec := serve(h, http.MethodPost, "/ops/providers/quotas/ors/reset-dev")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 locally, got %d", rec.Code)
	}
	if store.resetHits != 1 {
		t.Fatalf("expected reset to run once, got %d", store.resetHits)
	}
}

func TestProviderQuotasDetailUnknownProvider(t *testing.T) {
	h := newQuotaTestHandler("local", &stubStore{})
	rec := serve(h, http.MethodGet, "/ops/providers/quotas/unknown-provider")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown provider, got %d", rec.Code)
	}
}
