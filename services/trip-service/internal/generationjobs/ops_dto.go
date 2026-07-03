package generationjobs

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type OpsJobListResponse struct {
	Jobs       []OpsJobResponse `json:"jobs"`
	NextCursor *string          `json:"nextCursor"`
	NextOffset *int             `json:"nextOffset,omitempty"`
}

type OpsJobEnvelope struct {
	Job OpsJobResponse `json:"job"`
}

type OpsRetryResponse struct {
	Retried bool           `json:"retried"`
	NewJob  OpsJobResponse `json:"newJob"`
}

type OpsPayloadSummary struct {
	DayNumber             *int     `json:"dayNumber"`
	ItemIndex             *int     `json:"itemIndex"`
	Scope                 string   `json:"scope,omitempty"`
	TargetReductionAmount *float64 `json:"targetReductionAmount,omitempty"`
	Currency              *string  `json:"currency,omitempty"`
	HasInstruction        bool     `json:"hasInstruction"`
	HasConstraints        bool     `json:"hasConstraints"`
}

type OpsJobResponse struct {
	ID                        uuid.UUID                  `json:"id"`
	TripID                    uuid.UUID                  `json:"tripId"`
	WorkspaceID               *uuid.UUID                 `json:"workspaceId,omitempty"`
	Scope                     string                     `json:"scope,omitempty"`
	RequestedByUserID         uuid.UUID                  `json:"requestedByUserId"`
	JobType                   entity.GenerationJobType   `json:"jobType"`
	Status                    entity.GenerationJobStatus `json:"status"`
	PayloadSummary            *OpsPayloadSummary         `json:"payloadSummary,omitempty"`
	ErrorCode                 *string                    `json:"errorCode"`
	ErrorMessage              *string                    `json:"errorMessage"`
	ExpectedItineraryRevision int                        `json:"expectedItineraryRevision"`
	ResultItineraryRevision   *int                       `json:"resultItineraryRevision"`
	CorrelationID             *string                    `json:"correlationId"`
	RequestID                 *string                    `json:"requestId"`
	RetriedFromJobID          *uuid.UUID                 `json:"retriedFromJobId,omitempty"`
	CreatedAt                 time.Time                  `json:"createdAt"`
	StartedAt                 *time.Time                 `json:"startedAt"`
	CompletedAt               *time.Time                 `json:"completedAt"`
	CancelledAt               *time.Time                 `json:"cancelledAt"`
	UpdatedAt                 time.Time                  `json:"updatedAt"`
	DurationMs                *int64                     `json:"durationMs"`
	AttemptCount              int                        `json:"attemptCount"`
	CanRetry                  bool                       `json:"canRetry"`
	CanCancel                 bool                       `json:"canCancel"`
	CanMarkFailed             bool                       `json:"canMarkFailed"`
}

type OpsTripMetadata struct {
	TripID      uuid.UUID
	WorkspaceID *uuid.UUID
}

type OpsJobSummaryResponse struct {
	CountsByStatus    map[string]int     `json:"countsByStatus"`
	CountsByType      map[string]int     `json:"countsByType"`
	RecentFailures    []OpsRecentFailure `json:"recentFailures"`
	StaleRunningCount int                `json:"staleRunningCount"`
}

type OpsRecentFailure struct {
	JobID     uuid.UUID                `json:"jobId"`
	JobType   entity.GenerationJobType `json:"jobType"`
	ErrorCode string                   `json:"errorCode"`
	CreatedAt time.Time                `json:"createdAt"`
}

func NewOpsJobResponse(job *entity.GenerationJob, staleThreshold time.Duration, includePayload bool, metadata ...OpsTripMetadata) OpsJobResponse {
	resp := OpsJobResponse{
		ID:                        job.ID,
		TripID:                    job.TripID,
		RequestedByUserID:         job.RequestedByUserID,
		JobType:                   job.JobType,
		Status:                    job.Status,
		ErrorCode:                 job.ErrorCode,
		ErrorMessage:              job.ErrorMessage,
		ExpectedItineraryRevision: job.ExpectedItineraryRevision,
		ResultItineraryRevision:   job.ResultItineraryRevision,
		CorrelationID:             job.CorrelationID,
		RequestID:                 job.RequestID,
		RetriedFromJobID:          job.RetriedFromJobID,
		CreatedAt:                 job.CreatedAt,
		StartedAt:                 job.StartedAt,
		CompletedAt:               job.CompletedAt,
		CancelledAt:               job.CancelledAt,
		UpdatedAt:                 job.UpdatedAt,
		DurationMs:                durationMs(job),
		AttemptCount:              0,
		CanRetry:                  job.Status == entity.GenerationJobStatusFailed || job.Status == entity.GenerationJobStatusCancelled,
		CanCancel:                 job.Status == entity.GenerationJobStatusQueued,
		CanMarkFailed:             canMarkFailed(job, staleThreshold),
	}
	if len(metadata) > 0 {
		resp.WorkspaceID = metadata[0].WorkspaceID
		if metadata[0].WorkspaceID != nil {
			resp.Scope = "workspace"
		} else {
			resp.Scope = "personal"
		}
	}
	if includePayload {
		resp.PayloadSummary = summarizePayload(job)
	}
	return resp
}

func durationMs(job *entity.GenerationJob) *int64 {
	if job == nil || job.StartedAt == nil {
		return nil
	}
	end := time.Now()
	switch {
	case job.CompletedAt != nil:
		end = *job.CompletedAt
	case job.CancelledAt != nil:
		end = *job.CancelledAt
	}
	ms := end.Sub(*job.StartedAt).Milliseconds()
	if ms < 0 {
		ms = 0
	}
	return &ms
}

func canMarkFailed(job *entity.GenerationJob, staleThreshold time.Duration) bool {
	if job == nil || job.Status != entity.GenerationJobStatusRunning || job.StartedAt == nil || staleThreshold <= 0 {
		return false
	}
	return job.StartedAt.Before(time.Now().Add(-staleThreshold))
}
