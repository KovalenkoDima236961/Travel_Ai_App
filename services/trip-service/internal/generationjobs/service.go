package generationjobs

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	appservice "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetoptimization"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	tripobs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/observability"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/observability"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/triprepair"
)

type Repository interface {
	CreateGenerationJob(ctx context.Context, job *entity.GenerationJob) (*entity.GenerationJob, error)
	GetGenerationJobByID(ctx context.Context, id uuid.UUID) (*entity.GenerationJob, error)
	GetGenerationJobByIDAndTrip(ctx context.Context, id, tripID uuid.UUID) (*entity.GenerationJob, error)
	ListGenerationJobsByTrip(ctx context.Context, tripID uuid.UUID, limit int) ([]entity.GenerationJob, error)
	ListOpsGenerationJobs(ctx context.Context, filters OpsJobListFilters) ([]entity.GenerationJob, error)
	ListOpsTripMetadata(ctx context.Context, tripIDs []uuid.UUID) (map[uuid.UUID]OpsTripMetadata, error)
	CountOpsJobsByStatus(ctx context.Context) (map[entity.GenerationJobStatus]int, error)
	CountOpsJobsByType(ctx context.Context) (map[entity.GenerationJobType]int, error)
	ListRecentFailedOpsJobs(ctx context.Context, limit int) ([]entity.GenerationJob, error)
	CountStaleRunningGenerationJobs(ctx context.Context, startedBefore time.Time) (int, error)
	ClaimNextGenerationJob(ctx context.Context) (*entity.GenerationJob, error)
	ClaimGenerationJob(ctx context.Context, id uuid.UUID) (*entity.GenerationJob, error)
	CompleteGenerationJob(ctx context.Context, id uuid.UUID, resultItineraryRevision int) (*entity.GenerationJob, error)
	FailGenerationJob(ctx context.Context, id uuid.UUID, errorCode string, errorMessage string) (*entity.GenerationJob, error)
	ResetRunningGenerationJobToQueued(ctx context.Context, id uuid.UUID, errorCode string, errorMessage string) (*entity.GenerationJob, error)
	CancelQueuedGenerationJob(ctx context.Context, id uuid.UUID) (*entity.GenerationJob, error)
	CancelOpsGenerationJob(ctx context.Context, id uuid.UUID, errorCode, errorMessage string) (*entity.GenerationJob, error)
	MarkOpsGenerationJobFailed(ctx context.Context, id uuid.UUID, startedBefore time.Time, errorCode, errorMessage string) (*entity.GenerationJob, error)
	MarkStaleRunningGenerationJobsFailed(ctx context.Context, startedBefore time.Time, errorCode string, errorMessage string) (int64, error)
	CreateOpsAuditEvent(ctx context.Context, event OpsAuditEvent) error
	SetGenerationJobResultPayload(ctx context.Context, id uuid.UUID, payload json.RawMessage) error
}

type TripService interface {
	GetTripForActor(ctx context.Context, tripID, actorUserID uuid.UUID) (*entity.Trip, appservice.TripAccess, error)
	GenerateForActor(ctx context.Context, tripID, actorUserID uuid.UUID, expectedRevision int) (*entity.Trip, error)
	RegenerateDayForActor(ctx context.Context, tripID, actorUserID uuid.UUID, dayNumber int, instruction string, expectedRevision int) (*entity.Trip, error)
	RegenerateItemForActor(ctx context.Context, tripID, actorUserID uuid.UUID, dayNumber, itemIndex int, instruction string, expectedRevision int) (*entity.Trip, error)
	OptimizeBudgetDayForActor(ctx context.Context, tripID, actorUserID uuid.UUID, jobID *uuid.UUID, dayNumber int, instruction string, expectedRevision int, payload budgetoptimization.JobPayload) (*entity.Trip, error)
	PrepareTemplateAdaptation(ctx context.Context, templateID uuid.UUID, in appdto.CreateTemplateAdaptationInput) (*entity.Trip, json.RawMessage, error)
	AdaptTemplateForActor(ctx context.Context, tripID, actorUserID uuid.UUID, expectedRevision int, requestPayload json.RawMessage) (*entity.Trip, json.RawMessage, error)
	RepairItineraryForActor(ctx context.Context, tripID, actorUserID uuid.UUID, jobID *uuid.UUID, expectedRevision int, payload triprepair.JobPayload) (*entity.Trip, json.RawMessage, error)
	RecordGenerationJobFailed(ctx context.Context, tripID, requesterID, jobID uuid.UUID, jobType entity.GenerationJobType, errorCode, errorMessage string)
}

