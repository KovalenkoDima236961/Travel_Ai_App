package observability

import "github.com/prometheus/client_golang/prometheus"

var (
	activityEventsCreated = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "activity_events_created_total", Help: "Total trip activity events created."},
		[]string{"event_type"},
	)
	notificationsRequested = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "notifications_requested_total", Help: "Total notification requests sent by Trip Service."},
		[]string{"type", "result"},
	)
	calendarSyncTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "calendar_sync_total", Help: "Total calendar sync operations."},
		[]string{"provider", "result"},
	)
	budgetOptimizationJobsCreated = prometheus.NewCounter(
		prometheus.CounterOpts{Name: "budget_optimization_jobs_created_total", Help: "Total budget optimization jobs created."},
	)
	budgetOptimizationProposalsCreated = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "budget_optimization_proposals_created_total", Help: "Total budget optimization proposals created."},
		[]string{"status"},
	)
	summaryCacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "summary_cache_hits_total", Help: "Total trip summary cache hits."},
		[]string{"summary"},
	)
	summaryCacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "summary_cache_misses_total", Help: "Total trip summary cache misses."},
		[]string{"summary"},
	)
	summaryCacheEvictions = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "summary_cache_evictions_total", Help: "Total trip summary cache evictions."},
		[]string{"summary"},
	)
)

func init() {
	prometheus.MustRegister(
		activityEventsCreated,
		notificationsRequested,
		calendarSyncTotal,
		budgetOptimizationJobsCreated,
		budgetOptimizationProposalsCreated,
		summaryCacheHits,
		summaryCacheMisses,
		summaryCacheEvictions,
	)
}

func RecordSummaryCacheHit(summary string) {
	summaryCacheHits.WithLabelValues(summary).Inc()
}

func RecordSummaryCacheMiss(summary string) {
	summaryCacheMisses.WithLabelValues(summary).Inc()
}

func RecordSummaryCacheEviction(summary string) {
	summaryCacheEvictions.WithLabelValues(summary).Inc()
}

func RecordActivityEventCreated(eventType string) {
	activityEventsCreated.WithLabelValues(eventType).Inc()
}

func RecordNotificationsRequested(notificationType, result string, count int) {
	if count <= 0 {
		return
	}
	notificationsRequested.WithLabelValues(notificationType, result).Add(float64(count))
}

func RecordCalendarSync(provider, result string) {
	calendarSyncTotal.WithLabelValues(provider, result).Inc()
}

func RecordBudgetOptimizationJobCreated() {
	budgetOptimizationJobsCreated.Inc()
}

func RecordBudgetOptimizationProposalCreated(status string) {
	budgetOptimizationProposalsCreated.WithLabelValues(status).Inc()
}
