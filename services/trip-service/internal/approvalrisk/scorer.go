package approvalrisk

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

const (
	maxTopReasons       = 5
	maxSuggestedActions = 8
	maxAffectedItems    = 10

	defaultWalkingLimitKm     = 12.0
	veryHighWalkingMultiplier = 1.5
	denseDayItemThreshold     = 6
)

func NotApplicable(tripID uuid.UUID, reason string) Response {
	now := time.Now().UTC()
	return Response{
		TripID:              tripID,
		Status:              RiskLevelNotApplicable,
		Score:               nil,
		MaxScore:            MaxScore,
		GeneratedAt:         now,
		Factors:             []Factor{},
		TopReasons:          []string{},
		SuggestedActions:    []SuggestedAction{},
		Warnings:            []string{},
		NotApplicableReason: &reason,
	}
}

func UnknownSummary() QueueSummary {
	return QueueSummary{Status: RiskLevelUnknown, Score: nil, TopReasons: []string{}}
}

func QueueSummaryFromResponse(response Response) QueueSummary {
	return QueueSummary{
		Status:     response.Status,
		Score:      response.Score,
		TopReasons: append([]string(nil), response.TopReasons...),
	}
}

func Score(in Input) Response {
	if in.GeneratedAt.IsZero() {
		in.GeneratedAt = time.Now().UTC()
	}
	if in.WorkspaceID == nil {
		return NotApplicable(in.TripID, "personal_trip")
	}

	builder := factorBuilder{tripID: in.TripID, workspaceID: *in.WorkspaceID}
	builder.add(policyFactors(in)...)
	builder.add(budgetFactors(in)...)
	builder.add(budgetConfidenceFactors(in)...)
	builder.add(costEstimateFactors(in)...)
	builder.add(costSplittingFactors(in)...)
	builder.add(availabilityFactors(in)...)
	builder.add(verificationFactors(in)...)
	builder.add(metadataFactors(in)...)
	builder.add(routeFactors(in)...)
	builder.add(scheduleFactors(in)...)
	builder.add(accommodationFactors(in)...)
	for _, name := range in.SignalUnavailableNames {
		builder.add(signalUnavailableFactor(in.TripID, name))
	}

	factors := builder.factors
	sortFactors(factors)

	score := 0
	blockingCount := blockingPolicyCount(in)
	for _, factor := range factors {
		score += factor.Points
	}
	if score > MaxScore {
		score = MaxScore
	}
	if blockingCount >= 2 && score < 75 {
		score = 75
	} else if blockingCount > 0 && score < 50 {
		score = 50
	}
	if score > MaxScore {
		score = MaxScore
	}

	actions := suggestedActions(factors)
	summary := summaryFor(factors, blockingCount, len(actions))
	scorePtr := score
	return Response{
		TripID:           in.TripID,
		WorkspaceID:      in.WorkspaceID,
		Status:           RiskLevelFromScore(score),
		Score:            &scorePtr,
		MaxScore:         MaxScore,
		GeneratedAt:      in.GeneratedAt,
		Summary:          summary,
		Factors:          factors,
		TopReasons:       topReasons(factors),
		SuggestedActions: actions,
		Warnings: []string{
			"Risk score is a planning aid, not an approval decision.",
		},
	}
}

func verificationFactors(in Input) []Factor {
	signal := in.Verification
	if signal == nil {
		return nil
	}
	out := make([]Factor, 0, 2)
	if signal.UnavailableCount > 0 {
		out = append(out, Factor{
			Type:     "unavailable_travel_assumption",
			Severity: FactorSeverityHigh,
			Points:   minInt(signal.UnavailableCount*18, 36),
			Title:    "Travel details reported unavailable",
			Message:  fmt.Sprintf("%d transport or activity verification item(s) are unavailable.", signal.UnavailableCount),
			Source:   SourceVerification,
			Affected: affected("verification", signal.UnavailableCount, nil),
			SuggestedActions: []SuggestedAction{
				action("review_verification", "Review verification", ActionPriorityHigh),
			},
		})
	}
	attention := signal.StaleCount + signal.MissingCount
	if attention > 0 {
		out = append(out, Factor{
			Type:     "travel_assumptions_need_verification",
			Severity: severityFromPoints(minInt(attention*6, 24)),
			Points:   minInt(attention*6, 24),
			Title:    "Travel assumptions need verification",
			Message:  fmt.Sprintf("%d real-world travel detail(s) are stale or missing.", attention),
			Source:   SourceVerification,
			Affected: affected("verification", attention, nil),
			SuggestedActions: []SuggestedAction{
				action("review_verification", "Review verification", ActionPriorityMedium),
			},
		})
	}
	return out
}

type factorBuilder struct {
	tripID      uuid.UUID
	workspaceID uuid.UUID
	factors     []Factor
}

func (b *factorBuilder) add(factors ...Factor) {
	for _, factor := range factors {
		if factor.Points <= 0 {
			continue
		}
		if factor.Affected == nil {
			tripID := b.tripID
			factor.Affected = &AffectedTarget{TripID: &tripID}
		} else if factor.Affected.TripID == nil {
			tripID := b.tripID
			factor.Affected.TripID = &tripID
		}
		for i := range factor.SuggestedActions {
			if factor.SuggestedActions[i].Priority == "" {
				factor.SuggestedActions[i].Priority = priorityForSeverity(factor.Severity)
			}
			if factor.SuggestedActions[i].Target.TripID == nil {
				tripID := b.tripID
				factor.SuggestedActions[i].Target.TripID = &tripID
			}
		}
		b.factors = append(b.factors, factor)
	}
}

