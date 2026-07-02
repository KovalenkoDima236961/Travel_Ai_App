package generationjobs

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

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