type Service struct {
	repo      Repository
	trips     TripService
	cfg       Config
	publisher JobPublisher
}

type Option func(*Service)

func WithPublisher(publisher JobPublisher) Option {
	return func(s *Service) {
		s.publisher = publisher
	}
}

func NewService(repo Repository, trips TripService, cfg Config, opts ...Option) *Service {
	s := &Service{
		repo:  repo,
		trips: trips,
		cfg:   NormalizeConfig(cfg),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Service) Create(ctx context.Context, tripID uuid.UUID, req CreateRequest) (*entity.GenerationJob, error) {
	startedAt := time.Now()
	if !s.cfg.Enabled {
		return nil, ErrDisabled
	}

	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	trip, access, err := s.trips.GetTripForActor(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	if !access.CanEdit() {
		return nil, apperrs.ErrForbidden
	}

	expectedRevision, err := requireExpectedRevision(req.ExpectedItineraryRevision)
	if err != nil {
		return nil, err
	}
	if expectedRevision != trip.ItineraryRevision {
		return nil, apperrs.NewItineraryConflict(trip.ItineraryRevision)
	}

	instruction, err := normalizeInstruction(req.Instruction)
	if err != nil {
		return nil, err
	}
	if err := validateJobTarget(req.JobType, trip, req.DayNumber, req.ItemIndex); err != nil {
		return nil, err
	}

	ctx, requestID, correlationID := observability.EnsureRequestIDs(ctx)
	job, err := s.repo.CreateGenerationJob(ctx, &entity.GenerationJob{
		ID:                        uuid.New(),
		TripID:                    tripID,
		RequestedByUserID:         user.ID,
		JobType:                   req.JobType,
		Status:                    entity.GenerationJobStatusQueued,
		ExpectedItineraryRevision: expectedRevision,
		Instruction:               instruction,
		DayNumber:                 req.DayNumber,
		ItemIndex:                 req.ItemIndex,
		Payload:                   req.Payload,
		CorrelationID:             &correlationID,
		RequestID:                 &requestID,
	})
	if err != nil {
		return nil, err
	}
	recordGenerationJobCreated(req.JobType, time.Since(startedAt))
	if req.JobType == entity.GenerationJobTypeBudgetOptimizationDay {
		tripobs.RecordBudgetOptimizationJobCreated()
	}

	return s.dispatchGenerationJob(ctx, job)
}

// dispatchGenerationJob publishes a queued job to the worker queue (queue mode)
// or leaves it for the in-process poller (in-process mode), marking the job
// failed when a required publish cannot be completed and fail-open is disabled.
func (s *Service) dispatchGenerationJob(ctx context.Context, job *entity.GenerationJob) (*entity.GenerationJob, error) {
	if s.cfg.QueueMode() {
		recordGenerationJobDispatch(job.JobType, string(s.cfg.DispatchMode))
		if s.publisher == nil {
			recordGenerationJobDispatchFailed(job.JobType, ErrorJobDispatchFailed)
			failed, _ := s.repo.FailGenerationJob(ctx, job.ID, ErrorJobDispatchFailed, "Generation job could not be dispatched.")
			if failed != nil {
				recordGenerationJobStatus(failed.JobType, failed.Status)
			}
			return nil, ErrJobDispatchFailed
		}
		publishCtx, cancel := context.WithTimeout(ctx, s.cfg.PublishTimeout)
		err := s.publisher.PublishGenerationJob(publishCtx, NewQueueMessageFromContext(ctx, job))
		cancel()
		if err != nil {
			recordGenerationJobDispatchFailed(job.JobType, ErrorJobDispatchFailed)
			if s.cfg.PublishFailOpen {
				return job, nil
			}
			failed, _ := s.repo.FailGenerationJob(ctx, job.ID, ErrorJobDispatchFailed, "Generation job could not be dispatched.")
			if failed != nil {
				recordGenerationJobStatus(failed.JobType, failed.Status)
			}
			return nil, ErrJobDispatchFailed
		}
	} else {
		recordGenerationJobDispatch(job.JobType, string(s.cfg.DispatchMode))
	}
	return job, nil
}

// CreateTemplateAdaptation creates the draft trip and a queued template
// adaptation job, then dispatches it. The draft trip is created up front (like
// full generation) so the existing per-trip job status endpoint works, and the
// adaptation request is stored in the job payload for the worker.
func (s *Service) CreateTemplateAdaptation(ctx context.Context, templateID uuid.UUID, req CreateTemplateAdaptationRequest) (*entity.GenerationJob, error) {
	startedAt := time.Now()
	if !s.cfg.Enabled {
		return nil, ErrDisabled
	}

	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	fallback := true
	if req.FallbackToDeterministic != nil {
		fallback = *req.FallbackToDeterministic
	}
	in := appdto.CreateTemplateAdaptationInput{
		Title:                   req.Title,
		Destination:             req.Destination,
		StartDate:               req.StartDate,
		DurationDays:            req.DurationDays,
		WorkspaceID:             req.WorkspaceID,
		Travelers:               req.Travelers,
		Pace:                    req.Pace,
		Interests:               req.Interests,
		Avoid:                   req.Avoid,
		SpecialInstructions:     req.SpecialInstructions,
		FallbackToDeterministic: fallback,
	}
	if req.Budget != nil {
		in.BudgetAmount = req.Budget.Amount
		in.BudgetCurrency = req.Budget.Currency
	}

	trip, payload, err := s.trips.PrepareTemplateAdaptation(ctx, templateID, in)
	if err != nil {
		return nil, err
	}

	ctx, requestID, correlationID := observability.EnsureRequestIDs(ctx)
	job, err := s.repo.CreateGenerationJob(ctx, &entity.GenerationJob{
		ID:                        uuid.New(),
		TripID:                    trip.ID,
		RequestedByUserID:         user.ID,
		JobType:                   entity.GenerationJobTypeTemplateAdaptation,
		Status:                    entity.GenerationJobStatusQueued,
		ExpectedItineraryRevision: trip.ItineraryRevision,
		Payload:                   payload,
		CorrelationID:             &correlationID,
		RequestID:                 &requestID,
	})
	if err != nil {
		return nil, err
	}
	recordGenerationJobCreated(entity.GenerationJobTypeTemplateAdaptation, time.Since(startedAt))

	return s.dispatchGenerationJob(ctx, job)
}

func (s *Service) Get(ctx context.Context, tripID, jobID uuid.UUID) (*entity.GenerationJob, error) {
	if !s.cfg.Enabled {
		return nil, ErrDisabled
	}

	if err := s.requireViewAccess(ctx, tripID); err != nil {
		return nil, err
	}
	return s.repo.GetGenerationJobByIDAndTrip(ctx, jobID, tripID)
}

func (s *Service) List(ctx context.Context, tripID uuid.UUID, limit int) ([]entity.GenerationJob, int, error) {
	if !s.cfg.Enabled {
		return nil, 0, ErrDisabled
	}

	if err := s.requireViewAccess(ctx, tripID); err != nil {
		return nil, 0, err
	}
	if limit == 0 {
		limit = defaultLimit
	}
	if limit < 1 || limit > maxLimit {
		return nil, 0, apperrs.NewInvalidInput("limit must be between 1 and %d", maxLimit)
	}
	jobs, err := s.repo.ListGenerationJobsByTrip(ctx, tripID, limit)
	return jobs, limit, err
}

func (s *Service) Cancel(ctx context.Context, tripID, jobID uuid.UUID) (*entity.GenerationJob, error) {
	if !s.cfg.Enabled {
		return nil, ErrDisabled
	}

	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	_, access, err := s.trips.GetTripForActor(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	if !access.CanView() {
		return nil, domainerrs.ErrNotFound
	}
	job, err := s.repo.GetGenerationJobByIDAndTrip(ctx, jobID, tripID)
	if err != nil {
		return nil, err
	}
	if job.Status != entity.GenerationJobStatusQueued {
		return nil, ErrNotCancellable
	}
	if access.Level != appservice.AccessLevelOwner && job.RequestedByUserID != user.ID {
		return nil, apperrs.ErrForbidden
	}
	cancelled, err := s.repo.CancelQueuedGenerationJob(ctx, jobID)
	if err == nil {
		recordGenerationJobStatus(cancelled.JobType, cancelled.Status)
	}
	return cancelled, err
}

func (s *Service) requireViewAccess(ctx context.Context, tripID uuid.UUID) error {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	_, access, err := s.trips.GetTripForActor(ctx, tripID, user.ID)
	if err != nil {
		return err
	}
	if !access.CanView() {
		return domainerrs.ErrNotFound
	}
	return nil
}

func requireExpectedRevision(expected *int) (int, error) {
	if expected == nil {
		return 0, apperrs.ErrExpectedItineraryRevisionRequired
	}
	if *expected < 0 {
		return 0, apperrs.NewInvalidInput("expectedItineraryRevision must be >= 0")
	}
	return *expected, nil
}

func normalizeInstruction(input *string) (*string, error) {
	if input == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*input)
	if len(trimmed) > maxInstructionLength {
		return nil, apperrs.NewInvalidInput("instruction must be at most %d characters", maxInstructionLength)
	}
	if trimmed == "" {
		return nil, nil
	}
	return &trimmed, nil
}

func validateJobTarget(
	jobType entity.GenerationJobType,
	trip *entity.Trip,
	dayNumber *int,
	itemIndex *int,
) error {
	switch jobType {
	case entity.GenerationJobTypeFullGeneration,
		entity.GenerationJobTypePolicyRepair:
		return nil
	case entity.GenerationJobTypeDayRegeneration,
		entity.GenerationJobTypeQualityImprovementDay,
		entity.GenerationJobTypeBudgetOptimizationDay:
		if dayNumber == nil || *dayNumber < 1 {
			return apperrs.NewInvalidInput("dayNumber is required and must be > 0")
		}
		return requireDayExists(trip, *dayNumber)
	case entity.GenerationJobTypeItemRegeneration, entity.GenerationJobTypeQualityImprovementItem:
		if dayNumber == nil || *dayNumber < 1 {
			return apperrs.NewInvalidInput("dayNumber is required and must be > 0")
		}
		if itemIndex == nil || *itemIndex < 0 {
			return apperrs.NewInvalidInput("itemIndex is required and must be >= 0")
		}
		return requireItemExists(trip, *dayNumber, *itemIndex)
	default:
		return apperrs.NewInvalidInput("jobType is invalid")
	}
}

func requireDayExists(trip *entity.Trip, dayNumber int) error {
	_, _, err := getCurrentItineraryDay(trip, dayNumber)
	return err
}

func requireItemExists(trip *entity.Trip, dayNumber, itemIndex int) error {
	itinerary, dayIndex, err := getCurrentItineraryDay(trip, dayNumber)
	if err != nil {
		return err
	}
	if itemIndex >= len(itinerary.Days[dayIndex].Items) {
		return apperrs.NewInvalidInput("current itinerary is invalid")
	}
	return nil
}

func getCurrentItineraryDay(trip *entity.Trip, dayNumber int) (aggregate.Itinerary, int, error) {
	if trip == nil || len(trip.Itinerary) == 0 || strings.EqualFold(strings.TrimSpace(string(trip.Itinerary)), "null") {
		return aggregate.Itinerary{}, -1, apperrs.NewInvalidInput("current itinerary is invalid")
	}

	var itinerary aggregate.Itinerary
	if err := json.Unmarshal(trip.Itinerary, &itinerary); err != nil {
		return aggregate.Itinerary{}, -1, apperrs.NewInvalidInput("current itinerary is invalid")
	}
	for index := range itinerary.Days {
		if itinerary.Days[index].Day == dayNumber {
			return itinerary, index, nil
		}
	}
	return aggregate.Itinerary{}, -1, apperrs.NewInvalidInput("current itinerary is invalid")
}

func safeInstruction(job *entity.GenerationJob) string {
	if job == nil || job.Instruction == nil {
		return ""
	}
	return *job.Instruction
}

func isNoQueuedJob(err error) bool {
	return errors.Is(err, domainerrs.ErrNotFound)
}
