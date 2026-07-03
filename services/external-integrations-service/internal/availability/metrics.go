package availability

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	availabilitySearchRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "availability_search_requests_total", Help: "Total availability search requests."},
		[]string{"provider", "result"},
	)
	availabilitySearchDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "availability_search_duration_seconds",
			Help:    "Availability search duration.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"provider", "result"},
	)
	availabilityOptionsReturned = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "availability_options_returned_total", Help: "Total availability options returned."},
		[]string{"provider"},
	)
	availabilityCacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "availability_cache_hits_total", Help: "Total availability cache hits."},
		[]string{"provider"},
	)
	availabilityCacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "availability_cache_misses_total", Help: "Total availability cache misses."},
		[]string{"provider"},
	)
	availabilityFallback = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "availability_fallback_total", Help: "Total availability fallback uses."},
		[]string{"provider", "reason"},
	)
	availabilityErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "availability_errors_total", Help: "Total availability errors."},
		[]string{"provider", "error_code"},
	)
)

func init() {
	prometheus.MustRegister(
		availabilitySearchRequests,
		availabilitySearchDuration,
		availabilityOptionsReturned,
		availabilityCacheHits,
		availabilityCacheMisses,
		availabilityFallback,
		availabilityErrors,
	)
}

func recordAvailabilityRequest(provider, result string, duration time.Duration) {
	provider = metricValue(provider)
	result = metricValue(result)
	availabilitySearchRequests.WithLabelValues(provider, result).Inc()
	availabilitySearchDuration.WithLabelValues(provider, result).Observe(duration.Seconds())
}

func recordAvailabilityOptions(provider string, count int) {
	if count <= 0 {
		return
	}
	availabilityOptionsReturned.WithLabelValues(metricValue(provider)).Add(float64(count))
}

func recordAvailabilityCacheHit(provider string) {
	availabilityCacheHits.WithLabelValues(metricValue(provider)).Inc()
}

func recordAvailabilityCacheMiss(provider string) {
	availabilityCacheMisses.WithLabelValues(metricValue(provider)).Inc()
}

func recordAvailabilityFallback(provider, reason string) {
	availabilityFallback.WithLabelValues(metricValue(provider), metricValue(reason)).Inc()
}

func recordAvailabilityError(provider, code string) {
	availabilityErrors.WithLabelValues(metricValue(provider), metricValue(code)).Inc()
}

func metricValue(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}
	return value
}
