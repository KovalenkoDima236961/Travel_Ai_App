package aimodel

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateRegisterInputRejectsUnsafeDeploymentKey(t *testing.T) {
	input := RegisterDeploymentInput{
		DeploymentKey: "../candidate",
		ModelID:       uuid.New(),
		ModelVariant:  VariantFineTunedCandidate,
		AdapterID:     uuidPtr(uuid.New()),
		Status:        StatusRegistered,
		TrafficMode:   TrafficDisabled,
		PromptVersion: "prompt-v1",
		Reason:        "registration",
	}
	if err := validateRegisterInput(input); err == nil {
		t.Fatal("expected unsafe deployment key to be rejected")
	}
}

func TestValidateRegisterInputRequiresAdapterForCandidate(t *testing.T) {
	input := RegisterDeploymentInput{
		DeploymentKey: "candidate-v1",
		ModelID:       uuid.New(),
		ModelVariant:  VariantFineTunedCandidate,
		Status:        StatusRegistered,
		TrafficMode:   TrafficDisabled,
		PromptVersion: "prompt-v1",
		Reason:        "registration",
	}
	if err := validateRegisterInput(input); err == nil {
		t.Fatal("expected candidate without adapter to be rejected")
	}
}

func TestValidateStateTransitionRequiresMatchingTrafficMode(t *testing.T) {
	err := validateStateTransition(Deployment{ModelVariant: VariantFineTunedCandidate}, StatusShadow, TrafficPercentage)
	if err == nil {
		t.Fatal("expected mismatched shadow traffic mode to be rejected")
	}
}

func TestValidateStateTransitionBlocksBaselineCandidateStates(t *testing.T) {
	err := validateStateTransition(Deployment{ModelVariant: VariantGroundedBaseline}, StatusStagedRollout, TrafficPercentage)
	if err == nil {
		t.Fatal("expected baseline staged rollout to be rejected")
	}
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	return &id
}
