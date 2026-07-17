package search

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	searchRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "search_requests_total",
			Help: "Total global search requests.",
		},
		[]string{"service", "scope", "status"},
	)
	searchDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "search_duration_seconds",
			Help:    "Global search request duration.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "scope", "status"},
	)
	searchResultsCount = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "search_results_count",
			Help:    "Global search results returned.",
			Buckets: []float64{0, 1, 2, 5, 10, 20, 50},
		},
		[]string{"service", "scope"},
	)
	searchErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "search_errors_total",
			Help: "Global search errors.",
		},
		[]string{"service", "scope", "kind"},
	)
)

func init() {
	prometheus.MustRegister(
		searchRequestsTotal,
		searchDuration,
		searchResultsCount,
		searchErrorsTotal,
	)
}

func recordSearch(scope Scope, status string, duration time.Duration, resultCount int) {
	searchRequestsTotal.WithLabelValues("trip-service", string(scope), status).Inc()
	searchDuration.WithLabelValues("trip-service", string(scope), status).Observe(duration.Seconds())
	if status == "ok" {
		searchResultsCount.WithLabelValues("trip-service", string(scope)).Observe(float64(resultCount))
	}
}

func recordSearchError(scope Scope, kind string) {
	searchErrorsTotal.WithLabelValues("trip-service", string(scope), kind).Inc()
}
