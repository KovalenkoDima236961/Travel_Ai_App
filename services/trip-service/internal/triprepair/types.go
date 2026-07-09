package triprepair

import (
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/approvalrisk"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

type RepairMode string

const (
	RepairModePolicyCompliance       RepairMode = "policy_compliance"
	RepairModeReduceBudgetRisk       RepairMode = "reduce_budget_risk"
	RepairModeFixScheduleRisk        RepairMode = "fix_schedule_risk"
	RepairModeReduceWalking          RepairMode = "reduce_walking"
	RepairModeAddRestTime            RepairMode = "add_rest_time"
	RepairModeReplaceDisallowedItems RepairMode = "replace_disallowed_items"
	RepairModeSelectedIssues         RepairMode = "selected_issues"
)

type Constraints struct {
	PreserveConfirmedItems   bool `json:"preserveConfirmedItems"`
	MinimizeChanges          bool `json:"minimizeChanges"`
	PreserveUserEditedItems  bool `json:"preserveUserEditedItems"`
	DoNotChangeAccommodation bool `json:"doNotChangeAccommodation"`
	DoNotChangeDates         bool `json:"doNotChangeDates"`
	MaxChangedItems          *int `json:"maxChangedItems,omitempty"`
}

type JobPayload struct {
	RepairMode              RepairMode  `json:"repairMode"`
	SelectedIssueTypes      []string    `json:"selectedIssueTypes,omitempty"`
	SelectedRiskFactorTypes []string    `json:"selectedRiskFactorTypes,omitempty"`
	Constraints             Constraints `json:"constraints"`
	SpecialInstructions     string      `json:"specialInstructions,omitempty"`
}

type CreateJobRequest struct {
	ExpectedItineraryRevision *int        `json:"expectedItineraryRevision"`
	RepairMode                RepairMode  `json:"repairMode"`
	SelectedIssueTypes        []string    `json:"selectedIssueTypes"`
	SelectedRiskFactorTypes   []string    `json:"selectedRiskFactorTypes"`
	Constraints               Constraints `json:"constraints"`
	SpecialInstructions       string      `json:"specialInstructions"`
}

type ApplyRequest struct {
	ExpectedItineraryRevision *int `json:"expectedItineraryRevision"`
}

type DiscardRequest struct {
	Reason string `json:"reason"`
}

type Money struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type TripContext struct {
	Title        string `json:"title,omitempty"`
	Destination  string `json:"destination"`
	StartDate    string `json:"startDate,omitempty"`
	DurationDays int32  `json:"durationDays"`
	Budget       *Money `json:"budget,omitempty"`
	Travelers    int32  `json:"travelers"`
	Pace         string `json:"pace"`
}

type IssueAffected struct {
	DayNumber *int     `json:"dayNumber,omitempty"`
	ItemIndex *int     `json:"itemIndex,omitempty"`
	Name      string   `json:"name,omitempty"`
	Amount    *float64 `json:"amount,omitempty"`
	Currency  string   `json:"currency,omitempty"`
}

type Issue struct {
	Type     string         `json:"type"`
	Severity string         `json:"severity"`
	Message  string         `json:"message"`
	Affected *IssueAffected `json:"affected,omitempty"`
}

type Input struct {
	Trip             entity.Trip                     `json:"-"`
	CurrentItinerary aggregate.Itinerary             `json:"itinerary"`
	TripContext      TripContext                     `json:"tripContext"`
	Policy           *workspacepolicies.Policy       `json:"policy,omitempty"`
	PolicyEvaluation workspacepolicies.Evaluation    `json:"policyEvaluation"`
	ApprovalRisk     approvalrisk.Response           `json:"approvalRisk"`
	Issues           []Issue                         `json:"issues"`
	Constraints      JobPayload                      `json:"constraints"`
	UserProfile      *usercontext.UserProfile        `json:"userProfile,omitempty"`
	UserPreferences  *usercontext.UserPreferences    `json:"userPreferences,omitempty"`
	WeatherForecast  *weathercontext.WeatherForecast `json:"weatherContext,omitempty"`
}

type Summary struct {
	RepairMode          RepairMode `json:"repairMode"`
	ChangedItemCount    int        `json:"changedItemCount"`
	AddedItemCount      int        `json:"addedItemCount"`
	RemovedItemCount    int        `json:"removedItemCount"`
	MovedItemCount      int        `json:"movedItemCount"`
	EstimatedCostBefore *Money     `json:"estimatedCostBefore,omitempty"`
	EstimatedCostAfter  *Money     `json:"estimatedCostAfter,omitempty"`
	MajorChanges        []string   `json:"majorChanges"`
	IssuesAddressed     []string   `json:"issuesAddressed"`
	IssuesRemaining     []string   `json:"issuesRemaining"`
	Warnings            []string   `json:"warnings"`
}

type Change struct {
	Type         string         `json:"type"`
	DayNumber    *int           `json:"dayNumber,omitempty"`
	ItemIndex    *int           `json:"itemIndex,omitempty"`
	Before       map[string]any `json:"before,omitempty"`
	After        map[string]any `json:"after,omitempty"`
	FieldChanges []FieldChange  `json:"fieldChanges,omitempty"`
	Reason       string         `json:"reason,omitempty"`
}

type FieldChange struct {
	Field  string `json:"field"`
	Before any    `json:"before,omitempty"`
	After  any    `json:"after,omitempty"`
}

type Diff struct {
	DaysChanged   []Change `json:"daysChanged"`
	ItemsAdded    []Change `json:"itemsAdded"`
	ItemsRemoved  []Change `json:"itemsRemoved"`
	ItemsModified []Change `json:"itemsModified"`
	ItemsMoved    []Change `json:"itemsMoved"`
	Warnings      []string `json:"warnings,omitempty"`
}

type Validation struct {
	Valid    bool     `json:"valid"`
	Warnings []string `json:"warnings"`
}

type ProposalContent struct {
	RepairedItinerary aggregate.Itinerary `json:"repairedItinerary"`
	RepairSummary     Summary             `json:"repairSummary"`
	Changes           []Change            `json:"changes"`
	Diff              Diff                `json:"diff"`
	Validation        Validation          `json:"validation"`
}

type JobResultPayload struct {
	ProposalID uuid.UUID `json:"proposalId"`
}

func DecodeJobPayload(raw json.RawMessage) JobPayload {
	if len(raw) == 0 {
		return DefaultJobPayload(RepairModePolicyCompliance)
	}
	var payload JobPayload
	_ = json.Unmarshal(raw, &payload)
	payload.RepairMode = NormalizeRepairMode(payload.RepairMode)
	payload.SelectedIssueTypes = cleanStrings(payload.SelectedIssueTypes, 20)
	payload.SelectedRiskFactorTypes = cleanStrings(payload.SelectedRiskFactorTypes, 20)
	payload.SpecialInstructions = strings.TrimSpace(payload.SpecialInstructions)
	payload.Constraints = defaultConstraints(payload.Constraints)
	return payload
}

func DefaultJobPayload(mode RepairMode) JobPayload {
	return JobPayload{
		RepairMode: mode,
		Constraints: Constraints{
			PreserveConfirmedItems:  true,
			MinimizeChanges:         true,
			PreserveUserEditedItems: true,
			DoNotChangeDates:        true,
			MaxChangedItems:         intPtr(10),
		},
	}
}

func NormalizeRepairMode(mode RepairMode) RepairMode {
	normalized := RepairMode(strings.TrimSpace(strings.ToLower(string(mode))))
	if normalized == "" {
		return RepairModePolicyCompliance
	}
	return normalized
}

func ValidRepairMode(mode RepairMode) bool {
	switch NormalizeRepairMode(mode) {
	case RepairModePolicyCompliance,
		RepairModeReduceBudgetRisk,
		RepairModeFixScheduleRisk,
		RepairModeReduceWalking,
		RepairModeAddRestTime,
		RepairModeReplaceDisallowedItems,
		RepairModeSelectedIssues:
		return true
	default:
		return false
	}
}

func intPtr(value int) *int {
	return &value
}
