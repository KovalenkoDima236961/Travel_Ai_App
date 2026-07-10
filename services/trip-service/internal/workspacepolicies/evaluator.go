package workspacepolicies

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/analytics"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

const (
	maxAffectedItems                = 10
	availabilityConfidenceThreshold = 0.65
)

var ticketedTypes = tokenSet([]string{
	"ticket", "attraction", "museum", "tour", "activity", "event", "concert",
	"show", "theme_park", "guided_tour",
})

var lateExemptTypes = tokenSet([]string{
	"accommodation", "transport", "check_in", "checkin", "checkout", "check_out",
})

var restTypes = tokenSet([]string{
	"rest", "break", "free_time", "freetime", "leisure", "hotel_rest",
})

var transportTypes = tokenSet([]string{
	"transport", "transfer", "public_transport", "walking", "walk", "train", "bus", "metro",
	"taxi", "rideshare", "car", "rental_car", "flight", "bike", "ferry", "boat", "hiking",
})

type EvaluationInput struct {
	Trip                *entity.Trip
	Policy              *Policy
	Itinerary           aggregate.Itinerary
	AnalyticsByCurrency map[string]analytics.TripCostAnalytics
	CostSplitting       *CostSplittingSnapshot
	Converter           budget.CurrencyConverter
	ConversionEnabled   bool
}

type CostSplittingSnapshot struct {
	Currency          string
	TravelerCount     int
	UnassignedTotal   float64
	DefaultSplitCount int
	InvalidSplitCount int
}

func NotApplicableEvaluation(
	tripID uuid.UUID,
	workspaceID *uuid.UUID,
	reason string,
) Evaluation {
	return Evaluation{
		TripID:              tripID,
		WorkspaceID:         workspaceID,
		Status:              EvaluationNotApplicable,
		GeneratedAt:         time.Now().UTC(),
		Results:             []EvaluationResult{},
		Warnings:            []string{},
		NotApplicableReason: &reason,
	}
}

