package copilot

import "github.com/prometheus/client_golang/prometheus"

var (
	requestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "copilot_requests_total",
		Help: "Number of Trip Copilot requests.",
	}, []string{"intent", "mode", "status", "role"})
	durationSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "copilot_duration_seconds",
		Help: "Trip Copilot response duration.",
	}, []string{"intent", "mode", "status", "role"})
	aiFailuresTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "copilot_ai_failures_total",
		Help: "Trip Copilot AI provider failures.",
	}, []string{"intent", "mode"})
	validationFailuresTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "copilot_response_validation_failures_total",
		Help: "Trip Copilot response validation failures.",
	}, []string{"intent", "mode"})
	fallbacksTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "copilot_fallbacks_total",
		Help: "Trip Copilot deterministic fallbacks.",
	}, []string{"intent", "mode"})
	unsafeRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "copilot_unsafe_requests_total",
		Help: "Trip Copilot unsafe requests refused.",
	}, []string{"intent", "role"})
	actionSuggestionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "copilot_action_suggestions_total",
		Help: "Trip Copilot action suggestions returned.",
	}, []string{"intent", "mode", "role"})
)

func init() {
	prometheus.MustRegister(
		requestsTotal,
		durationSeconds,
		aiFailuresTotal,
		validationFailuresTotal,
		fallbacksTotal,
		unsafeRequestsTotal,
		actionSuggestionsTotal,
	)
}
