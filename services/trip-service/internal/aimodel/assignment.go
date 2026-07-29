package aimodel

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"
	"time"
)

type DecisionInput struct {
	Config    Config
	Context   RoutingContext
	Baseline  Deployment
	Candidate *Deployment
}

func Decide(input DecisionInput) RoutingDecision {
	cfg := NormalizeConfig(input.Config)
	ctx := normalizeRoutingContext(input.Context)
	baseline := input.Baseline
	if baseline.AssignmentSalt == "" {
		baseline.AssignmentSalt = firstNonEmpty(cfg.DeploymentAssignmentSalt, "baseline")
	}
	if baseline.ModelVariant == "" {
		baseline.ModelVariant = VariantGroundedBaseline
	}
	if baseline.TrafficMode == "" {
		baseline.TrafficMode = TrafficActive
	}
	if baseline.Status == "" {
		baseline.Status = StatusActive
	}

	decision := baselineDecision(baseline, "model_serving_disabled")
	if !cfg.ModelServingEnabled || !ctx.FeatureFlagsEnabled("ai_model_serving_enabled", true) {
		return decision
	}
	decision.Reason = "baseline_default"

	candidate := input.Candidate
	if candidate == nil {
		return decision
	}
	if !candidate.MatchesFeatureFlags(ctx.FeatureFlags) {
		decision.Reason = "candidate_feature_flag_disabled"
		return decision
	}
	if !candidate.CanServe() || !candidate.IsCandidate() {
		decision.Reason = "candidate_not_serving"
		return decision
	}

	salt := firstNonEmpty(candidate.AssignmentSalt, cfg.DeploymentAssignmentSalt, baseline.AssignmentSalt)
	stickyKey := stableAssignmentKey(ctx)
	bucket := DeterministicBucket(salt, stickyKey)
	decision.DeterministicBucket = &bucket

	if ctx.ForcedOpsTest && ctx.IsOpsUser && candidate.CanServe() {
		return candidateDecision(baseline, *candidate, AssignmentForcedOpsTest, bucket, "forced_ops_test")
	}
	if cfg.InternalRolloutEnabled && ctx.FeatureFlagsEnabled("ai_candidate_internal_rollout_enabled", false) &&
		ctx.IsOpsUser && candidate.SupportsInternal() {
		return candidateDecision(baseline, *candidate, AssignmentInternalCandidate, bucket, "internal_ops_user")
	}
	if cfg.AllowlistRolloutEnabled && ctx.FeatureFlagsEnabled("ai_candidate_allowlist_rollout_enabled", false) &&
		candidate.SupportsAllowlist() &&
		(candidate.HasUserAllowlist(ctx.UserID) || candidate.HasWorkspaceAllowlist(ctx.WorkspaceID)) {
		return candidateDecision(baseline, *candidate, AssignmentAllowlistCandidate, bucket, "allowlisted_subject")
	}
	if cfg.PercentageRolloutEnabled && cfg.UserOptInEnabled &&
		ctx.FeatureFlagsEnabled("ai_candidate_percentage_rollout_enabled", false) &&
		ctx.FeatureFlagsEnabled("ai_candidate_user_opt_in_enabled", false) &&
		ctx.UserOptInExperimentalAI && candidate.SupportsPercentage() &&
		bucket < percentToBucketLimit(firstPositive(candidate.RolloutPercent, cfg.CandidateRolloutPercent)) {
		return candidateDecision(baseline, *candidate, AssignmentPercentageCandidate, bucket, "opt_in_percentage_bucket")
	}
	if candidate.SupportsActiveCandidate() {
		return candidateDecision(baseline, *candidate, AssignmentPercentageCandidate, bucket, "active_candidate")
	}
	if cfg.ShadowEnabled && ctx.FeatureFlagsEnabled("ai_shadow_evaluation_enabled", false) &&
		candidate.SupportsShadow() &&
		bucket < percentToBucketLimit(firstPositive(candidate.ShadowSamplePercent, cfg.ShadowSamplePercent)) {
		decision.AssignmentType = AssignmentShadow
		shadow := *candidate
		decision.ShadowDeployment = &shadow
		decision.ShadowEligible = true
		decision.ComparisonRequired = true
		decision.Reason = "shadow_sample_bucket"
		decision.Metadata = decisionMetadata(decision)
		return decision
	}
	decision.Reason = "baseline_only"
	decision.Metadata = decisionMetadata(decision)
	return decision
}

