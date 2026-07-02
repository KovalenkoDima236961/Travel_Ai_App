package generationjobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	appservice "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetoptimization"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
)

func TestServiceCreateQueueModePublishesMessage(t *testing.T) {
	userID := uuid.New()
	tripID := uuid.New()
	repo := &fakeJobRepo{}
	publisher := &fakePublisher{}
	svc := NewService(
		repo,
		fakeTripService{trip: &entity.Trip{ID: tripID, ItineraryRevision: 7}},
		Config{Enabled: true, DispatchMode: DispatchModeQueue, PublishTimeout: time.Second},
		WithPublisher(publisher),
	)

	job, err := svc.Create(auth.WithUser(context.Background(), auth.AuthenticatedUser{ID: userID}), tripID, CreateRequest{
		JobType:                   entity.GenerationJobTypeFullGeneration,
		ExpectedItineraryRevision: intPtr(7),
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if len(publisher.messages) != 1 {
		t.Fatalf("expected one published message, got %d", len(publisher.messages))
	}
	msg := publisher.messages[0]
	if msg.JobID != job.ID || msg.TripID != tripID || msg.JobType != entity.GenerationJobTypeFullGeneration {
		t.Fatalf("published message mismatch: %+v", msg)
	}
}

func TestServiceCreateInProcessModeDoesNotPublish(t *testing.T) {
	tripID := uuid.New()
	publisher := &fakePublisher{}
	svc := NewService(
		&fakeJobRepo{},
		fakeTripService{trip: &entity.Trip{ID: tripID, ItineraryRevision: 3}},
		Config{Enabled: true, DispatchMode: DispatchModeInProcess},
		WithPublisher(publisher),
	)

	_, err := svc.Create(auth.WithUser(context.Background(), auth.AuthenticatedUser{ID: uuid.New()}), tripID, CreateRequest{
		JobType:                   entity.GenerationJobTypeFullGeneration,
		ExpectedItineraryRevision: intPtr(3),
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if len(publisher.messages) != 0 {
		t.Fatalf("expected no published messages, got %d", len(publisher.messages))
	}
}

func TestServiceCreateQueuePublishFailureMarksJobFailed(t *testing.T) {
	tripID := uuid.New()
	repo := &fakeJobRepo{}
	svc := NewService(
		repo,
		fakeTripService{trip: &entity.Trip{ID: tripID, ItineraryRevision: 2}},
		Config{Enabled: true, DispatchMode: DispatchModeQueue, PublishTimeout: time.Second},
		WithPublisher(&fakePublisher{err: errors.New("rabbit down")}),
	)

	_, err := svc.Create(auth.WithUser(context.Background(), auth.AuthenticatedUser{ID: uuid.New()}), tripID, CreateRequest{
		JobType:                   entity.GenerationJobTypeFullGeneration,
		ExpectedItineraryRevision: intPtr(2),
	})
	if !errors.Is(err, ErrJobDispatchFailed) {
		t.Fatalf("expected dispatch failure, got %v", err)
	}
	if repo.failedCode != ErrorJobDispatchFailed {
		t.Fatalf("expected job marked failed with %q, got %q", ErrorJobDispatchFailed, repo.failedCode)
	}
}

type fakePublisher struct {
	messages []QueueMessage
	err      error
}

func (f *fakePublisher) PublishGenerationJob(_ context.Context, msg QueueMessage) error {
	if f.err != nil {
		return f.err
	}
	f.messages = append(f.messages, msg)
	return nil
}

type fakeJobRepo struct {
	job        *entity.GenerationJob
	failedCode string
	staleCalls int
}

func (f *fakeJobRepo) CreateGenerationJob(_ context.Context, job *entity.GenerationJob) (*entity.GenerationJob, error) {
	now := time.Now()
	out := *job
	out.CreatedAt = now
	out.UpdatedAt = now
	f.job = &out
	return &out, nil
}

func (f *fakeJobRepo) GetGenerationJobByID(_ context.Context, id uuid.UUID) (*entity.GenerationJob, error) {
	if f.job != nil && f.job.ID == id {
		return f.job, nil
	}
	return nil, domainerrs.ErrNotFound
}

func (f *fakeJobRepo) GetGenerationJobByIDAndTrip(_ context.Context, id, tripID uuid.UUID) (*entity.GenerationJob, error) {
	if f.job != nil && f.job.ID == id && f.job.TripID == tripID {
		return f.job, nil
	}
	return nil, domainerrs.ErrNotFound
}

func (f *fakeJobRepo) ListGenerationJobsByTrip(_ context.Context, tripID uuid.UUID, _ int) ([]entity.GenerationJob, error) {
	if f.job != nil && f.job.TripID == tripID {
		return []entity.GenerationJob{*f.job}, nil
	}
	return []entity.GenerationJob{}, nil
}

func (f *fakeJobRepo) ClaimNextGenerationJob(context.Context) (*entity.GenerationJob, error) {
	return nil, domainerrs.ErrNotFound
}

func (f *fakeJobRepo) ClaimGenerationJob(_ context.Context, id uuid.UUID) (*entity.GenerationJob, error) {
	if f.job != nil && f.job.ID == id && f.job.Status == entity.GenerationJobStatusQueued {
		out := *f.job
		out.Status = entity.GenerationJobStatusRunning
		f.job = &out
		return &out, nil
	}
	return nil, domainerrs.ErrNotFound
}

func (f *fakeJobRepo) CompleteGenerationJob(_ context.Context, id uuid.UUID, revision int) (*entity.GenerationJob, error) {
	if f.job != nil && f.job.ID == id {
		out := *f.job
		out.Status = entity.GenerationJobStatusCompleted
		out.ResultItineraryRevision = &revision
		f.job = &out
		return &out, nil
	}
	return nil, domainerrs.ErrNotFound
}

func (f *fakeJobRepo) FailGenerationJob(_ context.Context, id uuid.UUID, code string, message string) (*entity.GenerationJob, error) {
	f.failedCode = code
	if f.job != nil && f.job.ID == id {
		out := *f.job
		out.Status = entity.GenerationJobStatusFailed
		out.ErrorCode = &code
		out.ErrorMessage = &message
		f.job = &out
		return &out, nil
	}
	return nil, domainerrs.ErrNotFound
}

func (f *fakeJobRepo) ResetRunningGenerationJobToQueued(_ context.Context, id uuid.UUID, code string, message string) (*entity.GenerationJob, error) {
	if f.job != nil && f.job.ID == id {
		out := *f.job
		out.Status = entity.GenerationJobStatusQueued
		out.ErrorCode = &code
		out.ErrorMessage = &message
		f.job = &out
		return &out, nil
	}
	return nil, domainerrs.ErrNotFound
}

func (f *fakeJobRepo) CancelQueuedGenerationJob(_ context.Context, id uuid.UUID) (*entity.GenerationJob, error) {
	if f.job != nil && f.job.ID == id {
		out := *f.job
		out.Status = entity.GenerationJobStatusCancelled
		f.job = &out
		return &out, nil
	}
	return nil, domainerrs.ErrNotFound
}

func (f *fakeJobRepo) MarkStaleRunningGenerationJobsFailed(context.Context, time.Time, string, string) (int64, error) {
	f.staleCalls++
	return 0, nil
}

type fakeTripService struct {
	trip *entity.Trip
}

func (f fakeTripService) GetTripForActor(context.Context, uuid.UUID, uuid.UUID) (*entity.Trip, appservice.TripAccess, error) {
	return f.trip, appservice.TripAccess{Level: appservice.AccessLevelOwner}, nil
}

func (f fakeTripService) GenerateForActor(context.Context, uuid.UUID, uuid.UUID, int) (*entity.Trip, error) {
	return &entity.Trip{ItineraryRevision: 1}, nil
}

func (f fakeTripService) RegenerateDayForActor(context.Context, uuid.UUID, uuid.UUID, int, string, int) (*entity.Trip, error) {
	return &entity.Trip{ItineraryRevision: 1}, nil
}

func (f fakeTripService) RegenerateItemForActor(context.Context, uuid.UUID, uuid.UUID, int, int, string, int) (*entity.Trip, error) {
	return &entity.Trip{ItineraryRevision: 1}, nil
}

func (f fakeTripService) OptimizeBudgetDayForActor(context.Context, uuid.UUID, uuid.UUID, *uuid.UUID, int, string, int, budgetoptimization.JobPayload) (*entity.Trip, error) {
	return &entity.Trip{ItineraryRevision: 1}, nil
}

func (f fakeTripService) RecordGenerationJobFailed(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, entity.GenerationJobType, string, string) {
}

func intPtr(v int) *int {
	return &v
}
