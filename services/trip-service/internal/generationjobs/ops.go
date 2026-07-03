package generationjobs

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetoptimization"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/observability"
)

const (
	OpsActionJobRetried      = "ops_job_retried"
	OpsActionJobCancelled    = "ops_job_cancelled"
	OpsActionJobMarkedFailed = "ops_job_marked_failed"

	opsCancelErrorCode     = "ops_cancelled"
	opsMarkFailedErrorCode = "ops_marked_failed"

	opsDefaultLimit = 50
	opsMaxLimit     = 200
	opsReasonMaxLen = 500
)

type OpsJobListFilters struct {
	Status        *entity.GenerationJobStatus
	JobType       *entity.GenerationJobType
	TripID        *uuid.UUID
	UserID        *uuid.UUID
	ErrorCode     string
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	Limit         int
	Offset        int
}

type OpsAuditEvent struct {
	ID          uuid.UUID
	ActorUserID uuid.UUID
	ActorEmail  string
	Action      string
	EntityType  string
	EntityID    uuid.UUID
	Reason      string
	Metadata    map[string]any
}

func (s *Service) OpsList(ctx context.Context, filters OpsJobListFilters) (OpsJobListResponse, error) {
	filters = normalizeOpsFilters(filters)
	jobs, err := s.repo.ListOpsGenerationJobs(ctx, filters)
	if err != nil {
		return OpsJobListResponse{}, err
	}
	items := make([]OpsJobResponse, 0, len(jobs))
	metadata, err := s.opsTripMetadata(ctx, collectOpsTripIDs(jobs))
	if err != nil {
		return OpsJobListResponse{}, err
	}
	for i := range jobs {
		items = append(items, NewOpsJobResponse(&jobs[i], 0, false, metadata[jobs[i].TripID]))
	}
	var nextOffset *int
	if len(jobs) == filters.Limit {
		v := filters.Offset + filters.Limit
		nextOffset = &v
	}
	return OpsJobListResponse{Jobs: items, NextOffset: nextOffset}, nil
}

func (s *Service) OpsGet(ctx context.Context, jobID uuid.UUID, staleThreshold time.Duration) (OpsJobEnvelope, error) {
	job, err := s.repo.GetGenerationJobByID(ctx, jobID)
	if err != nil {
		return OpsJobEnvelope{}, err
	}
	metadata, err := s.opsTripMetadata(ctx, []uuid.UUID{job.TripID})
	if err != nil {
		return OpsJobEnvelope{}, err
	}
	return OpsJobEnvelope{Job: NewOpsJobResponse(job, staleThreshold, true, metadata[job.TripID])}, nil
}

func (s *Service) OpsSummary(ctx context.Context, staleThreshold time.Duration) (OpsJobSummaryResponse, error) {
	statusCounts, err := s.repo.CountOpsJobsByStatus(ctx)
	if err != nil {
		return OpsJobSummaryResponse{}, err
	}
	typeCounts, err := s.repo.CountOpsJobsByType(ctx)
	if err != nil {
		return OpsJobSummaryResponse{}, err
	}
	recent, err := s.repo.ListRecentFailedOpsJobs(ctx, 10)
	if err != nil {
		return OpsJobSummaryResponse{}, err
	}
	staleCount, err := s.repo.CountStaleRunningGenerationJobs(ctx, time.Now().Add(-staleThreshold))
	if err != nil {
		return OpsJobSummaryResponse{}, err
	}

	failures := make([]OpsRecentFailure, 0, len(recent))
	for i := range recent {
		failures = append(failures, OpsRecentFailure{
			JobID:     recent[i].ID,
			JobType:   recent[i].JobType,
			ErrorCode: stringPtrValue(recent[i].ErrorCode),
			CreatedAt: recent[i].CreatedAt,
		})
	}

	return OpsJobSummaryResponse{
		CountsByStatus:    stringifyStatusCounts(statusCounts),
		CountsByType:      stringifyTypeCounts(typeCounts),
		RecentFailures:    failures,
		StaleRunningCount: staleCount,
	}, nil
}