func Evaluate(ctx context.Context, in EvaluationInput) Evaluation {
	if in.Trip == nil {
		return NotApplicableEvaluation(uuid.Nil, nil, "trip_not_found")
	}
	if in.Trip.WorkspaceID == nil {
		return NotApplicableEvaluation(in.Trip.ID, nil, "personal_trip")
	}
	if in.Policy == nil {
		return NotApplicableEvaluation(in.Trip.ID, in.Trip.WorkspaceID, "no_active_policy")
	}
	policyID := in.Policy.ID
	result := Evaluation{
		TripID:      in.Trip.ID,
		WorkspaceID: in.Trip.WorkspaceID,
		PolicyID:    &policyID,
		Status:      EvaluationOK,
		GeneratedAt: time.Now().UTC(),
		Results:     make([]EvaluationResult, 0, 13),
		Warnings:    []string{},
	}
	rules := in.Policy.Rules.Rules

	if rules.RequireTripBudget.Enabled {
		item := pass("requireTripBudget", rules.RequireTripBudget.Severity,
			"Trip budget is configured", "The trip has a budget amount and currency.")
		if in.Trip.BudgetAmount == nil || strings.TrimSpace(in.Trip.BudgetCurrency) == "" {
			item = violation("requireTripBudget", rules.RequireTripBudget.Severity,
				"Trip budget is required",
				"Set a trip budget amount and currency before approval.",
				nil, map[string]any{"budget": "configured"},
				action("set_trip_budget", "Set trip budget", nil, nil))
		}
		result.add(item)
	}

	if rules.MaxTripBudget.Enabled {
		rule := rules.MaxTripBudget
		costs, ok := analyticsFor(in, rule.Currency)
		if !ok {
			result.add(unknown("maxTripBudget", rule.Severity, "Maximum trip budget",
				"Not enough data to evaluate this rule."))
		} else {
			result.Warnings = appendUnique(result.Warnings, costs.Warnings...)
			item := pass("maxTripBudget", rule.Severity,
				"Trip is within the workspace limit", "Estimated trip cost is within the limit.")
			if costs.Summary.EstimatedTotal > rule.Amount {
				item = violation("maxTripBudget", rule.Severity,
					"Trip exceeds maximum budget",
					fmt.Sprintf("Estimated trip cost is %.2f %s, above the %.2f %s workspace limit.",
						costs.Summary.EstimatedTotal, rule.Currency, rule.Amount, rule.Currency),
					money(costs.Summary.EstimatedTotal, rule.Currency),
					money(rule.Amount, rule.Currency),
					action("open_budget_optimization", "Optimize budget", nil, nil),
					action("open_trip_analytics", "Open trip analytics", nil, nil))
			}
			result.add(item)
		}
	}

	if rules.MaxDailyBudget.Enabled {
		rule := rules.MaxDailyBudget
		costs, ok := analyticsFor(in, rule.Currency)
		if !ok {
			result.add(unknown("maxDailyBudget", rule.Severity, "Maximum daily budget",
				"Not enough data to evaluate this rule."))
		} else {
			affected := make([]AffectedItem, 0)
			for _, day := range costs.ByDay {
				if day.EstimatedTotal > rule.Amount {
					dayNumber := day.DayNumber
					amount := day.EstimatedTotal
					affected = appendLimited(affected, AffectedItem{
						DayNumber: &dayNumber, Amount: &amount, Currency: rule.Currency,
					})
				}
			}
			item := pass("maxDailyBudget", rule.Severity,
				"Daily costs are within the workspace limit", "Every day is within the daily limit.")
			if len(affected) > 0 {
				item = violation("maxDailyBudget", rule.Severity,
					"One or more days exceed the daily budget",
					fmt.Sprintf("%d day(s) exceed %.2f %s.", len(affected), rule.Amount, rule.Currency),
					map[string]any{"days": affected}, money(rule.Amount, rule.Currency),
					action("optimize_day_budget", "Optimize day budget", affected[0].DayNumber, nil))
				item.AffectedItems = affected
			}
			result.add(item)
		}
	}

	if rules.MaxItemCost.Enabled {
		result.add(evaluateMaxItemCost(ctx, in, rules.MaxItemCost, &result))
	}
	if rules.MaxAccommodationTotal.Enabled {
		result.add(evaluateAccommodation(ctx, in, "maxAccommodationTotal",
			rules.MaxAccommodationTotal, false, &result))
	}
	if rules.MaxAccommodationPerNight.Enabled {
		result.add(evaluateAccommodation(ctx, in, "maxAccommodationPerNight",
			rules.MaxAccommodationPerNight, true, &result))
	}

	if rules.RequireCostSplitting.Enabled {
		rule := rules.RequireCostSplitting
		if in.CostSplitting == nil {
			result.add(unknown("requireCostSplitting", rule.Severity, "Cost splitting",
				"Not enough data to evaluate this rule."))
		} else {
			summary := in.CostSplitting
			item := pass("requireCostSplitting", rule.Severity,
				"Cost splitting is configured", "Estimated costs are allocated to travelers.")
			switch {
			case summary.TravelerCount == 0:
				item = violation("requireCostSplitting", rule.Severity,
					"Travelers are required for cost splitting",
					"Add travelers before submitting for approval.", nil, nil,
					action("open_cost_splitting", "Configure cost splitting", nil, nil))
			case summary.InvalidSplitCount > 0 || summary.UnassignedTotal > 0:
				item = violation("requireCostSplitting", rule.Severity,
					"Cost splitting is incomplete",
					fmt.Sprintf("%d invalid split(s) and %.2f %s unassigned.",
						summary.InvalidSplitCount, summary.UnassignedTotal, summary.Currency),
					map[string]any{
						"invalidSplitCount": summary.InvalidSplitCount,
						"unassignedTotal":   summary.UnassignedTotal,
						"currency":          summary.Currency,
					}, nil, action("open_cost_splitting", "Review cost splitting", nil, nil))
			case summary.DefaultSplitCount > 0:
				item = unknown("requireCostSplitting", rule.Severity,
					"Some costs use the default split",
					fmt.Sprintf("%d cost(s) use the default equal split.", summary.DefaultSplitCount))
				item.SuggestedActions = []SuggestedAction{
					action("open_cost_splitting", "Review cost splitting", nil, nil),
				}
			}
			result.add(item)
		}
	}

	if rules.RequireAvailabilityForTicketedItems.Enabled {
		rule := rules.RequireAvailabilityForTicketedItems
		affected := make([]AffectedItem, 0)
		unavailable, lowConfidence := 0, 0
		for _, day := range in.Itinerary.Days {
			for itemIndex, item := range day.Items {
				if !isTicketed(item) {
					continue
				}
				check := item.AvailabilityCheck
				if check == nil || strings.TrimSpace(check.CheckedAt) == "" {
					dayNumber, index := day.Day, itemIndex
					affected = appendLimited(affected, AffectedItem{
						DayNumber: &dayNumber, ItemIndex: &index, Name: item.Name,
					})
					continue
				}
				if normalizeToken(check.Status) == "unavailable" {
					unavailable++
				}
				if check.MatchConfidence > 0 &&
					check.MatchConfidence < availabilityConfidenceThreshold {
					lowConfidence++
				}
			}
		}
		item := pass("requireAvailabilityForTicketedItems", rule.Severity,
			"Ticketed-item availability is checked", "Ticketed items have availability checks.")
		if len(affected) > 0 || unavailable > 0 || lowConfidence > 0 {
			item = violation("requireAvailabilityForTicketedItems", rule.Severity,
				"Ticketed items need availability review",
				fmt.Sprintf("%d unchecked, %d unavailable, and %d low-confidence item(s).",
					len(affected), unavailable, lowConfidence),
				map[string]any{
					"uncheckedCount":     len(affected),
					"unavailableCount":   unavailable,
					"lowConfidenceCount": lowConfidence,
				}, map[string]any{"availabilityCheck": "verified"},
				action("check_availability", "Check availability", nil, nil))
			item.AffectedItems = affected
		}
		result.add(item)
	}

	if rules.MaxWalkingKmPerDay.Enabled {
		rule := rules.MaxWalkingKmPerDay
		affected := make([]AffectedItem, 0)
		hasEstimate := false
		for _, day := range in.Itinerary.Days {
			total := 0.0
			for _, item := range day.Items {
				if item.WalkingDistanceKm != nil && *item.WalkingDistanceKm >= 0 {
					hasEstimate = true
					total += *item.WalkingDistanceKm
				}
			}
			if total > rule.Km {
				dayNumber, amount := day.Day, round2(total)
				affected = appendLimited(affected, AffectedItem{
					DayNumber: &dayNumber, Amount: &amount, Currency: "km",
				})
			}
		}
		switch {
		case !hasEstimate:
			result.add(unknown("maxWalkingKmPerDay", rule.Severity, "Walking distance",
				"Walking distance not available. Not enough data to evaluate this rule."))
		case len(affected) > 0:
			item := violation("maxWalkingKmPerDay", rule.Severity,
				"Daily walking limit exceeded",
				fmt.Sprintf("%d day(s) exceed %.2f km of walking.", len(affected), rule.Km),
				map[string]any{"days": affected}, map[string]any{"km": rule.Km},
				action("optimize_route", "Optimize route", affected[0].DayNumber, nil))
			item.AffectedItems = affected
			result.add(item)
		default:
			result.add(pass("maxWalkingKmPerDay", rule.Severity,
				"Walking distance is within the limit", "Estimated walking is within the daily limit."))
		}
	}

	if rules.NoLateActivitiesAfter.Enabled {
		result.add(evaluateLateActivities(in.Itinerary, rules.NoLateActivitiesAfter))
	}
	if rules.RequiredRestTimePerDay.Enabled {
		result.add(evaluateRestTime(in.Itinerary, rules.RequiredRestTimePerDay))
	}
	if rules.PreferredTransportModes.Enabled {
		result.add(evaluateTransport(in.Itinerary, rules.PreferredTransportModes))
	}
	if rules.MaxTransferHoursPerDay.Enabled {
		result.add(evaluateMaxTransferHours(in.Itinerary, rules.MaxTransferHoursPerDay))
	}
	if rules.DisallowedTransportModes.Enabled {
		result.add(evaluateDisallowedTransport(in.Itinerary, rules.DisallowedTransportModes))
	}
	if rules.DisallowedActivityTypes.Enabled {
		result.add(evaluateDisallowed(in.Itinerary, rules.DisallowedActivityTypes))
	}

	result.finish()
	return result
}

