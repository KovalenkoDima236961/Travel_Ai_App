package aiobservability

import "github.com/prometheus/client_golang/prometheus"

var (
	tracesStarted      = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ai_generation_traces_started_total", Help: "AI generation traces started."}, []string{"generation_type", "provider", "model"})
	tracesCompleted    = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ai_generation_traces_completed_total", Help: "AI generation traces completed."}, []string{"generation_type", "provider", "model", "status", "quality_status"})
	tracesFailed       = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ai_generation_traces_failed_total", Help: "AI generation traces failed."}, []string{"generation_type", "provider", "model", "error_code"})
	traceEvents        = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ai_generation_trace_events_total", Help: "AI generation trace events recorded."}, []string{"event_type", "event_status"})
	traceWriteFailures = prometheus.NewCounter(prometheus.CounterOpts{Name: "ai_generation_trace_write_failures_total", Help: "Trace persistence failures."})
	generationDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "ai_generation_duration_seconds", Help: "End-to-end traced AI generation duration."}, []string{"generation_type", "provider", "model", "status"})
	aiCallDuration     = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "ai_generation_ai_call_duration_seconds", Help: "Traced AI call duration."}, []string{"generation_type", "provider", "model"})
	repairDuration     = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "ai_generation_repair_duration_seconds", Help: "Traced repair duration."}, []string{"generation_type", "quality_status"})
	promptTokens       = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ai_generation_prompt_tokens_estimated", Help: "Estimated AI prompt tokens."}, []string{"generation_type", "provider", "model"})
	completionTokens   = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ai_generation_completion_tokens_estimated", Help: "Estimated AI completion tokens."}, []string{"generation_type", "provider", "model"})
)

func init() {
	prometheus.MustRegister(tracesStarted, tracesCompleted, tracesFailed, traceEvents, traceWriteFailures, generationDuration, aiCallDuration, repairDuration, promptTokens, completionTokens)
}
