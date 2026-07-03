package providerlimits

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	storage "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/storage/postgres"
)

// PostgresStore is the Postgres-backed QuotaStore. Reservations use a
// provider_daily_totals row locked FOR UPDATE so concurrent reservations for the
// same provider are serialized and can never exceed the quota.
type PostgresStore struct {
	db *storage.DB
}

// NewPostgresStore builds a Postgres quota store.
func NewPostgresStore(db *storage.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) Reserve(ctx context.Context, provider, operation string, date time.Time, cost, quota int64) (Reservation, error) {
	if cost < 1 {
		cost = 1
	}
	usageDate := date.UTC().Truncate(24 * time.Hour)

	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Reservation{}, fmt.Errorf("begin quota reservation tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`INSERT INTO provider_daily_totals (id, provider, usage_date)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (provider, usage_date) DO NOTHING`,
		uuid.New(), provider, usageDate,
	); err != nil {
		return Reservation{}, fmt.Errorf("ensure provider totals row: %w", err)
	}

	var used int64
	if err := tx.QueryRow(ctx,
		`SELECT used_count FROM provider_daily_totals
		 WHERE provider = $1 AND usage_date = $2 FOR UPDATE`,
		provider, usageDate,
	).Scan(&used); err != nil {
		return Reservation{}, fmt.Errorf("lock provider totals row: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO provider_daily_usage (id, provider, operation, usage_date)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (provider, operation, usage_date) DO NOTHING`,
		uuid.New(), provider, operation, usageDate,
	); err != nil {
		return Reservation{}, fmt.Errorf("ensure provider usage row: %w", err)
	}

	if quota > 0 && used+cost > quota {
		// provider_daily_totals is the per-provider aggregate and has no
		// last_*_at columns; those live on the per-operation usage rows.
		if _, err := tx.Exec(ctx,
			`UPDATE provider_daily_totals
			 SET blocked_count = blocked_count + $3, updated_at = NOW()
			 WHERE provider = $1 AND usage_date = $2`,
			provider, usageDate, cost,
		); err != nil {
			return Reservation{}, fmt.Errorf("increment totals blocked: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`UPDATE provider_daily_usage
			 SET blocked_count = blocked_count + $4, last_blocked_at = NOW(), updated_at = NOW()
			 WHERE provider = $1 AND operation = $2 AND usage_date = $3`,
			provider, operation, usageDate, cost,
		); err != nil {
			return Reservation{}, fmt.Errorf("increment usage blocked: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return Reservation{}, fmt.Errorf("commit quota reservation: %w", err)
		}
		return Reservation{
			Allowed:        false,
			QuotaExceeded:  true,
			DailyQuota:     quota,
			DailyUsed:      used,
			DailyRemaining: remaining(quota, used),
		}, nil
	}

	if _, err := tx.Exec(ctx,
		`UPDATE provider_daily_totals
		 SET used_count = used_count + $3, updated_at = NOW()
		 WHERE provider = $1 AND usage_date = $2`,
		provider, usageDate, cost,
	); err != nil {
		return Reservation{}, fmt.Errorf("increment totals used: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE provider_daily_usage
		 SET used_count = used_count + $4, last_allowed_at = NOW(), updated_at = NOW()
		 WHERE provider = $1 AND operation = $2 AND usage_date = $3`,
		provider, operation, usageDate, cost,
	); err != nil {
		return Reservation{}, fmt.Errorf("increment usage used: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Reservation{}, fmt.Errorf("commit quota reservation: %w", err)
	}

	newUsed := used + cost
	return Reservation{
		Allowed:        true,
		DailyQuota:     quota,
		DailyUsed:      newUsed,
		DailyRemaining: remaining(quota, newUsed),
	}, nil
}

func (s *PostgresStore) IncrementBlocked(ctx context.Context, provider, operation string, date time.Time, amount int64) error {
	return s.incrementCounter(ctx, provider, operation, date, amount, "blocked_count", "last_blocked_at")
}

func (s *PostgresStore) IncrementFallback(ctx context.Context, provider, operation string, date time.Time, amount int64) error {
	return s.incrementCounter(ctx, provider, operation, date, amount, "fallback_count", "last_fallback_at")
}

// incrementCounter additively bumps a counter column on the per-provider totals
// row and the per-operation usage row. Only the usage row carries the last_*_at
// timestamp (the totals table has no such columns). Best-effort; does not lock.
func (s *PostgresStore) incrementCounter(ctx context.Context, provider, operation string, date time.Time, amount int64, column, tsColumn string) error {
	if amount < 1 {
		amount = 1
	}
	usageDate := date.UTC().Truncate(24 * time.Hour)

	totalsQuery := fmt.Sprintf(
		`INSERT INTO provider_daily_totals (id, provider, usage_date, %s)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (provider, usage_date)
		 DO UPDATE SET %s = provider_daily_totals.%s + EXCLUDED.%s, updated_at = NOW()`,
		column, column, column, column,
	)
	if _, err := s.db.Exec(ctx, totalsQuery, uuid.New(), provider, usageDate, amount); err != nil {
		return fmt.Errorf("increment totals %s: %w", column, err)
	}

	usageQuery := fmt.Sprintf(
		`INSERT INTO provider_daily_usage (id, provider, operation, usage_date, %s, %s)
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 ON CONFLICT (provider, operation, usage_date)
		 DO UPDATE SET %s = provider_daily_usage.%s + EXCLUDED.%s, %s = NOW(), updated_at = NOW()`,
		column, tsColumn, column, column, column, tsColumn,
	)
	if _, err := s.db.Exec(ctx, usageQuery, uuid.New(), provider, operation, usageDate, amount); err != nil {
		return fmt.Errorf("increment usage %s: %w", column, err)
	}
	return nil
}

func (s *PostgresStore) ListUsageByDate(ctx context.Context, date time.Time) ([]OperationUsage, error) {
	usageDate := date.UTC().Truncate(24 * time.Hour)
	rows, err := s.db.Query(ctx,
		`SELECT provider, operation, usage_date, used_count, blocked_count, fallback_count,
		        last_allowed_at, last_blocked_at, last_fallback_at
		 FROM provider_daily_usage
		 WHERE usage_date = $1
		 ORDER BY provider, operation`,
		usageDate,
	)
	if err != nil {
		return nil, fmt.Errorf("list provider usage by date: %w", err)
	}
	defer rows.Close()
	return scanUsageRows(rows)
}

func (s *PostgresStore) ListUsageByProvider(ctx context.Context, provider string, from, to time.Time) ([]OperationUsage, error) {
	fromDate := from.UTC().Truncate(24 * time.Hour)
	toDate := to.UTC().Truncate(24 * time.Hour)
	rows, err := s.db.Query(ctx,
		`SELECT provider, operation, usage_date, used_count, blocked_count, fallback_count,
		        last_allowed_at, last_blocked_at, last_fallback_at
		 FROM provider_daily_usage
		 WHERE provider = $1 AND usage_date BETWEEN $2 AND $3
		 ORDER BY usage_date DESC, operation`,
		provider, fromDate, toDate,
	)
	if err != nil {
		return nil, fmt.Errorf("list provider usage by provider: %w", err)
	}
	defer rows.Close()
	return scanUsageRows(rows)
}

func (s *PostgresStore) ResetProviderForDate(ctx context.Context, provider string, date time.Time) error {
	usageDate := date.UTC().Truncate(24 * time.Hour)
	if _, err := s.db.Exec(ctx,
		`DELETE FROM provider_daily_usage WHERE provider = $1 AND usage_date = $2`,
		provider, usageDate,
	); err != nil {
		return fmt.Errorf("reset provider usage rows: %w", err)
	}
	if _, err := s.db.Exec(ctx,
		`DELETE FROM provider_daily_totals WHERE provider = $1 AND usage_date = $2`,
		provider, usageDate,
	); err != nil {
		return fmt.Errorf("reset provider totals row: %w", err)
	}
	return nil
}

func scanUsageRows(rows pgx.Rows) ([]OperationUsage, error) {
	out := make([]OperationUsage, 0)
	for rows.Next() {
		var u OperationUsage
		var lastAllowed, lastBlocked, lastFallback sql.NullTime
		if err := rows.Scan(
			&u.Provider,
			&u.Operation,
			&u.UsageDate,
			&u.UsedCount,
			&u.BlockedCount,
			&u.FallbackCount,
			&lastAllowed,
			&lastBlocked,
			&lastFallback,
		); err != nil {
			return nil, fmt.Errorf("scan provider usage row: %w", err)
		}
		u.LastAllowedAt = nullTimePtr(lastAllowed)
		u.LastBlockedAt = nullTimePtr(lastBlocked)
		u.LastFallbackAt = nullTimePtr(lastFallback)
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate provider usage rows: %w", err)
	}
	return out, nil
}

func nullTimePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}

func remaining(quota, used int64) int64 {
	if quota <= 0 {
		return 0
	}
	if used >= quota {
		return 0
	}
	return quota - used
}
