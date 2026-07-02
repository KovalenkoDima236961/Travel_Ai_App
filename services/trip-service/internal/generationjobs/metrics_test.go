package generationjobs

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func TestWorkerMetricsActiveGaugeBalancesOnComplete(t *testing.T) {
	job := &entity.GenerationJob{
		JobType:   entity.GenerationJobTypeQualityImprovementItem,
		CreatedAt: time.Now().Add(-2 * time.Second),
	}
	jobType := string(job.JobType)
	activeBefore := testutil.ToFloat64(workerActiveJobs.WithLabelValues(jobType))
	completedBefore := testutil.ToFloat64(workerJobsCompleted.WithLabelValues(jobType))

	startedAt := recordWorkerStart(job, false)
	if got := testutil.ToFloat64(workerActiveJobs.WithLabelValues(jobType)); got != activeBefore+1 {
		t.Fatalf("active jobs after start = %v, want %v", got, activeBefore+1)
	}

	recordWorkerComplete(job, startedAt, false)
	if got := testutil.ToFloat64(workerActiveJobs.WithLabelValues(jobType)); got != activeBefore {
		t.Fatalf("active jobs after complete = %v, want %v", got, activeBefore)
	}
	if got := testutil.ToFloat64(workerJobsCompleted.WithLabelValues(jobType)); got != completedBefore+1 {
		t.Fatalf("completed jobs = %v, want %v", got, completedBefore+1)
	}
}

func TestWorkerMetricsActiveGaugeBalancesOnFailure(t *testing.T) {
	job := &entity.GenerationJob{JobType: entity.GenerationJobTypeQualityImprovementDay}
	jobType := string(job.JobType)
	errorCode := "metrics_test_failure"
	activeBefore := testutil.ToFloat64(workerActiveJobs.WithLabelValues(jobType))
	failedBefore := testutil.ToFloat64(workerJobsFailed.WithLabelValues(jobType, errorCode))

	startedAt := recordWorkerStart(job, true)
	if got := testutil.ToFloat64(workerActiveJobs.WithLabelValues(jobType)); got != activeBefore+1 {
		t.Fatalf("active jobs after start = %v, want %v", got, activeBefore+1)
	}

	recordWorkerFailure(job, errorCode, startedAt, true)
	if got := testutil.ToFloat64(workerActiveJobs.WithLabelValues(jobType)); got != activeBefore {
		t.Fatalf("active jobs after failure = %v, want %v", got, activeBefore)
	}
	if got := testutil.ToFloat64(workerJobsFailed.WithLabelValues(jobType, errorCode)); got != failedBefore+1 {
		t.Fatalf("failed jobs = %v, want %v", got, failedBefore+1)
	}
}
