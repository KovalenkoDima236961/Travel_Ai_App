package generationjobs

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/observability"
)

func TestValidateQueueMessage(t *testing.T) {
	valid := QueueMessage{
		MessageID: uuid.New(),
		JobID:     uuid.New(),
		TripID:    uuid.New(),
		JobType:   entity.GenerationJobTypeFullGeneration,
		CreatedAt: time.Now(),
	}

	if err := ValidateQueueMessage(valid); err != nil {
		t.Fatalf("expected valid message, got %v", err)
	}

	tests := []struct {
		name string
		msg  QueueMessage
	}{
		{name: "missing message id", msg: QueueMessage{JobID: valid.JobID, TripID: valid.TripID, JobType: valid.JobType}},
		{name: "missing job id", msg: QueueMessage{MessageID: valid.MessageID, TripID: valid.TripID, JobType: valid.JobType}},
		{name: "missing trip id", msg: QueueMessage{MessageID: valid.MessageID, JobID: valid.JobID, JobType: valid.JobType}},
		{name: "unsupported job type", msg: QueueMessage{MessageID: valid.MessageID, JobID: valid.JobID, TripID: valid.TripID, JobType: "unknown"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateQueueMessage(tt.msg); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestNewQueueMessageFromContextPropagatesRequestIDs(t *testing.T) {
	requestID := "request-1"
	correlationID := "correlation-1"
	ctx := observability.ContextWithRequestIDs(context.Background(), requestID, correlationID)
	job := &entity.GenerationJob{
		ID:        uuid.New(),
		TripID:    uuid.New(),
		JobType:   entity.GenerationJobTypeFullGeneration,
		CreatedAt: time.Now(),
	}

	msg := NewQueueMessageFromContext(ctx, job)

	if msg.RequestID != requestID {
		t.Fatalf("request id = %q, want %q", msg.RequestID, requestID)
	}
	if msg.CorrelationID != correlationID {
		t.Fatalf("correlation id = %q, want %q", msg.CorrelationID, correlationID)
	}
	if msg.JobID != job.ID || msg.TripID != job.TripID || msg.JobType != job.JobType {
		t.Fatal("message does not include expected job identifiers")
	}
}

func TestNewQueueMessageFromContextUsesStoredJobRequestIDs(t *testing.T) {
	requestID := "stored-request"
	correlationID := "stored-correlation"
	job := &entity.GenerationJob{
		ID:            uuid.New(),
		TripID:        uuid.New(),
		JobType:       entity.GenerationJobTypeFullGeneration,
		CreatedAt:     time.Now(),
		RequestID:     &requestID,
		CorrelationID: &correlationID,
	}

	msg := NewQueueMessageFromContext(context.Background(), job)

	if msg.RequestID != requestID {
		t.Fatalf("request id = %q, want %q", msg.RequestID, requestID)
	}
	if msg.CorrelationID != correlationID {
		t.Fatalf("correlation id = %q, want %q", msg.CorrelationID, correlationID)
	}
}
