package alpha

import "github.com/prometheus/client_golang/prometheus"

var (
	alphaActiveUsers = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "alpha_active_users",
		Help: "Closed-alpha active users in the last 7 days.",
	})
	alphaTripGenerationTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "alpha_trip_generation_total",
		Help: "Closed-alpha itinerary generation events.",
	}, []string{"event"})
	alphaAISuccessTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "alpha_ai_success_total",
		Help: "Closed-alpha AI generation outcomes.",
	}, []string{"status"})
	alphaFeedbackTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "alpha_feedback_total",
		Help: "Closed-alpha feedback submitted by category.",
	}, []string{"category"})
	alphaBugReportsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "alpha_bug_reports_total",
		Help: "Closed-alpha bug reports submitted.",
	})
	alphaFeatureRequestsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "alpha_feature_requests_total",
		Help: "Closed-alpha feature requests submitted.",
	})
	alphaRetentionRate = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "alpha_retention_rate",
		Help: "Closed-alpha retained participant ratio.",
	})
	alphaGenerationLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "alpha_generation_latency",
		Help:    "Closed-alpha AI generation latency in seconds.",
		Buckets: []float64{0.5, 1, 2.5, 5, 10, 20, 40, 80, 160},
	})
	alphaOpenAICostEstimate = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "alpha_openai_cost_estimate",
		Help: "Closed-alpha estimated OpenAI cost in USD.",
	})
	alphaAnalyticsEventsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "alpha_analytics_events_total",
		Help: "Privacy-safe alpha analytics events accepted.",
	}, []string{"event", "feature"})
	alphaAnalyticsRejectedTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "alpha_analytics_rejected_total",
		Help: "Alpha analytics events rejected before persistence.",
	}, []string{"reason"})
)

func init() {
	prometheus.MustRegister(
		alphaActiveUsers,
		alphaTripGenerationTotal,
		alphaAISuccessTotal,
		alphaFeedbackTotal,
		alphaBugReportsTotal,
		alphaFeatureRequestsTotal,
		alphaRetentionRate,
		alphaGenerationLatency,
		alphaOpenAICostEstimate,
		alphaAnalyticsEventsTotal,
		alphaAnalyticsRejectedTotal,
	)
}
