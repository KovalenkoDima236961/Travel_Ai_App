// Package aimodel contains backend-only model serving, rollout, and online
// evaluation primitives for Trip Service.
package aimodel

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	TaskGroundedItineraryGeneration = "grounded_itinerary_generation"
)

type DeploymentStatus string

const (
	StatusRegistered    DeploymentStatus = "registered"
	StatusCandidate     DeploymentStatus = "candidate"
	StatusShadow        DeploymentStatus = "shadow"
	StatusInternal      DeploymentStatus = "internal"
	StatusAllowlist     DeploymentStatus = "allowlist"
	StatusStagedRollout DeploymentStatus = "staged_rollout"
	StatusActive        DeploymentStatus = "active"
	StatusPaused        DeploymentStatus = "paused"
	StatusRejected      DeploymentStatus = "rejected"
	StatusRetired       DeploymentStatus = "retired"
)

type ModelVariant string

const (
	VariantGroundedBaseline   ModelVariant = "grounded_baseline"
	VariantFineTunedCandidate ModelVariant = "fine_tuned_candidate"
)

type TrafficMode string

const (
	TrafficDisabled   TrafficMode = "disabled"
	TrafficShadow     TrafficMode = "shadow"
	TrafficInternal   TrafficMode = "internal"
	TrafficAllowlist  TrafficMode = "allowlist"
	TrafficPercentage TrafficMode = "percentage"
	TrafficActive     TrafficMode = "active"
)

type AssignmentType string

const (
	AssignmentBaselineOnly        AssignmentType = "baseline_only"
	AssignmentShadow              AssignmentType = "shadow"
	AssignmentInternalCandidate   AssignmentType = "internal_candidate"
	AssignmentAllowlistCandidate  AssignmentType = "allowlist_candidate"
	AssignmentPercentageCandidate AssignmentType = "percentage_candidate"
	AssignmentForcedOpsTest       AssignmentType = "forced_ops_test"
)

type InferenceMode string

const (
	InferenceModePrimary    InferenceMode = "primary"
	InferenceModeShadow     InferenceMode = "shadow"
	InferenceModeEvaluation InferenceMode = "evaluation"
)

type Deployment struct {
	ID                      uuid.UUID        `json:"id"`
	Environment             string           `json:"environment"`
	DeploymentKey           string           `json:"deploymentKey"`
	ModelID                 uuid.UUID        `json:"modelId"`
	AdapterID               *uuid.UUID       `json:"adapterId,omitempty"`
	ExperimentID            *uuid.UUID       `json:"experimentId,omitempty"`
	ModelVariant            ModelVariant     `json:"modelVariant"`
	Status                  DeploymentStatus `json:"status"`
	TaskType                string           `json:"taskType"`
	TrafficMode             TrafficMode      `json:"trafficMode"`
	ShadowSamplePercent     float64          `json:"shadowSamplePercent"`
	RolloutPercent          float64          `json:"rolloutPercent"`
	AllowlistedUserIDs      []uuid.UUID      `json:"allowlistedUserIds,omitempty"`
	AllowlistedWorkspaceIDs []uuid.UUID      `json:"allowlistedWorkspaceIds,omitempty"`
	InternalOnly            bool             `json:"internalOnly"`
	FeatureFlagKey          string           `json:"featureFlagKey,omitempty"`
	AssignmentSalt          string           `json:"assignmentSalt"`
	PromptVersion           string           `json:"promptVersion"`
	GroundingVersion        string           `json:"groundingVersion,omitempty"`
	ValidatorVersion        string           `json:"validatorVersion,omitempty"`
	Config                  map[string]any   `json:"config,omitempty"`
	ActivatedAt             *time.Time       `json:"activatedAt,omitempty"`
	PausedAt                *time.Time       `json:"pausedAt,omitempty"`
	RetiredAt               *time.Time       `json:"retiredAt,omitempty"`
	CreatedAt               time.Time        `json:"createdAt"`
	UpdatedAt               time.Time        `json:"updatedAt"`
}

func (d Deployment) IsBaseline() bool {
	return d.ModelVariant == VariantGroundedBaseline
}

func (d Deployment) IsCandidate() bool {
	return d.ModelVariant == VariantFineTunedCandidate
}

