package service

import "github.com/prometheus/client_golang/prometheus"

var (
	tripDataExportJobs  = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "trip_data_export_jobs_total", Help: "Trip export job outcomes."}, []string{"result"})
	tripDataExportBytes = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "trip_data_export_package_bytes", Help: "Size of completed private trip export packages.", Buckets: prometheus.ExponentialBuckets(1024, 4, 10)})
)

func init() { prometheus.MustRegister(tripDataExportJobs, tripDataExportBytes) }
