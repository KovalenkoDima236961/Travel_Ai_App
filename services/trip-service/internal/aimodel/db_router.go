package aimodel

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

type DBRouter struct {
	db       *storage.DB
	cfg      Config
	env      string
	fallback *StaticRouter
}

func NewDBRouter(db *storage.DB, cfg Config, environment string) *DBRouter {
	cfg = NormalizeConfig(cfg)
	env := strings.TrimSpace(environment)
	if env == "" {
		env = "local"
	}
	return &DBRouter{
		db:       db,
		cfg:      cfg,
		env:      env,
		fallback: NewStaticRouter(cfg, env),
	}
}

func (r *DBRouter) Decide(ctx context.Context, routing RoutingContext) (RoutingDecision, error) {
	if r == nil {
		return RoutingDecision{}, nil
	}
	if r.db == nil {
		return r.fallback.Decide(ctx, routing)
	}
	if !r.cfg.ModelServingEnabled {
		return r.fallback.Decide(ctx, routing)
	}
	if strings.TrimSpace(routing.Environment) == "" {
		routing.Environment = r.env
	}
	routing.FeatureFlags = defaultModelFeatureFlags(routing.FeatureFlags, r.cfg)

	baseline, err := r.loadBaseline(ctx, routing)
	if err != nil {
		if err == domainerrs.ErrNotFound {
			return r.fallback.Decide(ctx, routing)
		}
		return RoutingDecision{}, err
	}
	candidate, err := r.loadCandidate(ctx, routing)
	if err != nil && err != domainerrs.ErrNotFound {
		return RoutingDecision{}, err
	}
	decision := Decide(DecisionInput{
		Config:    r.cfg,
		Context:   routing,
		Baseline:  *baseline,
		Candidate: candidate,
	})
	if assignmentID, err := r.recordAssignment(ctx, routing, *baseline, decision); err == nil {
		if decision.Metadata == nil {
			decision.Metadata = map[string]any{}
		}
		decision.Metadata["requestAssignmentId"] = assignmentID.String()
	}
	return decision, nil
}

func (r *DBRouter) loadBaseline(ctx context.Context, routing RoutingContext) (*Deployment, error) {
	key := strings.TrimSpace(r.cfg.DefaultDeploymentKey)
	if key != "" {
		deployment, err := scanDeployment(r.db.QueryRow(ctx, `SELECT `+deploymentSelectColumns+`
FROM ai_model_deployments
WHERE environment = $1
  AND task_type = $2
  AND deployment_key = $3
  AND model_variant = 'grounded_baseline'
  AND status = 'active'
  AND traffic_mode = 'active'
LIMIT 1`, routing.Environment, routing.TaskType, key))
		if err == nil {
			return deployment, nil
		}
		if err != domainerrs.ErrNotFound {
			return nil, err
		}
	}
	return scanDeployment(r.db.QueryRow(ctx, `SELECT `+deploymentSelectColumns+`
FROM ai_model_deployments
WHERE environment = $1
  AND task_type = $2
  AND model_variant = 'grounded_baseline'
  AND status = 'active'
  AND traffic_mode = 'active'
ORDER BY updated_at DESC
LIMIT 1`, routing.Environment, routing.TaskType))
}

func (r *DBRouter) loadCandidate(ctx context.Context, routing RoutingContext) (*Deployment, error) {
	return scanDeployment(r.db.QueryRow(ctx, `SELECT `+deploymentSelectColumns+`
FROM ai_model_deployments
WHERE environment = $1
  AND task_type = $2
  AND model_variant = 'fine_tuned_candidate'
  AND status IN ('shadow','internal','allowlist','staged_rollout','active')
  AND traffic_mode <> 'disabled'
ORDER BY
  CASE status
    WHEN 'active' THEN 1
    WHEN 'staged_rollout' THEN 2
    WHEN 'allowlist' THEN 3
    WHEN 'internal' THEN 4
    WHEN 'shadow' THEN 5
    ELSE 99
  END,
  updated_at DESC
LIMIT 1`, routing.Environment, routing.TaskType))
}

func (r *DBRouter) recordAssignment(ctx context.Context, routing RoutingContext, baseline Deployment, decision RoutingDecision) (uuid.UUID, error) {
	if baseline.ID == uuid.Nil || strings.TrimSpace(routing.RequestKey) == "" {
		return uuid.Nil, fmt.Errorf("assignment persistence requires baseline deployment and request key")
	}
	candidate := candidateDeploymentForDecision(decision)
	decisionCandidateID := nullableDeploymentID(candidate)
	candidateUserVisible := decision.UserVisibleVariant == VariantFineTunedCandidate
	assignmentType := decision.AssignmentType
	bucket := decision.DeterministicBucket
	var scannedID pgtype.UUID
	err := r.db.QueryRow(ctx, `
INSERT INTO ai_model_request_assignments (
    id,
    request_key,
    user_id,
    workspace_id,
    trip_id,
    task_type,
    environment,
    baseline_deployment_id,
    candidate_deployment_id,
    assignment_type,
    bucket,
    candidate_user_visible
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
ON CONFLICT (request_key) DO UPDATE SET request_key = EXCLUDED.request_key
RETURNING id`,
		idArg(uuid.New()),
		strings.TrimSpace(routing.RequestKey),
		nullableIDArg(routing.UserID),
		nullableIDArg(routing.WorkspaceID),
		nullableIDArg(routing.TripID),
		firstNonEmpty(routing.TaskType, TaskGroundedItineraryGeneration),
		firstNonEmpty(routing.Environment, r.env, "local"),
		idArg(baseline.ID),
		decisionCandidateID,
		string(assignmentType),
		nullableIntArg(bucket),
		candidateUserVisible,
	).Scan(&scannedID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("record ai model request assignment: %w", err)
	}
	return uuid.UUID(scannedID.Bytes), nil
}

func candidateDeploymentForDecision(decision RoutingDecision) *Deployment {
	if decision.ShadowDeployment != nil {
		return decision.ShadowDeployment
	}
	if decision.UserVisibleVariant == VariantFineTunedCandidate {
		return &decision.PrimaryDeployment
	}
	return nil
}

func nullableDeploymentID(deployment *Deployment) any {
	if deployment == nil || deployment.ID == uuid.Nil {
		return nil
	}
	return idArg(deployment.ID)
}

func nullableIntArg(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func defaultModelFeatureFlags(flags map[string]bool, cfg Config) map[string]bool {
	out := make(map[string]bool, len(flags)+7)
	for key, value := range flags {
		out[key] = value
	}
	setDefault := func(key string, value bool) {
		if _, ok := out[key]; !ok {
			out[key] = value
		}
	}
	setDefault("ai_model_serving_enabled", cfg.ModelServingEnabled)
	setDefault("ai_shadow_evaluation_enabled", cfg.ShadowEnabled)
	setDefault("ai_candidate_internal_rollout_enabled", cfg.InternalRolloutEnabled)
	setDefault("ai_candidate_allowlist_rollout_enabled", cfg.AllowlistRolloutEnabled)
	setDefault("ai_candidate_percentage_rollout_enabled", cfg.PercentageRolloutEnabled)
	setDefault("ai_candidate_user_opt_in_enabled", cfg.UserOptInEnabled)
	setDefault("ai_automatic_guardrail_pause_enabled", cfg.AutomaticGuardrailPauseEnabled)
	return out
}
