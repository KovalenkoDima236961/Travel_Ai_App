package personalization

import "github.com/prometheus/client_golang/prometheus"

var (
	contextBuiltTotal               = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "personalization_context_built_total", Help: "Total personalization contexts built."}, []string{"source", "completeness_level"})
	contextBuildDuration            = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "personalization_context_build_duration_seconds", Help: "Duration of personalization context builds."}, []string{"source"})
	feedbackSubmittedTotal          = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "personalization_feedback_submitted_total", Help: "Total explicit personalization feedback submissions."}, []string{"feedback_type"})
	completenessScore               = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "preference_completeness_score", Help: "Latest observed preference completeness score by level."}, []string{"completeness_level"})
	personalizedRecommendationTotal = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "personalized_recommendation_generated_total", Help: "Total personalized recommendation payloads generated."}, []string{"source", "completeness_level"})
)

func init() {
	prometheus.MustRegister(contextBuiltTotal, contextBuildDuration, feedbackSubmittedTotal, completenessScore, personalizedRecommendationTotal)
}

func RecordRecommendation(source Source, completenessLevel string, count int) {
	if count > 0 {
		personalizedRecommendationTotal.WithLabelValues(string(source), completenessLevel).Add(float64(count))
	}
}