func policyFactors(in Input) []Factor {
	if in.PolicyEvaluation == nil {
		return policySummaryFactors(in)
	}
	evaluation := in.PolicyEvaluation
	if evaluation.Status == workspacepolicies.EvaluationNotApplicable {
		return nil
	}
	counts := map[workspacepolicies.Severity]int{}
	affectedBySeverity := map[workspacepolicies.Severity][]AffectedItem{}
	actionsBySeverity := map[workspacepolicies.Severity][]SuggestedAction{}
	for _, result := range evaluation.Results {
		if result.Status == workspacepolicies.ResultPassed {
			continue
		}
		counts[result.Severity]++
		affectedBySeverity[result.Severity] = append(
			affectedBySeverity[result.Severity],
			affectedFromPolicy(result.AffectedItems)...,
		)
		actionsBySeverity[result.Severity] = append(
			actionsBySeverity[result.Severity],
			actionsFromPolicy(result.SuggestedActions)...,
		)
	}

	out := make([]Factor, 0, 3)
	if count := counts[workspacepolicies.SeverityBlocking]; count > 0 {
		points := minInt(count*35, 50)
		severity := FactorSeverityHigh
		if count >= 2 {
			severity = FactorSeverityCritical
		}
		out = append(out, Factor{
			Type:     "workspace_policy_blocking",
			Severity: severity,
			Points:   points,
			Title:    "Blocking workspace policy violations",
			Message:  fmt.Sprintf("%d blocking workspace policy violation(s) need review.", count),
			Source:   SourceWorkspacePolicy,
			Affected: affected("policy", count, affectedBySeverity[workspacepolicies.SeverityBlocking]),
			SuggestedActions: append(
				actionsBySeverity[workspacepolicies.SeverityBlocking],
				action("repair_with_ai", "Repair with AI", ActionPriorityHigh),
				action("fix_policy_violation", "Fix policy violation", ActionPriorityHigh),
				action("open_approval_checklist", "Open approval checklist", ActionPriorityHigh),
			),
		})
	}
	if count := counts[workspacepolicies.SeverityWarning]; count > 0 {
		out = append(out, Factor{
			Type:     "workspace_policy_warning",
			Severity: severityFromPoints(count * 10),
			Points:   minInt(count*10, 30),
			Title:    "Workspace policy warnings",
			Message:  fmt.Sprintf("%d workspace policy warning(s) should be reviewed.", count),
			Source:   SourceWorkspacePolicy,
			Affected: affected("policy", count, affectedBySeverity[workspacepolicies.SeverityWarning]),
			SuggestedActions: append(
				actionsBySeverity[workspacepolicies.SeverityWarning],
				action("repair_with_ai", "Repair with AI", ActionPriorityMedium),
				action("fix_policy_violation", "Review policy warning", ActionPriorityMedium),
			),
		})
	}
	if count := counts[workspacepolicies.SeverityInfo]; count > 0 {
		out = append(out, Factor{
			Type:             "workspace_policy_info",
			Severity:         FactorSeverityLow,
			Points:           minInt(count*2, 10),
			Title:            "Workspace policy notes",
			Message:          fmt.Sprintf("%d workspace policy note(s) may need attention.", count),
			Source:           SourceWorkspacePolicy,
			Affected:         affected("policy", count, affectedBySeverity[workspacepolicies.SeverityInfo]),
			SuggestedActions: []SuggestedAction{action("open_approval_checklist", "Open approval checklist", ActionPriorityLow)},
		})
	}
	return out
}

func policySummaryFactors(in Input) []Factor {
	checks := in.ChecklistInput
	count := checks.PolicyBlockingCount
	out := make([]Factor, 0, 3)
	if count > 0 {
		severity := FactorSeverityHigh
		if count >= 2 {
			severity = FactorSeverityCritical
		}
		out = append(out, Factor{
			Type:     "workspace_policy_blocking",
			Severity: severity,
			Points:   minInt(count*35, 50),
			Title:    "Blocking workspace policy violations",
			Message:  fmt.Sprintf("%d blocking workspace policy violation(s) need review.", count),
			Source:   SourceWorkspacePolicy,
			Affected: affected("policy", count, nil),
			SuggestedActions: []SuggestedAction{
				action("repair_with_ai", "Repair with AI", ActionPriorityHigh),
				action("fix_policy_violation", "Fix policy violation", ActionPriorityHigh),
			},
		})
	}
	if count := checks.PolicyWarningCount; count > 0 {
		out = append(out, Factor{
			Type:     "workspace_policy_warning",
			Severity: severityFromPoints(count * 10),
			Points:   minInt(count*10, 30),
			Title:    "Workspace policy warnings",
			Message:  fmt.Sprintf("%d workspace policy warning(s) should be reviewed.", count),
			Source:   SourceWorkspacePolicy,
			Affected: affected("policy", count, nil),
			SuggestedActions: []SuggestedAction{
				action("repair_with_ai", "Repair with AI", ActionPriorityMedium),
				action("fix_policy_violation", "Review policy warning", ActionPriorityMedium),
			},
		})
	}
	if count := checks.PolicyInfoCount; count > 0 {
		out = append(out, Factor{
			Type:             "workspace_policy_info",
			Severity:         FactorSeverityLow,
			Points:           minInt(count*2, 10),
			Title:            "Workspace policy notes",
			Message:          fmt.Sprintf("%d workspace policy note(s) may need attention.", count),
			Source:           SourceWorkspacePolicy,
			Affected:         affected("policy", count, nil),
			SuggestedActions: []SuggestedAction{action("open_approval_checklist", "Open approval checklist", ActionPriorityLow)},
		})
	}
	return out
}

func budgetFactors(in Input) []Factor {
	checks := in.ChecklistInput
	out := make([]Factor, 0, 3)
	if !checks.HasTripBudget {
		out = append(out, Factor{
			Type:             "missing_trip_budget",
			Severity:         FactorSeverityMedium,
			Points:           8,
			Title:            "Trip budget missing",
			Message:          "No trip budget is set for reviewers to compare against.",
			Source:           SourceTripBudget,
			Affected:         affected("budget", 1, nil),
			SuggestedActions: []SuggestedAction{action("open_trip_analytics", "Review trip analytics", ActionPriorityMedium)},
		})
	} else if checks.TripBudgetAmount > 0 && checks.EstimatedTotal > checks.TripBudgetAmount {
		over := checks.EstimatedTotal - checks.TripBudgetAmount
		ratio := over / checks.TripBudgetAmount
		points := overBudgetPoints(ratio, 10, 18, 25)
		out = append(out, Factor{
			Type:     "trip_budget_exceeded",
			Severity: severityFromPoints(points),
			Points:   points,
			Title:    "Trip budget exceeded",
			Message: fmt.Sprintf(
				"Estimated costs exceed the trip budget by %.2f %s.",
				round2(over), currencyOrDefault(in.Trip.BudgetCurrency),
			),
			Source:   SourceTripBudget,
			Affected: affected("budget", 1, nil),
			SuggestedActions: []SuggestedAction{
				action("repair_with_ai", "Repair with AI", ActionPriorityHigh),
				action("open_budget_optimization", "Optimize budget", ActionPriorityHigh),
				action("open_trip_analytics", "Open trip analytics", ActionPriorityMedium),
			},
		})
	}

	if in.WorkspaceBudget != nil {
		signal := in.WorkspaceBudget
		if signal.OverBudgetAmount > 0 {
			ratio := 0.0
			if signal.Amount > 0 {
				ratio = signal.OverBudgetAmount / signal.Amount
			}
			points := overBudgetPoints(ratio, 12, 18, 25)
			out = append(out, Factor{
				Type:     "workspace_budget_exceeded",
				Severity: severityFromPoints(points),
				Points:   points,
				Title:    "Workspace budget exceeded",
				Message: fmt.Sprintf(
					"Active workspace budget is exceeded by %.2f %s.",
					round2(signal.OverBudgetAmount), currencyOrDefault(signal.Currency),
				),
				Source:   SourceWorkspaceBudget,
				Affected: affected("budget", 1, nil),
				SuggestedActions: []SuggestedAction{
					action("repair_with_ai", "Repair with AI", ActionPriorityHigh),
					action("open_workspace_budget", "Open workspace budget", ActionPriorityHigh),
					action("open_budget_optimization", "Optimize budget", ActionPriorityHigh),
				},
			})
		} else if signal.UtilizationPercent >= 90 {
			out = append(out, Factor{
				Type:     "workspace_budget_nearing_limit",
				Severity: FactorSeverityMedium,
				Points:   8,
				Title:    "Workspace budget nearing limit",
				Message: fmt.Sprintf(
					"Active workspace budget is %.0f%% utilized.",
					round2(signal.UtilizationPercent),
				),
				Source:           SourceWorkspaceBudget,
				Affected:         affected("budget", 1, nil),
				SuggestedActions: []SuggestedAction{action("open_workspace_budget", "Open workspace budget", ActionPriorityMedium)},
			})
		}
	}
	return out
}

