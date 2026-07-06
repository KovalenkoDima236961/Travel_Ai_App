package generationjobs

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func TestWorkerProcessesTemplateAdaptationJob(t *testing.T) {
	jobID := uuid.New()
	tripID := uuid.New()
	repo := &fakeJobRepo{job: &entity.GenerationJob{
		ID:      jobID,
		TripID:  tripID,
		JobType: entity.GenerationJobTypeTemplateAdaptation,
		Status:  entity.GenerationJobStatusQueued,
	}}
	worker := NewWorker(
		repo,
		fakeTripService{trip: &entity.Trip{ID: tripID, ItineraryRevision: 1}},
		Config{Enabled: true, WorkerEnabled: true, DispatchMode: DispatchModeQueue, MaxRunning: time.Second},
		zap.NewNop(),
	)

	result, err := worker.ProcessJobByID(context.Background(), jobID, true)
	if err != nil {
		t.Fatalf("process template adaptation job: %v", err)
	}
	if result.Status != ProcessStatusCompleted {
		t.Fatalf("expected completed, got %s (code=%s)", result.Status, result.ErrorCode)
	}
	if len(repo.resultPayloadSet) == 0 {
		t.Fatal("expected adaptation summary result payload to be persisted")
	}
}

func TestWorkerDoesNotStartPollingInQueueMode(t *testing.T) {
	repo := &fakeJobRepo{}
	worker := NewWorker(
		repo,
		fakeTripService{trip: &entity.Trip{ItineraryRevision: 1}},
		Config{
			Enabled:       true,
			WorkerEnabled: true,
			DispatchMode:  DispatchModeQueue,
			PollInterval:  time.Millisecond,
			MaxRunning:    time.Second,
		},
		zap.NewNop(),
	)

	stop := worker.Start(context.Background())
	if err := stop(context.Background()); err != nil {
		t.Fatalf("stop worker: %v", err)
	}
	if repo.staleCalls != 0 {
		t.Fatalf("expected queue mode not to run stale cleanup, got %d calls", repo.staleCalls)
	}
}