func (s *Service) OpsRetry(ctx context.Context, jobID uuid.UUID, reason string) (OpsRetryResponse, error) {
	startedAt := time.Now()
	reason, err := normalizeOpsReason(reason)
	if err != nil {
		recordOpsJobAction("retry", "invalid", time.Since(startedAt))
		return OpsRetryResponse{}, err
	}
	actor, err := auth.MustUserFromContext(ctx)
	if err != nil {
		recordOpsJobAction("retry", "unauthorized", time.Since(startedAt))
		return OpsRetryResponse{}, err
	}
	oldJob, err := s.repo.GetGenerationJobByID(ctx, jobID)
	if err != nil {
		recordOpsJobAction("retry", "not_found", time.Since(startedAt))
		return OpsRetryResponse{}, err
	}
	if oldJob.Status != entity.GenerationJobStatusFailed && oldJob.Status != entity.GenerationJobStatusCancelled {
		recordOpsJobAction("retry", "invalid_status", time.Since(startedAt))
		return OpsRetryResponse{}, ErrOpsInvalidAction
	}
	if !IsSupportedJobType(oldJob.JobType) {
		recordOpsJobAction("retry", "unsupported_type", time.Since(startedAt))
		return OpsRetryResponse{}, ErrOpsInvalidAction
	}

	trip, access, err := s.trips.GetTripForActor(ctx, oldJob.TripID, oldJob.RequestedByUserID)
	if err != nil {
		recordOpsJobAction("retry", "trip_unavailable", time.Since(startedAt))
		return OpsRetryResponse{}, err
	}
	if !access.CanEdit() {
		recordOpsJobAction("retry", "forbidden", time.Since(startedAt))
		return OpsRetryResponse{}, apperrs.ErrForbidden
	}

	ctx, requestID, correlationID := observability.EnsureRequestIDs(ctx)
	newJob, err := s.repo.CreateGenerationJob(ctx, &entity.GenerationJob{
		ID:                        uuid.New(),
		TripID:                    oldJob.TripID,
		RequestedByUserID:         oldJob.RequestedByUserID,
		JobType:                   oldJob.JobType,
		Status:                    entity.GenerationJobStatusQueued,
		ExpectedItineraryRevision: trip.ItineraryRevision,
		Instruction:               oldJob.Instruction,
		DayNumber:                 oldJob.DayNumber,
		ItemIndex:                 oldJob.ItemIndex,
		Payload:                   oldJob.Payload,
		CorrelationID:             &correlationID,
		RequestID:                 &requestID,
		RetriedFromJobID:          &oldJob.ID,
	})
	if err != nil {
		recordOpsJobAction("retry", "create_failed", time.Since(startedAt))
		return OpsRetryResponse{}, err
	}
	if err := s.dispatchOpsJob(ctx, newJob); err != nil {
		recordOpsJobAction("retry", "dispatch_failed", time.Since(startedAt))
		return OpsRetryResponse{}, err
	}
	_ = s.repo.CreateOpsAuditEvent(ctx, OpsAuditEvent{
		ID:          uuid.New(),
		ActorUserID: actor.ID,
		ActorEmail:  strings.ToLower(strings.TrimSpace(actor.Email)),
		Action:      OpsActionJobRetried,
		EntityType:  "generation_job",
		EntityID:    oldJob.ID,
		Reason:      reason,
		Metadata: map[string]any{
			"newJobId": newJob.ID.String(),
		},
	})
	recordOpsJobAction("retry", "success", time.Since(startedAt))
	return OpsRetryResponse{
		Retried: true,
		NewJob: NewOpsJobResponse(newJob, 0, false, OpsTripMetadata{
			TripID:      trip.ID,
			WorkspaceID: trip.WorkspaceID,
		}),
	}, nil
}