func budgetConfidenceFactors(in Input) []Factor {
	signal := in.BudgetConfidence
	if signal == nil {
		return nil
	}
	out := make([]Factor, 0, 2)
	if signal.Level == "very_low" || signal.Level == "low" || signal.RiskLevel == "high" || signal.RiskLevel == "critical" {
		points := 8
		severity := FactorSeverityMedium
		if signal.Level == "very_low" || signal.RiskLevel == "critical" {
			points = 16
			severity = FactorSeverityHigh
		}
		out = append(out, Factor{
			Type:     "budget_confidence_low",
			Severity: severity,
			Points:   points,
			Title:    "Budget confidence needs review",
			Message: fmt.Sprintf(
				"Budget confidence is %s with %s risk.",
				strings.ReplaceAll(signal.Level, "_", " "),
				strings.ReplaceAll(signal.RiskLevel, "_", " "),
			),
			Source:   SourceBudgetConfidence,
			Affected: affected("budget", 1, nil),
			SuggestedActions: []SuggestedAction{
				action("open_budget_confidence", "Improve budget confidence", ActionPriorityHigh),
				action("open_budget_optimization", "Optimize budget", ActionPriorityMedium),
			},
		})
	}
	for _, issueID := range signal.TopIssues {
		if issueID == "missing_accommodation_cost" || issueID == "missing_transport_prices" {
			out = append(out, Factor{
				Type:             "budget_confidence_missing_major_cost",
				Severity:         FactorSeverityMedium,
				Points:           8,
				Title:            "Major cost is missing",
				Message:          "Budget confidence found a missing major trip cost.",
				Source:           SourceBudgetConfidence,
				Affected:         affected("budget", 1, nil),
				SuggestedActions: []SuggestedAction{action("open_budget_confidence", "Improve budget confidence", ActionPriorityHigh)},
			})
			break
		}
	}
	return out
}

func costEstimateFactors(in Input) []Factor {
	checks := in.ChecklistInput
	out := make([]Factor, 0, 3)
	if checks.MissingEstimateCount > 0 {
		points := countThresholdPoints(checks.MissingEstimateCount, 5, 10, 15)
		out = append(out, Factor{
			Type:     "missing_cost_estimates",
			Severity: severityFromPoints(points),
			Points:   points,
			Title:    "Missing cost estimates",
			Message:  fmt.Sprintf("%d item(s) likely need cost estimates.", checks.MissingEstimateCount),
			Source:   SourceCostAnalytics,
			Affected: affected("budget", checks.MissingEstimateCount, nil),
			SuggestedActions: []SuggestedAction{
				action("add_missing_costs", "Add missing costs", ActionPriorityMedium),
				action("open_trip_analytics", "Open trip analytics", ActionPriorityMedium),
			},
		})
	}
	lowConfidenceCount := lowConfidenceEstimateCount(in.Itinerary)
	if lowConfidenceCount > 0 {
		out = append(out, Factor{
			Type:             "low_confidence_cost_estimates",
			Severity:         FactorSeverityMedium,
			Points:           8,
			Title:            "Low-confidence cost estimates",
			Message:          fmt.Sprintf("%d cost estimate(s) have low or uncertain confidence.", lowConfidenceCount),
			Source:           SourceCostAnalytics,
			Affected:         affected("budget", lowConfidenceCount, nil),
			SuggestedActions: []SuggestedAction{action("open_trip_analytics", "Review cost estimates", ActionPriorityMedium)},
		})
	}
	return out
}

func costSplittingFactors(in Input) []Factor {
	checks := in.ChecklistInput
	out := make([]Factor, 0, 4)
	if checks.EstimatedTotal > 0 && checks.TravelerCount == 0 {
		out = append(out, Factor{
			Type:             "cost_splitting_no_travelers",
			Severity:         FactorSeverityMedium,
			Points:           10,
			Title:            "No travelers configured for cost splitting",
			Message:          "This trip has estimated costs but no active travelers for splitting.",
			Source:           SourceCostSplitting,
			Affected:         affected("cost_splitting", 1, nil),
			SuggestedActions: []SuggestedAction{action("open_cost_splitting", "Configure cost splitting", ActionPriorityMedium)},
		})
	}
	if checks.UnassignedCostCount > 0 {
		out = append(out, Factor{
			Type:             "cost_splitting_unassigned_costs",
			Severity:         FactorSeverityHigh,
			Points:           12,
			Title:            "Unassigned costs",
			Message:          fmt.Sprintf("%d cost(s) are not assigned to travelers.", checks.UnassignedCostCount),
			Source:           SourceCostSplitting,
			Affected:         affected("cost_splitting", checks.UnassignedCostCount, nil),
			SuggestedActions: []SuggestedAction{action("open_cost_splitting", "Review cost splitting", ActionPriorityHigh)},
		})
	}
	if checks.InvalidSplitCount > 0 {
		out = append(out, Factor{
			Type:             "cost_splitting_invalid_rules",
			Severity:         FactorSeverityHigh,
			Points:           15,
			Title:            "Invalid cost split rules",
			Message:          fmt.Sprintf("%d cost split rule(s) are invalid.", checks.InvalidSplitCount),
			Source:           SourceCostSplitting,
			Affected:         affected("cost_splitting", checks.InvalidSplitCount, nil),
			SuggestedActions: []SuggestedAction{action("open_cost_splitting", "Fix split rules", ActionPriorityHigh)},
		})
	}
	if checks.DefaultSplitCount >= 5 {
		out = append(out, Factor{
			Type:             "cost_splitting_many_default_splits",
			Severity:         FactorSeverityLow,
			Points:           5,
			Title:            "Many default splits",
			Message:          fmt.Sprintf("%d costs use the default equal split.", checks.DefaultSplitCount),
			Source:           SourceCostSplitting,
			Affected:         affected("cost_splitting", checks.DefaultSplitCount, nil),
			SuggestedActions: []SuggestedAction{action("open_cost_splitting", "Review default splits", ActionPriorityLow)},
		})
	}
	return out
}

