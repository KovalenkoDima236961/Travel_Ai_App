package aimodel

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestShadowEvaluationPayloadValidateRejectsRawlessMissingReferences(t *testing.T) {
	now := time.Now().UTC()
	payload := ShadowEvaluationPayload{
		RequestAssignmentID:   uuid.New(),
		TripID:                uuid.New(),
		GenerationJobID:       uuid.New(),
		CandidateDeploymentID: uuid.New(),
		BaselineDeploymentID:  uuid.New(),
		PromptVersion:         "itinerary_generation_v1",
		ValidatorVersion:      "aivalidation_v1",
		ExpiresAt:             now.Add(time.Hour),
	}

	err := payload.Validate(now)
	if err == nil || !strings.Contains(err.Error(), "inputSnapshotRef") {
		t.Fatalf("expected missing input snapshot ref error, got %v", err)
	}
}

func TestShadowEvaluationPayloadValidateAcceptsSafeReferences(t *testing.T) {
	now := time.Now().UTC()
	payload := ShadowEvaluationPayload{
		RequestAssignmentID:   uuid.New(),
		TripID:                uuid.New(),
		GenerationJobID:       uuid.New(),
		CandidateDeploymentID: uuid.New(),
		BaselineDeploymentID:  uuid.New(),
		InputSnapshotRef:      "generation-job:00000000-0000-0000-0000-000000000001",
		GroundingSnapshotRef:  "trace:00000000-0000-0000-0000-000000000002",
		PromptVersion:         "itinerary_generation_v1",
		ValidatorVersion:      "aivalidation_v1",
		ExpiresAt:             now.Add(time.Hour),
	}

	if err := payload.Validate(now); err != nil {
		t.Fatalf("expected valid shadow payload, got %v", err)
	}
}

func TestShadowCapacitySkipReason(t *testing.T) {
	cfg := NormalizeConfig(Config{ShadowEnabled: true, ShadowMaxConcurrent: 2, ShadowSkipWhenQueueDepthAbove: 10})
	if got := ShadowCapacitySkipReason(cfg, CapacityState{Enabled: true, Concurrent: 2}); got != "shadow_concurrency_limit" {
		t.Fatalf("expected concurrency skip, got %q", got)
	}
	if got := ShadowCapacitySkipReason(cfg, CapacityState{Enabled: true, QueueDepth: 11}); got != "shadow_queue_depth" {
		t.Fatalf("expected queue-depth skip, got %q", got)
	}
	if got := ShadowCapacitySkipReason(cfg, CapacityState{Enabled: true, QueueDepth: 1, Concurrent: 1}); got != "" {
		t.Fatalf("expected no skip reason, got %q", got)
	}
}
