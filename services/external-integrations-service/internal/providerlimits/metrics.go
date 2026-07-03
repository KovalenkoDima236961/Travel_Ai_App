package providerlimits

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

// Provider-limit metrics. Labels are intentionally low cardinality: only the
// bounded provider, operation, result, and reason values ever appear. userId,
// tripId, destination, cacheKey, and raw errors are never used as labels.
var (
	limitRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "provider_limit_requests_total",
			Help: "Total provider limit checks by result.",
		},
		[]string{"provider", "operation", "result"},
	)
	quotaUsed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "provider_quota_used_total",
			Help: "Total provider quota units consumed.",
		},
		[]string{"provider", "operation"},
	)
	quotaBlocked = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "provider_quota_blocked_total",
			Help: "Total provider calls blocked by rate limit or quota.",
		},
		[]string{"provider", "operation", "reason"},
	)
	quotaRemaining = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "provider_quota_remaining",
			Help: "Remaining daily provider quota (0 when unlimited).",
		},
		[]string{"provider"},
	)
	rateTokensAvailable = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "provider_rate_limit_tokens_available",
			Help: "Approximate rate-limit tokens available for a provider.",
		},
		[]string{"provider"},
	)
	fallbackDueToLimit = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "provider_fallback_due_to_limit_total",
			Help: "Total fallbacks triggered by a provider limit.",
		},
		[]string{"provider", "operation", "reason"},
	)
	limitCheckDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "provider_limit_check_duration_seconds",
			Help:    "Duration of provider limit checks.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"provider", "operation", "result"},
	)
)

func init() {
	prometheus.MustRegister(
		limitRequests,
		quotaUsed,
		quotaBlocked,
		quotaRemaining,
		rateTokensAvailable,
		fallbackDueToLimit,
		limitCheckDuration,
	)
}

func recordLimitRequest(provider, operation, result string, seconds float64) {
	provider = metricLabel(provider)
	operation = metricLabel(operation)
	result = metricLabel(result)
	limitRequests.WithLabelValues(provider, operation, result).Inc()
	limitCheckDuration.WithLabelValues(provider, operation, result).Observe(seconds)
}

func recordQuotaUsed(provider, operation string, amount int64) {
	quotaUsed.WithLabelValues(metricLabel(provider), metricLabel(operation)).Add(float64(amount))
}

func recordQuotaBlocked(provider, operation, reason string) {
	quotaBlocked.WithLabelValues(metricLabel(provider), metricLabel(operation), metricLabel(reason)).Inc()
}

func setQuotaRemaining(provider string, remaining int64) {
	quotaRemaining.WithLabelValues(metricLabel(provider)).Set(float64(remaining))
}

func setRateTokensAvailable(provider string, tokens float64) {
	rateTokensAvailable.WithLabelValues(metricLabel(provider)).Set(tokens)
}

func recordFallbackDueToLimit(provider, operation, reason string) {
	fallbackDueToLimit.WithLabelValues(metricLabel(provider), metricLabel(operation), metricLabel(reason)).Inc()
}

func metricLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}
	return value
}