func (s *Service) OpsCancel(ctx context.Context, jobID uuid.UUID, reason string) (OpsJobEnvelope, error) {
	startedAt := time.Now()
	reason, err := normalizeOpsReason(reason)
	if err != nil {
		recordOpsJobAction("cancel", "invalid", time.Since(startedAt))
		return OpsJobEnvelope{}, err
	}
	actor, err := auth.MustUserFromContext(ctx)
	if err != nil {
		recordOpsJobAction("cancel", "unauthorized", time.Since(startedAt))
		return OpsJobEnvelope{}, err
	}
	job, err := s.repo.GetGenerationJobByID(ctx, jobID)
	if err != nil {
		recordOpsJobAction("cancel", "not_found", time.Since(startedAt))
		return OpsJobEnvelope{}, err
	}
	if job.Status != entity.GenerationJobStatusQueued {
		recordOpsJobAction("cancel", "invalid_status", time.Since(startedAt))
		return OpsJobEnvelope{}, ErrNotCancellable
	}
	cancelled, err := s.repo.CancelOpsGenerationJob(ctx, jobID, opsCancelErrorCode, reason)
	if err != nil {
		recordOpsJobAction("cancel", "update_failed", time.Since(startedAt))
		return OpsJobEnvelope{}, err
	}
	_ = s.repo.CreateOpsAuditEvent(ctx, OpsAuditEvent{
		ID:          uuid.New(),
		ActorUserID: actor.ID,
		ActorEmail:  strings.ToLower(strings.TrimSpace(actor.Email)),
		Action:      OpsActionJobCancelled,
		EntityType:  "generation_job",
		EntityID:    jobID,
		Reason:      reason,
	})
	recordGenerationJobStatus(cancelled.JobType, cancelled.Status)
	recordOpsJobAction("cancel", "success", time.Since(startedAt))
	metadata, err := s.opsTripMetadata(ctx, []uuid.UUID{cancelled.TripID})
	if err != nil {
		return OpsJobEnvelope{}, err
	}
	return OpsJobEnvelope{Job: NewOpsJobResponse(cancelled, 0, true, metadata[cancelled.TripID])}, nil
}

func (s *Service) OpsMarkFailed(ctx context.Context, jobID uuid.UUID, reason string, staleThreshold time.Duration) (OpsJobEnvelope, error) {
	startedAt := time.Now()
	reason, err := normalizeOpsReason(reason)
	if err != nil {
		recordOpsJobAction("mark_failed", "invalid", time.Since(startedAt))
		return OpsJobEnvelope{}, err
	}
	actor, err := auth.MustUserFromContext(ctx)
	if err != nil {
		recordOpsJobAction("mark_failed", "unauthorized", time.Since(startedAt))
		return OpsJobEnvelope{}, err
	}
	job, err := s.repo.GetGenerationJobByID(ctx, jobID)
	if err != nil {
		recordOpsJobAction("mark_failed", "not_found", time.Since(startedAt))
		return OpsJobEnvelope{}, err
	}
	if job.Status != entity.GenerationJobStatusRunning {
		recordOpsJobAction("mark_failed", "invalid_status", time.Since(startedAt))
		return OpsJobEnvelope{}, ErrOpsInvalidAction
	}
	startedBefore := time.Now().Add(-staleThreshold)
	if job.StartedAt == nil || job.StartedAt.After(startedBefore) {
		recordOpsJobAction("mark_failed", "not_stale", time.Since(startedAt))
		return OpsJobEnvelope{}, ErrOpsJobNotStale
	}
	failed, err := s.repo.MarkOpsGenerationJobFailed(ctx, jobID, startedBefore, opsMarkFailedErrorCode, reason)
	if err != nil {
		recordOpsJobAction("mark_failed", "update_failed", time.Since(startedAt))
		return OpsJobEnvelope{}, err
	}
	_ = s.repo.CreateOpsAuditEvent(ctx, OpsAuditEvent{
		ID:          uuid.New(),
		ActorUserID: actor.ID,
		ActorEmail:  strings.ToLower(strings.TrimSpace(actor.Email)),
		Action:      OpsActionJobMarkedFailed,
		EntityType:  "generation_job",
		EntityID:    jobID,
		Reason:      reason,
	})
	recordGenerationJobStatus(failed.JobType, failed.Status)
	recordOpsJobAction("mark_failed", "success", time.Since(startedAt))
	metadata, err := s.opsTripMetadata(ctx, []uuid.UUID{failed.TripID})
	if err != nil {
		return OpsJobEnvelope{}, err
	}
	return OpsJobEnvelope{Job: NewOpsJobResponse(failed, staleThreshold, true, metadata[failed.TripID])}, nil
}

