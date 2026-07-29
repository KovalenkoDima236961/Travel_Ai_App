package aimodel

import (
	"testing"

	"github.com/google/uuid"
)

func TestDeterministicBucketStableForSameSubject(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000101")
	ctx := RoutingContext{UserID: &userID, RequestKey: "request-a"}

	first := DeterministicBucket("salt-v1", stableAssignmentKey(ctx))
	second := DeterministicBucket("salt-v1", stableAssignmentKey(ctx))
	if first != second {
		t.Fatalf("expected stable bucket, got %d and %d", first, second)
	}
	if first < 0 || first > 9999 {
		t.Fatalf("bucket outside 0..9999: %d", first)
	}

	ctx.RequestKey = "request-b"
	third := DeterministicBucket("salt-v1", stableAssignmentKey(ctx))
	if third != first {
		t.Fatalf("same user must remain sticky across requests, got %d and %d", first, third)
	}
}

func TestDecideBaselineOnlyWhenServingFeatureDisabled(t *testing.T) {
	decision := Decide(DecisionInput{
		Config:   Config{ModelServingEnabled: false},
		Context:  RoutingContext{FeatureFlags: map[string]bool{"ai_model_serving_enabled": false}},
		Baseline: testBaseline(),
		Candidate: ptrDeployment(testCandidate(func(d *Deployment) {
			d.Status = StatusShadow
			d.TrafficMode = TrafficShadow
			d.ShadowSamplePercent = 100
		})),
	})

	if decision.AssignmentType != AssignmentBaselineOnly {
		t.Fatalf("expected baseline-only assignment, got %s", decision.AssignmentType)
	}
	if decision.ShadowDeployment != nil {
		t.Fatal("disabled model serving must not assign shadow candidate")
	}
}

func TestDecidePausedCandidateFallsBackToBaseline(t *testing.T) {
	candidate := testCandidate(func(d *Deployment) {
		d.Status = StatusPaused
		d.TrafficMode = TrafficAllowlist
		d.AllowlistedUserIDs = []uuid.UUID{uuid.MustParse("00000000-0000-0000-0000-000000000102")}
	})
	userID := candidate.AllowlistedUserIDs[0]

	decision := Decide(DecisionInput{
		Config: testConfig(),
		Context: RoutingContext{
			UserID: userIDPtr(userID),
			FeatureFlags: map[string]bool{
				"ai_model_serving_enabled":               true,
				"ai_candidate_allowlist_rollout_enabled": true,
			},
		},
		Baseline:  testBaseline(),
		Candidate: &candidate,
	})

	if decision.AssignmentType != AssignmentBaselineOnly {
		t.Fatalf("expected paused candidate to fall back to baseline, got %s", decision.AssignmentType)
	}
}

func TestDecidePrecedenceInternalBeforeShadow(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000103")
	candidate := testCandidate(func(d *Deployment) {
		d.Status = StatusInternal
		d.TrafficMode = TrafficInternal
		d.ShadowSamplePercent = 100
	})

	decision := Decide(DecisionInput{
		Config: testConfig(),
		Context: RoutingContext{
			UserID:    userIDPtr(userID),
			IsOpsUser: true,
			FeatureFlags: map[string]bool{
				"ai_model_serving_enabled":              true,
				"ai_candidate_internal_rollout_enabled": true,
				"ai_shadow_evaluation_enabled":          true,
			},
		},
		Baseline:  testBaseline(),
		Candidate: &candidate,
	})

	if decision.AssignmentType != AssignmentInternalCandidate {
		t.Fatalf("expected internal rollout before shadow, got %s", decision.AssignmentType)
	}
	if decision.PrimaryDeployment.ID != candidate.ID || decision.ShadowDeployment != nil {
		t.Fatalf("internal candidate must be primary without shadow, got %+v", decision)
	}
}

func TestDecideAllowlistedUserGetsCandidate(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000104")
	candidate := testCandidate(func(d *Deployment) {
		d.Status = StatusAllowlist
		d.TrafficMode = TrafficAllowlist
		d.AllowlistedUserIDs = []uuid.UUID{userID}
	})

	decision := Decide(DecisionInput{
		Config: testConfig(),
		Context: RoutingContext{
			UserID: userIDPtr(userID),
			FeatureFlags: map[string]bool{
				"ai_model_serving_enabled":               true,
				"ai_candidate_allowlist_rollout_enabled": true,
			},
		},
		Baseline:  testBaseline(),
		Candidate: &candidate,
	})

	if decision.AssignmentType != AssignmentAllowlistCandidate {
		t.Fatalf("expected allowlist candidate, got %s", decision.AssignmentType)
	}
	if decision.UserVisibleVariant != VariantFineTunedCandidate {
		t.Fatalf("expected candidate user-visible variant, got %s", decision.UserVisibleVariant)
	}
}

