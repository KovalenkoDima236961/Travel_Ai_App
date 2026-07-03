package analytics

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
)

const (
	CategoryUnknown = "unknown"

	SourceUnknown = "unknown"

	ConfidenceUnknown = "unknown"

	InsightSeverityInfo     = "info"
	InsightSeverityWarning  = "warning"
	InsightSeverityCritical = "critical"

	ActionOptimizeBudget    = "optimize_budget"
	ActionCheckAvailability = "check_availability"
	ActionUpdatePrice       = "update_price"
	ActionOpenItem          = "open_item"
	ActionOpenTrip          = "open_trip"
	ActionExportReport      = "export_report"

	PlanningDisclaimer = "Costs are estimates for planning purposes only. Provider prices, availability, exchange rates, and booking costs may change."
)

type TripCostAnalytics struct {
	TripID                 uuid.UUID                `json:"tripId"`
	WorkspaceID            *uuid.UUID               `json:"workspaceId"`
	Currency               string                   `json:"currency"`
	GeneratedAt            time.Time                `json:"generatedAt"`
	Summary                CostAnalyticsSummary     `json:"summary"`
	ByDay                  []CostByDay              `json:"byDay"`
	ByCategory             []CostAmountBreakdown    `json:"byCategory"`
	BySource               []CostAmountBreakdown    `json:"bySource"`
	ByConfidence           []CostAmountBreakdown    `json:"byConfidence"`
	OriginalCurrencyTotals []OriginalCurrencyTotal  `json:"originalCurrencyTotals"`
	ExpensiveItems         []ExpensiveCostItem      `json:"expensiveItems"`
	Insights               []CostInsight            `json:"insights"`
	Warnings               []string                 `json:"warnings"`
	ExchangeRateInfo       *budget.ExchangeRateInfo `json:"exchangeRateInfo,omitempty"`
}

type WorkspaceCostAnalytics struct {
	WorkspaceID    uuid.UUID                 `json:"workspaceId"`
	Currency       string                    `json:"currency"`
	GeneratedAt    time.Time                 `json:"generatedAt"`
	DateRange      DateRange                 `json:"dateRange"`
	Summary        WorkspaceAnalyticsSummary `json:"summary"`
	ByTrip         []TripCostSummary         `json:"byTrip"`
	ByCategory     []CostAmountBreakdown     `json:"byCategory"`
	BySource       []CostAmountBreakdown     `json:"bySource"`
	ByMonth        []CostByMonth             `json:"byMonth"`
	ExpensiveTrips []TripCostSummary         `json:"expensiveTrips"`
	ExpensiveItems []ExpensiveCostItem       `json:"expensiveItems"`
	Insights       []CostInsight             `json:"insights"`
	Warnings       []string                  `json:"warnings"`
}

type DateRange struct {
	From *string `json:"from"`
	To   *string `json:"to"`
}

type CostAnalyticsSummary struct {
	BudgetAmount              *float64 `json:"budgetAmount"`
	EstimatedTotal            float64  `json:"estimatedTotal"`
	RemainingAmount           *float64 `json:"remainingAmount"`
	OverBudgetAmount          *float64 `json:"overBudgetAmount"`
	BudgetUtilizationPercent  *float64 `json:"budgetUtilizationPercent"`
	ItemEstimatedTotal        float64  `json:"itemEstimatedTotal"`
	AccommodationTotal        *float64 `json:"accommodationTotal"`
	MissingEstimateCount      int      `json:"missingEstimateCount"`
	UncertainEstimateCount    int      `json:"uncertainEstimateCount"`
	ConvertedItemCount        int      `json:"convertedItemCount"`
	UnconvertedItemCount      int      `json:"unconvertedItemCount"`
	IncompleteBudgetDataCount int      `json:"incompleteBudgetDataCount,omitempty"`
}

type WorkspaceAnalyticsSummary struct {
	TripCount                 int      `json:"tripCount"`
	EstimatedTotal            float64  `json:"estimatedTotal"`
	BudgetTotal               *float64 `json:"budgetTotal"`
	OverBudgetTripCount       int      `json:"overBudgetTripCount"`
	MissingEstimateCount      int      `json:"missingEstimateCount"`
	UncertainEstimateCount    int      `json:"uncertainEstimateCount"`
	ConvertedItemCount        int      `json:"convertedItemCount"`
	UnconvertedItemCount      int      `json:"unconvertedItemCount"`
	IncompleteBudgetTripCount int      `json:"incompleteBudgetTripCount"`
}

type CostByDay struct {
	DayNumber            int                 `json:"dayNumber"`
	Date                 *string             `json:"date"`
	EstimatedTotal       float64             `json:"estimatedTotal"`
	BudgetShare          *float64            `json:"budgetShare"`
	OverBudgetAmount     *float64            `json:"overBudgetAmount"`
	MissingEstimateCount int                 `json:"missingEstimateCount"`
	TopItems             []ExpensiveCostItem `json:"topItems"`
}

type CostAmountBreakdown struct {
	Name       string  `json:"name,omitempty"`
	Category   string  `json:"category,omitempty"`
	Source     string  `json:"source,omitempty"`
	Confidence string  `json:"confidence,omitempty"`
	Amount     float64 `json:"amount"`
	Percentage float64 `json:"percentage"`
	ItemCount  int     `json:"itemCount"`
}

type OriginalCurrencyTotal struct {
	Currency        string   `json:"currency"`
	Amount          float64  `json:"amount"`
	ConvertedAmount *float64 `json:"convertedAmount"`
}

type ExpensiveCostItem struct {
	TripID           *uuid.UUID `json:"tripId,omitempty"`
	TripTitle        string     `json:"tripTitle,omitempty"`
	Destination      string     `json:"destination,omitempty"`
	DayNumber        int        `json:"dayNumber,omitempty"`
	ItemIndex        int        `json:"itemIndex,omitempty"`
	Name             string     `json:"name"`
	Type             string     `json:"type"`
	Category         string     `json:"category"`
	Amount           float64    `json:"amount"`
	Currency         string     `json:"currency"`
	ConvertedAmount  *float64   `json:"convertedAmount"`
	Source           string     `json:"source"`
	Confidence       string     `json:"confidence"`
	PercentageOfTrip float64    `json:"percentageOfTrip"`
}

type TripCostSummary struct {
	TripID               uuid.UUID `json:"tripId"`
	Title                string    `json:"title"`
	Destination          string    `json:"destination"`
	StartDate            *string   `json:"startDate"`
	EndDate              *string   `json:"endDate"`
	BudgetAmount         *float64  `json:"budgetAmount"`
	EstimatedTotal       float64   `json:"estimatedTotal"`
	OverBudgetAmount     *float64  `json:"overBudgetAmount"`
	MissingEstimateCount int       `json:"missingEstimateCount"`
	WorkspaceID          uuid.UUID `json:"workspaceId"`
}

type CostByMonth struct {
	Month          string  `json:"month"`
	EstimatedTotal float64 `json:"estimatedTotal"`
	TripCount      int     `json:"tripCount"`
}

type CostInsight struct {
	Type     string             `json:"type"`
	Severity string             `json:"severity"`
	Title    string             `json:"title"`
	Message  string             `json:"message"`
	Action   *CostInsightAction `json:"action,omitempty"`
}

type CostInsightAction struct {
	Type      string     `json:"type"`
	TripID    *uuid.UUID `json:"tripId,omitempty"`
	DayNumber *int       `json:"dayNumber,omitempty"`
	ItemIndex *int       `json:"itemIndex,omitempty"`
}
