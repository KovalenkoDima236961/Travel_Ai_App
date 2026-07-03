package providerlimits

import (
	"context"
	"math"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/observability"
)

// ProviderLimit is the resolved rate-limit and daily-quota configuration for the
// active provider of a single category.
type ProviderLimit struct {
	Category      string
	Provider      string
	RatePerMinute int
	Burst         int
	DailyQuota    int64
}

// Unlimited reports whether the provider has no daily quota cap.
func (p ProviderLimit) UnlimitedQuota() bool { return p.DailyQuota <= 0 }

// Guard is the central enforcement point. It is safe for concurrent use.
type Guard struct {
	enabled  bool
	failOpen bool
	loc      *time.Location
	maxWait  time.Duration
	limiter  *Limiter
	store    QuotaStore
	registry map[string]ProviderLimit // category -> limit
	log      *zap.Logger
}

// GuardParams configures a Guard.
type GuardParams struct {
	Enabled  bool
	FailOpen bool
	Location *time.Location
	MaxWait  time.Duration
	Limiter  *Limiter
	Store    QuotaStore
	Limits   []ProviderLimit
	Logger   *zap.Logger
}

// NewGuard builds a Guard from resolved parameters.
func NewGuard(p GuardParams) *Guard {
	loc := p.Location
	if loc == nil {
		loc = time.UTC
	}
	limiter := p.Limiter
	if limiter == nil {
		limiter = NewLimiter()
	}
	log := p.Logger
	if log == nil {
		log = zap.NewNop()
	}
	registry := make(map[string]ProviderLimit, len(p.Limits))
	for _, limit := range p.Limits {
		registry[limit.Category] = limit
	}
	return &Guard{
		enabled:  p.Enabled,
		failOpen: p.FailOpen,
		loc:      loc,
		maxWait:  p.MaxWait,
		limiter:  limiter,
		store:    p.Store,
		registry: registry,
		log:      log,
	}
}

// Enabled reports whether enforcement is active.
func (g *Guard) Enabled() bool { return g.enabled }

// FailOpen reports the fail-open policy.
func (g *Guard) FailOpen() bool { return g.failOpen }

// Store exposes the quota store for Ops read queries.
func (g *Guard) Store() QuotaStore { return g.store }

// Location returns the timezone used to compute the usage date.
func (g *Guard) Location() *time.Location { return g.loc }

// Today returns the current usage date in the guard's timezone, truncated to a
// UTC day boundary (v1 stores dates as UTC calendar days).
func (g *Guard) Today() time.Time {
	return time.Now().In(g.loc).UTC().Truncate(24 * time.Hour)
}

// Limits returns all configured provider limits, ordered by category.
func (g *Guard) Limits() []ProviderLimit {
	order := []string{CategoryPlaces, CategoryRoutes, CategoryWeather, CategoryCalendar, CategoryExchangeRate, CategoryPrice, CategoryAvailability}
	out := make([]ProviderLimit, 0, len(order))
	for _, category := range order {
		if limit, ok := g.registry[category]; ok {
			out = append(out, limit)
		}
	}
	return out
}

// LimitForCategory returns the configured limit for a category.
func (g *Guard) LimitForCategory(category string) (ProviderLimit, bool) {
	limit, ok := g.registry[category]
	return limit, ok
}

