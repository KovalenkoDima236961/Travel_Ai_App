package featureflags

import "github.com/prometheus/client_golang/prometheus"

var (
	featureFlagEvaluations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "feature_flag_evaluations_total",
		Help: "Feature flag evaluations by source and resolved result.",
	}, []string{"flag", "source", "result"})
	featureFlagDisabledRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "feature_flag_disabled_requests_total",
		Help: "Requests denied by server-side feature controls.",
	}, []string{"flag", "route", "service"})
	featureFlagCacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "feature_flag_cache_hits_total", Help: "Feature flag cache hits.",
	})
	featureFlagCacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "feature_flag_cache_misses_total", Help: "Feature flag cache misses.",
	})
	featureFlagUpdates = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "feature_flag_update_total", Help: "Feature flag changes recorded through ops controls.",
	}, []string{"flag", "action"})
)

func init() {
	prometheus.MustRegister(featureFlagEvaluations, featureFlagDisabledRequests, featureFlagCacheHits, featureFlagCacheMisses, featureFlagUpdates)
}

func RecordDisabledRequest(flag, route string) {
	featureFlagDisabledRequests.WithLabelValues(flag, route, "trip-service").Inc()
}