func TestDecidePercentageRequiresUserOptIn(t *testing.T) {
	userID := uuid.MustParse("00000000-0000-0000-0000-000000000105")
	candidate := testCandidate(func(d *Deployment) {
		d.Status = StatusStagedRollout
		d.TrafficMode = TrafficPercentage
		d.RolloutPercent = 100
	})

	decision := Decide(DecisionInput{
		Config: testConfig(),
		Context: RoutingContext{
			UserID: userIDPtr(userID),
			FeatureFlags: map[string]bool{
				"ai_model_serving_enabled":                true,
				"ai_candidate_percentage_rollout_enabled": true,
				"ai_candidate_user_opt_in_enabled":        true,
			},
		},
		Baseline:  testBaseline(),
		Candidate: &candidate,
	})
	if decision.AssignmentType != AssignmentBaselineOnly {
		t.Fatalf("expected baseline without opt-in, got %s", decision.AssignmentType)
	}

	decision = Decide(DecisionInput{
		Config: testConfig(),
		Context: RoutingContext{
			UserID:                  userIDPtr(userID),
			UserOptInExperimentalAI: true,
			FeatureFlags: map[string]bool{
				"ai_model_serving_enabled":                true,
				"ai_candidate_percentage_rollout_enabled": true,
				"ai_candidate_user_opt_in_enabled":        true,
			},
		},
		Baseline:  testBaseline(),
		Candidate: &candidate,
	})
	if decision.AssignmentType != AssignmentPercentageCandidate {
		t.Fatalf("expected opted-in percentage candidate, got %s", decision.AssignmentType)
	}
}

func TestDecideShadowKeepsBaselinePrimary(t *testing.T) {
	candidate := testCandidate(func(d *Deployment) {
		d.Status = StatusShadow
		d.TrafficMode = TrafficShadow
		d.ShadowSamplePercent = 100
	})

	decision := Decide(DecisionInput{
		Config: testConfig(),
		Context: RoutingContext{
			RequestKey: "shadow-request",
			FeatureFlags: map[string]bool{
				"ai_model_serving_enabled":     true,
				"ai_shadow_evaluation_enabled": true,
			},
		},
		Baseline:  testBaseline(),
		Candidate: &candidate,
	})

	if decision.AssignmentType != AssignmentShadow {
		t.Fatalf("expected shadow assignment, got %s", decision.AssignmentType)
	}
	if decision.PrimaryDeployment.ModelVariant != VariantGroundedBaseline {
		t.Fatalf("shadow assignment must keep baseline primary, got %s", decision.PrimaryDeployment.ModelVariant)
	}
	if decision.ShadowDeployment == nil || !decision.ComparisonRequired {
		t.Fatalf("expected shadow deployment and comparison requirement, got %+v", decision)
	}
}

func testConfig() Config {
	return Config{
		ModelServingEnabled:      true,
		ShadowEnabled:            true,
		ShadowSamplePercent:      100,
		InternalRolloutEnabled:   true,
		AllowlistRolloutEnabled:  true,
		PercentageRolloutEnabled: true,
		UserOptInEnabled:         true,
		DeploymentAssignmentSalt: "salt-v1",
		Guardrails:               DefaultGuardrailConfig(),
	}
}

func testBaseline() Deployment {
	return Deployment{
		ID:             uuid.MustParse("00000000-0000-0000-0000-000000000201"),
		DeploymentKey:  "grounded-baseline",
		ModelVariant:   VariantGroundedBaseline,
		Status:         StatusActive,
		TrafficMode:    TrafficActive,
		TaskType:       TaskGroundedItineraryGeneration,
		AssignmentSalt: "salt-v1",
		PromptVersion:  "itinerary_generation_v1",
	}
}

func testCandidate(mutators ...func(*Deployment)) Deployment {
	d := Deployment{
		ID:             uuid.MustParse("00000000-0000-0000-0000-000000000202"),
		DeploymentKey:  "candidate-v1",
		ModelVariant:   VariantFineTunedCandidate,
		Status:         StatusCandidate,
		TrafficMode:    TrafficDisabled,
		TaskType:       TaskGroundedItineraryGeneration,
		AssignmentSalt: "salt-v1",
		PromptVersion:  "itinerary_generation_v1",
	}
	for _, mutate := range mutators {
		mutate(&d)
	}
	return d
}

func ptrDeployment(d Deployment) *Deployment { return &d }

func userIDPtr(id uuid.UUID) *uuid.UUID { return &id }