func (s *Service) dispatchOpsJob(ctx context.Context, job *entity.GenerationJob) error {
	recordGenerationJobDispatch(job.JobType, string(s.cfg.DispatchMode))
	if !s.cfg.QueueMode() {
		return nil
	}
	if s.publisher == nil {
		recordGenerationJobDispatchFailed(job.JobType, ErrorJobDispatchFailed)
		_, _ = s.repo.FailGenerationJob(ctx, job.ID, ErrorJobDispatchFailed, "Generation job could not be dispatched.")
		return ErrJobDispatchFailed
	}
	publishCtx, cancel := context.WithTimeout(ctx, s.cfg.PublishTimeout)
	defer cancel()
	if err := s.publisher.PublishGenerationJob(publishCtx, NewQueueMessageFromContext(ctx, job)); err != nil {
		recordGenerationJobDispatchFailed(job.JobType, ErrorJobDispatchFailed)
		if s.cfg.PublishFailOpen {
			return nil
		}
		_, _ = s.repo.FailGenerationJob(ctx, job.ID, ErrorJobDispatchFailed, "Generation job could not be dispatched.")
		return ErrJobDispatchFailed
	}
	return nil
}

func normalizeOpsFilters(filters OpsJobListFilters) OpsJobListFilters {
	if filters.Limit == 0 {
		filters.Limit = opsDefaultLimit
	}
	if filters.Limit < 1 {
		filters.Limit = 1
	}
	if filters.Limit > opsMaxLimit {
		filters.Limit = opsMaxLimit
	}
	if filters.Offset < 0 {
		filters.Offset = 0
	}
	filters.ErrorCode = strings.TrimSpace(filters.ErrorCode)
	return filters
}

func (s *Service) opsTripMetadata(ctx context.Context, tripIDs []uuid.UUID) (map[uuid.UUID]OpsTripMetadata, error) {
	metadata, err := s.repo.ListOpsTripMetadata(ctx, tripIDs)
	if err != nil {
		return nil, err
	}
	for _, tripID := range tripIDs {
		if _, ok := metadata[tripID]; !ok {
			metadata[tripID] = OpsTripMetadata{TripID: tripID}
		}
	}
	return metadata, nil
}

func collectOpsTripIDs(jobs []entity.GenerationJob) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(jobs))
	ids := make([]uuid.UUID, 0, len(jobs))
	for i := range jobs {
		if _, ok := seen[jobs[i].TripID]; ok {
			continue
		}
		seen[jobs[i].TripID] = struct{}{}
		ids = append(ids, jobs[i].TripID)
	}
	return ids
}

func normalizeOpsReason(reason string) (string, error) {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "", apperrs.NewInvalidInput("reason is required")
	}
	if len(reason) > opsReasonMaxLen {
		reason = reason[:opsReasonMaxLen]
	}
	return reason, nil
}

func stringifyStatusCounts(counts map[entity.GenerationJobStatus]int) map[string]int {
	out := map[string]int{}
	for _, status := range []entity.GenerationJobStatus{
		entity.GenerationJobStatusQueued,
		entity.GenerationJobStatusRunning,
		entity.GenerationJobStatusCompleted,
		entity.GenerationJobStatusFailed,
		entity.GenerationJobStatusCancelled,
	} {
		out[string(status)] = counts[status]
	}
	return out
}

func stringifyTypeCounts(counts map[entity.GenerationJobType]int) map[string]int {
	out := map[string]int{}
	for jobType, count := range counts {
		out[string(jobType)] = count
	}
	return out
}

func summarizePayload(job *entity.GenerationJob) *OpsPayloadSummary {
	if job == nil {
		return nil
	}
	summary := &OpsPayloadSummary{
		DayNumber:      job.DayNumber,
		ItemIndex:      job.ItemIndex,
		HasInstruction: job.Instruction != nil && strings.TrimSpace(*job.Instruction) != "",
	}
	if job.JobType == entity.GenerationJobTypeBudgetOptimizationDay {
		payload := budgetoptimization.DecodeJobPayload(job.Payload)
		summary.Scope = "day"
		summary.TargetReductionAmount = payload.TargetReductionAmount
		if strings.TrimSpace(payload.Currency) != "" {
			currency := payload.Currency
			summary.Currency = &currency
		}
		summary.HasConstraints = payload.Constraints != nil
		return summary
	}
	if len(job.Payload) > 0 {
		var raw map[string]any
		if err := json.Unmarshal(job.Payload, &raw); err == nil {
			if scope, ok := raw["scope"].(string); ok {
				scope = strings.TrimSpace(scope)
				if scope != "" {
					summary.Scope = scope
				}
			}
		}
	}
	return summary
}
