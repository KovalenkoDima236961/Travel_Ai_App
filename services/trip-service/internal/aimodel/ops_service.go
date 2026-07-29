package aimodel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	storage "github.com/KovalenkoDima236961/Travel_Ai_App/internal/platform/storage/postgres"
)

var deploymentKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,127}$`)

type RegisterDeploymentInput struct {
	Environment             string           `json:"environment"`
	DeploymentKey           string           `json:"deploymentKey"`
	ModelID                 uuid.UUID        `json:"modelId"`
	AdapterID               *uuid.UUID       `json:"adapterId,omitempty"`
	ExperimentID            *uuid.UUID       `json:"experimentId,omitempty"`
	ModelVariant            ModelVariant     `json:"modelVariant"`
	Status                  DeploymentStatus `json:"status,omitempty"`
	TaskType                string           `json:"taskType"`
	TrafficMode             TrafficMode      `json:"trafficMode,omitempty"`
	ShadowSamplePercent     float64          `json:"shadowSamplePercent,omitempty"`
	RolloutPercent          float64          `json:"rolloutPercent,omitempty"`
	AllowlistedUserIDs      []uuid.UUID      `json:"allowlistedUserIds,omitempty"`
	AllowlistedWorkspaceIDs []uuid.UUID      `json:"allowlistedWorkspaceIds,omitempty"`
	InternalOnly            *bool            `json:"internalOnly,omitempty"`
	FeatureFlagKey          string           `json:"featureFlagKey,omitempty"`
	AssignmentSalt          string           `json:"assignmentSalt,omitempty"`
	PromptVersion           string           `json:"promptVersion"`
	GroundingVersion        string           `json:"groundingVersion,omitempty"`
	ValidatorVersion        string           `json:"validatorVersion,omitempty"`
	Config                  map[string]any   `json:"config,omitempty"`
	Reason                  string           `json:"reason"`
	ActorUserID             *uuid.UUID       `json:"-"`
	RequestID               string           `json:"-"`
}

type ShadowRolloutInput struct {
	ShadowSamplePercent float64    `json:"shadowSamplePercent"`
	Reason              string     `json:"reason"`
	ActorUserID         *uuid.UUID `json:"-"`
	RequestID           string     `json:"-"`
}

type DeploymentActionInput struct {
	Reason      string     `json:"reason"`
	ActorUserID *uuid.UUID `json:"-"`
	RequestID   string     `json:"-"`
}

type OnlineSummary struct {
	Deployment          Deployment     `json:"deployment"`
	ComparisonCount     int            `json:"comparisonCount"`
	CompletedCount      int            `json:"completedCount"`
	PendingCount        int            `json:"pendingCount"`
	FailureCount        int            `json:"failureCount"`
	GuardrailCounts     map[string]int `json:"guardrailCounts"`
	StatusCounts        map[string]int `json:"statusCounts"`
	BaselineLatencyAvg  float64        `json:"baselineLatencyAvgMs"`
	CandidateLatencyAvg float64        `json:"candidateLatencyAvgMs"`
	CandidateLatencyP95 float64        `json:"candidateLatencyP95Ms"`
	RecentWindowHours   int            `json:"recentWindowHours"`
	GeneratedAt         time.Time      `json:"generatedAt"`
}

type OpsService struct {
	db          *storage.DB
	cfg         Config
	environment string
	log         *zap.Logger
	now         func() time.Time
}

func NewOpsService(db *storage.DB, cfg Config, environment string, log *zap.Logger) *OpsService {
	if log == nil {
		log = zap.NewNop()
	}
	return &OpsService{
		db:          db,
		cfg:         NormalizeConfig(cfg),
		environment: strings.ToLower(strings.TrimSpace(environment)),
		log:         log,
		now:         time.Now,
	}
}

func (s *OpsService) Enabled() bool {
	return s != nil && s.db != nil
}

func (s *OpsService) RegisterDeployment(ctx context.Context, input RegisterDeploymentInput) (*Deployment, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("ai model ops is not configured")
	}
	input = s.normalizeRegisterInput(input)
	if err := validateRegisterInput(input); err != nil {
		return nil, err
	}
	allowlistedUsers, err := marshalUUIDArray(input.AllowlistedUserIDs)
	if err != nil {
		return nil, err
	}
	allowlistedWorkspaces, err := marshalUUIDArray(input.AllowlistedWorkspaceIDs)
	if err != nil {
		return nil, err
	}
	configJSON, err := marshalObject(input.Config)
	if err != nil {
		return nil, err
	}
	deploymentID := uuid.New()
	now := s.now().UTC()
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin ai model deployment registration: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
INSERT INTO ai_model_deployments (
    id, environment, deployment_key, model_id, adapter_id, experiment_id,
    model_variant, status, task_type, traffic_mode, shadow_sample_percent,
    rollout_percent, allowlisted_user_ids, allowlisted_workspace_ids,
    internal_only, feature_flag_key, assignment_salt, prompt_version,
    grounding_version, validator_version, config_json, created_by_user_id,
    updated_by_user_id, created_at, updated_at
) VALUES (
    $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb,$14::jsonb,$15,NULLIF($16,''),
    $17,$18,NULLIF($19,''),NULLIF($20,''),$21::jsonb,$22,$22,$23,$23
)
RETURNING `+deploymentSelectColumns,
		idArg(deploymentID),
		input.Environment,
		input.DeploymentKey,
		idArg(input.ModelID),
		nullableIDArg(input.AdapterID),
		nullableIDArg(input.ExperimentID),
		string(input.ModelVariant),
		string(input.Status),
		input.TaskType,
		string(input.TrafficMode),
		input.ShadowSamplePercent,
		input.RolloutPercent,
		string(allowlistedUsers),
		string(allowlistedWorkspaces),
		*input.InternalOnly,
		strings.TrimSpace(input.FeatureFlagKey),
		input.AssignmentSalt,
		input.PromptVersion,
		input.GroundingVersion,
		input.ValidatorVersion,
		string(configJSON),
		nullableIDArg(input.ActorUserID),
		now,
	)
	deployment, err := scanDeployment(row)
	if err != nil {
		return nil, fmt.Errorf("register ai model deployment: %w", err)
	}
	if err := insertDeploymentEventTx(ctx, tx, deployment.ID, input.ActorUserID, "created", "", string(deployment.Status), nil, deploymentEventConfig(deployment), input.Reason, input.RequestID, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit ai model deployment registration: %w", err)
	}
	return deployment, nil
}