func availabilityFactors(in Input) []Factor {
	checks := in.ChecklistInput
	out := make([]Factor, 0, 5)
	if checks.AvailabilityUnavailableCount > 0 {
		out = append(out, Factor{
			Type:             "availability_unavailable",
			Severity:         FactorSeverityHigh,
			Points:           25,
			Title:            "Unavailable important items",
			Message:          fmt.Sprintf("%d item(s) are marked unavailable.", checks.AvailabilityUnavailableCount),
			Source:           SourceAvailability,
			Affected:         affected("availability", checks.AvailabilityUnavailableCount, topAvailabilityAffected(in.Itinerary, "unavailable")),
			SuggestedActions: []SuggestedAction{action("check_availability", "Check availability", ActionPriorityHigh)},
		})
	}
	if checks.AvailabilityUncheckedCount > 0 {
		points := countThresholdPoints(checks.AvailabilityUncheckedCount, 8, 14, 20)
		out = append(out, Factor{
			Type:     "availability_unchecked",
			Severity: severityFromPoints(points),
			Points:   points,
			Title:    "Unchecked bookable items",
			Message:  fmt.Sprintf("%d bookable item(s) do not have stored availability checks.", checks.AvailabilityUncheckedCount),
			Source:   SourceAvailability,
			Affected: affected("availability", checks.AvailabilityUncheckedCount, topUncheckedAffected(in.Itinerary)),
			SuggestedActions: []SuggestedAction{
				action("check_availability", "Check availability", ActionPriorityHigh),
				action("open_item", "Open item", ActionPriorityMedium),
			},
		})
	}
	if checks.AvailabilityLowConfidenceCount > 0 {
		out = append(out, Factor{
			Type:             "availability_low_confidence",
			Severity:         FactorSeverityMedium,
			Points:           8,
			Title:            "Low-confidence availability matches",
			Message:          fmt.Sprintf("%d availability match(es) have low confidence.", checks.AvailabilityLowConfidenceCount),
			Source:           SourceAvailability,
			Affected:         affected("availability", checks.AvailabilityLowConfidenceCount, topAvailabilityAffected(in.Itinerary, "low_confidence")),
			SuggestedActions: []SuggestedAction{action("check_availability", "Recheck availability", ActionPriorityMedium)},
		})
	}
	if checks.AvailabilityFallbackCount > 0 {
		out = append(out, Factor{
			Type:             "availability_fallback_used",
			Severity:         FactorSeverityMedium,
			Points:           6,
			Title:            "Fallback availability used",
			Message:          fmt.Sprintf("%d item(s) use fallback availability data.", checks.AvailabilityFallbackCount),
			Source:           SourceAvailability,
			Affected:         affected("availability", checks.AvailabilityFallbackCount, topAvailabilityAffected(in.Itinerary, "fallback")),
			SuggestedActions: []SuggestedAction{action("check_availability", "Verify availability", ActionPriorityMedium)},
		})
	}
	return out
}

func metadataFactors(in Input) []Factor {
	out := make([]Factor, 0, 3)
	if in.Metadata.TemplateFallbackUsed {
		out = append(out, Factor{
			Type:             "ai_fallback_used",
			Severity:         FactorSeverityMedium,
			Points:           10,
			Title:            "AI template fallback used",
			Message:          "This trip was created with deterministic fallback after AI template adaptation could not be used.",
			Source:           SourceTemplate,
			Affected:         affected("ai", 1, nil),
			SuggestedActions: []SuggestedAction{action("review_ai_adaptation", "Review AI adaptation", ActionPriorityMedium)},
		})
	}
	if in.Metadata.TemplateWarningCount > 0 {
		points := minInt(5+in.Metadata.TemplateWarningCount*2, 12)
		out = append(out, Factor{
			Type:             "ai_adaptation_warnings",
			Severity:         FactorSeverityMedium,
			Points:           points,
			Title:            "AI adaptation warnings",
			Message:          fmt.Sprintf("%d AI template adaptation warning(s) need review.", in.Metadata.TemplateWarningCount),
			Source:           SourceTemplate,
			Affected:         affected("ai", in.Metadata.TemplateWarningCount, nil),
			SuggestedActions: []SuggestedAction{action("review_ai_adaptation", "Review AI adaptation", ActionPriorityMedium)},
		})
	}
	if in.Metadata.ValidationRepairUsed {
		out = append(out, Factor{
			Type:             "ai_output_repaired",
			Severity:         FactorSeverityMedium,
			Points:           8,
			Title:            "Generated output was repaired",
			Message:          "The generated itinerary needed validation repair before it was saved.",
			Source:           SourceAIGeneration,
			Affected:         affected("ai", 1, nil),
			SuggestedActions: []SuggestedAction{action("review_ai_adaptation", "Review generated itinerary", ActionPriorityMedium)},
		})
	}
	return out
}