func evaluateMaxItemCost(
	ctx context.Context,
	in EvaluationInput,
	rule ItemCostRule,
	evaluation *Evaluation,
) EvaluationResult {
	categories := tokenSet(rule.Categories)
	affected := make([]AffectedItem, 0)
	comparable := 0
	for _, day := range in.Itinerary.Days {
		for itemIndex, item := range day.Items {
			if item.EstimatedCost == nil || item.EstimatedCost.Amount == nil {
				continue
			}
			category := itemCategory(item)
			if len(categories) > 0 {
				if _, ok := categories[category]; !ok {
					continue
				}
			}
			amount, ok := convertCost(ctx, *item.EstimatedCost.Amount,
				item.EstimatedCost.Currency, in.Itinerary.Currency, rule.Currency, in)
			if !ok {
				evaluation.Warnings = appendUnique(evaluation.Warnings,
					fmt.Sprintf("Cost for %q could not be converted to %s.", item.Name, rule.Currency))
				continue
			}
			comparable++
			if amount > rule.Amount {
				dayNumber, index, rounded := day.Day, itemIndex, round2(amount)
				affected = appendLimited(affected, AffectedItem{
					DayNumber: &dayNumber, ItemIndex: &index, Name: item.Name,
					Amount: &rounded, Currency: rule.Currency,
				})
			}
		}
	}
	if comparable == 0 {
		return unknown("maxItemCost", rule.Severity, "Maximum item cost",
			"Not enough data to evaluate this rule.")
	}
	if len(affected) == 0 {
		return pass("maxItemCost", rule.Severity,
			"Item costs are within the limit", "Comparable item costs are within the limit.")
	}
	item := violation("maxItemCost", rule.Severity,
		"One or more items exceed the cost limit",
		fmt.Sprintf("%d item(s) exceed %.2f %s.", len(affected), rule.Amount, rule.Currency),
		map[string]any{"items": affected}, money(rule.Amount, rule.Currency),
		action("open_item", "Open item", affected[0].DayNumber, affected[0].ItemIndex),
		action("update_price", "Update price", affected[0].DayNumber, affected[0].ItemIndex),
		action("replace_item", "Replace item", affected[0].DayNumber, affected[0].ItemIndex))
	item.AffectedItems = affected
	return item
}

