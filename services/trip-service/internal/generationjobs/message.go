package generationjobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/observability"
)

const (
	MessageTypeTripGenerationJob = "trip_generation_job"
	HeaderAttempts               = "x-attempts"
	HeaderRequestID              = "x-request-id"
	HeaderCorrelationID          = "x-correlation-id"
	HeaderSourceService          = "x-source-service"
	HeaderMessageType            = "x-message-type"
	SourceTripService            = "trip-service"
	ContentTypeJSON              = "application/json"
)

type QueueMessage struct {
	MessageID     uuid.UUID                `json:"messageId"`
	JobID         uuid.UUID                `json:"jobId"`
	TripID        uuid.UUID                `json:"tripId"`
	JobType       entity.GenerationJobType `json:"jobType"`
	CreatedAt     time.Time                `json:"createdAt"`
	CorrelationID string                   `json:"correlationId,omitempty"`
	RequestID     string                   `json:"requestId,omitempty"`
}

func NewQueueMessage(job *entity.GenerationJob) QueueMessage {
	return NewQueueMessageFromContext(context.Background(), job)
}

func NewQueueMessageFromContext(ctx context.Context, job *entity.GenerationJob) QueueMessage {
	requestID := ""
	correlationID := ""
	if job != nil {
		requestID = stringPtrValue(job.RequestID)
		correlationID = stringPtrValue(job.CorrelationID)
	}
	if strings.TrimSpace(requestID) == "" || strings.TrimSpace(correlationID) == "" {
		ctx, ctxRequestID, ctxCorrelationID := observability.EnsureRequestIDs(ctx)
		_ = ctx
		if strings.TrimSpace(requestID) == "" {
			requestID = ctxRequestID
		}
		if strings.TrimSpace(correlationID) == "" {
			correlationID = ctxCorrelationID
		}
	}

	return QueueMessage{
		MessageID:     uuid.New(),
		JobID:         job.ID,
		TripID:        job.TripID,
		JobType:       job.JobType,
		CreatedAt:     time.Now().UTC(),
		CorrelationID: strings.TrimSpace(correlationID),
		RequestID:     strings.TrimSpace(requestID),
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
		entity.GenerationJobTypeBudgetOptimizationDay,
		entity.GenerationJobTypeTemplateAdaptation,
		entity.GenerationJobTypePolicyRepair:
		return true
	default:
		return false
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
