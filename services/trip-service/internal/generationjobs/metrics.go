package generationjobs

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

var aiJobBuckets = []float64{1, 5, 10, 30, 60, 120, 300, 600, 1200}

var (
	generationJobsCreated = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "generation_jobs_created_total",
			Help: "Total generation jobs created.",
		},
		[]string{"job_type"},
	)
	generationJobsDispatch = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "generation_jobs_dispatch_total",
			Help: "Total generation job dispatch attempts.",
		},
		[]string{"job_type", "dispatch_mode"},
	)
	generationJobsDispatchFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "generation_jobs_dispatch_failed_total",
			Help: "Total failed generation job dispatch attempts.",
		},
		[]string{"job_type", "error_code"},
	)
	generationJobsStatus = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "generation_jobs_status_total",
			Help: "Total generation job status transitions observed.",
		},
		[]string{"job_type", "status"},
	)
	generationJobCreationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "generation_job_creation_duration_seconds",
			Help:    "Generation job creation request duration.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"job_type"},
	)
	inProcessWorkerJobsStarted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "inprocess_worker_jobs_started_total",
			Help: "Total in-process generation worker jobs started.",
		},
		[]string{"job_type"},
	)
	inProcessWorkerJobsCompleted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "inprocess_worker_jobs_completed_total",
			Help: "Total in-process generation worker jobs completed.",
		},
		[]string{"job_type"},
	)
	inProcessWorkerJobsFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "inprocess_worker_jobs_failed_total",
			Help: "Total in-process generation worker jobs failed.",
		},
		[]string{"job_type", "error_code"},
	)
	inProcessWorkerJobDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "inprocess_worker_job_duration_seconds",
			Help:    "In-process generation worker job duration.",
			Buckets: aiJobBuckets,
		},
		[]string{"job_type"},
	)
	workerActiveJobs = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "worker_active_jobs",
			Help: "Active worker jobs.",
		},
		[]string{"job_type"},
	)
	workerJobsStarted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "worker_jobs_started_total",
			Help: "Total worker jobs started.",
		},
		[]string{"job_type"},
	)
	workerJobsCompleted = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "worker_jobs_completed_total",
			Help: "Total worker jobs completed.",
		},
		[]string{"job_type"},
	)
	workerJobsFailed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "worker_jobs_failed_total",
			Help: "Total worker jobs failed.",
		},
		[]string{"job_type", "error_code"},
	)
	workerJobDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "worker_job_duration_seconds",
			Help:    "Worker job processing duration.",
			Buckets: aiJobBuckets,
		},
		[]string{"job_type"},
	)
	workerJobQueueDelay = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "worker_job_queue_delay_seconds",
			Help:    "Worker job queue delay from message/job creation to processing start.",
			Buckets: aiJobBuckets,
		},
		[]string{"job_type"},
	)
)

func init() {
	prometheus.MustRegister(
		generationJobsCreated,
		generationJobsDispatch,
		generationJobsDispatchFailed,
		generationJobsStatus,
		generationJobCreationDuration,
		inProcessWorkerJobsStarted,
		inProcessWorkerJobsCompleted,
		inProcessWorkerJobsFailed,
		inProcessWorkerJobDuration,
		workerActiveJobs,
		workerJobsStarted,
		workerJobsCompleted,
		workerJobsFailed,
		workerJobDuration,
		workerJobQueueDelay,
	)
}

func recordGenerationJobCreated(jobType entity.GenerationJobType, duration time.Duration) {
	label := string(jobType)
	generationJobsCreated.WithLabelValues(label).Inc()
	generationJobsStatus.WithLabelValues(label, string(entity.GenerationJobStatusQueued)).Inc()
	generationJobCreationDuration.WithLabelValues(label).Observe(duration.Seconds())
}

func recordGenerationJobDispatch(jobType entity.GenerationJobType, dispatchMode string) {
	generationJobsDispatch.WithLabelValues(string(jobType), dispatchMode).Inc()
}

func recordGenerationJobDispatchFailed(jobType entity.GenerationJobType, errorCode string) {
	generationJobsDispatchFailed.WithLabelValues(string(jobType), errorCode).Inc()
}

func recordGenerationJobStatus(jobType entity.GenerationJobType, status entity.GenerationJobStatus) {
	generationJobsStatus.WithLabelValues(string(jobType), string(status)).Inc()
}

func recordWorkerStart(job *entity.GenerationJob, inProcess bool) time.Time {
	start := time.Now()
	jobType := string(job.JobType)
	workerActiveJobs.WithLabelValues(jobType).Inc()
	workerJobsStarted.WithLabelValues(jobType).Inc()
	if !job.CreatedAt.IsZero() {
		workerJobQueueDelay.WithLabelValues(jobType).Observe(start.Sub(job.CreatedAt).Seconds())
	}
	if inProcess {
		inProcessWorkerJobsStarted.WithLabelValues(jobType).Inc()
	}
	return start
}

func recordWorkerComplete(job *entity.GenerationJob, startedAt time.Time, inProcess bool) {
	jobType := string(job.JobType)
	workerJobsCompleted.WithLabelValues(jobType).Inc()
	workerJobDuration.WithLabelValues(jobType).Observe(time.Since(startedAt).Seconds())
	workerActiveJobs.WithLabelValues(jobType).Dec()
	if inProcess {
		inProcessWorkerJobsCompleted.WithLabelValues(jobType).Inc()
		inProcessWorkerJobDuration.WithLabelValues(jobType).Observe(time.Since(startedAt).Seconds())
	}
}

func recordWorkerFailure(job *entity.GenerationJob, errorCode string, startedAt time.Time, inProcess bool) {
	jobType := string(job.JobType)
	workerJobsFailed.WithLabelValues(jobType, errorCode).Inc()
	workerJobDuration.WithLabelValues(jobType).Observe(time.Since(startedAt).Seconds())
	workerActiveJobs.WithLabelValues(jobType).Dec()
	if inProcess {
		inProcessWorkerJobsFailed.WithLabelValues(jobType, errorCode).Inc()
		inProcessWorkerJobDuration.WithLabelValues(jobType).Observe(time.Since(startedAt).Seconds())
	}
}