func evaluateAccommodation(
	ctx context.Context,
	in EvaluationInput,
	key string,
	rule MoneyRule,
	perNight bool,
	evaluation *Evaluation,
) EvaluationResult {
	accommodation := in.Trip.Accommodation
	if accommodation == nil || accommodation.EstimatedCost == nil ||
		accommodation.EstimatedCost.Amount == nil {
		return unknown(key, rule.Severity, "Accommodation cost",
			"Not enough data to evaluate this rule.")
	}
	amount, ok := convertCost(ctx, *accommodation.EstimatedCost.Amount,
		accommodation.EstimatedCost.Currency, in.Itinerary.Currency, rule.Currency, in)
	if !ok {
		evaluation.Warnings = appendUnique(evaluation.Warnings,
			"Accommodation cost could not be converted to "+rule.Currency+".")
		return unknown(key, rule.Severity, "Accommodation cost",
			"Not enough data to evaluate this rule.")
	}
	label := "Accommodation total"
	if perNight {
		nights, ok := accommodationNights(accommodation)
		if !ok {
			return unknown(key, rule.Severity, "Accommodation cost per night",
				"Not enough data to evaluate this rule.")
		}
		amount /= float64(nights)
		label = "Accommodation cost per night"
	}
	if amount <= rule.Amount {
		return pass(key, rule.Severity, label+" is within the limit",
			label+" is within the workspace limit.")
	}
	return violation(key, rule.Severity, label+" exceeds the limit",
		fmt.Sprintf("%s is %.2f %s, above the %.2f %s limit.",
			label, amount, rule.Currency, rule.Amount, rule.Currency),
		money(round2(amount), rule.Currency), money(rule.Amount, rule.Currency),
		action("open_accommodation", "Open accommodation", nil, nil))
}

