package observability

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	externalProviderRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "external_provider_requests_total", Help: "Total external provider requests."},
		[]string{"provider", "operation", "result"},
	)
	externalProviderDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "external_provider_duration_seconds",
			Help:    "External provider request duration.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"provider", "operation", "result"},
	)
	externalProviderFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "external_provider_failures_total", Help: "Total external provider failures."},
		[]string{"provider", "operation", "error_code"},
	)
	externalProviderFallback = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "external_provider_fallback_total", Help: "Total external provider fallback uses."},
		[]string{"provider", "operation", "fallback_provider"},
	)
	externalProviderCacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "external_provider_cache_hits_total", Help: "Total external provider cache hits."},
		[]string{"provider", "operation"},
	)
	externalProviderCacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "external_provider_cache_misses_total", Help: "Total external provider cache misses."},
		[]string{"provider", "operation"},
	)
)

func init() {
	prometheus.MustRegister(
		externalProviderRequests,
		externalProviderDuration,
		externalProviderFailures,
		externalProviderFallback,
		externalProviderCacheHits,
		externalProviderCacheMisses,
	)
}

func RecordProviderRequest(provider, operation, result string, duration time.Duration) {
	provider = normalize(provider)
	operation = normalize(operation)
	result = normalize(result)
	externalProviderRequests.WithLabelValues(provider, operation, result).Inc()
	externalProviderDuration.WithLabelValues(provider, operation, result).Observe(duration.Seconds())
}

func RecordProviderFailure(provider, operation, errorCode string) {
	externalProviderFailures.WithLabelValues(normalize(provider), normalize(operation), normalize(errorCode)).Inc()
}

func RecordProviderFallback(provider, operation, fallbackProvider string) {
	externalProviderFallback.WithLabelValues(normalize(provider), normalize(operation), normalize(fallbackProvider)).Inc()
}

func RecordProviderCacheHit(provider, operation string) {
	externalProviderCacheHits.WithLabelValues(normalize(provider), normalize(operation)).Inc()
}

func RecordProviderCacheMiss(provider, operation string) {
	externalProviderCacheMisses.WithLabelValues(normalize(provider), normalize(operation)).Inc()
}

func normalize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}
	return value
}
