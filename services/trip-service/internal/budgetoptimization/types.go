package budgetoptimization

import (
	"encoding/json"
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/planningconstraints"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

const (
	ScopeDay = "day"

	ChangeReplaceItem    = "replace_item"
	ChangeRemoveItem     = "remove_item"
	ChangeAddItem        = "add_item"
	ChangeModifyItemCost = "modify_item_cost"
	ChangeReorderItem    = "reorder_item"
	ChangeKeepItem       = "keep_item"

	ConfidenceLow    = "low"
	ConfidenceMedium = "medium"
	ConfidenceHigh   = "high"
)

type Constraints struct {
	PreserveMustSeeItems      bool     `json:"preserveMustSeeItems"`
	MaxWalkingIncreaseKm      *float64 `json:"maxWalkingIncreaseKm,omitempty"`
	KeepMealCount             bool     `json:"keepMealCount"`
	AvoidReplacingManualCosts bool     `json:"avoidReplacingManualCosts"`
}

type JobPayload struct {
	TargetReductionAmount *float64     `json:"targetReductionAmount,omitempty"`
	Currency              string       `json:"currency,omitempty"`
	Constraints           *Constraints `json:"constraints,omitempty"`
}

func DecodeJobPayload(raw json.RawMessage) JobPayload {
	if len(raw) == 0 {
		return JobPayload{}
	}
	var payload JobPayload
	_ = json.Unmarshal(raw, &payload)
	payload.Currency = strings.ToUpper(strings.TrimSpace(payload.Currency))
	return payload
}

type CreateJobRequest struct {
	Scope                     string       `json:"scope"`
	DayNumber                 *int         `json:"dayNumber"`
	TargetReductionAmount     *float64     `json:"targetReductionAmount"`
	Currency                  string       `json:"currency"`
	ExpectedItineraryRevision *int         `json:"expectedItineraryRevision"`
	Constraints               *Constraints `json:"constraints"`
	Instruction               *string      `json:"instruction"`
}

type ApplyRequest struct {
	ExpectedItineraryRevision *int `json:"expectedItineraryRevision"`
}

type ExpensiveItem struct {
	ItemIndex       int     `json:"itemIndex"`
	ItemName        string  `json:"itemName"`
	ItemType        string  `json:"itemType"`
	Amount          float64 `json:"amount"`
	Currency        string  `json:"currency"`
	Category        string  `json:"category,omitempty"`
	Source          string  `json:"source,omitempty"`
	Confidence      string  `json:"confidence,omitempty"`
	ShareOfDayTotal float64 `json:"shareOfDayTotal,omitempty"`
}

type BudgetContext struct {
	Currency              string          `json:"currency"`
	TripBudget            *float64        `json:"tripBudget,omitempty"`
	TripEstimatedTotal    float64         `json:"tripEstimatedTotal"`
	DayEstimatedTotal     float64         `json:"dayEstimatedTotal"`
	DailyBudgetShare      *float64        `json:"dailyBudgetShare,omitempty"`
	TargetReductionAmount float64         `json:"targetReductionAmount"`
	ExpensiveItems        []ExpensiveItem `json:"expensiveItems"`
}

type OptimizeDayInput struct {
	Trip                       entity.Trip                              `json:"-"`
	CurrentItinerary           aggregate.Itinerary                      `json:"currentItinerary"`
	DayNumber                  int                                      `json:"dayNumber"`
	CurrentDay                 aggregate.ItineraryDay                   `json:"currentDay"`
	BudgetSummary              budget.Summary                           `json:"budgetSummary"`
	BudgetContext              BudgetContext                            `json:"budgetContext"`
	Constraints                Constraints                              `json:"constraints"`
	Instruction                string                                   `json:"instruction,omitempty"`
	UserProfile                *usercontext.UserProfile                 `json:"userProfile,omitempty"`
	UserPreferences            *usercontext.UserPreferences             `json:"userPreferences,omitempty"`
	WeatherForecast            *weathercontext.WeatherForecast          `json:"weatherForecast,omitempty"`
	Accommodation              *aggregate.Accommodation                 `json:"accommodation,omitempty"`
	WorkspacePolicyConstraints *workspacepolicies.AIConstraints         `json:"workspacePolicyConstraints,omitempty"`
	PlanningConstraints        *planningconstraints.PlanningConstraints `json:"planningConstraints,omitempty"`
}

type ProposalContent struct {
	Summary                   string                 `json:"summary"`
	Scope                     string                 `json:"scope"`
	DayNumber                 int                    `json:"dayNumber"`
	Currency                  string                 `json:"currency"`
	BaseDayEstimatedTotal     float64                `json:"baseDayEstimatedTotal"`
	ProposedDayEstimatedTotal float64                `json:"proposedDayEstimatedTotal"`
	EstimatedSavingsAmount    float64                `json:"estimatedSavingsAmount"`
	Confidence                string                 `json:"confidence"`
	Changes                   []ProposalChange       `json:"changes"`
	PreservedItems            []PreservedItem        `json:"preservedItems"`
	Tradeoffs                 []string               `json:"tradeoffs"`
	Warnings                  []string               `json:"warnings"`
	ProposedDay               aggregate.ItineraryDay `json:"proposedDay"`
}

type ProposalChange struct {
	Type                   string   `json:"type"`
	OldItemIndex           *int     `json:"oldItemIndex,omitempty"`
	OldItemName            string   `json:"oldItemName,omitempty"`
	NewItemIndex           *int     `json:"newItemIndex,omitempty"`
	NewItemName            string   `json:"newItemName,omitempty"`
	Reason                 string   `json:"reason,omitempty"`
	EstimatedSavingsAmount *float64 `json:"estimatedSavingsAmount,omitempty"`
	Currency               string   `json:"currency,omitempty"`
}

type PreservedItem struct {
	ItemIndex int    `json:"itemIndex"`
	ItemName  string `json:"itemName"`
	Reason    string `json:"reason,omitempty"`
}