func evaluateLateActivities(
	itinerary aggregate.Itinerary,
	rule LateActivityRule,
) EvaluationResult {
	limit, _ := minutesFromHHMM(rule.Time)
	affected := make([]AffectedItem, 0)
	evaluable := 0
	for _, day := range itinerary.Days {
		for itemIndex, item := range day.Items {
			if _, exempt := lateExemptTypes[normalizeToken(item.Type)]; exempt {
				continue
			}
			value := item.EndTime
			if value == "" {
				value = item.Time
			}
			minutes, ok := minutesFromHHMM(value)
			if !ok {
				continue
			}
			evaluable++
			if minutes > limit {
				dayNumber, index := day.Day, itemIndex
				affected = appendLimited(affected, AffectedItem{
					DayNumber: &dayNumber, ItemIndex: &index, Name: item.Name,
				})
			}
		}
	}
	if evaluable == 0 {
		return unknown("noLateActivitiesAfter", rule.Severity, "Late activities",
			"Not enough data to evaluate this rule.")
	}
	if len(affected) == 0 {
		return pass("noLateActivitiesAfter", rule.Severity,
			"No activities are scheduled too late", "Activities finish by "+rule.Time+".")
	}
	item := violation("noLateActivitiesAfter", rule.Severity,
		"Activities are scheduled too late",
		fmt.Sprintf("%d item(s) are scheduled after %s.", len(affected), rule.Time),
		map[string]any{"items": affected}, map[string]any{"latestTime": rule.Time},
		action("open_item", "Open item", affected[0].DayNumber, affected[0].ItemIndex),
		action("regenerate_day", "Regenerate day", affected[0].DayNumber, nil))
	item.AffectedItems = affected
	return item
}

func evaluateRestTime(itinerary aggregate.Itinerary, rule RestTimeRule) EvaluationResult {
	affected := make([]AffectedItem, 0)
	hasAnyDuration := false
	for _, day := range itinerary.Days {
		total := 0
		for _, item := range day.Items {
			if _, ok := restTypes[normalizeToken(item.Type)]; !ok {
				continue
			}
			duration, ok := itemDurationMinutes(item)
			if ok {
				hasAnyDuration = true
				total += duration
			}
		}
		if total < rule.Minutes {
			dayNumber := day.Day
			amount := float64(total)
			affected = appendLimited(affected, AffectedItem{DayNumber: &dayNumber, Amount: &amount})
		}
	}
	if !hasAnyDuration && rule.Minutes > 0 {
		return unknown("requiredRestTimePerDay", rule.Severity, "Required rest time",
			"Not enough data to evaluate this rule.")
	}
	if len(affected) == 0 {
		return pass("requiredRestTimePerDay", rule.Severity,
			"Daily rest time is included", "Every day includes the required rest time.")
	}
	item := violation("requiredRestTimePerDay", rule.Severity,
		"Some days need more rest time",
		fmt.Sprintf("%d day(s) include less than %d minutes of rest.", len(affected), rule.Minutes),
		map[string]any{"days": affected}, map[string]any{"minutes": rule.Minutes},
		action("regenerate_day", "Regenerate day", affected[0].DayNumber, nil),
		action("add_rest_block", "Add rest block", affected[0].DayNumber, nil))
	item.AffectedItems = affected
	return item
}

