package aivalidation

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	validationTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_generation_validation_total",
			Help: "Total AI generation validation runs.",
		},
		[]string{"service", "generation_type", "status"},
	)
	validationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ai_generation_validation_duration_seconds",
			Help:    "AI generation validation duration.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "generation_type", "status"},
	)
	validationIssueCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_generation_validation_issue_count",
			Help: "Total AI generation validation issues.",
		},
		[]string{"service", "generation_type", "issue_category", "severity"},
	)
	repairAttemptsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_generation_repair_attempts_total",
			Help: "Total AI generation repair attempts.",
		},
		[]string{"service", "generation_type", "status"},
	)
	repairSuccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_generation_repair_success_total",
			Help: "Total AI generation repair successes.",
		},
		[]string{"service", "generation_type"},
	)
	repairFailureTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_generation_repair_failure_total",
			Help: "Total AI generation repair failures.",
		},
		[]string{"service", "generation_type"},
	)
	blockedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_generation_blocked_total",
			Help: "Total AI generation outputs blocked by validation.",
		},
		[]string{"service", "generation_type", "status"},
	)
	savedWithWarningsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ai_generation_saved_with_warnings_total",
			Help: "Total AI generation outputs saved with validation warnings.",
		},
		[]string{"service", "generation_type"},
	)
)

func init() {
	prometheus.MustRegister(
		validationTotal,
		validationDuration,
		validationIssueCount,
		repairAttemptsTotal,
		repairSuccessTotal,
		repairFailureTotal,
		blockedTotal,
		savedWithWarningsTotal,
	)
}

func recordValidation(generationType GenerationType, status GenerationQualityStatus, issues []ValidationIssue, duration time.Duration) {
	labels := []string{"trip-service", string(generationType), string(status)}
	validationTotal.WithLabelValues(labels...).Inc()
	validationDuration.WithLabelValues(labels...).Observe(duration.Seconds())
	for _, issue := range issues {
		validationIssueCount.WithLabelValues(
			"trip-service",
			string(generationType),
			string(issue.Category),
			string(issue.Severity),
		).Inc()
	}
}

func recordRepairAttempt(generationType GenerationType, status string) {
	repairAttemptsTotal.WithLabelValues("trip-service", string(generationType), status).Inc()
}

func recordRepairSuccess(generationType GenerationType) {
	repairSuccessTotal.WithLabelValues("trip-service", string(generationType)).Inc()
}

func recordRepairFailure(generationType GenerationType) {
	repairFailureTotal.WithLabelValues("trip-service", string(generationType)).Inc()
}

func recordBlocked(generationType GenerationType, status GenerationQualityStatus) {
	blockedTotal.WithLabelValues("trip-service", string(generationType), string(status)).Inc()
}

func recordSavedWithWarnings(generationType GenerationType) {
	savedWithWarningsTotal.WithLabelValues("trip-service", string(generationType)).Inc()
}