func (s *OpsService) EnableShadow(ctx context.Context, id uuid.UUID, input ShadowRolloutInput) (*Deployment, error) {
	if input.ShadowSamplePercent < 0 || input.ShadowSamplePercent > 100 {
		return nil, fmt.Errorf("shadowSamplePercent must be between 0 and 100")
	}
	return s.updateDeploymentState(ctx, id, deploymentStateUpdate{
		Action:              "enabled_shadow",
		NewStatus:           StatusShadow,
		NewTrafficMode:      TrafficShadow,
		ShadowSamplePercent: &input.ShadowSamplePercent,
		RolloutPercent:      floatPtr(0),
		Reason:              input.Reason,
		ActorUserID:         input.ActorUserID,
		RequestID:           input.RequestID,
	})
}

func (s *OpsService) PauseDeployment(ctx context.Context, id uuid.UUID, input DeploymentActionInput) (*Deployment, error) {
	return s.updateDeploymentState(ctx, id, deploymentStateUpdate{
		Action:              "paused",
		NewStatus:           StatusPaused,
		NewTrafficMode:      TrafficDisabled,
		ShadowSamplePercent: floatPtr(0),
		RolloutPercent:      floatPtr(0),
		SetPausedAt:         true,
		Reason:              input.Reason,
		ActorUserID:         input.ActorUserID,
		RequestID:           input.RequestID,
	})
}

func (s *OpsService) RollbackDeployment(ctx context.Context, id uuid.UUID, input DeploymentActionInput) (*Deployment, error) {
	return s.updateDeploymentState(ctx, id, deploymentStateUpdate{
		Action:              "rollback",
		NewStatus:           StatusRetired,
		NewTrafficMode:      TrafficDisabled,
		ShadowSamplePercent: floatPtr(0),
		RolloutPercent:      floatPtr(0),
		SetPausedAt:         true,
		SetRetiredAt:        true,
		Reason:              input.Reason,
		ActorUserID:         input.ActorUserID,
		RequestID:           input.RequestID,
	})
}