func evaluateTransport(itinerary aggregate.Itinerary, rule TransportRule) EvaluationResult {
	preferred := tokenSet(rule.Modes)
	if len(preferred) == 0 {
		return unknown("preferredTransportModes", rule.Severity, "Preferred transport",
			"Not enough data to evaluate this rule.")
	}
	affected := make([]AffectedItem, 0)
	transportCount := 0
	for _, day := range itinerary.Days {
		for itemIndex, item := range day.Items {
			itemType := normalizeToken(item.Type)
			mode := transportModeForItem(item)
			if _, isTransport := transportTypes[itemType]; !isTransport {
				if _, isMode := transportTypes[mode]; !isMode {
					continue
				}
			}
			transportCount++
			if _, ok := preferred[mode]; !ok {
				dayNumber, index := day.Day, itemIndex
				affected = appendLimited(affected, AffectedItem{
					DayNumber: &dayNumber, ItemIndex: &index, Name: item.Name,
				})
			}
		}
	}
	if transportCount == 0 {
		return pass("preferredTransportModes", rule.Severity,
			"No transport conflicts found", "No transport items require comparison.")
	}
	if len(affected) == 0 {
		return pass("preferredTransportModes", rule.Severity,
			"Preferred transport is used", "Transport items use preferred modes.")
	}
	item := violation("preferredTransportModes", rule.Severity,
		"Some transport does not match workspace preferences",
		fmt.Sprintf("%d transport item(s) use non-preferred modes.", len(affected)),
		map[string]any{"items": affected}, map[string]any{"modes": rule.Modes},
		action("replace_transport", "Replace transport", affected[0].DayNumber, affected[0].ItemIndex))
	item.AffectedItems = affected
	return item
}

func evaluateMaxTransferHours(
	itinerary aggregate.Itinerary,
	rule TransferHoursRule,
) EvaluationResult {
	limit := int(math.Round(rule.Hours * 60))
	if limit <= 0 {
		return unknown("maxTransferHoursPerDay", rule.Severity, "Maximum transfer time",
			"Not enough data to evaluate this rule.")
	}
	affected := make([]AffectedItem, 0)
	transferCount := 0
	hasDuration := false
	for _, day := range itinerary.Days {
		for itemIndex, item := range day.Items {
			if !isTransferItem(item) {
				continue
			}
			transferCount++
			duration, ok := itemDurationMinutes(item)
			if !ok && item.Transfer != nil && item.Transfer.EstimatedDurationMinutes != nil {
				duration = *item.Transfer.EstimatedDurationMinutes
				ok = true
			}
			if !ok {
				continue
			}
			hasDuration = true
			if duration > limit {
				dayNumber, index := day.Day, itemIndex
				amount := float64(duration)
				affected = appendLimited(affected, AffectedItem{
					DayNumber: &dayNumber, ItemIndex: &index, Name: item.Name, Amount: &amount,
				})
			}
		}
	}
	if transferCount == 0 {
		return pass("maxTransferHoursPerDay", rule.Severity,
			"No transfer-day conflicts found", "No transfer items require comparison.")
	}
	if !hasDuration {
		return unknown("maxTransferHoursPerDay", rule.Severity, "Maximum transfer time",
			"Transfer duration estimates are missing.")
	}
	if len(affected) == 0 {
		return pass("maxTransferHoursPerDay", rule.Severity,
			"Transfer times are within the workspace limit", "Transfer days fit the maximum transfer time.")
	}
	item := violation("maxTransferHoursPerDay", rule.Severity,
		"Transfer day is too long",
		fmt.Sprintf("%d transfer item(s) exceed %.1f hour(s).", len(affected), rule.Hours),
		map[string]any{"items": affected}, map[string]any{"hours": rule.Hours},
		action("repair_with_ai", "Repair with AI", affected[0].DayNumber, nil),
		action("replace_transport", "Change transport", affected[0].DayNumber, affected[0].ItemIndex))
	item.AffectedItems = affected
	return item
}

func evaluateDisallowedTransport(
	itinerary aggregate.Itinerary,
	rule TransportRule,
) EvaluationResult {
	disallowed := tokenSet(rule.Modes)
	if len(disallowed) == 0 {
		return pass("disallowedTransportModes", rule.Severity,
			"No disallowed transport modes configured", "No transport modes are disallowed.")
	}
	affected := make([]AffectedItem, 0)
	for _, day := range itinerary.Days {
		for itemIndex, item := range day.Items {
			mode := transportModeForItem(item)
			if mode == "" {
				continue
			}
			if _, ok := disallowed[mode]; ok {
				dayNumber, index := day.Day, itemIndex
				affected = appendLimited(affected, AffectedItem{
					DayNumber: &dayNumber, ItemIndex: &index, Name: item.Name,
				})
			}
		}
	}
	if len(affected) == 0 {
		return pass("disallowedTransportModes", rule.Severity,
			"No disallowed transport modes found", "The itinerary avoids disallowed transport modes.")
	}
	item := violation("disallowedTransportModes", rule.Severity,
		"Disallowed transport mode is used",
		fmt.Sprintf("%d transport item(s) use a disallowed mode.", len(affected)),
		map[string]any{"items": affected}, map[string]any{"modes": rule.Modes},
		action("replace_transport", "Replace transport", affected[0].DayNumber, affected[0].ItemIndex),
		action("repair_with_ai", "Repair with AI", affected[0].DayNumber, nil))
	item.AffectedItems = affected
	return item
}

