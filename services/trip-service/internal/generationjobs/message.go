package generationjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

const (
	MessageTypeTripGenerationJob = "trip_generation_job"
	HeaderAttempts               = "x-attempts"
	HeaderSourceService          = "x-source-service"
	HeaderMessageType            = "x-message-type"
	SourceTripService            = "trip-service"
	ContentTypeJSON              = "application/json"
)

type QueueMessage struct {
	MessageID uuid.UUID                `json:"messageId"`
	JobID     uuid.UUID                `json:"jobId"`
	TripID    uuid.UUID                `json:"tripId"`
	JobType   entity.GenerationJobType `json:"jobType"`
	CreatedAt time.Time                `json:"createdAt"`
}

func NewQueueMessage(job *entity.GenerationJob) QueueMessage {
	return QueueMessage{
		MessageID: uuid.New(),
		JobID:     job.ID,
		TripID:    job.TripID,
		JobType:   job.JobType,
		CreatedAt: time.Now().UTC(),
	}
}

type JobPublisher interface {
	PublishGenerationJob(ctx context.Context, msg QueueMessage) error
}

func ValidateQueueMessage(msg QueueMessage) error {
	if msg.MessageID == uuid.Nil {
		return fmt.Errorf("messageId is required")
	}
	if msg.JobID == uuid.Nil {
		return fmt.Errorf("jobId is required")
	}
	if msg.TripID == uuid.Nil {
		return fmt.Errorf("tripId is required")
	}
	if !IsSupportedJobType(msg.JobType) {
		return fmt.Errorf("jobType is invalid")
	}
	return nil
}

func IsSupportedJobType(jobType entity.GenerationJobType) bool {
	switch jobType {
	case entity.GenerationJobTypeFullGeneration,
		entity.GenerationJobTypeDayRegeneration,
		entity.GenerationJobTypeItemRegeneration,
		entity.GenerationJobTypeQualityImprovementDay,
		entity.GenerationJobTypeQualityImprovementItem,
		entity.GenerationJobTypeBudgetOptimizationDay:
		return true
	default:
		return false
	}
}
