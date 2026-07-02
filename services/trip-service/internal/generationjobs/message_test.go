package generationjobs

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
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