func evaluateDisallowed(
	itinerary aggregate.Itinerary,
	rule ActivityTypesRule,
) EvaluationResult {
	disallowed := tokenSet(rule.Types)
	affected := make([]AffectedItem, 0)
	for _, day := range itinerary.Days {
		for itemIndex, item := range day.Items {
			values := []string{
				normalizeToken(item.Type),
				normalizeToken(item.Category),
				itemCategory(item),
			}
			matched := false
			for _, value := range values {
				if _, ok := disallowed[value]; ok {
					matched = true
					break
				}
			}
			if matched {
				dayNumber, index := day.Day, itemIndex
				affected = appendLimited(affected, AffectedItem{
					DayNumber: &dayNumber, ItemIndex: &index, Name: item.Name,
				})
			}
		}
	}
	if len(affected) == 0 {
		return pass("disallowedActivityTypes", rule.Severity,
			"No disallowed activity types found", "The itinerary avoids disallowed activity types.")
	}
	item := violation("disallowedActivityTypes", rule.Severity,
		"Disallowed activity types are present",
		fmt.Sprintf("%d item(s) match a disallowed activity type.", len(affected)),
		map[string]any{"items": affected}, map[string]any{"types": rule.Types},
		action("replace_item", "Replace item", affected[0].DayNumber, affected[0].ItemIndex),
		action("regenerate_day", "Regenerate day", affected[0].DayNumber, nil))
	item.AffectedItems = affected
	return item
}

func (e *Evaluation) add(result EvaluationResult) {
	e.Results = append(e.Results, result)
	e.Summary.RulesChecked++
	switch result.Status {
	case ResultPassed:
		e.Summary.PassedCount++
	default:
		switch result.Severity {
		case SeverityBlocking:
			e.Summary.BlockingCount++
		case SeverityWarning:
			e.Summary.WarningCount++
		default:
			e.Summary.InfoCount++
		}
	}
}

func (e *Evaluation) finish() {
	switch {
	case e.Summary.BlockingCount > 0:
		e.Status = EvaluationBlocking
	case e.Summary.WarningCount > 0:
		e.Status = EvaluationWarning
	case e.Summary.InfoCount > 0:
		e.Status = EvaluationInfo
	default:
		e.Status = EvaluationOK
	}
}

func pass(key string, severity Severity, title, message string) EvaluationResult {
	return EvaluationResult{
		RuleKey: key, Status: ResultPassed, Severity: severity, Title: title, Message: message,
		AffectedItems: []AffectedItem{}, SuggestedActions: []SuggestedAction{},
	}
}

func violation(
	key string,
	severity Severity,
	title, message string,
	actual, expected any,
	actions ...SuggestedAction,
) EvaluationResult {
	return EvaluationResult{
		RuleKey: key, Status: ResultViolation, Severity: severity, Title: title, Message: message,
		Actual: actual, Expected: expected, AffectedItems: []AffectedItem{},
		SuggestedActions: actions,
	}
}

func unknown(key string, configured Severity, title, message string) EvaluationResult {
	severity := configured
	status := ResultInfoUnknown
	if configured == SeverityWarning || configured == SeverityBlocking {
		severity = SeverityWarning
		status = ResultWarningUnknown
	}
	return EvaluationResult{
		RuleKey: key, Status: status, Severity: severity, Title: title, Message: message,
		AffectedItems: []AffectedItem{}, SuggestedActions: []SuggestedAction{},
	}
}

