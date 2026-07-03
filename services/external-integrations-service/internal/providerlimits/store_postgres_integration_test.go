package providerlimits

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"

	storage "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/storage/postgres"
)

// newIntegrationStore connects to the database named by EIS_TEST_DATABASE_URL,
// applies the provider-limit migration, and truncates the tables. Tests that use
// it are skipped when the env var is unset so the default suite stays hermetic.
func newIntegrationStore(t *testing.T) *PostgresStore {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("EIS_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("EIS_TEST_DATABASE_URL not set; skipping Postgres integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	db := &storage.DB{Pool: pool, Builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)}

	migration, err := os.ReadFile("../../migrations/000002_create_provider_limits_tables.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	for _, stmt := range strings.Split(string(migration), ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatalf("apply migration statement: %v\n%s", err, stmt)
		}
	}
	if _, err := pool.Exec(ctx, "TRUNCATE provider_daily_usage, provider_daily_totals"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return NewPostgresStore(db)
}

func TestPostgresStoreReserveCreatesAndIncrements(t *testing.T) {
	store := newIntegrationStore(t)
	ctx := context.Background()
	date := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)

	first, err := store.Reserve(ctx, "ors", OpRouteEstimate, date, 1, 10)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if !first.Allowed || first.DailyUsed != 1 || first.DailyRemaining != 9 {
		t.Fatalf("unexpected first reservation: %+v", first)
	}

	second, err := store.Reserve(ctx, "ors", OpRouteEstimate, date, 1, 10)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if second.DailyUsed != 2 {
		t.Fatalf("expected used=2 after two reservations, got %d", second.DailyUsed)
	}

	rows, err := store.ListUsageByDate(ctx, date)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(rows) != 1 || rows[0].UsedCount != 2 || rows[0].Provider != "ors" || rows[0].Operation != OpRouteEstimate {
		t.Fatalf("unexpected usage rows: %+v", rows)
	}
	if rows[0].LastAllowedAt == nil {
		t.Fatal("expected last_allowed_at to be set")
	}
}

func TestPostgresStoreQuotaExceededBlocks(t *testing.T) {
	store := newIntegrationStore(t)
	ctx := context.Background()
	date := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)

	if _, err := store.Reserve(ctx, "ors", OpRouteEstimate, date, 1, 1); err != nil {
		t.Fatalf("reserve: %v", err)
	}
	blocked, err := store.Reserve(ctx, "ors", OpRouteEstimate, date, 1, 1)
	if err != nil {
		t.Fatalf("reserve: %v", err)
	}
	if blocked.Allowed || !blocked.QuotaExceeded {
		t.Fatalf("expected quota exceeded, got %+v", blocked)
	}
	rows, _ := store.ListUsageByDate(ctx, date)
	if len(rows) != 1 || rows[0].UsedCount != 1 || rows[0].BlockedCount != 1 {
		t.Fatalf("expected used=1 blocked=1, got %+v", rows)
	}
}

func TestPostgresStoreConcurrentReservationsDoNotExceedQuota(t *testing.T) {
	store := newIntegrationStore(t)
	ctx := context.Background()
	date := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)
	const quota = 25

	var wg sync.WaitGroup
	var mu sync.Mutex
	allowed := 0
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := store.Reserve(ctx, "ors", OpRouteEstimate, date, 1, quota)
			if err != nil {
				return
			}
			if res.Allowed {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if allowed != quota {
		t.Fatalf("expected exactly %d allowed under concurrency, got %d", quota, allowed)
	}
	rows, _ := store.ListUsageByDate(ctx, date)
	if rows[0].UsedCount != quota {
		t.Fatalf("expected used_count=%d, got %d", quota, rows[0].UsedCount)
	}
}

func TestPostgresStoreFallbackAndBlockedAndReset(t *testing.T) {
	store := newIntegrationStore(t)
	ctx := context.Background()
	date := time.Date(2026, 7, 3, 0, 0, 0, 0, time.UTC)

	if err := store.IncrementBlocked(ctx, "ors", OpRouteEstimate, date, 1); err != nil {
		t.Fatalf("blocked: %v", err)
	}
	if err := store.IncrementFallback(ctx, "ors", OpRouteEstimate, date, 1); err != nil {
		t.Fatalf("fallback: %v", err)
	}
	rows, _ := store.ListUsageByDate(ctx, date)
	if len(rows) != 1 || rows[0].BlockedCount != 1 || rows[0].FallbackCount != 1 {
		t.Fatalf("expected blocked=1 fallback=1, got %+v", rows)
	}

	if err := store.ResetProviderForDate(ctx, "ors", date); err != nil {
		t.Fatalf("reset: %v", err)
	}
	rows, _ = store.ListUsageByDate(ctx, date)
	if len(rows) != 0 {
		t.Fatalf("expected no rows after reset, got %+v", rows)
	}
}