func routeFactors(in Input) []Factor {
	route := in.Trip.Route
	if route == nil || len(route.Stops) == 0 {
		return nil
	}
	out := make([]Factor, 0, 6)
	stopCount := len(route.Stops)
	if in.Trip.Days > 0 && stopCount > in.Trip.Days {
		out = append(out, Factor{
			Type:             "too_many_stops_for_duration",
			Severity:         FactorSeverityHigh,
			Points:           18,
			Title:            "Too many stops for trip duration",
			Message:          fmt.Sprintf("%d stop(s) in %d day(s) may be rushed.", stopCount, in.Trip.Days),
			Source:           SourceRoute,
			Affected:         affected("route", stopCount, nil),
			SuggestedActions: []SuggestedAction{action("repair_with_ai", "Repair with AI", ActionPriorityHigh)},
		})
	}

	longTransfers := 0
	missingEstimates := 0
	highCost := 0.0
	lowConfidenceSelections := 0
	unverifiedSelections := 0
	missingSelections := 0
	for _, leg := range route.Legs {
		duration := effectiveRouteLegDuration(leg)
		if duration == nil || *duration <= 0 {
			missingEstimates++
		} else if *duration > 6*60 {
			longTransfers++
		}
		if amount := effectiveRouteLegCostAmount(leg); amount != nil {
			highCost += *amount
		}
		if leg.SelectedTransportOption == nil {
			if routeLegNeedsSelection(leg.Mode) {
				missingSelections++
			}
			continue
		}
		if selectedTransportLowConfidence(leg.SelectedTransportOption) {
			lowConfidenceSelections++
		}
		if selectedTransportUnverified(leg.SelectedTransportOption) {
			unverifiedSelections++
		}
	}
	if longTransfers > 0 {
		out = append(out, Factor{
			Type:             "long_transfer_day",
			Severity:         FactorSeverityHigh,
			Points:           minInt(12+longTransfers*4, 24),
			Title:            "Long transfer day",
			Message:          fmt.Sprintf("%d transfer leg(s) exceed six hours.", longTransfers),
			Source:           SourceRoute,
			Affected:         affected("route", longTransfers, nil),
			SuggestedActions: []SuggestedAction{action("repair_with_ai", "Repair with AI", ActionPriorityHigh)},
		})
	}
	if missingEstimates > 0 {
		out = append(out, Factor{
			Type:             "route_estimate_missing",
			Severity:         FactorSeverityMedium,
			Points:           minInt(5+missingEstimates*3, 15),
			Title:            "Route estimate missing",
			Message:          fmt.Sprintf("%d route leg(s) are missing duration estimates.", missingEstimates),
			Source:           SourceRoute,
			Affected:         affected("route", missingEstimates, nil),
			SuggestedActions: []SuggestedAction{action("review_route", "Review route", ActionPriorityMedium)},
		})
	}
	if in.Trip.BudgetAmount != nil && *in.Trip.BudgetAmount > 0 && highCost > *in.Trip.BudgetAmount*0.35 {
		out = append(out, Factor{
			Type:     "high_transport_cost",
			Severity: FactorSeverityMedium,
			Points:   10,
			Title:    "High transport cost",
			Message:  "Estimated transfer costs are a large share of the trip budget.",
			Source:   SourceRoute,
			Affected: affected("route", len(route.Legs), nil),
			SuggestedActions: []SuggestedAction{
				action("repair_with_ai", "Repair with AI", ActionPriorityMedium),
				action("optimize_budget", "Optimize budget", ActionPriorityMedium),
			},
		})
	}
	if lowConfidenceSelections > 0 {
		out = append(out, Factor{
			Type:             "transport_option_low_confidence",
			Severity:         FactorSeverityMedium,
			Points:           minInt(5+lowConfidenceSelections*3, 15),
			Title:            "Low-confidence transport selection",
			Message:          fmt.Sprintf("%d selected transport option(s) have low or medium confidence.", lowConfidenceSelections),
			Source:           SourceRoute,
			Affected:         affected("route", lowConfidenceSelections, nil),
			SuggestedActions: []SuggestedAction{action("review_route", "Review route", ActionPriorityMedium)},
		})
	}
	if unverifiedSelections > 0 {
		out = append(out, Factor{
			Type:             "transport_option_unverified",
			Severity:         FactorSeverityMedium,
			Points:           minInt(6+unverifiedSelections*3, 18),
			Title:            "Transport selection needs verification",
			Message:          fmt.Sprintf("%d selected transport option(s) come from mock, manual, or unknown availability data.", unverifiedSelections),
			Source:           SourceRoute,
			Affected:         affected("route", unverifiedSelections, nil),
			SuggestedActions: []SuggestedAction{action("review_route", "Review route", ActionPriorityMedium)},
		})
	}
	if missingSelections > 0 {
		out = append(out, Factor{
			Type:             "missing_transport_option_for_required_leg",
			Severity:         FactorSeverityMedium,
			Points:           minInt(4+missingSelections*2, 12),
			Title:            "Transport option not selected",
			Message:          fmt.Sprintf("%d scheduled transport leg(s) do not have a selected provider option.", missingSelections),
			Source:           SourceRoute,
			Affected:         affected("route", missingSelections, nil),
			SuggestedActions: []SuggestedAction{action("review_route", "Review route", ActionPriorityMedium)},
		})
	}
	styles := tokenSet(route.Preferences.TripStyles)
	if _, ok := styles["hiking"]; ok && denseHikingDays(in.Itinerary) > 0 {
		out = append(out, Factor{
			Type:             "hiking_day_too_dense",
			Severity:         FactorSeverityMedium,
			Points:           10,
			Title:            "Hiking day may be too dense",
			Message:          "Hiking style is selected and one or more days are densely scheduled.",
			Source:           SourceRoute,
			Affected:         affected("route", 1, nil),
			SuggestedActions: []SuggestedAction{action("repair_with_ai", "Repair with AI", ActionPriorityMedium)},
		})
	}
	if _, ok := styles["camping"]; ok && !routeHasCampingAccommodation(route) {
		out = append(out, Factor{
			Type:             "camping_accommodation_missing",
			Severity:         FactorSeverityMedium,
			Points:           8,
			Title:            "Camping accommodation not configured",
			Message:          "Camping style is selected, but no route stop is marked as campsite or cabin.",
			Source:           SourceRoute,
			Affected:         affected("route", 1, nil),
			SuggestedActions: []SuggestedAction{action("review_route", "Review route", ActionPriorityMedium)},
		})
	}
	return out
}

func effectiveRouteLegDuration(leg aggregate.RouteLeg) *int {
	if leg.SelectedTransportOption != nil && leg.SelectedTransportOption.DurationMinutes > 0 {
		duration := leg.SelectedTransportOption.DurationMinutes
		return &duration
	}
	return leg.EstimatedDurationMinutes
}

func effectiveRouteLegCostAmount(leg aggregate.RouteLeg) *float64 {
	if leg.SelectedTransportOption != nil && leg.SelectedTransportOption.EstimatedPrice != nil {
		amount := leg.SelectedTransportOption.EstimatedPrice.Amount
		return &amount
	}
	if leg.EstimatedCost != nil && leg.EstimatedCost.Amount != nil {
		amount := *leg.EstimatedCost.Amount
		return &amount
	}
	return nil
}

