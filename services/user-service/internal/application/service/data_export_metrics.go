package service

import "github.com/prometheus/client_golang/prometheus"

var (
	accountDataExportJobs  = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "account_data_export_jobs_total", Help: "Account export job outcomes."}, []string{"result"})
	accountDataExportBytes = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "account_data_export_package_bytes", Help: "Size of completed private account export packages.", Buckets: prometheus.ExponentialBuckets(1024, 4, 10)})
)

func init() { prometheus.MustRegister(accountDataExportJobs, accountDataExportBytes) }