func (s *OpsService) OnlineSummary(ctx context.Context, id uuid.UUID) (*OnlineSummary, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("ai model ops is not configured")
	}
	deployment, err := s.GetDeployment(ctx, id)
	if err != nil {
		return nil, err
	}
	summary := OnlineSummary{
		Deployment:        *deployment,
		GuardrailCounts:   map[string]int{},
		StatusCounts:      map[string]int{},
		RecentWindowHours: 24,
		GeneratedAt:       s.now().UTC(),
	}
	if err := s.db.QueryRow(ctx, `
SELECT
    COUNT(*)::int,
    COUNT(*) FILTER (WHERE comparison_status = 'completed')::int,
    COUNT(*) FILTER (WHERE comparison_status = 'pending')::int,
    COUNT(*) FILTER (WHERE comparison_status IN ('baseline_failed','candidate_failed','timed_out','invalid'))::int,
    COALESCE(AVG(baseline_latency_ms), 0)::float8,
    COALESCE(AVG(candidate_latency_ms), 0)::float8,
    COALESCE(percentile_cont(0.95) WITHIN GROUP (ORDER BY candidate_latency_ms), 0)::float8
FROM ai_model_online_comparisons
WHERE candidate_deployment_id = $1
  AND created_at >= NOW() - INTERVAL '24 hours'`,
		idArg(id),
	).Scan(
		&summary.ComparisonCount,
		&summary.CompletedCount,
		&summary.PendingCount,
		&summary.FailureCount,
		&summary.BaselineLatencyAvg,
		&summary.CandidateLatencyAvg,
		&summary.CandidateLatencyP95,
	); err != nil {
		return nil, fmt.Errorf("query ai model online summary: %w", err)
	}
	guardrailCounts, err := s.countComparisonField(ctx, id, "guardrail_status")
	if err != nil {
		return nil, err
	}
	statusCounts, err := s.countComparisonField(ctx, id, "comparison_status")
	if err != nil {
		return nil, err
	}
	summary.GuardrailCounts = guardrailCounts
	summary.StatusCounts = statusCounts
	return &summary, nil
}

func (s *OpsService) GetDeployment(ctx context.Context, id uuid.UUID) (*Deployment, error) {
	row := s.db.QueryRow(ctx, `SELECT `+deploymentSelectColumns+` FROM ai_model_deployments WHERE id = $1`, idArg(id))
	deployment, err := scanDeployment(row)
	if errors.Is(err, domainerrs.ErrNotFound) {
		return nil, domainerrs.ErrNotFound
	}
	return deployment, err
}

type deploymentStateUpdate struct {
	Action              string
	NewStatus           DeploymentStatus
	NewTrafficMode      TrafficMode
	ShadowSamplePercent *float64
	RolloutPercent      *float64
	SetPausedAt         bool
	SetRetiredAt        bool
	Reason              string
	ActorUserID         *uuid.UUID
	RequestID           string
}

func (s *OpsService) updateDeploymentState(ctx context.Context, id uuid.UUID, update deploymentStateUpdate) (*Deployment, error) {
	if !s.Enabled() {
		return nil, fmt.Errorf("ai model ops is not configured")
	}
	if strings.TrimSpace(update.Reason) == "" {
		return nil, fmt.Errorf("reason is required")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("begin ai model deployment update: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	oldDeployment, err := scanDeployment(tx.QueryRow(ctx, `SELECT `+deploymentSelectColumns+` FROM ai_model_deployments WHERE id = $1 FOR UPDATE`, idArg(id)))
	if err != nil {
		return nil, err
	}
	shadowSample := oldDeployment.ShadowSamplePercent
	if update.ShadowSamplePercent != nil {
		shadowSample = *update.ShadowSamplePercent
	}
	rolloutPercent := oldDeployment.RolloutPercent
	if update.RolloutPercent != nil {
		rolloutPercent = *update.RolloutPercent
	}
	if err := validateStateTransition(*oldDeployment, update.NewStatus, update.NewTrafficMode); err != nil {
		return nil, err
	}
	now := s.now().UTC()
	var pausedAtExpr, retiredAtExpr string
	if update.SetPausedAt {
		pausedAtExpr = "NOW()"
	} else {
		pausedAtExpr = "paused_at"
	}
	if update.SetRetiredAt {
		retiredAtExpr = "NOW()"
	} else {
		retiredAtExpr = "retired_at"
	}
	query := fmt.Sprintf(`
UPDATE ai_model_deployments
SET status = $2,
    traffic_mode = $3,
    shadow_sample_percent = $4,
    rollout_percent = $5,
    updated_by_user_id = $6,
    updated_at = $7,
    paused_at = %s,
    retired_at = %s
WHERE id = $1
RETURNING `+deploymentSelectColumns, pausedAtExpr, retiredAtExpr)
	newDeployment, err := scanDeployment(tx.QueryRow(ctx, query,
		idArg(id),
		string(update.NewStatus),
		string(update.NewTrafficMode),
		shadowSample,
		rolloutPercent,
		nullableIDArg(update.ActorUserID),
		now,
	))
	if err != nil {
		return nil, fmt.Errorf("update ai model deployment: %w", err)
	}
	if err := insertDeploymentEventTx(ctx, tx, id, update.ActorUserID, update.Action, string(oldDeployment.Status), string(newDeployment.Status), deploymentEventConfig(oldDeployment), deploymentEventConfig(newDeployment), update.Reason, update.RequestID, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit ai model deployment update: %w", err)
	}
	return newDeployment, nil
}

func (s *OpsService) countComparisonField(ctx context.Context, id uuid.UUID, field string) (map[string]int, error) {
	if field != "guardrail_status" && field != "comparison_status" {
		return nil, fmt.Errorf("unsupported comparison count field")
	}
	rows, err := s.db.Query(ctx, fmt.Sprintf(`
SELECT %s, COUNT(*)::int
FROM ai_model_online_comparisons
WHERE candidate_deployment_id = $1
  AND created_at >= NOW() - INTERVAL '24 hours'
GROUP BY %s`, field, field), idArg(id))
	if err != nil {
		return nil, fmt.Errorf("query ai model %s counts: %w", field, err)
	}
	defer rows.Close()
	out := map[string]int{}
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return nil, fmt.Errorf("scan ai model %s count: %w", field, err)
		}
		out[key] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ai model %s counts: %w", field, err)
	}
	return out, nil
}

