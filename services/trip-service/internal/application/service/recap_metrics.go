package service

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	recapMetricLabelNames     = []string{"mode", "status", "trip_type", "ai_provider", "fallback"}
	recapGeneratedTotal       = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "trip_recap_generated_total", Help: "Number of trip recaps generated."}, recapMetricLabelNames)
	recapGenerationDuration   = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "trip_recap_generation_duration_seconds", Help: "Duration of AI trip recap generation."}, recapMetricLabelNames)
	recapAIFailuresTotal      = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "trip_recap_ai_failures_total", Help: "Trip recap AI generation failures."}, recapMetricLabelNames)
	recapFallbacksTotal       = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "trip_recap_fallbacks_total", Help: "Deterministic trip recap fallbacks."}, recapMetricLabelNames)
	recapUpdatedTotal         = prometheus.NewCounter(prometheus.CounterOpts{Name: "trip_recap_updated_total", Help: "Trip recap updates."})
	recapFinalizedTotal       = prometheus.NewCounter(prometheus.CounterOpts{Name: "trip_recap_finalized_total", Help: "Finalized trip recaps."})
	recapLearningAppliedTotal = prometheus.NewCounter(prometheus.CounterOpts{Name: "trip_recap_learning_applied_total", Help: "Applied recap learning signals."})
	recapTemplateCreatedTotal = prometheus.NewCounter(prometheus.CounterOpts{Name: "trip_recap_template_created_total", Help: "Templates created from recaps."})
)

func init() {
	prometheus.MustRegister(recapGeneratedTotal, recapGenerationDuration, recapAIFailuresTotal, recapFallbacksTotal, recapUpdatedTotal, recapFinalizedTotal, recapLearningAppliedTotal, recapTemplateCreatedTotal)
}

func recapMetricLabels(mode, status, tripType, aiProvider string, fallback bool) []string {
	if tripType == "" {
		tripType = "unknown"
	}
	return []string{mode, status, tripType, aiProvider, fmt.Sprint(fallback)}
}
