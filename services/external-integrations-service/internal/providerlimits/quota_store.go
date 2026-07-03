package providerlimits

import (
	"context"
	"time"
)

// Reservation is the outcome of an atomic daily-quota reservation.
type Reservation struct {
	Allowed        bool
	QuotaExceeded  bool
	DailyQuota     int64
	DailyUsed      int64
	DailyRemaining int64
}

// OperationUsage is a single provider+operation counter row for a usage date.
type OperationUsage struct {
	Provider       string
	Operation      string
	UsageDate      time.Time
	UsedCount      int64
	BlockedCount   int64
	FallbackCount  int64
	LastAllowedAt  *time.Time
	LastBlockedAt  *time.Time
	LastFallbackAt *time.Time
}

// QuotaStore persists daily provider usage so counters survive restarts. The
// Reserve method must be atomic: concurrent reservations for the same provider
// must never let usage exceed the quota.
type QuotaStore interface {
	// Reserve atomically checks the provider's daily quota and, when allowed,
	// increments used_count; otherwise it increments blocked_count. A quota of 0
	// means unlimited.
	Reserve(ctx context.Context, provider, operation string, date time.Time, cost, quota int64) (Reservation, error)
	// IncrementBlocked records a blocked call that never reached the provider
	// (e.g. an in-memory rate-limit denial), so no quota is consumed.
	IncrementBlocked(ctx context.Context, provider, operation string, date time.Time, amount int64) error
	// IncrementFallback records that a limited call fell back to mock/cache.
	IncrementFallback(ctx context.Context, provider, operation string, date time.Time, amount int64) error
	// ListUsageByDate returns every operation row for the given usage date.
	ListUsageByDate(ctx context.Context, date time.Time) ([]OperationUsage, error)
	// ListUsageByProvider returns rows for one provider across a date range.
	ListUsageByProvider(ctx context.Context, provider string, from, to time.Time) ([]OperationUsage, error)
	// ResetProviderForDate clears a provider's counters for one date (dev only).
	ResetProviderForDate(ctx context.Context, provider string, date time.Time) error
}