func routeLegNeedsSelection(mode string) bool {
	switch normalizeToken(mode) {
	case aggregate.TransportModeTrain, aggregate.TransportModeBus, aggregate.TransportModeFlight, aggregate.TransportModeFerry, aggregate.TransportModeBoat, aggregate.TransportModePublicTransport:
		return true
	default:
		return false
	}
}

func selectedTransportLowConfidence(option *aggregate.SelectedTransportOption) bool {
	if option == nil {
		return false
	}
	switch normalizeToken(option.Confidence) {
	case "low", "medium", "":
		return true
	default:
		return false
	}
}

func selectedTransportUnverified(option *aggregate.SelectedTransportOption) bool {
	if option == nil {
		return false
	}
	switch normalizeToken(option.Provider) {
	case "", "mock", "manual":
		return true
	default:
		return normalizeToken(option.Status) == "unknown"
	}
}

func denseHikingDays(itinerary aggregate.Itinerary) int {
	count := 0
	for _, day := range itinerary.Days {
		if len(day.Items) >= denseDayItemThreshold && !hasRestBlock(day) {
			count++
		}
	}
	return count
}

func routeHasCampingAccommodation(route *aggregate.TripRoute) bool {
	if route == nil {
		return false
	}
	for _, stop := range route.Stops {
		switch normalizeToken(stop.AccommodationHint) {
		case "campsite", "camping", "cabin", "campervan":
			return true
		}
	}
	return false
}

func tokenSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		if normalized := normalizeToken(value); normalized != "" {
			out[normalized] = struct{}{}
		}
	}
	return out
}

func scheduleFactors(in Input) []Factor {
	out := make([]Factor, 0, 4)
	walkingAffected := make([]AffectedItem, 0)
	veryHighWalking := false
	overlaps := 0
	impossible := 0
	denseWithoutRest := 0
	for _, day := range in.Itinerary.Days {
		totalWalk := 0.0
		for _, item := range day.Items {
			if item.WalkingDistanceKm != nil && *item.WalkingDistanceKm > 0 {
				totalWalk += *item.WalkingDistanceKm
			}
		}
		if totalWalk > defaultWalkingLimitKm {
			dayNumber := day.Day
			amount := round2(totalWalk)
			walkingAffected = appendLimited(walkingAffected, AffectedItem{
				DayNumber: &dayNumber,
				Amount:    &amount,
				Currency:  "km",
			})
			if totalWalk > defaultWalkingLimitKm*veryHighWalkingMultiplier {
				veryHighWalking = true
			}
		}
		dayOverlaps, dayImpossible := scheduleIssues(day)
		overlaps += dayOverlaps
		impossible += dayImpossible
		if len(day.Items) >= denseDayItemThreshold && !hasRestBlock(day) {
			denseWithoutRest++
		}
	}
	if len(walkingAffected) > 0 {
		points := 10
		severity := FactorSeverityMedium
		if veryHighWalking {
			points = 18
			severity = FactorSeverityHigh
		}
		out = append(out, Factor{
			Type:     "walking_distance_high",
			Severity: severity,
			Points:   points,
			Title:    "High walking distance",
			Message:  fmt.Sprintf("%d day(s) have high estimated walking distance.", len(walkingAffected)),
			Source:   SourceWalkingDistance,
			Affected: affected("route", len(walkingAffected), walkingAffected),
			SuggestedActions: []SuggestedAction{
				action("repair_with_ai", "Repair with AI", ActionPriorityMedium),
				action("optimize_route", "Optimize route", ActionPriorityMedium),
				action("regenerate_day", "Regenerate day", ActionPriorityMedium),
			},
		})
	}
	if overlaps > 0 {
		out = append(out, Factor{
			Type:             "schedule_overlaps",
			Severity:         FactorSeverityHigh,
			Points:           15,
			Title:            "Overlapping itinerary items",
			Message:          fmt.Sprintf("%d item overlap(s) were found in the schedule.", overlaps),
			Source:           SourceSchedule,
			Affected:         affected("schedule", overlaps, nil),
			SuggestedActions: []SuggestedAction{action("open_item", "Review schedule", ActionPriorityHigh)},
		})
	}
	if impossible > 0 {
		out = append(out, Factor{
			Type:             "schedule_impossible_timing",
			Severity:         FactorSeverityHigh,
			Points:           20,
			Title:            "Impossible timing",
			Message:          fmt.Sprintf("%d itinerary item(s) have impossible timing or negative duration.", impossible),
			Source:           SourceSchedule,
			Affected:         affected("schedule", impossible, nil),
			SuggestedActions: []SuggestedAction{action("open_item", "Fix timing", ActionPriorityHigh)},
		})
	}
	if denseWithoutRest > 0 {
		out = append(out, Factor{
			Type:     "dense_days_without_rest",
			Severity: FactorSeverityMedium,
			Points:   6,
			Title:    "Dense days without rest blocks",
			Message:  fmt.Sprintf("%d dense day(s) do not include a rest block.", denseWithoutRest),
			Source:   SourceSchedule,
			Affected: affected("schedule", denseWithoutRest, nil),
			SuggestedActions: []SuggestedAction{
				action("repair_with_ai", "Repair with AI", ActionPriorityMedium),
				action("regenerate_day", "Regenerate day", ActionPriorityMedium),
			},
		})
	}
	return out
}

func accommodationFactors(in Input) []Factor {
	out := make([]Factor, 0, 2)
	if in.Trip.Days > 1 && in.Trip.Accommodation == nil {
		out = append(out, Factor{
			Type:             "accommodation_missing",
			Severity:         FactorSeverityMedium,
			Points:           8,
			Title:            "Accommodation missing",
			Message:          "This overnight trip does not have accommodation details.",
			Source:           SourceAccommodation,
			Affected:         affected("accommodation", 1, nil),
			SuggestedActions: []SuggestedAction{action("open_accommodation", "Add accommodation", ActionPriorityMedium)},
		})
		return out
	}
	if in.Trip.Accommodation != nil && itineraryUsesLocationContext(in.Itinerary) {
		place := in.Trip.Accommodation.Place
		if place == nil || place.Latitude == nil || place.Longitude == nil {
			out = append(out, Factor{
				Type:             "accommodation_location_incomplete",
				Severity:         FactorSeverityLow,
				Points:           5,
				Title:            "Accommodation location incomplete",
				Message:          "Accommodation lacks a mapped place or coordinates for route review.",
				Source:           SourceAccommodation,
				Affected:         affected("accommodation", 1, nil),
				SuggestedActions: []SuggestedAction{action("open_accommodation", "Update accommodation", ActionPriorityLow)},
			})
		}
	}
	return out
}

