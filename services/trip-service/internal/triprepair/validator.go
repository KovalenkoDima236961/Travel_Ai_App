package triprepair

import (
	"encoding/json"
	"strings"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

const (
	maxSpecialInstructionsLength = 1000
	defaultMaxChangedItems       = 10
)

func (r CreateJobRequest) NormalizeAndPayload() (*JobPayload, json.RawMessage, error) {
	if r.ExpectedItineraryRevision == nil {
		return nil, nil, apperrs.ErrExpectedItineraryRevisionRequired
	}
	if *r.ExpectedItineraryRevision < 0 {
		return nil, nil, apperrs.NewInvalidInput("expectedItineraryRevision must be >= 0")
	}
	mode := NormalizeRepairMode(r.RepairMode)
	if !ValidRepairMode(mode) {
		return nil, nil, apperrs.NewInvalidInput("repairMode is invalid")
	}
	selectedIssueTypes := cleanStrings(r.SelectedIssueTypes, 20)
	selectedRiskFactorTypes := cleanStrings(r.SelectedRiskFactorTypes, 20)
	constraints := defaultConstraints(r.Constraints)
	if constraints.MaxChangedItems != nil && (*constraints.MaxChangedItems < 1 || *constraints.MaxChangedItems > 50) {
		return nil, nil, apperrs.NewInvalidInput("constraints.maxChangedItems must be between 1 and 50")
	}
	instructions := strings.TrimSpace(r.SpecialInstructions)
	if len(instructions) > maxSpecialInstructionsLength {
		return nil, nil, apperrs.NewInvalidInput("specialInstructions must be at most %d characters", maxSpecialInstructionsLength)
	}

	payload := &JobPayload{
		RepairMode:              mode,
		SelectedIssueTypes:      selectedIssueTypes,
		SelectedRiskFactorTypes: selectedRiskFactorTypes,
		Constraints:             constraints,
		SpecialInstructions:     instructions,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	return payload, raw, nil
}

func NormalizeProposalContent(
	content *ProposalContent,
	base aggregate.Itinerary,
	trip entity.Trip,
	payload JobPayload,
) error {
	if content == nil {
		return apperrs.NewDependencyError("repair proposal is required")
	}
	if strings.TrimSpace(string(content.RepairSummary.RepairMode)) == "" {
		content.RepairSummary.RepairMode = NormalizeRepairMode(payload.RepairMode)
	} else {
		content.RepairSummary.RepairMode = NormalizeRepairMode(content.RepairSummary.RepairMode)
	}
	if !ValidRepairMode(content.RepairSummary.RepairMode) {
		return apperrs.NewDependencyError("repair proposal mode is invalid")
	}
	if len(content.RepairedItinerary.Days) == 0 {
		return apperrs.NewDependencyError("repair proposal itinerary is required")
	}
	if payload.Constraints.DoNotChangeDates && len(content.RepairedItinerary.Days) != len(base.Days) {
		return apperrs.NewDependencyError("repair proposal changed itinerary duration")
	}
	for index := range content.RepairedItinerary.Days {
		day := &content.RepairedItinerary.Days[index]
		if day.Day < 1 {
			return apperrs.NewDependencyError("repair proposal day number is invalid")
		}
		if payload.Constraints.DoNotChangeDates && index < len(base.Days) && day.Day != base.Days[index].Day {
			return apperrs.NewDependencyError("repair proposal changed itinerary day numbers")
		}
		day.Title = strings.TrimSpace(day.Title)
		if day.Title == "" {
			return apperrs.NewDependencyError("repair proposal day title is required")
		}
		if len(day.Items) == 0 {
			return apperrs.NewDependencyError("repair proposal day items are required")
		}
		for itemIndex := range day.Items {
			item := &day.Items[itemIndex]
			item.Time = strings.TrimSpace(item.Time)
			item.Type = strings.TrimSpace(item.Type)
			item.Name = strings.TrimSpace(item.Name)
			item.Note = strings.TrimSpace(item.Note)
			if item.Time == "" || item.Type == "" || item.Name == "" {
				return apperrs.NewDependencyError("repair proposal item is invalid")
			}
			if item.EstimatedCost != nil && item.EstimatedCost.Amount != nil && *item.EstimatedCost.Amount < 0 {
				return apperrs.NewDependencyError("repair proposal item cost is invalid")
			}
		}
	}
	if payload.Constraints.DoNotChangeDates {
		if strings.TrimSpace(content.RepairedItinerary.Destination) == "" {
			content.RepairedItinerary.Destination = base.Destination
		}
		if strings.TrimSpace(content.RepairedItinerary.Destination) != "" &&
			strings.TrimSpace(base.Destination) != "" &&
			!strings.EqualFold(content.RepairedItinerary.Destination, base.Destination) {
			return apperrs.NewDependencyError("repair proposal changed destination")
		}
	}
	if strings.TrimSpace(content.RepairedItinerary.Destination) == "" {
		content.RepairedItinerary.Destination = trip.Destination
	}
	if content.RepairedItinerary.Travelers == 0 {
		content.RepairedItinerary.Travelers = trip.Travelers
	}
	if strings.TrimSpace(content.RepairedItinerary.Pace) == "" {
		content.RepairedItinerary.Pace = trip.Pace
	}
	if strings.TrimSpace(content.RepairedItinerary.Currency) == "" {
		content.RepairedItinerary.Currency = trip.BudgetCurrency
	}
	if content.RepairedItinerary.TotalBudget == nil {
		content.RepairedItinerary.TotalBudget = trip.BudgetAmount
	}
	if strings.TrimSpace(content.RepairedItinerary.Source) == "" {
		content.RepairedItinerary.Source = "ai_policy_repair"
	}
	content.Validation.Valid = true
	content.Validation.Warnings = cleanStrings(content.Validation.Warnings, 20)
	content.RepairSummary.MajorChanges = cleanStrings(content.RepairSummary.MajorChanges, 20)
	content.RepairSummary.IssuesAddressed = cleanStrings(content.RepairSummary.IssuesAddressed, 50)
	content.RepairSummary.IssuesRemaining = cleanStrings(content.RepairSummary.IssuesRemaining, 50)
	content.RepairSummary.Warnings = cleanStrings(content.RepairSummary.Warnings, 50)
	if len(content.RepairSummary.Warnings) == 0 {
		content.RepairSummary.Warnings = []string{"Availability and prices should be checked again after repair."}
	}
	content.Changes = capChanges(content.Changes, 100)
	content.Diff = BuildDiff(base, content.RepairedItinerary)
	return nil
}

func defaultConstraints(in Constraints) Constraints {
	out := in
	if out.MaxChangedItems == nil {
		out.MaxChangedItems = intPtr(defaultMaxChangedItems)
	}
	return out
}

func cleanStrings(values []string, limit int) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if len(out) >= limit {
			break
		}
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func capChanges(values []Change, limit int) []Change {
	if len(values) <= limit {
		return values
	}
	return values[:limit]
}
