package jobqueue

import (
	"context"
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/pkg/observability"
)

func TestEnsureMessageRequestIDsUsesContext(t *testing.T) {
	ctx := observability.ContextWithRequestIDs(context.Background(), "ctx-request", "ctx-correlation")

	gotCtx, requestID, correlationID := ensureMessageRequestIDs(ctx, generationjobs.QueueMessage{})

	if requestID != "ctx-request" {
		t.Fatalf("request id = %q, want ctx-request", requestID)
	}
	if correlationID != "ctx-correlation" {
		t.Fatalf("correlation id = %q, want ctx-correlation", correlationID)
	}
	if got := observability.RequestIDFromContext(gotCtx); got != requestID {
		t.Fatalf("context request id = %q, want %q", got, requestID)
	}
	if got := observability.CorrelationIDFromContext(gotCtx); got != correlationID {
		t.Fatalf("context correlation id = %q, want %q", got, correlationID)
	}
}

func TestEnsureMessageRequestIDsPrefersMessageValues(t *testing.T) {
	ctx := observability.ContextWithRequestIDs(context.Background(), "ctx-request", "ctx-correlation")
	msg := generationjobs.QueueMessage{
		RequestID:     "message-request",
		CorrelationID: "message-correlation",
	}

	gotCtx, requestID, correlationID := ensureMessageRequestIDs(ctx, msg)

	if requestID != "message-request" {
		t.Fatalf("request id = %q, want message-request", requestID)
	}
	if correlationID != "message-correlation" {
		t.Fatalf("correlation id = %q, want message-correlation", correlationID)
	}
	if got := observability.RequestIDFromContext(gotCtx); got != requestID {
		t.Fatalf("context request id = %q, want %q", got, requestID)
	}
	if got := observability.CorrelationIDFromContext(gotCtx); got != correlationID {
		t.Fatalf("context correlation id = %q, want %q", got, correlationID)
	}
}