func action(kind, label string, dayNumber, itemIndex *int) SuggestedAction {
	return SuggestedAction{Type: kind, Label: label, DayNumber: dayNumber, ItemIndex: itemIndex}
}

func analyticsFor(in EvaluationInput, currency string) (analytics.TripCostAnalytics, bool) {
	value, ok := in.AnalyticsByCurrency[strings.ToUpper(strings.TrimSpace(currency))]
	return value, ok
}

func convertCost(
	ctx context.Context,
	amount float64,
	from, fallback, to string,
	in EvaluationInput,
) (float64, bool) {
	from = strings.ToUpper(strings.TrimSpace(from))
	if from == "" {
		from = strings.ToUpper(strings.TrimSpace(fallback))
	}
	to = strings.ToUpper(strings.TrimSpace(to))
	if from == "" || to == "" {
		return 0, false
	}
	if from == to {
		return round2(amount), true
	}
	if !in.ConversionEnabled || in.Converter == nil {
		return 0, false
	}
	result, err := in.Converter.Convert(ctx, amount, from, to)
	if err != nil || result == nil {
		return 0, false
	}
	return round2(result.ConvertedAmount), true
}

func accommodationNights(accommodation *aggregate.Accommodation) (int, bool) {
	checkIn, err := time.Parse("2006-01-02", accommodation.CheckInDate)
	if err != nil {
		return 0, false
	}
	checkOut, err := time.Parse("2006-01-02", accommodation.CheckOutDate)
	if err != nil || !checkOut.After(checkIn) {
		return 0, false
	}
	return int(checkOut.Sub(checkIn).Hours() / 24), true
}

func isTicketed(item aggregate.ItineraryItem) bool {
	if _, ok := ticketedTypes[normalizeToken(item.Type)]; ok {
		return true
	}
	if _, ok := ticketedTypes[itemCategory(item)]; ok {
		return true
	}
	return item.Place != nil && item.EstimatedCost != nil &&
		normalizeToken(item.EstimatedCost.Category) == "ticket"
}

func itemCategory(item aggregate.ItineraryItem) string {
	if value := normalizeToken(item.Category); value != "" {
		return value
	}
	if item.EstimatedCost != nil {
		if value := normalizeToken(item.EstimatedCost.Category); value != "" {
			return value
		}
	}
	if item.Place != nil {
		if value := normalizeToken(item.Place.Category); value != "" {
			return value
		}
	}
	return normalizeToken(item.Type)
}

func isTransferItem(item aggregate.ItineraryItem) bool {
	return normalizeToken(item.Type) == "transfer" || item.Transfer != nil
}

func transportModeForItem(item aggregate.ItineraryItem) string {
	if item.Transfer != nil {
		if mode := normalizeToken(item.Transfer.Mode); mode != "" {
			return mode
		}
	}
	if mode := normalizeToken(item.TransportMode); mode != "" {
		return mode
	}
	itemType := normalizeToken(item.Type)
	if _, ok := transportTypes[itemType]; ok {
		return itemType
	}
	return ""
}

func itemDurationMinutes(item aggregate.ItineraryItem) (int, bool) {
	if item.DurationMinutes != nil && *item.DurationMinutes >= 0 {
		return *item.DurationMinutes, true
	}
	start, startOK := minutesFromHHMM(item.Time)
	end, endOK := minutesFromHHMM(item.EndTime)
	if !startOK || !endOK || end < start {
		return 0, false
	}
	return end - start, true
}

func minutesFromHHMM(value string) (int, bool) {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	return parsed.Hour()*60 + parsed.Minute(), true
}

func money(amount float64, currency string) map[string]any {
	return map[string]any{"amount": round2(amount), "currency": currency}
}

func tokenSet(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value = normalizeToken(value); value != "" {
			result[value] = struct{}{}
		}
	}
	return result
}

func appendLimited(values []AffectedItem, value AffectedItem) []AffectedItem {
	if len(values) >= maxAffectedItems {
		return values
	}
	return append(values, value)
}

func appendUnique(values []string, additions ...string) []string {
	seen := make(map[string]struct{}, len(values)+len(additions))
	for _, value := range values {
		seen[value] = struct{}{}
	}
	for _, value := range additions {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
