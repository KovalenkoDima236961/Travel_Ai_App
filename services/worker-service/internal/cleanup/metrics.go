package cleanup

import "github.com/prometheus/client_golang/prometheus"

var (
	runsTotal    = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "cleanup_runs_total", Help: "Completed cleanup runs by task, status, and dry-run mode."}, []string{"task", "status", "dry_run"})
	deletedRows  = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "cleanup_deleted_rows_total", Help: "Rows deleted by cleanup task."}, []string{"task"})
	deletedFiles = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "cleanup_deleted_files_total", Help: "Files deleted by cleanup task."}, []string{"task"})
	bytesFreed   = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "cleanup_bytes_freed_total", Help: "Bytes freed by cleanup task."}, []string{"task"})
	duration     = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "cleanup_duration_seconds", Help: "Cleanup task duration."}, []string{"task"})
	errorsTotal  = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "cleanup_errors_total", Help: "Failed cleanup runs by task."}, []string{"task"})
	lastSuccess  = prometheus.NewGaugeVec(prometheus.GaugeOpts{Name: "cleanup_last_success_timestamp_seconds", Help: "Unix timestamp of the last successful cleanup run."}, []string{"task"})
)

func init() {
	prometheus.MustRegister(runsTotal, deletedRows, deletedFiles, bytesFreed, duration, errorsTotal, lastSuccess)
}

func recordResult(result Result, status string, nowUnix float64) {
	runsTotal.WithLabelValues(result.TaskName, status, boolLabel(result.DryRun)).Inc()
	duration.WithLabelValues(result.TaskName).Observe(float64(result.DurationMS) / 1000)
	if status == StatusSucceeded {
		lastSuccess.WithLabelValues(result.TaskName).Set(nowUnix)
	}
	if result.DryRun {
		return
	}
	deletedRows.WithLabelValues(result.TaskName).Add(float64(result.DeletedCount))
	deletedFiles.WithLabelValues(result.TaskName).Add(float64(result.FileDeletedCount))
	bytesFreed.WithLabelValues(result.TaskName).Add(float64(result.BytesFreed))
	if status == StatusFailed {
		errorsTotal.WithLabelValues(result.TaskName).Inc()
	}
}

func boolLabel(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
