package budgetconfidence

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type ConfidenceLevel string

const (
	LevelVeryLow  ConfidenceLevel = "very_low"
	LevelLow      ConfidenceLevel = "low"
	LevelMedium   ConfidenceLevel = "medium"
	LevelHigh     ConfidenceLevel = "high"
	LevelVeryHigh ConfidenceLevel = "very_high"
)

type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

type Category string

const (
	CategoryTransport     Category = "transport"
	CategoryAccommodation Category = "accommodation"
	CategoryActivities    Category = "activities"
	CategoryTickets       Category = "tickets"
	CategoryFood          Category = "food"
	CategoryShopping      Category = "shopping"
	CategoryFuel          Category = "fuel"
	CategoryParking       Category = "parking"
	CategoryTolls         Category = "tolls"
	CategoryGroceries     Category = "groceries"
	CategoryCamping       Category = "camping"
	CategoryHealthSafety  Category = "health_safety"
	CategoryOther         Category = "other"
)

type CoverageCategory string

const (
	CoverageTransport        CoverageCategory = "transport"
	CoverageAccommodation    CoverageCategory = "accommodation"
	CoverageActivities       CoverageCategory = "activities"
	CoverageFood             CoverageCategory = "food"
	CoverageShopping         CoverageCategory = "shopping"
	CoverageFuelParkingTolls CoverageCategory = "fuelParkingTolls"
	CoverageOther            CoverageCategory = "other"
)

type Source string

const (
	SourceActualReceiptExpense                    Source = "actual_receipt_expense"
	SourceActualManualExpense                     Source = "actual_manual_expense"
	SourceProviderPrice                           Source = "provider_price"
	SourceSelectedTransportOptionHighConfidence   Source = "selected_transport_option_high_confidence"
	SourceSelectedTransportOptionMediumConfidence Source = "selected_transport_option_medium_confidence"
	SourceSelectedTransportOptionLowConfidence    Source = "selected_transport_option_low_confidence"
	SourceManualEstimate                          Source = "manual_estimate"
	SourceAIEstimateHighConfidence                Source = "ai_estimate_high_confidence"
	SourceAIEstimateMediumConfidence              Source = "ai_estimate_medium_confidence"
	SourceAIEstimateLowConfidence                 Source = "ai_estimate_low_confidence"
	SourceMockEstimate                            Source = "mock_estimate"
	SourceMissingCost                             Source = "missing_cost"
	SourceUnknown                                 Source = "unknown_source"
)

type EntityType string

const (
	EntityItineraryItem           EntityType = "itinerary_item"
	EntityAccommodation           EntityType = "accommodation"
	EntityRouteLeg                EntityType = "route_leg"
	EntitySelectedTransportOption EntityType = "selected_transport_option"
	EntityExpense                 EntityType = "expense"
	EntityReceiptExpense          EntityType = "receipt_expense"
	EntityManualBudget            EntityType = "manual_budget"
	EntityWorkspaceBudget         EntityType = "workspace_budget"
	EntityUnknown                 EntityType = "unknown"
)

type IssueSeverity string

const (
	SeverityInfo     IssueSeverity = "info"
	SeverityWarning  IssueSeverity = "warning"
	SeverityHigh     IssueSeverity = "high"
	SeverityCritical IssueSeverity = "critical"
)

type IssueCategory string

const (
	IssueCategoryTransport          IssueCategory = "transport"
	IssueCategoryAccommodation      IssueCategory = "accommodation"
	IssueCategoryActivities         IssueCategory = "activities"
	IssueCategoryFood               IssueCategory = "food"
	IssueCategoryActualSpend        IssueCategory = "actual_spend"
	IssueCategoryCurrency           IssueCategory = "currency"
	IssueCategoryCoverage           IssueCategory = "coverage"
	IssueCategoryProviderConfidence IssueCategory = "provider_confidence"
	IssueCategoryBudgetLimit        IssueCategory = "budget_limit"
	IssueCategoryDataQuality        IssueCategory = "data_quality"
	IssueCategoryOther              IssueCategory = "other"
)

type RecommendationPriority string

const (
	PriorityLow    RecommendationPriority = "low"
	PriorityMedium RecommendationPriority = "medium"
	PriorityHigh   RecommendationPriority = "high"
)

type Money struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type Action struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

type Coverage struct {
	Overall          int  `json:"overall"`
	Transport        *int `json:"transport"`
	Accommodation    *int `json:"accommodation"`
	Activities       *int `json:"activities"`
	Food             *int `json:"food"`
	Shopping         *int `json:"shopping"`
	FuelParkingTolls *int `json:"fuelParkingTolls"`
	Other            *int `json:"other"`
}

