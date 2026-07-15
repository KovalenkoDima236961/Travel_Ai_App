package calendar

import (
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	calendarFreeBusyRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "calendar_free_busy_requests_total", Help: "Total calendar free/busy requests."},
		[]string{"provider", "result"},
	)
	calendarFreeBusyDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "calendar_free_busy_request_duration_seconds",
			Help:    "Calendar free/busy request duration.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"provider", "result"},
	)
	calendarFreeBusyFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "calendar_free_busy_failures_total", Help: "Total calendar free/busy failures."},
		[]string{"provider", "error_code"},
	)
	calendarFreeBusyBlocks = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "calendar_free_busy_busy_blocks_count", Help: "Total busy blocks returned by free/busy import."},
		[]string{"provider"},
	)
)

func init() {
	prometheus.MustRegister(
		calendarFreeBusyRequests,
		calendarFreeBusyDuration,
		calendarFreeBusyFailures,
		calendarFreeBusyBlocks,
	)
}

func recordCalendarFreeBusyRequest(provider, result string, duration time.Duration) {
	calendarFreeBusyRequests.WithLabelValues(calendarMetricValue(provider), calendarMetricValue(result)).Inc()
	calendarFreeBusyDuration.WithLabelValues(calendarMetricValue(provider), calendarMetricValue(result)).Observe(duration.Seconds())
}

func recordCalendarFreeBusyFailure(provider, code string) {
	calendarFreeBusyFailures.WithLabelValues(calendarMetricValue(provider), calendarMetricValue(code)).Inc()
	recordCalendarFreeBusyRequest(provider, code, 0)
}

func recordCalendarFreeBusyBlocks(provider string, count int) {
	if count <= 0 {
		return
	}
	calendarFreeBusyBlocks.WithLabelValues(calendarMetricValue(provider)).Add(float64(count))
}

func calendarMetricValue(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}
	return value
}