func signalUnavailableFactor(tripID uuid.UUID, name string) Factor {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "risk signal"
	}
	return Factor{
		Type:     "risk_signal_unavailable",
		Severity: FactorSeverityMedium,
		Points:   5,
		Title:    "Risk signal unavailable",
		Message:  fmt.Sprintf("%s could not be evaluated. The score may be incomplete.", name),
		Source:   SourceApprovalChecklist,
		Affected: &AffectedTarget{
			TripID:   &tripID,
			Category: "risk_signal",
		},
		SuggestedActions: []SuggestedAction{{
			Type:     "open_approval_checklist",
			Label:    "Open approval checklist",
			Priority: ActionPriorityMedium,
			Target:   SuggestedActionTarget{TripID: &tripID},
		}},
	}
}

func suggestedActions(factors []Factor) []SuggestedAction {
	type candidate struct {
		action   SuggestedAction
		severity FactorSeverity
		points   int
	}
	byKey := make(map[string]candidate)
	for _, factor := range factors {
		for _, action := range factor.SuggestedActions {
			key := actionKey(action)
			current, exists := byKey[key]
			next := candidate{action: action, severity: factor.Severity, points: factor.Points}
			if !exists || candidateLess(current, next) {
				byKey[key] = next
			}
		}
	}
	items := make([]candidate, 0, len(byKey))
	for _, item := range byKey {
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if severityRank(items[i].severity) != severityRank(items[j].severity) {
			return severityRank(items[i].severity) > severityRank(items[j].severity)
		}
		if items[i].points != items[j].points {
			return items[i].points > items[j].points
		}
		return priorityRank(items[i].action.Priority) > priorityRank(items[j].action.Priority)
	})
	if len(items) > maxSuggestedActions {
		items = items[:maxSuggestedActions]
	}
	out := make([]SuggestedAction, 0, len(items))
	for _, item := range items {
		out = append(out, item.action)
	}
	return out
}

func summaryFor(factors []Factor, blockingPolicyCount int, actionCount int) Summary {
	summary := Summary{
		FactorCount:                  len(factors),
		BlockingPolicyViolationCount: blockingPolicyCount,
		SuggestedActionCount:         actionCount,
	}
	for _, factor := range factors {
		switch factor.Severity {
		case FactorSeverityCritical:
			summary.CriticalFactorCount++
		case FactorSeverityHigh:
			summary.HighFactorCount++
		case FactorSeverityMedium:
			summary.MediumFactorCount++
		default:
			summary.LowFactorCount++
		}
	}
	return summary
}

func topReasons(factors []Factor) []string {
	if len(factors) == 0 {
		return []string{}
	}
	items := append([]Factor(nil), factors...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Points != items[j].Points {
			return items[i].Points > items[j].Points
		}
		return severityRank(items[i].Severity) > severityRank(items[j].Severity)
	})
	if len(items) > maxTopReasons {
		items = items[:maxTopReasons]
	}
	out := make([]string, 0, len(items))
	for _, factor := range items {
		out = append(out, shortReason(factor))
	}
	return out
}

func blockingPolicyCount(in Input) int {
	if in.PolicyEvaluation != nil {
		count := 0
		for _, result := range in.PolicyEvaluation.Results {
			if result.Status != workspacepolicies.ResultPassed &&
				result.Severity == workspacepolicies.SeverityBlocking {
				count++
			}
		}
		return count
	}
	return in.ChecklistInput.PolicyBlockingCount
}

func affected(category string, count int, items []AffectedItem) *AffectedTarget {
	if len(items) > maxAffectedItems {
		items = items[:maxAffectedItems]
	}
	return &AffectedTarget{
		Category:      category,
		AffectedCount: count,
		AffectedItems: items,
	}
}

func affectedFromPolicy(items []workspacepolicies.AffectedItem) []AffectedItem {
	out := make([]AffectedItem, 0, minInt(len(items), maxAffectedItems))
	for _, item := range items {
		out = appendLimited(out, AffectedItem{
			DayNumber: item.DayNumber,
			ItemIndex: item.ItemIndex,
			Name:      item.Name,
			Amount:    item.Amount,
			Currency:  item.Currency,
		})
	}
	return out
}

func actionsFromPolicy(actions []workspacepolicies.SuggestedAction) []SuggestedAction {
	out := make([]SuggestedAction, 0, len(actions))
	for _, candidate := range actions {
		actionType := normalizePolicyAction(candidate.Type)
		out = append(out, SuggestedAction{
			Type:  actionType,
			Label: policyActionLabel(actionType, candidate.Label),
			Target: SuggestedActionTarget{
				DayNumber: candidate.DayNumber,
				ItemIndex: candidate.ItemIndex,
			},
		})
	}
	return out
}

func normalizePolicyAction(actionType string) string {
	switch actionType {
	case "open_budget_optimization", "open_trip_analytics", "open_workspace_budget",
		"open_cost_splitting", "check_availability", "open_item",
		"open_accommodation", "fix_policy_violation", "regenerate_day",
		"optimize_route", "add_missing_costs", "review_ai_adaptation",
		"open_approval_checklist", "repair_with_ai":
		return actionType
	case "optimize_day_budget":
		return "open_budget_optimization"
	case "update_price":
		return "add_missing_costs"
	default:
		return "fix_policy_violation"
	}
}

func policyActionLabel(actionType string, fallback string) string {
	if strings.TrimSpace(fallback) != "" && actionType != "fix_policy_violation" {
		return fallback
	}
	switch actionType {
	case "open_budget_optimization":
		return "Optimize budget"
	case "add_missing_costs":
		return "Update costs"
	case "open_cost_splitting":
		return "Review cost splitting"
	case "check_availability":
		return "Check availability"
	case "regenerate_day":
		return "Regenerate day"
	case "optimize_route":
		return "Optimize route"
	case "repair_with_ai":
		return "Repair with AI"
	default:
		return "Fix policy violation"
	}
}

func action(actionType, label string, priority SuggestedActionPriority) SuggestedAction {
	return SuggestedAction{Type: actionType, Label: label, Priority: priority}
}

func appendLimited(values []AffectedItem, value AffectedItem) []AffectedItem {
	if len(values) >= maxAffectedItems {
		return values
	}
	return append(values, value)
}

func countThresholdPoints(count, low, medium, high int) int {
	switch {
	case count <= 0:
		return 0
	case count <= 2:
		return low
	case count <= 5:
		return medium
	default:
		return high
	}
}

func overBudgetPoints(ratio float64, low, medium, high int) int {
	switch {
	case ratio <= 0.10:
		return low
	case ratio <= 0.30:
		return medium
	default:
		return high
	}
}

