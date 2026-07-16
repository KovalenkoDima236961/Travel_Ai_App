package approvalrisk

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/approvals"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

const MaxScore = 100

type RiskLevel string

const (
	RiskLevelLow           RiskLevel = "low"
	RiskLevelMedium        RiskLevel = "medium"
	RiskLevelHigh          RiskLevel = "high"
	RiskLevelCritical      RiskLevel = "critical"
	RiskLevelUnknown       RiskLevel = "unknown"
	RiskLevelNotApplicable RiskLevel = "not_applicable"
)

func RiskLevelFromScore(score int) RiskLevel {
	switch {
	case score < 25:
		return RiskLevelLow
	case score < 50:
		return RiskLevelMedium
	case score < 75:
		return RiskLevelHigh
	default:
		return RiskLevelCritical
	}
}

type FactorSeverity string

const (
	FactorSeverityLow      FactorSeverity = "low"
	FactorSeverityMedium   FactorSeverity = "medium"
	FactorSeverityHigh     FactorSeverity = "high"
	FactorSeverityCritical FactorSeverity = "critical"
)

type FactorSource string

const (
	SourceWorkspacePolicy   FactorSource = "workspace_policy"
	SourceApprovalChecklist FactorSource = "approval_checklist"
	SourceTripBudget        FactorSource = "trip_budget"
	SourceBudgetConfidence  FactorSource = "budget_confidence"
	SourceWorkspaceBudget   FactorSource = "workspace_budget"
	SourceCostAnalytics     FactorSource = "cost_analytics"
	SourceCostSplitting     FactorSource = "cost_splitting"
	SourceAvailability      FactorSource = "availability"
	SourceAIGeneration      FactorSource = "ai_generation"
	SourceTemplate          FactorSource = "template_adaptation"
	SourceItineraryQuality  FactorSource = "itinerary_quality"
	SourceWalkingDistance   FactorSource = "walking_distance"
	SourceSchedule          FactorSource = "schedule"
	SourceAccommodation     FactorSource = "accommodation"
	SourceRoute             FactorSource = "route"
)

type SuggestedActionPriority string

const (
	ActionPriorityLow    SuggestedActionPriority = "low"
	ActionPriorityMedium SuggestedActionPriority = "medium"
	ActionPriorityHigh   SuggestedActionPriority = "high"
)

type SuggestedAction struct {
	Type     string                  `json:"type"`
	Label    string                  `json:"label"`
	Priority SuggestedActionPriority `json:"priority,omitempty"`
	Target   SuggestedActionTarget   `json:"target,omitempty"`
}

type SuggestedActionTarget struct {
	TripID      *uuid.UUID `json:"tripId,omitempty"`
	WorkspaceID *uuid.UUID `json:"workspaceId,omitempty"`
	DayNumber   *int       `json:"dayNumber,omitempty"`
	ItemIndex   *int       `json:"itemIndex,omitempty"`
	Category    string     `json:"category,omitempty"`
}

type AffectedTarget struct {
	TripID        *uuid.UUID     `json:"tripId,omitempty"`
	DayNumber     *int           `json:"dayNumber,omitempty"`
	ItemIndex     *int           `json:"itemIndex,omitempty"`
	Category      string         `json:"category,omitempty"`
	AffectedCount int            `json:"affectedCount,omitempty"`
	AffectedItems []AffectedItem `json:"affectedItems,omitempty"`
}

type AffectedItem struct {
	DayNumber *int     `json:"dayNumber,omitempty"`
	ItemIndex *int     `json:"itemIndex,omitempty"`
	Name      string   `json:"name,omitempty"`
	Category  string   `json:"category,omitempty"`
	Amount    *float64 `json:"amount,omitempty"`
	Currency  string   `json:"currency,omitempty"`
}

type Factor struct {
	Type             string            `json:"type"`
	Severity         FactorSeverity    `json:"severity"`
	Points           int               `json:"points"`
	Title            string            `json:"title"`
	Message          string            `json:"message"`
	Source           FactorSource      `json:"source"`
	Affected         *AffectedTarget   `json:"affected,omitempty"`
	SuggestedActions []SuggestedAction `json:"suggestedActions,omitempty"`
}

type Summary struct {
	FactorCount                  int `json:"factorCount"`
	CriticalFactorCount          int `json:"criticalFactorCount"`
	HighFactorCount              int `json:"highFactorCount"`
	MediumFactorCount            int `json:"mediumFactorCount"`
	LowFactorCount               int `json:"lowFactorCount"`
	BlockingPolicyViolationCount int `json:"blockingPolicyViolationCount"`
	SuggestedActionCount         int `json:"suggestedActionCount"`
}

type Response struct {
	TripID              uuid.UUID         `json:"tripId"`
	WorkspaceID         *uuid.UUID        `json:"workspaceId"`
	Status              RiskLevel         `json:"status"`
	Score               *int              `json:"score"`
	MaxScore            int               `json:"maxScore"`
	GeneratedAt         time.Time         `json:"generatedAt"`
	Summary             Summary           `json:"summary"`
	Factors             []Factor          `json:"factors"`
	TopReasons          []string          `json:"topReasons"`
	SuggestedActions    []SuggestedAction `json:"suggestedActions"`
	Warnings            []string          `json:"warnings"`
	NotApplicableReason *string           `json:"notApplicableReason"`
}

type QueueSummary struct {
	Status     RiskLevel `json:"status"`
	Score      *int      `json:"score"`
	TopReasons []string  `json:"topReasons,omitempty"`
}

type WorkspaceBudgetSignal struct {
	Amount             float64
	Currency           string
	EstimatedTotal     float64
	OverBudgetAmount   float64
	UtilizationPercent float64
}

type BudgetConfidenceSignal struct {
	Score     int
	Level     string
	RiskLevel string
	TopIssues []string
}

type MetadataSignal struct {
	Source               string
	TemplateFallbackUsed bool
	TemplateWarningCount int
	ValidationRepairUsed bool
}

type TripContext struct {
	BudgetAmount   *float64
	BudgetCurrency string
	Days           int
	Accommodation  *aggregate.Accommodation
	Route          *aggregate.TripRoute
}

type Input struct {
	TripID                 uuid.UUID
	WorkspaceID            *uuid.UUID
	GeneratedAt            time.Time
	Trip                   TripContext
	ChecklistInput         approvals.ChecklistInput
	PolicyEvaluation       *workspacepolicies.Evaluation
	Itinerary              aggregate.Itinerary
	WorkspaceBudget        *WorkspaceBudgetSignal
	BudgetConfidence       *BudgetConfidenceSignal
	Metadata               MetadataSignal
	SignalUnavailableNames []string
}