type SourceQuality struct {
	Source       Source `json:"source"`
	ItemCount    int    `json:"itemCount"`
	TotalAmount  Money  `json:"totalAmount"`
	QualityScore int    `json:"qualityScore"`
	Reason       string `json:"reason,omitempty"`
}

type PlannedVsActual struct {
	OverallDifference        Money                       `json:"overallDifference"`
	OverallDifferencePercent *float64                    `json:"overallDifferencePercent"`
	Categories               []PlannedVsActualByCategory `json:"categories"`
}

type PlannedVsActualByCategory struct {
	Category          Category `json:"category"`
	Estimated         Money    `json:"estimated"`
	Actual            Money    `json:"actual"`
	DifferencePercent *float64 `json:"differencePercent"`
	Status            string   `json:"status"`
}

type Issue struct {
	ID             string        `json:"id"`
	Severity       IssueSeverity `json:"severity"`
	Category       IssueCategory `json:"category"`
	Title          string        `json:"title"`
	Description    string        `json:"description"`
	Recommendation string        `json:"recommendation"`
	Action         *Action       `json:"action,omitempty"`
}

type Recommendation struct {
	ID          string                 `json:"id"`
	Label       string                 `json:"label"`
	Description string                 `json:"description"`
	Href        string                 `json:"href"`
	Priority    RecommendationPriority `json:"priority"`
}

type Response struct {
	TripID          uuid.UUID        `json:"tripId"`
	Score           int              `json:"score"`
	Level           ConfidenceLevel  `json:"level"`
	RiskLevel       RiskLevel        `json:"riskLevel"`
	Summary         string           `json:"summary"`
	Currency        string           `json:"currency"`
	EstimatedTotal  Money            `json:"estimatedTotal"`
	ActualTotal     Money            `json:"actualTotal"`
	TripBudget      *Money           `json:"tripBudget"`
	Coverage        Coverage         `json:"coverage"`
	SourceQuality   []SourceQuality  `json:"sourceQuality"`
	PlannedVsActual PlannedVsActual  `json:"plannedVsActual"`
	Issues          []Issue          `json:"issues"`
	Recommendations []Recommendation `json:"recommendations"`
	Warnings        []string         `json:"warnings"`
	ComputedAt      time.Time        `json:"computedAt"`
	Debug           map[string]any   `json:"debug,omitempty"`
}

type Config struct {
	Enabled                         bool
	FailOpen                        bool
	LargeExpenseReceiptThreshold    float64
	ActualSpendHighThresholdPercent float64
	PlannedActualGapWarningPercent  float64
	PlannedActualGapHighPercent     float64
}

func DefaultConfig() Config {
	return Config{
		Enabled:                         true,
		FailOpen:                        true,
		LargeExpenseReceiptThreshold:    100,
		ActualSpendHighThresholdPercent: 80,
		PlannedActualGapWarningPercent:  20,
		PlannedActualGapHighPercent:     40,
	}
}

type Options struct {
	Currency     string
	IncludeDebug bool
}

type CurrencyConverter interface {
	Convert(ctx context.Context, amount float64, from string, to string) (*budget.CurrencyConversionResult, error)
}

type Input struct {
	Trip                    *entity.Trip
	Itinerary               aggregate.Itinerary
	BudgetSummary           *budget.Summary
	Expenses                []entity.TripExpense
	Receipts                []entity.TripExpenseReceipt
	ReceiptOCR              map[uuid.UUID]*entity.ReceiptOCRResult
	Converter               CurrencyConverter
	ConversionEnabled       bool
	ConversionFailOpen      bool
	Currency                string
	Now                     time.Time
	Config                  Config
	IncludeDebug            bool
	ExpenseLoadFailed       bool
	ReceiptLoadFailed       bool
	BudgetSummaryLoadFailed bool
	AdditionalWarnings      []string
}

type costRecord struct {
	ID               string
	EntityType       EntityType
	Category         Category
	Amount           *Money
	OriginalAmount   *Money
	Source           Source
	Confidence       string
	QualityScore     int
	IsActual         bool
	IsEstimate       bool
	Missing          bool
	DayNumber        *int
	ItemIndex        *int
	RouteLegID       string
	ExpenseID        uuid.UUID
	ReceiptBacked    bool
	ConversionFailed bool
	ConversionApprox bool
	Metadata         map[string]any
}
