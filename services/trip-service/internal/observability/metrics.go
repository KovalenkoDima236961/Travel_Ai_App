package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/verification"
)

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
	verificationRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "trip_verification_requests_total", Help: "Total private trip verification reads."},
		[]string{"result"},
	)
	verificationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{Name: "trip_verification_duration_seconds", Help: "Duration of trip verification reads."},
		[]string{"result"},
	)
	verificationScore = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "trip_verification_score", Help: "Most recently computed trip verification score by readiness level."},
		[]string{"level"},
	)
	verificationStatusObservations = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "trip_verification_status_observations_total", Help: "Verification detail statuses observed while evaluating private trips."},
		[]string{"scope", "status", "source", "provider", "fallback_used"},
	)
	verificationStaleItems = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "trip_verification_stale_items_total", Help: "Stale verification details observed by scope."},
		[]string{"scope"},
	)
	verificationActions = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "trip_verification_actions_total", Help: "Explicit verification actions requested by users."},
		[]string{"action_type", "result"},
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
		verificationRequests,
		verificationDuration,
		verificationScore,
		verificationStatusObservations,
		verificationStaleItems,
		verificationActions,
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

func RecordVerificationRead(result string, duration time.Duration) {
	verificationRequests.WithLabelValues(result).Inc()
	verificationDuration.WithLabelValues(result).Observe(duration.Seconds())
}

// RecordVerificationComputed intentionally records only aggregate, private
// attributes. It never labels metrics with a trip, user, itinerary item, or
// provider response identifier.
func RecordVerificationComputed(response verification.Response) {
	verificationScore.WithLabelValues(string(response.Level)).Set(float64(response.Score))
	for _, section := range response.Sections {
		for _, detail := range section.Details {
			fallbackUsed := "false"
			if fallback, ok := detail.Metadata["fallbackUsed"].(bool); ok && fallback {
				fallbackUsed = "true"
			}
			provider := detail.Provider
			if provider == "" {
				provider = "unknown"
			}
			verificationStatusObservations.WithLabelValues(
				string(detail.Scope),
				string(detail.Status),
				string(detail.Source),
				provider,
				fallbackUsed,
			).Inc()
			if detail.Status == verification.StatusStale {
				verificationStaleItems.WithLabelValues(string(detail.Scope)).Inc()
			}
		}
	}
}

func RecordVerificationAction(actionType, result string) {
	verificationActions.WithLabelValues(actionType, result).Inc()
}