func (s *OpsService) normalizeRegisterInput(input RegisterDeploymentInput) RegisterDeploymentInput {
	input.Environment = strings.TrimSpace(input.Environment)
	if input.Environment == "" {
		input.Environment = s.environment
	}
	if input.Environment == "" {
		input.Environment = "local"
	}
	input.DeploymentKey = strings.TrimSpace(input.DeploymentKey)
	input.TaskType = strings.TrimSpace(input.TaskType)
	if input.TaskType == "" {
		input.TaskType = TaskGroundedItineraryGeneration
	}
	if input.Status == "" {
		input.Status = StatusRegistered
	}
	if input.TrafficMode == "" {
		input.TrafficMode = TrafficDisabled
	}
	if input.InternalOnly == nil {
		defaultInternalOnly := true
		input.InternalOnly = &defaultInternalOnly
	}
	input.AssignmentSalt = strings.TrimSpace(input.AssignmentSalt)
	if input.AssignmentSalt == "" {
		input.AssignmentSalt = strings.TrimSpace(s.cfg.DeploymentAssignmentSalt)
	}
	if input.AssignmentSalt == "" {
		input.AssignmentSalt = input.DeploymentKey + ":" + uuid.NewString()
	}
	input.PromptVersion = strings.TrimSpace(input.PromptVersion)
	input.GroundingVersion = strings.TrimSpace(input.GroundingVersion)
	input.ValidatorVersion = strings.TrimSpace(input.ValidatorVersion)
	input.FeatureFlagKey = strings.TrimSpace(input.FeatureFlagKey)
	input.Reason = strings.TrimSpace(input.Reason)
	if input.Config == nil {
		input.Config = map[string]any{}
	}
	return input
}

func validateRegisterInput(input RegisterDeploymentInput) error {
	if !validDeploymentKey(input.DeploymentKey) {
		return fmt.Errorf("deploymentKey must be 1-128 characters and contain only letters, numbers, dots, underscores, colons, or hyphens")
	}
	if input.ModelID == uuid.Nil {
		return fmt.Errorf("modelId is required")
	}
	if !validModelVariant(input.ModelVariant) {
		return fmt.Errorf("modelVariant is unsupported")
	}
	if input.ModelVariant == VariantFineTunedCandidate && input.AdapterID == nil {
		return fmt.Errorf("adapterId is required for fine_tuned_candidate")
	}
	if !validDeploymentStatus(input.Status) {
		return fmt.Errorf("status is unsupported")
	}
	if !validTrafficMode(input.TrafficMode) {
		return fmt.Errorf("trafficMode is unsupported")
	}
	if strings.TrimSpace(input.PromptVersion) == "" {
		return fmt.Errorf("promptVersion is required")
	}
	if strings.TrimSpace(input.Reason) == "" {
		return fmt.Errorf("reason is required")
	}
	if input.ShadowSamplePercent < 0 || input.ShadowSamplePercent > 100 {
		return fmt.Errorf("shadowSamplePercent must be between 0 and 100")
	}
	if input.RolloutPercent < 0 || input.RolloutPercent > 100 {
		return fmt.Errorf("rolloutPercent must be between 0 and 100")
	}
	return validateStateTransition(Deployment{ModelVariant: input.ModelVariant}, input.Status, input.TrafficMode)
}

