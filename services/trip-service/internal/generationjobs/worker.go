package generationjobs

import (
	"context"
	"errors"
	"strings"
	"time"

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

func NewWorker(repo Repository, trips TripService, cfg Config, log *zap.Logger) *Worker {
	return &Worker{
		repo:  repo,
		trips: trips,
		cfg:   NormalizeConfig(cfg),
		log:   log,
	}
}

func (w *Worker) Start(parent context.Context) func(context.Context) error {
	if !w.cfg.Enabled || !w.cfg.WorkerEnabled {
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

	processCtx, cancel := context.WithTimeout(ctx, w.cfg.MaxRunning)
	defer cancel()

	updatedTrip, processErr := w.process(processCtx, job)
	if processErr != nil {
		code, message := classifyJobError(processErr)
		if errors.Is(processCtx.Err(), context.DeadlineExceeded) {
			code = ErrorAIGeneration
			message = "generation job timed out"
		}
		w.failJob(ctx, job, code, message)
		return true
	}

	completed, err := w.repo.CompleteGenerationJob(ctx, job.ID, updatedTrip.ItineraryRevision)
	if err != nil {
		w.log.Warn("failed to complete generation job",
			zap.String("job_id", job.ID.String()),
			zap.Error(err),
		)
		return true
	}
	w.log.Info("generation job completed",
		zap.String("job_id", completed.ID.String()),
		zap.String("trip_id", completed.TripID.String()),
		zap.Int("itinerary_revision", updatedTrip.ItineraryRevision),
	)
	return true
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

func (w *Worker) failJob(ctx context.Context, job *entity.GenerationJob, code, message string) {
	message = truncateSafeMessage(message)
	failed, err := w.repo.FailGenerationJob(ctx, job.ID, code, message)
	if err != nil {
		w.log.Warn("failed to mark generation job failed",
			zap.String("job_id", job.ID.String()),
			zap.String("error_code", code),
			zap.Error(err),
		)
		return
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

func classifyJobError(err error) (string, string) {
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
