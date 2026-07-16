package triphealth

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/approvalrisk"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetconfidence"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

type Level string

const (
	LevelReady          Level = "ready"
	LevelAlmostReady    Level = "almost_ready"
	LevelNeedsAttention Level = "needs_attention"
	LevelNotReady       Level = "not_ready"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type Category string

const (
	CategoryItinerary     Category = "itinerary"
	CategoryRoute         Category = "route"
	CategoryTransport     Category = "transport"
	CategoryBudget        Category = "budget"
	CategoryAvailability  Category = "availability"
	CategoryCollaboration Category = "collaboration"
	CategoryChecklist     Category = "checklist"
	CategoryReminders     Category = "reminders"
	CategoryAccommodation Category = "accommodation"
	CategoryExpenses      Category = "expenses"
	CategoryPolicy        Category = "policy"
	CategoryApproval      Category = "approval"
	CategoryOffline       Category = "offline"
	CategoryDataQuality   Category = "data_quality"
	CategoryPublicShare   Category = "public_share"
	CategoryOther         Category = "other"
)

type Status string

const (
	StatusOpen     Status = "open"
	StatusResolved Status = "resolved"
	StatusIgnored  Status = "ignored"
)

type Action struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Href  string `json:"href"`
}

type Issue struct {
	ID             string         `json:"id"`
	Category       Category       `json:"category"`
	Severity       Severity       `json:"severity"`
	Status         Status         `json:"status"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Impact         string         `json:"impact,omitempty"`
	Recommendation string         `json:"recommendation,omitempty"`
	Action         *Action        `json:"action,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type CategorySummary struct {
	Category        Category `json:"category"`
	Score           int      `json:"score"`
	OpenIssueCount  int      `json:"openIssueCount"`
	HighestSeverity Severity `json:"highestSeverity"`
}

type TopFix struct {
	IssueID string `json:"issueId"`
	Label   string `json:"label"`
	Href    string `json:"href"`
}

type ComputedFrom struct {
	ItineraryRevision  int        `json:"itineraryRevision"`
	RouteUpdatedAt     *time.Time `json:"routeUpdatedAt,omitempty"`
	BudgetUpdatedAt    *time.Time `json:"budgetUpdatedAt,omitempty"`
	ChecklistUpdatedAt *time.Time `json:"checklistUpdatedAt,omitempty"`
	RemindersUpdatedAt *time.Time `json:"remindersUpdatedAt,omitempty"`
}

type Response struct {
	TripID       uuid.UUID         `json:"tripId"`
	Score        int               `json:"score"`
	Level        Level             `json:"level"`
	Summary      string            `json:"summary"`
	GeneratedAt  time.Time         `json:"generatedAt"`
	Categories   []CategorySummary `json:"categories"`
	Issues       []Issue           `json:"issues"`
	TopFixes     []TopFix          `json:"topFixes"`
	ComputedFrom ComputedFrom      `json:"computedFrom"`
	Debug        map[string]any    `json:"debug,omitempty"`
}

type Config struct {
	Enabled                         bool
	IncludeDebug                    bool
	LargeExpenseReceiptThreshold    float64
	DefaultMaxWalkingKmPerDay       float64
	DefaultMaxTransferMinutesPerDay int
}

func DefaultConfig() Config {
	return Config{
		Enabled:                         true,
		IncludeDebug:                    false,
		LargeExpenseReceiptThreshold:    100,
		DefaultMaxWalkingKmPerDay:       12,
		DefaultMaxTransferMinutesPerDay: 8 * 60,
	}
}

type Options struct {
	IncludeResolved bool
	IncludeDebug    bool
}

type ExpenseReceiptSignal struct {
	ExpenseID    uuid.UUID
	ReceiptCount int
}

type ReceiptOCRSignal struct {
	ReceiptID  uuid.UUID
	Confidence entity.ReceiptOCRConfidence
	Warnings   []string
}

type Snapshot struct {
	Trip                       *entity.Trip
	Itinerary                  aggregate.Itinerary
	Budget                     *budget.Summary
	BudgetConfidence           *budgetconfidence.Response
	BudgetLoadFailed           bool
	BudgetConfidenceLoadFailed bool
	Collaborators              []entity.TripCollaborator
	AvailabilityResponses      []entity.TripAvailabilityResponse
	Polls                      []entity.TripPoll
	Checklist                  *entity.TripChecklist
	Reminders                  []entity.TripReminder
	Expenses                   []entity.TripExpense
	Settlements                []entity.TripSettlement
	ExpenseReceiptSignals      []ExpenseReceiptSignal
	ReceiptOCRSignals          []ReceiptOCRSignal
	PolicyEvaluation           *workspacepolicies.Evaluation
	PolicyLoadFailed           bool
	ApprovalRisk               *approvalrisk.Response
	ApprovalRiskLoadFailed     bool
	Approval                   *entity.TripApprovalFields
	ApprovalLoadFailed         bool
	SubsystemFailures          []string
	Now                        time.Time
	Config                     Config
}
