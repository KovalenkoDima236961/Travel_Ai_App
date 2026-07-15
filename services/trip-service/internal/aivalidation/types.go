package aivalidation

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/planningconstraints"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

const ValidatorVersion = "v1"

type GenerationQualityStatus string

const (
	StatusNotValidated            GenerationQualityStatus = "not_validated"
	StatusValidated               GenerationQualityStatus = "validated"
	StatusValidatedWithWarnings   GenerationQualityStatus = "validated_with_warnings"
	StatusRepairedAndValidated    GenerationQualityStatus = "repaired_and_validated"
	StatusRepairedWithWarnings    GenerationQualityStatus = "repaired_with_warnings"
	StatusRepairFailed            GenerationQualityStatus = "repair_failed"
	StatusSchemaInvalid           GenerationQualityStatus = "schema_invalid"
	StatusBlockedByPolicy         GenerationQualityStatus = "blocked_by_policy"
	StatusBlockedByCriticalIssues GenerationQualityStatus = "blocked_by_critical_issues"
	StatusAIOutputInvalid         GenerationQualityStatus = "ai_output_invalid"
)

type IssueSeverity string

const (
	SeverityInfo     IssueSeverity = "info"
	SeverityWarning  IssueSeverity = "warning"
	SeverityHigh     IssueSeverity = "high"
	SeverityCritical IssueSeverity = "critical"
	SeverityBlocking IssueSeverity = "blocking"
)

type IssueCategory string

const (
	CategorySchema           IssueCategory = "schema"
	CategoryItinerary        IssueCategory = "itinerary"
	CategoryRoute            IssueCategory = "route"
	CategoryTransport        IssueCategory = "transport"
	CategoryTime             IssueCategory = "time"
	CategoryBudget           IssueCategory = "budget"
	CategoryPolicy           IssueCategory = "policy"
	CategoryWeather          IssueCategory = "weather"
	CategoryOpeningHours     IssueCategory = "opening_hours"
	CategoryPlace            IssueCategory = "place"
	CategoryAccommodation    IssueCategory = "accommodation"
	CategoryGroupPreferences IssueCategory = "group_preferences"
	CategoryDataQuality      IssueCategory = "data_quality"
	CategoryOther            IssueCategory = "other"
)

type IssueFixability string

const (
	FixableByAI   IssueFixability = "fixable_by_ai"
	FixableByUser IssueFixability = "fixable_by_user"
	NonBlocking   IssueFixability = "non_blocking"
	NotFixable    IssueFixability = "not_fixable"
)

type GenerationType string

const (
	GenerationTypeFullItinerary          GenerationType = "full_itinerary"
	GenerationTypeDayRegeneration        GenerationType = "day_regeneration"
	GenerationTypeItemRegeneration       GenerationType = "item_regeneration"
	GenerationTypeQualityImprovementDay  GenerationType = "quality_improvement_day"
	GenerationTypeQualityImprovementItem GenerationType = "quality_improvement_item"
	GenerationTypeTemplateAdaptation     GenerationType = "template_adaptation"
	GenerationTypePolicyRepair           GenerationType = "policy_repair"
	GenerationTypeBudgetOptimizationDay  GenerationType = "budget_optimization_day"
	GenerationTypeRouteAlternativeApply  GenerationType = "route_alternative_apply"
)

type RepairScopeType string

const (
	RepairScopeFullOutput     RepairScopeType = "full_output"
	RepairScopeDay            RepairScopeType = "day"
	RepairScopeItem           RepairScopeType = "item"
	RepairScopeRouteLeg       RepairScopeType = "route_leg"
	RepairScopeBudgetSection  RepairScopeType = "budget_section"
	RepairScopePolicyIssues   RepairScopeType = "policy_issues"
	RepairScopeSelectedIssues RepairScopeType = "selected_issues"
)

type MinimumSaveLevel string

const (
	MinimumSaveLevelNoBlockingIssues MinimumSaveLevel = "no_blocking_issues"
)