func DeterministicBucket(salt string, stableKey string) int {
	normalizedSalt := strings.TrimSpace(salt)
	if normalizedSalt == "" {
		normalizedSalt = "missing-salt"
	}
	normalizedKey := strings.TrimSpace(stableKey)
	if normalizedKey == "" {
		normalizedKey = "anonymous"
	}
	digest := sha256.Sum256([]byte(normalizedSalt + ":" + normalizedKey))
	return int(binary.BigEndian.Uint64(digest[:8]) % 10000)
}

func stableAssignmentKey(ctx RoutingContext) string {
	switch {
	case ctx.UserID != nil:
		return "user:" + ctx.UserID.String()
	case ctx.WorkspaceID != nil:
		return "workspace:" + ctx.WorkspaceID.String()
	case ctx.TripID != nil:
		return "trip:" + ctx.TripID.String()
	default:
		return "request:" + strings.TrimSpace(ctx.RequestKey)
	}
}

func baselineDecision(baseline Deployment, reason string) RoutingDecision {
	decision := RoutingDecision{
		PrimaryDeployment:  baseline,
		UserVisibleVariant: baseline.ModelVariant,
		AssignmentType:     AssignmentBaselineOnly,
		Reason:             reason,
		Metadata:           map[string]any{},
	}
	decision.Metadata = decisionMetadata(decision)
	return decision
}

func candidateDecision(
	baseline Deployment,
	candidate Deployment,
	assignmentType AssignmentType,
	bucket int,
	reason string,
) RoutingDecision {
	decision := RoutingDecision{
		PrimaryDeployment:   candidate,
		UserVisibleVariant:  candidate.ModelVariant,
		AssignmentType:      assignmentType,
		DeterministicBucket: &bucket,
		Reason:              reason,
		Metadata:            map[string]any{},
	}
	if baseline.ID != candidate.ID {
		decision.Metadata = map[string]any{
			"baselineDeploymentKey": baseline.DeploymentKey,
		}
	}
	decision.Metadata = decisionMetadata(decision)
	return decision
}

func decisionMetadata(decision RoutingDecision) map[string]any {
	metadata := map[string]any{
		"deploymentKey":  decision.PrimaryDeployment.DeploymentKey,
		"modelVariant":   string(decision.UserVisibleVariant),
		"assignmentType": string(decision.AssignmentType),
		"inferenceMode":  string(InferenceModePrimary),
	}
	if decision.DeterministicBucket != nil {
		metadata["deterministicBucket"] = *decision.DeterministicBucket
	}
	if decision.PrimaryDeployment.PromptVersion != "" {
		metadata["promptVersion"] = decision.PrimaryDeployment.PromptVersion
	}
	if decision.PrimaryDeployment.GroundingVersion != "" {
		metadata["groundingVersion"] = decision.PrimaryDeployment.GroundingVersion
	}
	if decision.PrimaryDeployment.ValidatorVersion != "" {
		metadata["validatorVersion"] = decision.PrimaryDeployment.ValidatorVersion
	}
	if decision.ShadowDeployment != nil {
		metadata["shadowDeploymentKey"] = decision.ShadowDeployment.DeploymentKey
	}
	return metadata
}

func normalizeRoutingContext(ctx RoutingContext) RoutingContext {
	if ctx.TaskType == "" {
		ctx.TaskType = TaskGroundedItineraryGeneration
	}
	if ctx.Environment == "" {
		ctx.Environment = "local"
	}
	if ctx.FeatureFlags == nil {
		ctx.FeatureFlags = map[string]bool{}
	}
	if ctx.RequestTimestamp.IsZero() {
		ctx.RequestTimestamp = time.Now().UTC()
	}
	return ctx
}

func (ctx RoutingContext) FeatureFlagsEnabled(key string, fallback bool) bool {
	if ctx.FeatureFlags == nil {
		return fallback
	}
	value, ok := ctx.FeatureFlags[key]
	if !ok {
		return fallback
	}
	return value
}

func percentToBucketLimit(percent float64) int {
	percent = clampPercent(percent)
	return int(percent * 100)
}

func firstPositive(values ...float64) float64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
