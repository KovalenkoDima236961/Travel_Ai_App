package aimodel

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const JobTypeShadowGenerationEvaluation = "ai_shadow_generation_evaluation"

type ShadowEvaluationPayload struct {
	RequestAssignmentID   uuid.UUID `json:"requestAssignmentId"`
	TripID                uuid.UUID `json:"tripId"`
	GenerationJobID       uuid.UUID `json:"generationJobId"`
	CandidateDeploymentID uuid.UUID `json:"candidateDeploymentId"`
	BaselineDeploymentID  uuid.UUID `json:"baselineDeploymentId"`
	InputSnapshotRef      string    `json:"inputSnapshotRef"`
	GroundingSnapshotRef  string    `json:"groundingSnapshotRef"`
	PromptVersion         string    `json:"promptVersion"`
	ValidatorVersion      string    `json:"validatorVersion"`
	ExpiresAt             time.Time `json:"expiresAt"`
}

func (p ShadowEvaluationPayload) Validate(now time.Time) error {
	if p.RequestAssignmentID == uuid.Nil {
		return fmt.Errorf("requestAssignmentId is required")
	}
	if p.TripID == uuid.Nil {
		return fmt.Errorf("tripId is required")
	}
	if p.GenerationJobID == uuid.Nil {
		return fmt.Errorf("generationJobId is required")
	}
	if p.CandidateDeploymentID == uuid.Nil {
		return fmt.Errorf("candidateDeploymentId is required")
	}
	if p.BaselineDeploymentID == uuid.Nil {
		return fmt.Errorf("baselineDeploymentId is required")
	}
	if strings.TrimSpace(p.InputSnapshotRef) == "" {
		return fmt.Errorf("inputSnapshotRef is required")
	}
	if strings.TrimSpace(p.GroundingSnapshotRef) == "" {
		return fmt.Errorf("groundingSnapshotRef is required")
	}
	if strings.TrimSpace(p.PromptVersion) == "" {
		return fmt.Errorf("promptVersion is required")
	}
	if strings.TrimSpace(p.ValidatorVersion) == "" {
		return fmt.Errorf("validatorVersion is required")
	}
	if p.ExpiresAt.IsZero() || !p.ExpiresAt.After(now) {
		return fmt.Errorf("shadow evaluation payload is expired")
	}
	return nil
}

type CapacityState struct {
	Enabled    bool
	QueueDepth int
	Concurrent int
}

func ShadowCapacitySkipReason(cfg Config, state CapacityState) string {
	cfg = NormalizeConfig(cfg)
	if !cfg.ShadowEnabled || !state.Enabled {
		return "shadow_disabled"
	}
	if state.Concurrent >= cfg.ShadowMaxConcurrent {
		return "shadow_concurrency_limit"
	}
	if state.QueueDepth > cfg.ShadowSkipWhenQueueDepthAbove {
		return "shadow_queue_depth"
	}
	return ""
}