type ValidationIssue struct {
	ID          string          `json:"id"`
	Category    IssueCategory   `json:"category"`
	Severity    IssueSeverity   `json:"severity"`
	Title       string          `json:"title"`
	Description string          `json:"description,omitempty"`
	Fixability  IssueFixability `json:"fixability"`
	DayNumber   *int            `json:"dayNumber,omitempty"`
	ItemIndex   *int            `json:"itemIndex,omitempty"`
	RouteLegID  string          `json:"routeLegId,omitempty"`
	RuleKey     string          `json:"ruleKey,omitempty"`
}

type ValidationContext struct {
	ExpectedDayCount int  `json:"expectedDayCount,omitempty"`
	RepairAllowed    bool `json:"repairAllowed"`
}

type ValidationInput struct {
	GenerationType      GenerationType                           `json:"generationType"`
	Trip                entity.Trip                              `json:"-"`
	Itinerary           aggregate.Itinerary                      `json:"itinerary"`
	PlanningConstraints *planningconstraints.PlanningConstraints `json:"planningConstraints,omitempty"`
	BudgetSummary       *budget.Summary                          `json:"budgetSummary,omitempty"`
	PolicyEvaluation    *workspacepolicies.Evaluation            `json:"policyEvaluation,omitempty"`
	WeatherForecast     *weathercontext.WeatherForecast          `json:"weather,omitempty"`
	Context             ValidationContext                        `json:"context"`
}

type ValidationResult struct {
	Valid            bool                    `json:"valid"`
	SaveAllowed      bool                    `json:"saveAllowed"`
	Issues           []ValidationIssue       `json:"issues"`
	BlockingIssues   []ValidationIssue       `json:"blockingIssues"`
	RepairableIssues []ValidationIssue       `json:"repairableIssues"`
	Warnings         []string                `json:"warnings"`
	QualityStatus    GenerationQualityStatus `json:"qualityStatus"`
}

type RepairScope struct {
	Type       RepairScopeType `json:"type"`
	DayNumber  *int            `json:"dayNumber,omitempty"`
	ItemIndex  *int            `json:"itemIndex,omitempty"`
	RouteLegID string          `json:"routeLegId,omitempty"`
}

type RepairConstraints struct {
	PreserveUnaffectedDays  bool   `json:"preserveUnaffectedDays"`
	PreserveUserEditedItems bool   `json:"preserveUserEditedItems"`
	OutputLanguage          string `json:"outputLanguage"`
}

type RepairGenerationOutputRequest struct {
	GenerationType   GenerationType      `json:"generationType"`
	CurrentOutput    aggregate.Itinerary `json:"currentOutput"`
	ValidationIssues []ValidationIssue   `json:"validationIssues"`
	PlanningContext  PlanningContext     `json:"planningContext"`
	RepairScope      RepairScope         `json:"repairScope"`
	Constraints      RepairConstraints   `json:"constraints"`
}

type PlanningContext struct {
	Trip                entity.Trip                              `json:"trip"`
	Route               *aggregate.TripRoute                     `json:"route,omitempty"`
	Accommodation       *aggregate.Accommodation                 `json:"accommodation,omitempty"`
	PlanningConstraints *planningconstraints.PlanningConstraints `json:"planningConstraints,omitempty"`
	WeatherForecast     *weathercontext.WeatherForecast          `json:"weatherForecast,omitempty"`
	BudgetSummary       *budget.Summary                          `json:"budgetSummary,omitempty"`
	WorkspacePolicy     *workspacepolicies.Evaluation            `json:"workspacePolicy,omitempty"`
}