// CheckAndReserve applies the rate limit and daily quota for a provider call.
// The returned error is non-nil only for unexpected internal failures; limit
// conditions are conveyed via the Decision. Callers should inspect the Decision
// and either proceed, fall back, or surface a controlled error.
func (g *Guard) CheckAndReserve(ctx context.Context, call ProviderCall) (Decision, error) {
	category := CategoryForOperation(call.Operation)
	decision := Decision{
		Provider:  call.Provider,
		Operation: call.Operation,
		Category:  category,
	}
	if !g.enabled {
		decision.Allowed = true
		decision.Reason = ReasonDisabled
		return decision, nil
	}

	cost := call.Cost
	if cost < 1 {
		cost = 1
	}
	limit, hasLimit := g.registry[category]
	decision.DailyQuota = limit.DailyQuota
	if !hasLimit {
		// Unknown category: allow but do not track. Should not happen for the
		// bounded operation set.
		decision.Allowed = true
		decision.Reason = ReasonAllowed
		return decision, nil
	}

	start := time.Now()
	date := g.Today()

	// 1. In-memory per-minute rate limit. A denial never reaches the provider,
	// so it consumes no daily quota.
	if allowed, wait := g.rateAllow(category, limit, cost); !allowed {
		retryAfter := int(math.Ceil(wait.Seconds()))
		if retryAfter < 1 {
			retryAfter = 1
		}
		decision.Limited = true
		decision.Reason = ReasonRateLimited
		decision.RetryAfterSeconds = retryAfter
		decision.DailyRemaining = 0
		g.recordBlocked(ctx, call.Provider, call.Operation, date, cost)
		recordLimitRequest(call.Provider, call.Operation, ReasonRateLimited, time.Since(start).Seconds())
		recordQuotaBlocked(call.Provider, call.Operation, ReasonRateLimited)
		g.logLimit(ctx, "provider_rate_limited", decision)
		return decision, nil
	}
	if tokens, ok := g.limiter.Tokens(category); ok {
		setRateTokensAvailable(call.Provider, tokens)
	}

	// 2. Postgres-backed daily quota reservation.
	res, err := g.store.Reserve(ctx, call.Provider, call.Operation, date, cost, limit.DailyQuota)
	if err != nil {
		g.log.Warn("provider_limits_unavailable",
			zap.String("provider", call.Provider),
			zap.String("operation", call.Operation),
			zap.Bool("failOpen", g.failOpen),
			zap.Error(err),
		)
		recordLimitRequest(call.Provider, call.Operation, ReasonLimitsUnavailable, time.Since(start).Seconds())
		if g.failOpen {
			decision.Allowed = true
			decision.Reason = ReasonFailOpen
			return decision, nil
		}
		decision.Unavailable = true
		decision.Reason = ReasonLimitsUnavailable
		decision.RetryAfterSeconds = 30
		return decision, nil
	}

	decision.DailyQuota = res.DailyQuota
	decision.DailyUsed = res.DailyUsed
	decision.DailyRemaining = res.DailyRemaining
	setQuotaRemaining(call.Provider, res.DailyRemaining)

	if res.QuotaExceeded {
		decision.QuotaExceeded = true
		decision.Reason = ReasonQuotaExceeded
		decision.RetryAfterSeconds = g.secondsUntilTomorrow()
		recordLimitRequest(call.Provider, call.Operation, ReasonQuotaExceeded, time.Since(start).Seconds())
		recordQuotaBlocked(call.Provider, call.Operation, ReasonQuotaExceeded)
		g.logLimit(ctx, "provider_quota_exceeded", decision)
		return decision, nil
	}

	decision.Allowed = true
	decision.Reason = ReasonAllowed
	recordLimitRequest(call.Provider, call.Operation, ReasonAllowed, time.Since(start).Seconds())
	recordQuotaUsed(call.Provider, call.Operation, cost)
	return decision, nil
}

// RecordFallback records that a limited call fell back to mock/cache.
func (g *Guard) RecordFallback(ctx context.Context, provider, operation, reason string) {
	if !g.enabled {
		return
	}
	if err := g.store.IncrementFallback(ctx, provider, operation, g.Today(), 1); err != nil {
		g.log.Warn("failed to record provider fallback",
			zap.String("provider", provider),
			zap.String("operation", operation),
			zap.Error(err),
		)
	}
	recordFallbackDueToLimit(provider, operation, reason)
	g.log.Info("provider_fallback_due_to_limit",
		zap.String("provider", provider),
		zap.String("operation", operation),
		zap.String("reason", reason),
	)
}

// rateAllow applies the token bucket, honoring MaxWait when configured.
func (g *Guard) rateAllow(category string, limit ProviderLimit, cost int64) (bool, time.Duration) {
	spec := RateSpec{PerMinute: limit.RatePerMinute, Burst: limit.Burst}
	allowed, wait := g.limiter.Allow(category, spec, cost)
	if allowed || g.maxWait <= 0 || wait > g.maxWait {
		return allowed, wait
	}
	// Bounded wait: sleep then try once more. Default MaxWait is 0 (fail fast).
	timer := time.NewTimer(wait)
	defer timer.Stop()
	<-timer.C
	return g.limiter.Allow(category, spec, cost)
}

func (g *Guard) recordBlocked(ctx context.Context, provider, operation string, date time.Time, cost int64) {
	if err := g.store.IncrementBlocked(ctx, provider, operation, date, cost); err != nil {
		g.log.Warn("failed to record provider blocked",
			zap.String("provider", provider),
			zap.String("operation", operation),
			zap.Error(err),
		)
	}
}

func (g *Guard) secondsUntilTomorrow() int {
	now := time.Now().In(g.loc)
	tomorrow := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, g.loc).Add(24 * time.Hour)
	seconds := int(math.Ceil(tomorrow.Sub(now).Seconds()))
	if seconds < 1 {
		seconds = 1
	}
	return seconds
}

func (g *Guard) logLimit(ctx context.Context, msg string, d Decision) {
	fields := []zap.Field{
		zap.String("provider", d.Provider),
		zap.String("operation", d.Operation),
		zap.String("quotaDate", g.Today().Format("2006-01-02")),
		zap.Int64("dailyUsed", d.DailyUsed),
		zap.Int64("dailyQuota", d.DailyQuota),
		zap.Int64("dailyRemaining", d.DailyRemaining),
		zap.String("reason", d.Reason),
		zap.Int("retryAfterSeconds", d.RetryAfterSeconds),
	}
	fields = append(fields, observability.RequestIDFields(ctx)...)
	g.log.Warn(msg, fields...)
}
