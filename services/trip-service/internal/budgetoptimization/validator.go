package budgetoptimization

import (
	"encoding/json"
	"regexp"
	"strings"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

const maxInstructionLength = 2000

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

func (r CreateJobRequest) NormalizeAndPayload() (*JobPayload, json.RawMessage, error) {
	scope := strings.TrimSpace(strings.ToLower(r.Scope))
	if scope == "" {
		scope = ScopeDay
	}
	if scope != ScopeDay {
		return nil, nil, apperrs.NewInvalidInput("scope must be day")
	}
	if r.DayNumber == nil || *r.DayNumber < 1 {
		return nil, nil, apperrs.NewInvalidInput("dayNumber is required and must be > 0")
	}
	if r.ExpectedItineraryRevision == nil {
		return nil, nil, apperrs.ErrExpectedItineraryRevisionRequired
	}
	if *r.ExpectedItineraryRevision < 0 {
		return nil, nil, apperrs.NewInvalidInput("expectedItineraryRevision must be >= 0")
	}
	if r.TargetReductionAmount != nil && *r.TargetReductionAmount < 0 {
		return nil, nil, apperrs.NewInvalidInput("targetReductionAmount must be >= 0")
	}
	currency := strings.ToUpper(strings.TrimSpace(r.Currency))
	if currency != "" && !currencyPattern.MatchString(currency) {
		return nil, nil, apperrs.NewInvalidInput("currency must be a 3-letter uppercase code")
	}
	if r.Constraints != nil && r.Constraints.MaxWalkingIncreaseKm != nil && *r.Constraints.MaxWalkingIncreaseKm < 0 {
		return nil, nil, apperrs.NewInvalidInput("constraints.maxWalkingIncreaseKm must be >= 0")
	}
	if r.Instruction != nil && len(strings.TrimSpace(*r.Instruction)) > maxInstructionLength {
		return nil, nil, apperrs.NewInvalidInput("instruction must be at most %d characters", maxInstructionLength)
	}

	payload := &JobPayload{
		TargetReductionAmount: r.TargetReductionAmount,
		Currency:              currency,
		Constraints:           r.Constraints,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	return payload, raw, nil
}

func NormalizeProposalContent(content *ProposalContent, dayNumber int, currency string) error {
	if content == nil {
		return apperrs.NewDependencyError("budget optimization proposal is required")
	}
	content.Summary = strings.TrimSpace(content.Summary)
	if content.Summary == "" {
		return apperrs.NewDependencyError("budget optimization proposal summary is required")
	}
	content.Scope = strings.TrimSpace(strings.ToLower(content.Scope))
	if content.Scope == "" {
		content.Scope = ScopeDay
	}
	if content.Scope != ScopeDay {
		return apperrs.NewDependencyError("budget optimization proposal scope is invalid")
	}
	if content.DayNumber == 0 {
		content.DayNumber = dayNumber
	}
	if content.DayNumber != dayNumber {
		return apperrs.NewDependencyError("budget optimization proposal day number is invalid")
	}
	content.Currency = strings.ToUpper(strings.TrimSpace(content.Currency))
	if content.Currency == "" {
		content.Currency = currency
	}
	if !currencyPattern.MatchString(content.Currency) {
		return apperrs.NewDependencyError("budget optimization proposal currency is invalid")
	}
	content.Confidence = strings.ToLower(strings.TrimSpace(content.Confidence))
	if content.Confidence == "" {
		content.Confidence = ConfidenceMedium
	}
	switch content.Confidence {
	case ConfidenceLow, ConfidenceMedium, ConfidenceHigh:
	default:
		return apperrs.NewDependencyError("budget optimization proposal confidence is invalid")
	}
	if content.BaseDayEstimatedTotal < 0 ||
		content.ProposedDayEstimatedTotal < 0 ||
		content.EstimatedSavingsAmount < 0 {
		return apperrs.NewDependencyError("budget optimization proposal amounts must be >= 0")
	}
	if len(content.Changes) == 0 || content.EstimatedSavingsAmount <= 0 {
		return apperrs.NewDependencyError("no_optimization_found")
	}
	if content.ProposedDayEstimatedTotal > content.BaseDayEstimatedTotal {
		return apperrs.NewDependencyError("budget optimization proposal does not reduce estimated cost")
	}

	day, err := NormalizeProposedDay(&content.ProposedDay, dayNumber)
	if err != nil {
		return err
	}
	content.ProposedDay = day
	for index := range content.Changes {
		change := &content.Changes[index]
		change.Type = strings.TrimSpace(strings.ToLower(change.Type))
		if !validChangeType(change.Type) {
			return apperrs.NewDependencyError("budget optimization proposal change type is invalid")
		}
		change.OldItemName = strings.TrimSpace(change.OldItemName)
		change.NewItemName = strings.TrimSpace(change.NewItemName)
		change.Reason = strings.TrimSpace(change.Reason)
		change.Currency = strings.ToUpper(strings.TrimSpace(change.Currency))
		if change.Currency == "" {
			change.Currency = content.Currency
		}
		if !currencyPattern.MatchString(change.Currency) {
			return apperrs.NewDependencyError("budget optimization proposal change currency is invalid")
		}
		if change.EstimatedSavingsAmount != nil && *change.EstimatedSavingsAmount < 0 {
			return apperrs.NewDependencyError("budget optimization proposal change savings must be >= 0")
		}
	}
	return nil
}

func NormalizeProposedDay(day *aggregate.ItineraryDay, dayNumber int) (aggregate.ItineraryDay, error) {
	if day == nil {
		return aggregate.ItineraryDay{}, apperrs.NewDependencyError("proposedDay is required")
	}
	normalized := *day
	normalized.Day = dayNumber
	normalized.Title = strings.TrimSpace(normalized.Title)
	if normalized.Title == "" {
		return aggregate.ItineraryDay{}, apperrs.NewDependencyError("proposedDay.title is required")
	}
	if len(normalized.Items) == 0 || len(normalized.Items) > 30 {
		return aggregate.ItineraryDay{}, apperrs.NewDependencyError("proposedDay.items count is invalid")
	}
	for index := range normalized.Items {
		item := &normalized.Items[index]
		item.Time = strings.TrimSpace(item.Time)
		item.Type = strings.TrimSpace(item.Type)
		item.Name = strings.TrimSpace(item.Name)
		item.Note = strings.TrimSpace(item.Note)
		if item.Time == "" || item.Type == "" || item.Name == "" {
			return aggregate.ItineraryDay{}, apperrs.NewDependencyError("proposedDay item fields are required")
		}
		if err := budget.NormalizeEstimatedCost(item.EstimatedCost, budget.SourceAI); err != nil {
			return aggregate.ItineraryDay{}, apperrs.NewDependencyError("proposedDay estimatedCost is invalid: %s", err.Error())
		}
	}
	return normalized, nil
}

func validChangeType(value string) bool {
	switch value {
	case ChangeReplaceItem, ChangeRemoveItem, ChangeAddItem,
		ChangeModifyItemCost, ChangeReorderItem, ChangeKeepItem:
		return true
	default:
		return false
	}
}