func severityFromPoints(points int) FactorSeverity {
	switch {
	case points >= 25:
		return FactorSeverityCritical
	case points >= 12:
		return FactorSeverityHigh
	case points >= 6:
		return FactorSeverityMedium
	default:
		return FactorSeverityLow
	}
}

func priorityForSeverity(severity FactorSeverity) SuggestedActionPriority {
	switch severity {
	case FactorSeverityCritical, FactorSeverityHigh:
		return ActionPriorityHigh
	case FactorSeverityMedium:
		return ActionPriorityMedium
	default:
		return ActionPriorityLow
	}
}

func sortFactors(factors []Factor) {
	sort.SliceStable(factors, func(i, j int) bool {
		if severityRank(factors[i].Severity) != severityRank(factors[j].Severity) {
			return severityRank(factors[i].Severity) > severityRank(factors[j].Severity)
		}
		if factors[i].Points != factors[j].Points {
			return factors[i].Points > factors[j].Points
		}
		return factors[i].Type < factors[j].Type
	})
}

func severityRank(severity FactorSeverity) int {
	switch severity {
	case FactorSeverityCritical:
		return 4
	case FactorSeverityHigh:
		return 3
	case FactorSeverityMedium:
		return 2
	default:
		return 1
	}
}

func priorityRank(priority SuggestedActionPriority) int {
	switch priority {
	case ActionPriorityHigh:
		return 3
	case ActionPriorityMedium:
		return 2
	default:
		return 1
	}
}

func candidateLess(a, b struct {
	action   SuggestedAction
	severity FactorSeverity
	points   int
}) bool {
	if severityRank(a.severity) != severityRank(b.severity) {
		return severityRank(a.severity) < severityRank(b.severity)
	}
	if a.points != b.points {
		return a.points < b.points
	}
	return priorityRank(a.action.Priority) < priorityRank(b.action.Priority)
}

func actionKey(action SuggestedAction) string {
	target := action.Target
	return fmt.Sprintf("%s:%s:%s:%s:%s",
		action.Type,
		uuidPtrString(target.TripID),
		uuidPtrString(target.WorkspaceID),
		intPtrString(target.DayNumber),
		intPtrString(target.ItemIndex),
	)
}

func uuidPtrString(value *uuid.UUID) string {
	if value == nil {
		return ""
	}
	return value.String()
}

func intPtrString(value *int) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
}

func shortReason(factor Factor) string {
	message := strings.TrimSpace(factor.Message)
	if message == "" {
		message = factor.Title
	}
	if len([]rune(message)) <= 120 {
		return message
	}
	runes := []rune(message)
	return strings.TrimSpace(string(runes[:117])) + "..."
}

func lowConfidenceEstimateCount(itinerary aggregate.Itinerary) int {
	count := 0
	for _, day := range itinerary.Days {
		for _, item := range day.Items {
			if item.EstimatedCost != nil {
				confidence := normalizeToken(item.EstimatedCost.Confidence)
				if confidence == "low" || confidence == "unknown" || confidence == "uncertain" {
					count++
				}
			}
			if item.PriceEnrichment != nil &&
				item.PriceEnrichment.MatchConfidence > 0 &&
				item.PriceEnrichment.MatchConfidence < 0.65 {
				count++
			}
		}
	}
	return count
}

func topUncheckedAffected(itinerary aggregate.Itinerary) []AffectedItem {
	out := make([]AffectedItem, 0, maxAffectedItems)
	for _, day := range itinerary.Days {
		for itemIndex, item := range day.Items {
			if item.Place == nil || item.PriceEnrichment != nil || item.AvailabilityCheck != nil {
				continue
			}
			dayNumber, index := day.Day, itemIndex
			out = appendLimited(out, AffectedItem{
				DayNumber: &dayNumber,
				ItemIndex: &index,
				Name:      item.Name,
				Category:  itemCategory(item),
			})
		}
	}
	return out
}

func topAvailabilityAffected(itinerary aggregate.Itinerary, kind string) []AffectedItem {
	out := make([]AffectedItem, 0, maxAffectedItems)
	for _, day := range itinerary.Days {
		for itemIndex, item := range day.Items {
			check := item.AvailabilityCheck
			if check == nil {
				continue
			}
			matches := false
			switch kind {
			case "unavailable":
				matches = normalizeToken(check.Status) == "unavailable"
			case "low_confidence":
				matches = check.MatchConfidence > 0 && check.MatchConfidence < 0.65
			case "fallback":
				matches = check.FallbackUsed
			}
			if !matches {
				continue
			}
			dayNumber, index := day.Day, itemIndex
			out = appendLimited(out, AffectedItem{
				DayNumber: &dayNumber,
				ItemIndex: &index,
				Name:      item.Name,
				Category:  itemCategory(item),
			})
		}
	}
	return out
}

func scheduleIssues(day aggregate.ItineraryDay) (overlaps int, impossible int) {
	type interval struct {
		start int
		end   int
	}
	intervals := make([]interval, 0, len(day.Items))
	for _, item := range day.Items {
		start, startOK := minutesFromHHMM(item.Time)
		end, endOK := minutesFromHHMM(item.EndTime)
		if item.DurationMinutes != nil && *item.DurationMinutes < 0 {
			impossible++
			continue
		}
		if !startOK || !endOK {
			continue
		}
		if end < start {
			impossible++
			continue
		}
		if end == start {
			continue
		}
		intervals = append(intervals, interval{start: start, end: end})
	}
	sort.SliceStable(intervals, func(i, j int) bool {
		return intervals[i].start < intervals[j].start
	})
	for i := 1; i < len(intervals); i++ {
		if intervals[i].start < intervals[i-1].end {
			overlaps++
		}
	}
	return overlaps, impossible
}

func hasRestBlock(day aggregate.ItineraryDay) bool {
	for _, item := range day.Items {
		token := normalizeToken(item.Type)
		if token == "rest" || token == "break" || token == "free_time" || token == "freetime" {
			return true
		}
	}
	return false
}

func itineraryUsesLocationContext(itinerary aggregate.Itinerary) bool {
	for _, day := range itinerary.Days {
		for _, item := range day.Items {
			if item.WalkingDistanceKm != nil || item.Place != nil {
				return true
			}
		}
	}
	return false
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

func minutesFromHHMM(value string) (int, bool) {
	parsed, err := time.Parse("15:04", strings.TrimSpace(value))
	if err != nil {
		return 0, false
	}
	return parsed.Hour()*60 + parsed.Minute(), true
}

func normalizeToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func currencyOrDefault(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value == "" {
		return "EUR"
	}
	return value
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
