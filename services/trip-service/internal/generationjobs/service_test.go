package generationjobs

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
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

func TestServiceCreateTemplateAdaptationDispatches(t *testing.T) {
	userID := uuid.New()
	tripID := uuid.New()
	templateID := uuid.New()
	repo := &fakeJobRepo{}
	publisher := &fakePublisher{}
	svc := NewService(
		repo,
		fakeTripService{trip: &entity.Trip{ID: tripID, ItineraryRevision: 0}},
		Config{Enabled: true, DispatchMode: DispatchModeQueue, PublishTimeout: time.Second},
		WithPublisher(publisher),
	)

	job, err := svc.CreateTemplateAdaptation(
		auth.WithUser(context.Background(), auth.AuthenticatedUser{ID: userID}),
		templateID,
		CreateTemplateAdaptationRequest{
			Title:        "Vienna weekend",
			Destination:  "Vienna",
			StartDate:    "2026-09-10",
			DurationDays: 3,
		},
	)
	if err != nil {
		t.Fatalf("create template adaptation job: %v", err)
	}
	if job.JobType != entity.GenerationJobTypeTemplateAdaptation {
		t.Fatalf("expected template_adaptation job type, got %s", job.JobType)
	}
	if job.TripID != tripID {
		t.Fatalf("expected job attached to draft trip %s, got %s", tripID, job.TripID)
	}
	if len(publisher.messages) != 1 {
		t.Fatalf("expected one published message, got %d", len(publisher.messages))
	}
	if publisher.messages[0].JobType != entity.GenerationJobTypeTemplateAdaptation {
		t.Fatalf("expected template_adaptation queue message, got %s", publisher.messages[0].JobType)
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
	job              *entity.GenerationJob
	failedCode       string
	staleCalls       int
	resultPayloadSet json.RawMessage
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

func (f *fakeJobRepo) ListOpsGenerationJobs(context.Context, OpsJobListFilters) ([]entity.GenerationJob, error) {
	if f.job == nil {
		return []entity.GenerationJob{}, nil
	}
	return []entity.GenerationJob{*f.job}, nil
}

func (f *fakeJobRepo) ListOpsTripMetadata(_ context.Context, tripIDs []uuid.UUID) (map[uuid.UUID]OpsTripMetadata, error) {
	out := make(map[uuid.UUID]OpsTripMetadata, len(tripIDs))
	for _, tripID := range tripIDs {
		out[tripID] = OpsTripMetadata{TripID: tripID}
	}
	return out, nil
}

func (f *fakeJobRepo) CountOpsJobsByStatus(context.Context) (map[entity.GenerationJobStatus]int, error) {
	out := map[entity.GenerationJobStatus]int{}
	if f.job != nil {
		out[f.job.Status] = 1
	}
	return out, nil
}

func (f *fakeJobRepo) CountOpsJobsByType(context.Context) (map[entity.GenerationJobType]int, error) {
	out := map[entity.GenerationJobType]int{}
	if f.job != nil {
		out[f.job.JobType] = 1
	}
	return out, nil
}

func (f *fakeJobRepo) ListRecentFailedOpsJobs(context.Context, int) ([]entity.GenerationJob, error) {
	if f.job != nil && f.job.Status == entity.GenerationJobStatusFailed {
		return []entity.GenerationJob{*f.job}, nil
	}
	return []entity.GenerationJob{}, nil
}

func (f *fakeJobRepo) CountStaleRunningGenerationJobs(context.Context, time.Time) (int, error) {
	return 0, nil
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

func (f *fakeJobRepo) CancelOpsGenerationJob(_ context.Context, id uuid.UUID, code, message string) (*entity.GenerationJob, error) {
	if f.job != nil && f.job.ID == id {
		out := *f.job
		out.Status = entity.GenerationJobStatusCancelled
		out.ErrorCode = &code
		out.ErrorMessage = &message
		f.job = &out
		return &out, nil
	}
	return nil, domainerrs.ErrNotFound
}

func (f *fakeJobRepo) MarkOpsGenerationJobFailed(_ context.Context, id uuid.UUID, _ time.Time, code, message string) (*entity.GenerationJob, error) {
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

func (f *fakeJobRepo) MarkStaleRunningGenerationJobsFailed(context.Context, time.Time, string, string) (int64, error) {
	f.staleCalls++
	return 0, nil
}

func (f *fakeJobRepo) CreateOpsAuditEvent(context.Context, OpsAuditEvent) error {
	return nil
}

func (f *fakeJobRepo) SetGenerationJobResultPayload(_ context.Context, _ uuid.UUID, payload json.RawMessage) error {
	f.resultPayloadSet = payload
	if f.job != nil {
		f.job.ResultPayload = payload
	}
	return nil
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

func (f fakeTripService) PrepareTemplateAdaptation(context.Context, uuid.UUID, appdto.CreateTemplateAdaptationInput) (*entity.Trip, json.RawMessage, error) {
	return f.trip, json.RawMessage(`{}`), nil
}

func (f fakeTripService) AdaptTemplateForActor(context.Context, uuid.UUID, uuid.UUID, int, json.RawMessage) (*entity.Trip, json.RawMessage, error) {
	return &entity.Trip{ItineraryRevision: 1}, json.RawMessage(`{}`), nil
}

func (f fakeTripService) RecordGenerationJobFailed(context.Context, uuid.UUID, uuid.UUID, uuid.UUID, entity.GenerationJobType, string, string) {
}

func intPtr(v int) *int {
	return &v
}
