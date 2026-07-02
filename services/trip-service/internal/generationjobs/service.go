package generationjobs

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	appservice "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetoptimization"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
)

type Repository interface {
	CreateGenerationJob(ctx context.Context, job *entity.GenerationJob) (*entity.GenerationJob, error)
	GetGenerationJobByID(ctx context.Context, id uuid.UUID) (*entity.GenerationJob, error)
	GetGenerationJobByIDAndTrip(ctx context.Context, id, tripID uuid.UUID) (*entity.GenerationJob, error)
	ListGenerationJobsByTrip(ctx context.Context, tripID uuid.UUID, limit int) ([]entity.GenerationJob, error)
	ClaimNextGenerationJob(ctx context.Context) (*entity.GenerationJob, error)
	CompleteGenerationJob(ctx context.Context, id uuid.UUID, resultItineraryRevision int) (*entity.GenerationJob, error)
	FailGenerationJob(ctx context.Context, id uuid.UUID, errorCode string, errorMessage string) (*entity.GenerationJob, error)
	CancelQueuedGenerationJob(ctx context.Context, id uuid.UUID) (*entity.GenerationJob, error)
	MarkStaleRunningGenerationJobsFailed(ctx context.Context, startedBefore time.Time, errorCode string, errorMessage string) (int64, error)
}

type TripService interface {
	GetTripForActor(ctx context.Context, tripID, actorUserID uuid.UUID) (*entity.Trip, appservice.TripAccess, error)
	GenerateForActor(ctx context.Context, tripID, actorUserID uuid.UUID, expectedRevision int) (*entity.Trip, error)
	RegenerateDayForActor(ctx context.Context, tripID, actorUserID uuid.UUID, dayNumber int, instruction string, expectedRevision int) (*entity.Trip, error)
	RegenerateItemForActor(ctx context.Context, tripID, actorUserID uuid.UUID, dayNumber, itemIndex int, instruction string, expectedRevision int) (*entity.Trip, error)
	OptimizeBudgetDayForActor(ctx context.Context, tripID, actorUserID uuid.UUID, jobID *uuid.UUID, dayNumber int, instruction string, expectedRevision int, payload budgetoptimization.JobPayload) (*entity.Trip, error)
	RecordGenerationJobFailed(ctx context.Context, tripID, requesterID, jobID uuid.UUID, jobType entity.GenerationJobType, errorCode, errorMessage string)
}

type Service struct {
	repo  Repository
	trips TripService
	cfg   Config
}

func NewService(repo Repository, trips TripService, cfg Config) *Service {
	return &Service{
		repo:  repo,
		trips: trips,
		cfg:   NormalizeConfig(cfg),
	}
}

func (s *Service) Create(ctx context.Context, tripID uuid.UUID, req CreateRequest) (*entity.GenerationJob, error) {
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

	return s.repo.CreateGenerationJob(ctx, &entity.GenerationJob{
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
	})
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
	return s.repo.CancelQueuedGenerationJob(ctx, jobID)
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
	case entity.GenerationJobTypeFullGeneration:
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
