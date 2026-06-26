package generationjobs

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type CreateRequest struct {
	JobType                   entity.GenerationJobType `json:"jobType"`
	ExpectedItineraryRevision *int                     `json:"expectedItineraryRevision"`
	Instruction               *string                  `json:"instruction"`
	DayNumber                 *int                     `json:"dayNumber"`
	ItemIndex                 *int                     `json:"itemIndex"`
}

type JobEnvelope struct {
	Job JobResponse `json:"job"`
}

type ListResponse struct {
	Items []JobResponse `json:"items"`
	Limit int           `json:"limit"`
}

type JobResponse struct {
	ID                        uuid.UUID                  `json:"id"`
	TripID                    uuid.UUID                  `json:"tripId"`
	RequestedByUserID         uuid.UUID                  `json:"requestedByUserId"`
	JobType                   entity.GenerationJobType   `json:"jobType"`
	Status                    entity.GenerationJobStatus `json:"status"`
	ExpectedItineraryRevision int                        `json:"expectedItineraryRevision"`
	Instruction               *string                    `json:"instruction"`
	DayNumber                 *int                       `json:"dayNumber"`
	ItemIndex                 *int                       `json:"itemIndex"`
	ErrorCode                 *string                    `json:"errorCode"`
	ErrorMessage              *string                    `json:"errorMessage"`
	ResultItineraryRevision   *int                       `json:"resultItineraryRevision"`
	CreatedAt                 time.Time                  `json:"createdAt"`
	StartedAt                 *time.Time                 `json:"startedAt"`
	CompletedAt               *time.Time                 `json:"completedAt"`
	CancelledAt               *time.Time                 `json:"cancelledAt"`
	UpdatedAt                 time.Time                  `json:"updatedAt"`
}

func NewJobEnvelope(job *entity.GenerationJob) JobEnvelope {
	return JobEnvelope{Job: NewJobResponse(job)}
}

func NewListResponse(jobs []entity.GenerationJob, limit int) ListResponse {
	items := make([]JobResponse, 0, len(jobs))
	for i := range jobs {
		items = append(items, NewJobResponse(&jobs[i]))
	}
	return ListResponse{Items: items, Limit: limit}
}

func NewJobResponse(job *entity.GenerationJob) JobResponse {
	return JobResponse{
		ID:                        job.ID,
		TripID:                    job.TripID,
		RequestedByUserID:         job.RequestedByUserID,
		JobType:                   job.JobType,
		Status:                    job.Status,
		ExpectedItineraryRevision: job.ExpectedItineraryRevision,
		Instruction:               job.Instruction,
		DayNumber:                 job.DayNumber,
		ItemIndex:                 job.ItemIndex,
		ErrorCode:                 job.ErrorCode,
		ErrorMessage:              job.ErrorMessage,
		ResultItineraryRevision:   job.ResultItineraryRevision,
		CreatedAt:                 job.CreatedAt,
		StartedAt:                 job.StartedAt,
		CompletedAt:               job.CompletedAt,
		CancelledAt:               job.CancelledAt,
		UpdatedAt:                 job.UpdatedAt,
	}
}