func validDeploymentKey(key string) bool {
	return deploymentKeyPattern.MatchString(strings.TrimSpace(key)) && !strings.Contains(key, "..")
}

func validModelVariant(value ModelVariant) bool {
	return value == VariantGroundedBaseline || value == VariantFineTunedCandidate
}

func validDeploymentStatus(value DeploymentStatus) bool {
	switch value {
	case StatusRegistered, StatusCandidate, StatusShadow, StatusInternal, StatusAllowlist, StatusStagedRollout, StatusActive, StatusPaused, StatusRejected, StatusRetired:
		return true
	default:
		return false
	}
}

func validTrafficMode(value TrafficMode) bool {
	switch value {
	case TrafficDisabled, TrafficShadow, TrafficInternal, TrafficAllowlist, TrafficPercentage, TrafficActive:
		return true
	default:
		return false
	}
}

func validateStateTransition(deployment Deployment, status DeploymentStatus, mode TrafficMode) error {
	if !validDeploymentStatus(status) || !validTrafficMode(mode) {
		return fmt.Errorf("unsupported deployment state")
	}
	switch status {
	case StatusRegistered, StatusCandidate, StatusPaused, StatusRejected, StatusRetired:
		if mode != TrafficDisabled {
			return fmt.Errorf("%s deployments must use disabled traffic mode", status)
		}
	case StatusShadow:
		if mode != TrafficShadow {
			return fmt.Errorf("shadow deployments must use shadow traffic mode")
		}
	case StatusInternal:
		if mode != TrafficInternal {
			return fmt.Errorf("internal deployments must use internal traffic mode")
		}
	case StatusAllowlist:
		if mode != TrafficAllowlist {
			return fmt.Errorf("allowlist deployments must use allowlist traffic mode")
		}
	case StatusStagedRollout:
		if mode != TrafficPercentage {
			return fmt.Errorf("staged rollout deployments must use percentage traffic mode")
		}
	case StatusActive:
		if mode != TrafficActive {
			return fmt.Errorf("active deployments must use active traffic mode")
		}
	}
	if deployment.ModelVariant == VariantGroundedBaseline {
		switch status {
		case StatusShadow, StatusInternal, StatusAllowlist, StatusStagedRollout:
			return fmt.Errorf("grounded_baseline deployments cannot use candidate rollout states")
		}
	}
	return nil
}

const deploymentSelectColumns = `
id,
environment,
deployment_key,
model_id,
adapter_id,
experiment_id,
model_variant,
status,
task_type,
traffic_mode,
shadow_sample_percent::float8,
rollout_percent::float8,
allowlisted_user_ids::text,
allowlisted_workspace_ids::text,
internal_only,
COALESCE(feature_flag_key, ''),
assignment_salt,
prompt_version,
COALESCE(grounding_version, ''),
COALESCE(validator_version, ''),
config_json::text,
activated_at,
paused_at,
retired_at,
created_at,
updated_at`