func (d Deployment) CanServe() bool {
	switch d.Status {
	case StatusPaused, StatusRejected, StatusRetired:
		return false
	}
	return d.TrafficMode != TrafficDisabled
}

func (d Deployment) SupportsShadow() bool {
	return d.CanServe() && d.IsCandidate() && d.Status == StatusShadow && d.TrafficMode == TrafficShadow
}

func (d Deployment) SupportsInternal() bool {
	return d.CanServe() && d.IsCandidate() && d.Status == StatusInternal && d.TrafficMode == TrafficInternal
}

func (d Deployment) SupportsAllowlist() bool {
	return d.CanServe() && d.IsCandidate() && d.Status == StatusAllowlist && d.TrafficMode == TrafficAllowlist
}

func (d Deployment) SupportsPercentage() bool {
	return d.CanServe() && d.IsCandidate() && d.Status == StatusStagedRollout && d.TrafficMode == TrafficPercentage
}

func (d Deployment) SupportsActiveCandidate() bool {
	return d.CanServe() && d.IsCandidate() && d.Status == StatusActive && d.TrafficMode == TrafficActive
}

func (d Deployment) MatchesFeatureFlags(flags map[string]bool) bool {
	key := strings.TrimSpace(d.FeatureFlagKey)
	if key == "" {
		return true
	}
	enabled, ok := flags[key]
	return ok && enabled
}

func (d Deployment) HasUserAllowlist(userID *uuid.UUID) bool {
	if userID == nil {
		return false
	}
	for _, id := range d.AllowlistedUserIDs {
		if id == *userID {
			return true
		}
	}
	return false
}

func (d Deployment) HasWorkspaceAllowlist(workspaceID *uuid.UUID) bool {
	if workspaceID == nil {
		return false
	}
	for _, id := range d.AllowlistedWorkspaceIDs {
		if id == *workspaceID {
			return true
		}
	}
	return false
}

type RoutingContext struct {
	RequestKey              string
	UserID                  *uuid.UUID
	WorkspaceID             *uuid.UUID
	TripID                  *uuid.UUID
	TaskType                string
	Environment             string
	AuthenticatedRole       string
	IsOpsUser               bool
	UserOptInExperimentalAI bool
	ForcedOpsTest           bool
	FeatureFlags            map[string]bool
	RequestTimestamp        time.Time
}

type RoutingDecision struct {
	PrimaryDeployment   Deployment     `json:"primaryDeployment"`
	ShadowDeployment    *Deployment    `json:"shadowDeployment,omitempty"`
	UserVisibleVariant  ModelVariant   `json:"userVisibleVariant"`
	AssignmentType      AssignmentType `json:"assignmentType"`
	DeterministicBucket *int           `json:"deterministicBucket,omitempty"`
	Reason              string         `json:"reason"`
	ShadowEligible      bool           `json:"shadowEligible"`
	ComparisonRequired  bool           `json:"comparisonRequired"`
	Metadata            map[string]any `json:"metadata,omitempty"`
}

type RequestAssignment struct {
	ID                    uuid.UUID
	RequestKey            string
	UserID                *uuid.UUID
	WorkspaceID           *uuid.UUID
	TripID                *uuid.UUID
	TaskType              string
	Environment           string
	BaselineDeploymentID  uuid.UUID
	CandidateDeploymentID *uuid.UUID
	AssignmentType        AssignmentType
	Bucket                *int
	CandidateUserVisible  bool
	CreatedAt             time.Time
}

type OnlineComparison struct {
	ID                       uuid.UUID
	RequestAssignmentID      uuid.UUID
	RequestKey               string
	TripID                   *uuid.UUID
	TaskType                 string
	BaselineDeploymentID     uuid.UUID
	CandidateDeploymentID    uuid.UUID
	BaselineResultStatus     string
	CandidateResultStatus    string
	BaselineMetrics          map[string]any
	CandidateMetrics         map[string]any
	MetricDeltas             map[string]any
	BaselineLatencyMS        *int
	CandidateLatencyMS       *int
	BaselineRepairAttempted  bool
	CandidateRepairAttempted bool
	BaselineErrorCode        string
	CandidateErrorCode       string
	GuardrailStatus          string
	ComparisonStatus         string
	CreatedAt                time.Time
	CompletedAt              *time.Time
}
