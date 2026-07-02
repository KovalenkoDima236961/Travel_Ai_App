package generationjobs

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetoptimization"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
)

type Worker struct {
	repo  Repository
	trips TripService
	cfg   Config
	log   *zap.Logger
}

type ProcessStatus string

const (
	ProcessStatusCompleted ProcessStatus = "completed"
	ProcessStatusFailed    ProcessStatus = "failed"
	ProcessStatusSkipped   ProcessStatus = "skipped"
)

type ProcessResult struct {
	Status                  ProcessStatus
	Job                     *entity.GenerationJob
	ErrorCode               string
	ErrorMessage            string
	Retryable               bool
	ResultItineraryRevision *int
}

func NewWorker(repo Repository, trips TripService, cfg Config, log *zap.Logger) *Worker {
	if log == nil {
		log = zap.NewNop()
	}
	return &Worker{
		repo:  repo,
		trips: trips,
		cfg:   NormalizeConfig(cfg),
		log:   log,
	}
}

func (w *Worker) Start(parent context.Context) func(context.Context) error {
	if !w.cfg.Enabled || !w.cfg.WorkerEnabled || w.cfg.QueueMode() {
		w.log.Info("generation job worker disabled")
		return func(context.Context) error { return nil }
	}

	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})

	if err := w.failStaleRunningJobs(ctx); err != nil {
		w.log.Warn("failed to mark stale generation jobs failed", zap.Error(err))
	}

	go func() {
		defer close(done)
		w.run(ctx)
	}()

	return func(stopCtx context.Context) error {
		cancel()
		select {
		case <-done:
			return nil
		case <-stopCtx.Done():
			return stopCtx.Err()
		}
	}
}

func (w *Worker) run(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			processed := w.processNext(ctx)
			interval := w.cfg.PollInterval
			if processed {
				interval = 0
			}
			timer.Reset(interval)
		}
	}
}

func (w *Worker) processNext(ctx context.Context) bool {
	job, err := w.repo.ClaimNextGenerationJob(ctx)
	if err != nil {
		if isNoQueuedJob(err) {
			return false
		}
		w.log.Warn("failed to claim generation job", zap.Error(err))
		return false
	}

	w.log.Info("generation job claimed",
		zap.String("job_id", job.ID.String()),
		zap.String("trip_id", job.TripID.String()),
		zap.String("job_type", string(job.JobType)),
	)

	if _, err := w.processClaimedJob(ctx, job, true); err != nil {
		w.log.Warn("generation job processing persistence failed",
			zap.String("job_id", job.ID.String()),
			zap.Error(err),
		)
	}
	return true
}

func (w *Worker) ProcessJobByID(ctx context.Context, jobID uuid.UUID, failOnError bool) (ProcessResult, error) {
	job, claimed, err := w.claimGenerationJob(ctx, jobID)
	if err != nil {
		return ProcessResult{}, err
	}
	if !claimed {
		return ProcessResult{
			Status: ProcessStatusSkipped,
			Job:    job,
		}, nil
	}
	return w.processClaimedJob(ctx, job, failOnError)
}

func (w *Worker) ResetRunningJobForRetry(ctx context.Context, jobID uuid.UUID, code, message string) error {
	_, err := w.repo.ResetRunningGenerationJobToQueued(ctx, jobID, code, truncateSafeMessage(message))
	return err
}

func (w *Worker) FailClaimedJob(ctx context.Context, job *entity.GenerationJob, code, message string) error {
	if job == nil {
		return domainerrs.ErrNotFound
	}
	return w.failJob(ctx, job, code, message)
}

func (w *Worker) claimGenerationJob(ctx context.Context, jobID uuid.UUID) (*entity.GenerationJob, bool, error) {
	job, err := w.repo.ClaimGenerationJob(ctx, jobID)
	if err == nil {
		w.log.Info("generation job claimed",
			zap.String("job_id", job.ID.String()),
			zap.String("trip_id", job.TripID.String()),
			zap.String("job_type", string(job.JobType)),
		)
		return job, true, nil
	}
	if !isNoQueuedJob(err) {
		return nil, false, err
	}

	job, err = w.repo.GetGenerationJobByID(ctx, jobID)
	if err != nil {
		return nil, false, err
	}

	switch job.Status {
	case entity.GenerationJobStatusRunning,
		entity.GenerationJobStatusCompleted,
		entity.GenerationJobStatusFailed,
		entity.GenerationJobStatusCancelled:
		w.log.Info("generation job message skipped",
			zap.String("job_id", job.ID.String()),
			zap.String("trip_id", job.TripID.String()),
			zap.String("job_type", string(job.JobType)),
			zap.String("status", string(job.Status)),
		)
		return job, false, nil
	default:
		return job, false, ErrJobAlreadyFinished
	}
}

func (w *Worker) processClaimedJob(ctx context.Context, job *entity.GenerationJob, failOnError bool) (ProcessResult, error) {
	processCtx, cancel := context.WithTimeout(ctx, w.cfg.MaxRunning)
	defer cancel()

	updatedTrip, processErr := w.process(processCtx, job)
	if processErr != nil {
		code, message := ClassifyJobError(processErr)
		if errors.Is(processCtx.Err(), context.DeadlineExceeded) {
			code = ErrorAIGeneration
			message = "generation job timed out"
		}
		result := ProcessResult{
			Status:       ProcessStatusFailed,
			Job:          job,
			ErrorCode:    code,
			ErrorMessage: message,
			Retryable:    IsRetryableErrorCode(code),
		}
		if !failOnError {
			return result, nil
		}
		return result, w.failJob(ctx, job, code, message)
	}

	completed, err := w.repo.CompleteGenerationJob(ctx, job.ID, updatedTrip.ItineraryRevision)
	if err != nil {
		w.log.Warn("failed to complete generation job",
			zap.String("job_id", job.ID.String()),
			zap.Error(err),
		)
		return ProcessResult{}, err
	}
	w.log.Info("generation job completed",
		zap.String("job_id", completed.ID.String()),
		zap.String("trip_id", completed.TripID.String()),
		zap.Int("itinerary_revision", updatedTrip.ItineraryRevision),
	)
	revision := updatedTrip.ItineraryRevision
	return ProcessResult{
		Status:                  ProcessStatusCompleted,
		Job:                     completed,
		ResultItineraryRevision: &revision,
	}, nil
}