func scanDeployment(row pgx.Row) (*Deployment, error) {
	var (
		deployment                                    Deployment
		id, modelID, adapterID, experimentID          pgtype.UUID
		allowlistedUsersRaw, allowlistedWorkspacesRaw string
		configRaw                                     string
		activatedAt, pausedAt, retiredAt              pgtype.Timestamp
	)
	err := row.Scan(
		&id,
		&deployment.Environment,
		&deployment.DeploymentKey,
		&modelID,
		&adapterID,
		&experimentID,
		&deployment.ModelVariant,
		&deployment.Status,
		&deployment.TaskType,
		&deployment.TrafficMode,
		&deployment.ShadowSamplePercent,
		&deployment.RolloutPercent,
		&allowlistedUsersRaw,
		&allowlistedWorkspacesRaw,
		&deployment.InternalOnly,
		&deployment.FeatureFlagKey,
		&deployment.AssignmentSalt,
		&deployment.PromptVersion,
		&deployment.GroundingVersion,
		&deployment.ValidatorVersion,
		&configRaw,
		&activatedAt,
		&pausedAt,
		&retiredAt,
		&deployment.CreatedAt,
		&deployment.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domainerrs.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	deployment.ID = uuid.UUID(id.Bytes)
	deployment.ModelID = uuid.UUID(modelID.Bytes)
	deployment.AdapterID = uuidPtrFromPg(adapterID)
	deployment.ExperimentID = uuidPtrFromPg(experimentID)
	deployment.ActivatedAt = timePtrFromPg(activatedAt)
	deployment.PausedAt = timePtrFromPg(pausedAt)
	deployment.RetiredAt = timePtrFromPg(retiredAt)
	deployment.AllowlistedUserIDs = parseUUIDArray(allowlistedUsersRaw)
	deployment.AllowlistedWorkspaceIDs = parseUUIDArray(allowlistedWorkspacesRaw)
	deployment.Config = map[string]any{}
	if strings.TrimSpace(configRaw) != "" {
		_ = json.Unmarshal([]byte(configRaw), &deployment.Config)
	}
	return &deployment, nil
}

func insertDeploymentEventTx(ctx context.Context, tx pgx.Tx, deploymentID uuid.UUID, actorUserID *uuid.UUID, action, oldStatus, newStatus string, oldConfig, newConfig map[string]any, reason, requestID string, now time.Time) error {
	oldRaw, err := marshalNullableObject(oldConfig)
	if err != nil {
		return err
	}
	newRaw, err := marshalNullableObject(newConfig)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
INSERT INTO ai_model_deployment_events (
    id, deployment_id, actor_user_id, action, old_status, new_status,
    old_config, new_config, reason, request_id, created_at
) VALUES ($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),NULLIF($7::jsonb, 'null'::jsonb),NULLIF($8::jsonb, 'null'::jsonb),$9,NULLIF($10,''),$11)`,
		idArg(uuid.New()),
		idArg(deploymentID),
		nullableIDArg(actorUserID),
		action,
		oldStatus,
		newStatus,
		string(oldRaw),
		string(newRaw),
		strings.TrimSpace(reason),
		strings.TrimSpace(requestID),
		now,
	)
	if err != nil {
		return fmt.Errorf("insert ai model deployment event: %w", err)
	}
	return nil
}

func deploymentEventConfig(deployment *Deployment) map[string]any {
	if deployment == nil {
		return nil
	}
	return map[string]any{
		"deploymentKey":       deployment.DeploymentKey,
		"status":              deployment.Status,
		"trafficMode":         deployment.TrafficMode,
		"shadowSamplePercent": deployment.ShadowSamplePercent,
		"rolloutPercent":      deployment.RolloutPercent,
		"internalOnly":        deployment.InternalOnly,
		"featureFlagKey":      deployment.FeatureFlagKey,
		"promptVersion":       deployment.PromptVersion,
		"groundingVersion":    deployment.GroundingVersion,
		"validatorVersion":    deployment.ValidatorVersion,
	}
}

func marshalUUIDArray(values []uuid.UUID) ([]byte, error) {
	out := make([]string, 0, len(values))
	for _, id := range values {
		if id != uuid.Nil {
			out = append(out, id.String())
		}
	}
	return json.Marshal(out)
}

func parseUUIDArray(raw string) []uuid.UUID {
	var values []string
	if err := json.Unmarshal([]byte(raw), &values); err != nil {
		return nil
	}
	out := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		id, err := uuid.Parse(strings.TrimSpace(value))
		if err == nil && id != uuid.Nil {
			out = append(out, id)
		}
	}
	return out
}

func marshalObject(value map[string]any) ([]byte, error) {
	if value == nil {
		value = map[string]any{}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal JSON object: %w", err)
	}
	return raw, nil
}

func marshalNullableObject(value map[string]any) ([]byte, error) {
	if value == nil {
		return []byte("null"), nil
	}
	return marshalObject(value)
}

func uuidPtrFromPg(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	id := uuid.UUID(value.Bytes)
	return &id
}

func timePtrFromPg(value pgtype.Timestamp) *time.Time {
	if !value.Valid {
		return nil
	}
	out := value.Time
	return &out
}

func floatPtr(value float64) *float64 {
	return &value
}
