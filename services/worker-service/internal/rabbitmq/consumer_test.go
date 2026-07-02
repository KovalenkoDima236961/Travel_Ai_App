package rabbitmq

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
)

func TestDecodeMessage(t *testing.T) {
	requestID := "body-request"
	correlationID := "body-correlation"
	msg := generationjobs.QueueMessage{
		MessageID:     uuid.New(),
		JobID:         uuid.New(),
		TripID:        uuid.New(),
		JobType:       entity.GenerationJobTypeBudgetOptimizationDay,
		CreatedAt:     time.Now(),
		RequestID:     requestID,
		CorrelationID: correlationID,
	}
	body, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	got, err := decodeMessage(amqp.Delivery{
		ContentType: generationjobs.ContentTypeJSON,
		Body:        body,
	})
	if err != nil {
		t.Fatalf("decode valid message: %v", err)
	}
	if got.JobID != msg.JobID {
		t.Fatalf("job id mismatch: got %s want %s", got.JobID, msg.JobID)
	}
	if got.RequestID != requestID {
		t.Fatalf("request id mismatch: got %q want %q", got.RequestID, requestID)
	}
	if got.CorrelationID != correlationID {
		t.Fatalf("correlation id mismatch: got %q want %q", got.CorrelationID, correlationID)
	}
}

func TestDecodeMessageFallsBackToRequestIDHeaders(t *testing.T) {
	msg := generationjobs.QueueMessage{
		MessageID: uuid.New(),
		JobID:     uuid.New(),
		TripID:    uuid.New(),
		JobType:   entity.GenerationJobTypeBudgetOptimizationDay,
		CreatedAt: time.Now(),
	}
	body, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	got, err := decodeMessage(amqp.Delivery{
		ContentType: generationjobs.ContentTypeJSON,
		Headers: amqp.Table{
			generationjobs.HeaderRequestID:     "header-request",
			generationjobs.HeaderCorrelationID: "header-correlation",
		},
		Body: body,
	})
	if err != nil {
		t.Fatalf("decode valid message: %v", err)
	}
	if got.RequestID != "header-request" {
		t.Fatalf("request id mismatch: got %q want header-request", got.RequestID)
	}
	if got.CorrelationID != "header-correlation" {
		t.Fatalf("correlation id mismatch: got %q want header-correlation", got.CorrelationID)
	}
}

func TestDecodeMessageRejectsInvalid(t *testing.T) {
	_, err := decodeMessage(amqp.Delivery{
		ContentType: generationjobs.ContentTypeJSON,
		Body:        []byte(`{"messageId":"00000000-0000-0000-0000-000000000000"}`),
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestReadAttempt(t *testing.T) {
	if got := readAttempt(amqp.Table{generationjobs.HeaderAttempts: int32(2)}); got != 2 {
		t.Fatalf("got %d want 2", got)
	}
	if got := readAttempt(amqp.Table{}); got != 0 {
		t.Fatalf("got %d want 0", got)
	}
}