type RepairChange struct {
	Type        string         `json:"type"`
	Description string         `json:"description,omitempty"`
	DayNumber   *int           `json:"dayNumber,omitempty"`
	ItemIndex   *int           `json:"itemIndex,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type RepairGenerationOutputResponse struct {
	RepairedOutput aggregate.Itinerary `json:"repairedOutput"`
	ChangesMade    []RepairChange      `json:"changesMade"`
	Warnings       []string            `json:"warnings"`
}

type GenerationRepairAttempt struct {
	Attempt         int            `json:"attempt"`
	RepairScope     RepairScope    `json:"repairScope"`
	TargetIssueIDs  []string       `json:"targetIssueIds"`
	IssuesFixed     []string       `json:"issuesFixed"`
	IssuesRemaining []string       `json:"issuesRemaining"`
	DurationMS      int64          `json:"durationMs"`
	AIProviderMode  string         `json:"aiProviderMode,omitempty"`
	ChangesMade     []RepairChange `json:"changesMade,omitempty"`
	Warnings        []string       `json:"warnings,omitempty"`
}

type GenerationQualityMetadata struct {
	Status             GenerationQualityStatus   `json:"status"`
	ValidatedAt        time.Time                 `json:"validatedAt"`
	ValidatorVersion   string                    `json:"validatorVersion"`
	RepairAttempts     int                       `json:"repairAttempts"`
	MaxRepairAttempts  int                       `json:"maxRepairAttempts"`
	BlockingIssueCount int                       `json:"blockingIssueCount"`
	CriticalIssueCount int                       `json:"criticalIssueCount"`
	HighIssueCount     int                       `json:"highIssueCount"`
	WarningIssueCount  int                       `json:"warningIssueCount"`
	RemainingIssues    []ValidationIssue         `json:"remainingIssues"`
	RepairedIssues     []ValidationIssue         `json:"repairedIssues"`
	Warnings           []string                  `json:"warnings"`
	RepairAttemptLog   []GenerationRepairAttempt `json:"repairAttemptsLog,omitempty"`
}

type PipelineInput struct {
	GenerationType      GenerationType
	AIOutput            aggregate.Itinerary
	Trip                entity.Trip
	PlanningConstraints *planningconstraints.PlanningConstraints
	WeatherForecast     *weathercontext.WeatherForecast
	RepairAllowed       bool
	MaxRepairAttempts   int
	MinimumSaveLevel    MinimumSaveLevel
	OutputLanguage      string
	JobID               *uuid.UUID
}

type PipelineResult struct {
	FinalOutput       aggregate.Itinerary       `json:"finalOutput"`
	InitialValidation ValidationResult          `json:"initialValidation"`
	FinalValidation   ValidationResult          `json:"finalValidation"`
	RepairAttempts    int                       `json:"repairAttempts"`
	Repairs           []GenerationRepairAttempt `json:"repairs"`
	GenerationQuality GenerationQualityMetadata `json:"generationQuality"`
	SaveAllowed       bool                      `json:"saveAllowed"`
	UserMessage       string                    `json:"userMessage"`
}

type AIOutputValidator interface {
	Validate(ctx context.Context, input ValidationInput) (ValidationResult, error)
}

type AIRepairCoordinator interface {
	Repair(ctx context.Context, input RepairInput) (RepairResult, error)
}

type GenerationReliabilityPipeline interface {
	ValidateAndRepair(ctx context.Context, input PipelineInput) (PipelineResult, error)
}

type RepairInput struct {
	PipelineInput    PipelineInput
	CurrentOutput    aggregate.Itinerary
	ValidationResult ValidationResult
	Attempt          int
	PolicyEvaluation *workspacepolicies.Evaluation
	BudgetSummary    *budget.Summary
}

type RepairResult struct {
	Output   aggregate.Itinerary
	Attempt  GenerationRepairAttempt
	Warnings []string
}

type RepairClient interface {
	RepairGenerationOutput(ctx context.Context, request RepairGenerationOutputRequest) (*RepairGenerationOutputResponse, error)
	ProviderMode() string
}

type PolicyEvaluator interface {
	EvaluatePolicyForItinerary(ctx context.Context, trip entity.Trip, itinerary aggregate.Itinerary) (*workspacepolicies.Evaluation, error)
}

func (m GenerationQualityMetadata) ToMap() map[string]any {
	raw, err := json.Marshal(m)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func MetadataEnvelope(quality GenerationQualityMetadata) map[string]any {
	return map[string]any{"generationQuality": quality.ToMap()}
}