func (w *Worker) process(ctx context.Context, job *entity.GenerationJob) (*entity.Trip, error) {
	switch job.JobType {
	case entity.GenerationJobTypeFullGeneration:
		return w.trips.GenerateForActor(
			ctx,
			job.TripID,
			job.RequestedByUserID,
			job.ExpectedItineraryRevision,
		)
	case entity.GenerationJobTypeDayRegeneration, entity.GenerationJobTypeQualityImprovementDay:
		if job.DayNumber == nil {
			return nil, apperrs.NewInvalidInput("dayNumber is required")
		}
		return w.trips.RegenerateDayForActor(
			ctx,
			job.TripID,
			job.RequestedByUserID,
			*job.DayNumber,
			safeInstruction(job),
			job.ExpectedItineraryRevision,
		)
	case entity.GenerationJobTypeItemRegeneration, entity.GenerationJobTypeQualityImprovementItem:
		if job.DayNumber == nil || job.ItemIndex == nil {
			return nil, apperrs.NewInvalidInput("dayNumber and itemIndex are required")
		}
		return w.trips.RegenerateItemForActor(
			ctx,
			job.TripID,
			job.RequestedByUserID,
			*job.DayNumber,
			*job.ItemIndex,
			safeInstruction(job),
			job.ExpectedItineraryRevision,
		)
	case entity.GenerationJobTypeBudgetOptimizationDay:
		if job.DayNumber == nil {
			return nil, apperrs.NewInvalidInput("dayNumber is required")
		}
		return w.trips.OptimizeBudgetDayForActor(
			ctx,
			job.TripID,
			job.RequestedByUserID,
			&job.ID,
			*job.DayNumber,
			safeInstruction(job),
			job.ExpectedItineraryRevision,
			budgetoptimization.DecodeJobPayload(job.Payload),
		)
	default:
		return nil, apperrs.NewInvalidInput("jobType is invalid")
	}
}

func (w *Worker) failJob(ctx context.Context, job *entity.GenerationJob, code, message string) error {
	message = truncateSafeMessage(message)
	failed, err := w.repo.FailGenerationJob(ctx, job.ID, code, message)
	if err != nil {
		w.log.Warn("failed to mark generation job failed",
			zap.String("job_id", job.ID.String()),
			zap.String("error_code", code),
			zap.Error(err),
		)
		return err
	}
	w.log.Warn("generation job failed",
		zap.String("job_id", failed.ID.String()),
		zap.String("trip_id", failed.TripID.String()),
		zap.String("error_code", code),
		zap.String("error_message", message),
	)
	w.trips.RecordGenerationJobFailed(
		ctx,
		failed.TripID,
		failed.RequestedByUserID,
		failed.ID,
		failed.JobType,
		code,
		message,
	)
	return nil
}

func (w *Worker) failStaleRunningJobs(ctx context.Context) error {
	startedBefore := time.Now().Add(-w.cfg.MaxRunning)
	count, err := w.repo.MarkStaleRunningGenerationJobsFailed(
		ctx,
		startedBefore,
		ErrorWorkerRestarted,
		"Generation job was interrupted by service restart.",
	)
	if err != nil {
		return err
	}
	if count > 0 {
		w.log.Warn("stale running generation jobs marked failed", zap.Int64("count", count))
	}
	return nil
}

func ClassifyJobError(err error) (string, string) {
	if err == nil {
		return "", ""
	}

	var invalid *apperrs.InvalidInputError
	var dependency *apperrs.DependencyError
	var conflict *apperrs.ItineraryConflictError
	switch {
	case errors.As(err, &conflict):
		return ErrorItineraryConflict, "The itinerary changed while this job was running."
	case errors.As(err, &invalid), errors.Is(err, apperrs.ErrExpectedItineraryRevisionRequired):
		return ErrorValidationFailed, err.Error()
	case errors.As(err, &dependency):
		if strings.Contains(dependency.Error(), ErrorNoOptimizationFound) {
			return ErrorNoOptimizationFound, "No useful lower-cost proposal was found."
		}
		return ErrorAIGeneration, dependency.Error()
	case errors.Is(err, apperrs.ErrForbidden):
		return ErrorPermissionDenied, "You no longer have permission to modify this trip."
	case errors.Is(err, domainerrs.ErrNotFound):
		return ErrorTripNotFound, "Trip not found."
	case errors.Is(err, context.Canceled):
		return ErrorCancelled, "Generation job was cancelled."
	default:
		return ErrorUnknown, "Generation job failed."
	}
}

func IsRetryableErrorCode(code string) bool {
	switch code {
	case ErrorAIGeneration, ErrorEnrichment, ErrorUnknown:
		return true
	default:
		return false
	}
}

func truncateSafeMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return "Generation job failed."
	}
	if len(message) <= maxInstructionLength {
		return message
	}
	return message[:maxInstructionLength]
}
