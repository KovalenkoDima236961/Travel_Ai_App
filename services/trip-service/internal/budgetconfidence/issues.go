package budgetconfidence

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func detectIssues(
	tripID uuid.UUID,
	in Input,
	records []costRecord,
	coverage Coverage,
	planned PlannedVsActual,
	estimatedTotal float64,
	actualTotal float64,
	collection collectionResult,
	preliminaryScore int,
) []Issue {
	issues := make([]Issue, 0)
	tripBudget := tripBudgetMoney(in.Trip, coverageCurrency(in, records))

	if tripBudget == nil {
		issues = append(issues, issue(
			"budget_missing",
			SeverityWarning,
			IssueCategoryBudgetLimit,
			"Budget missing",
			"No trip budget is set.",
			"Set a trip budget so estimates and actual expenses have a target.",
			action(tripID, "Set budget", "budget"),
		))
	} else if tripBudget.Amount > 0 {
		if estimatedTotal > tripBudget.Amount {
			overPercent := (estimatedTotal - tripBudget.Amount) / tripBudget.Amount
			issues = append(issues, issue(
				"budget_exceeded_estimated",
				severityForOverBudget(overPercent),
				IssueCategoryBudgetLimit,
				"Estimated budget exceeded",
				fmt.Sprintf("Estimated costs are %.0f%% over the trip budget.", overPercent*100),
				"Reduce planned costs, add cheaper alternatives, or update the budget.",
				action(tripID, "Review budget", "budget"),
			))
		}
		if actualTotal > tripBudget.Amount {
			overPercent := (actualTotal - tripBudget.Amount) / tripBudget.Amount
			issues = append(issues, issue(
				"budget_exceeded_actual",
				severityForOverBudget(overPercent),
				IssueCategoryActualSpend,
				"Actual budget exceeded",
				fmt.Sprintf("Recorded actual expenses are %.0f%% over the trip budget.", overPercent*100),
				"Review expenses and update the budget or spending plan.",
				action(tripID, "Review expenses", "expenses"),
			))
		}
		if actualTotal > 0 &&
			(actualTotal/tripBudget.Amount)*100 >= normalizedConfig(in.Config).ActualSpendHighThresholdPercent &&
			isTripInProgressOrFuture(in.Trip, in.Now) {
			issues = append(issues, issue(
				"actual_spend_high_before_trip_end",
				SeverityHigh,
				IssueCategoryActualSpend,
				"Actual spending is high",
				"Actual expenses already use most of the trip budget.",
				"Review recent expenses before adding more planned costs.",
				action(tripID, "Review expenses", "expenses"),
			))
		}
	}

	missingAccommodation := countRecords(records, func(record costRecord) bool {
		return record.Missing && record.Category == CategoryAccommodation
	})
	if missingAccommodation > 0 {
		issues = append(issues, issue(
			"missing_accommodation_cost",
			SeverityHigh,
			IssueCategoryAccommodation,
			"Accommodation cost is missing",
			"Overnight trips need an accommodation estimate for a reliable budget.",
			"Add accommodation cost.",
			action(tripID, "Add accommodation cost", "accommodation"),
		))
	}

	missingTransport := countRecords(records, func(record costRecord) bool {
		return record.Missing && record.Category == CategoryTransport
	})
	if missingTransport > 0 {
		severity := SeverityWarning
		if missingTransport >= 2 {
			severity = SeverityHigh
		}
		issues = append(issues, issue(
			"missing_transport_prices",
			severity,
			IssueCategoryTransport,
			"Transport prices are missing",
			fmt.Sprintf("%d route leg(s) do not have selected transport prices or estimates.", missingTransport),
			"Attach transport options or add route leg estimates.",
			action(tripID, "Confirm transport prices", "route"),
		))
	}

	mockTransport := countRecords(records, func(record costRecord) bool {
		return record.Category == CategoryTransport &&
			(record.Source == SourceMockEstimate || record.Source == SourceSelectedTransportOptionLowConfidence)
	})
	if mockTransport > 0 {
		issues = append(issues, issue(
			"transport_mock_prices",
			SeverityWarning,
			IssueCategoryProviderConfidence,
			"Transport prices need confirmation",
			"Some selected transport prices are mock or low-confidence estimates.",
			"Confirm transport prices before booking.",
			action(tripID, "Find transport", "route"),
		))
	}

	missingActivities := countRecords(records, func(record costRecord) bool {
		return record.Missing && (record.Category == CategoryActivities || record.Category == CategoryTickets)
	})
	if missingActivities > 0 {
		severity := SeverityWarning
		if missingActivities >= 3 {
			severity = SeverityHigh
		}
		issues = append(issues, issue(
			"missing_activity_prices",
			severity,
			IssueCategoryActivities,
			"Activity prices are missing",
			fmt.Sprintf("%d ticketed or paid activity item(s) have no price estimate.", missingActivities),
			"Add missing activity and ticket prices.",
			action(tripID, "Add missing activity prices", "budget"),
		))
	}

	foodEstimates := 0
	foodAI := 0
	foodActual := 0
	for _, record := range records {
		if record.Category != CategoryFood && record.Category != CategoryGroceries {
			continue
		}
		if record.IsActual {
			foodActual++
		}
		if record.IsEstimate && !record.Missing {
			foodEstimates++
			if record.Source == SourceAIEstimateHighConfidence ||
				record.Source == SourceAIEstimateMediumConfidence ||
				record.Source == SourceAIEstimateLowConfidence {
				foodAI++
			}
		}
	}
	if foodEstimates > 0 && foodAI*2 >= foodEstimates {
		issues = append(issues, issue(
			"food_costs_ai_estimated",
			SeverityWarning,
			IssueCategoryFood,
			"Food costs are mostly AI-estimated",
			"Most food costs are based on AI estimates.",
			"Review or manually adjust food estimates.",
			action(tripID, "Review food budget", "budget"),
		))
	}
	if in.Trip != nil && in.Trip.Days > 1 && foodEstimates == 0 && foodActual == 0 {
		issues = append(issues, issue(
			"missing_food_budget",
			SeverityWarning,
			IssueCategoryFood,
			"Food budget is missing",
			"Multi-day trips usually need food or grocery estimates.",
			"Add a food budget estimate.",
			action(tripID, "Review food budget", "budget"),
		))
	}

	if collection.ConversionFailureCount > 0 {
		severity := SeverityWarning
		if collection.ConversionFailureCount >= 3 || collection.ConversionFailureAmount >= 250 {
			severity = SeverityHigh
		}
		issues = append(issues, issue(
			"currency_conversion_unavailable",
			severity,
			IssueCategoryCurrency,
			"Currency conversion unavailable",
			"Some costs could not be converted into the requested budget currency.",
			"Review currency conversion warnings.",
			action(tripID, "Review conversions", "budget"),
		))
	}
	if collection.ConversionApproxCount > 0 {
		issues = append(issues, issue(
			"currency_conversion_approximate",
			SeverityInfo,
			IssueCategoryCurrency,
			"Currency conversion is approximate",
			"Some converted costs use fallback or approximate exchange-rate data.",
			"Verify major foreign-currency prices before paying.",
			action(tripID, "Review conversions", "budget"),
		))
	}

	cfg := normalizedConfig(in.Config)
	for _, category := range planned.Categories {
		if category.DifferencePercent == nil || category.Actual.Amount <= 0 {
			continue
		}
		gap := absFloat(*category.DifferencePercent)
		if gap < cfg.PlannedActualGapWarningPercent {
			continue
		}
		severity := SeverityWarning
		if gap >= cfg.PlannedActualGapHighPercent {
			severity = SeverityHigh
		}
		issues = append(issues, issue(
			"planned_actual_gap:"+string(category.Category),
			severity,
			IssueCategoryActualSpend,
			"Actual spending differs from estimate",
			fmt.Sprintf("%s actual spending differs from the estimate by %.0f%%.", category.Category, gap),
			"Review estimates or recorded expenses in this category.",
			action(tripID, "Review spending", "expenses"),
		))
	}

	unlinkedReceipts := 0
	for _, receipt := range in.Receipts {
		if receipt.DeletedAt == nil && receipt.ExpenseID == nil {
			unlinkedReceipts++
		}
	}
	if unlinkedReceipts > 0 {
		issues = append(issues, issue(
			"receipt_not_linked",
			SeverityInfo,
			IssueCategoryDataQuality,
			"Receipt is not linked to an expense",
			fmt.Sprintf("%d uploaded receipt(s) have not been converted or linked to an expense.", unlinkedReceipts),
			"Convert receipt data into an expense when it represents real spending.",
			action(tripID, "Review receipts", "expenses"),
		))
	}

	largeWithoutReceipt := 0
	receiptBacked := map[uuid.UUID]bool{}
	for _, record := range records {
		if record.ReceiptBacked {
			receiptBacked[record.ExpenseID] = true
		}
	}
	for _, expense := range in.Expenses {
		if expense.Status == entity.ExpenseStatusDeleted {
			continue
		}
		if expense.Amount >= cfg.LargeExpenseReceiptThreshold && !receiptBacked[expense.ID] {
			largeWithoutReceipt++
		}
	}
	if largeWithoutReceipt > 0 {
		issues = append(issues, issue(
			"large_expense_without_receipt",
			SeverityInfo,
			IssueCategoryDataQuality,
			"Large expense has no receipt",
			fmt.Sprintf("%d large expense(s) do not have an attached receipt.", largeWithoutReceipt),
			"Attach receipts to improve actual-spend confidence.",
			action(tripID, "Link receipt", "expenses"),
		))
	}

	if in.ExpenseLoadFailed || in.ReceiptLoadFailed || in.BudgetSummaryLoadFailed {
		issues = append(issues, issue(
			"budget_confidence_data_partial",
			SeverityWarning,
			IssueCategoryDataQuality,
			"Budget confidence is partially evaluated",
			"One or more budget confidence data sources could not be loaded.",
			"Retry after the subsystem is available.",
			action(tripID, "Review budget", "budget"),
		))
	}

	if preliminaryScore < 55 {
		severity := SeverityWarning
		if preliminaryScore < 30 {
			severity = SeverityHigh
		}
		issues = append(issues, issue(
			"low_overall_confidence",
			severity,
			IssueCategoryCoverage,
			"Budget confidence is low",
			"Coverage, source quality, or actual-vs-planned data are not strong enough yet.",
			"Improve budget confidence by confirming major costs first.",
			action(tripID, "Improve budget confidence", "budget"),
		))
	}

	sort.SliceStable(issues, func(i, j int) bool {
		if severityRank(issues[i].Severity) != severityRank(issues[j].Severity) {
			return severityRank(issues[i].Severity) > severityRank(issues[j].Severity)
		}
		return issues[i].ID < issues[j].ID
	})
	return dedupeIssues(issues)
}

func buildRecommendations(tripID uuid.UUID, issues []Issue) []Recommendation {
	byID := map[string]Recommendation{}
	for _, issue := range issues {
		switch issue.ID {
		case "missing_accommodation_cost":
			byID["add_accommodation_cost"] = recommendation("add_accommodation_cost", "Add accommodation cost", "Accommodation is a major trip cost and is currently missing.", tripID, "accommodation", PriorityHigh)
		case "missing_transport_prices":
			byID["attach_transport_options"] = recommendation("attach_transport_options", "Attach transport options", "Route legs without selected prices lower budget confidence.", tripID, "route", PriorityHigh)
		case "transport_mock_prices":
			byID["confirm_transport_prices"] = recommendation("confirm_transport_prices", "Confirm transport prices", "Replace mock or low-confidence transport prices with real provider options.", tripID, "route", PriorityMedium)
		case "missing_activity_prices":
			byID["add_missing_activity_prices"] = recommendation("add_missing_activity_prices", "Add missing activity prices", "Ticketed activities without prices can understate the budget.", tripID, "budget", PriorityMedium)
		case "food_costs_ai_estimated", "missing_food_budget":
			byID["review_food_budget"] = recommendation("review_food_budget", "Review food budget", "Food costs are missing or mostly AI-estimated.", tripID, "budget", PriorityMedium)
		case "currency_conversion_unavailable", "currency_conversion_approximate":
			byID["review_currency_conversion_warnings"] = recommendation("review_currency_conversion_warnings", "Review currency conversion warnings", "Some totals depend on approximate or unavailable conversion data.", tripID, "budget", PriorityMedium)
		case "receipt_not_linked":
			byID["convert_receipt_into_expense"] = recommendation("convert_receipt_into_expense", "Convert receipt into expense", "Uploaded receipts do not count as actual spend until linked or converted.", tripID, "expenses", PriorityLow)
		case "large_expense_without_receipt":
			byID["link_large_expense_receipt"] = recommendation("link_large_expense_receipt", "Link large expense receipt", "Receipt-backed actual expenses improve confidence.", tripID, "expenses", PriorityLow)
		case "budget_exceeded_estimated", "budget_exceeded_actual", "actual_spend_high_before_trip_end":
			byID["adjust_budget"] = recommendation("adjust_budget", "Adjust budget", "The current planned or actual spending is above the budget target.", tripID, "budget", PriorityHigh)
			byID["run_budget_optimization"] = recommendation("run_budget_optimization", "Run budget optimization", "Ask the planner for cheaper alternatives while preserving confirmed costs.", tripID, "budget", PriorityMedium)
		}
	}
	if len(byID) == 0 {
		return []Recommendation{recommendation("add_actual_expenses", "Add actual expenses", "Actual expenses improve budget accuracy once spending starts.", tripID, "expenses", PriorityLow)}
	}
	out := make([]Recommendation, 0, len(byID))
	for _, rec := range byID {
		out = append(out, rec)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if priorityRank(out[i].Priority) != priorityRank(out[j].Priority) {
			return priorityRank(out[i].Priority) > priorityRank(out[j].Priority)
		}
		return out[i].ID < out[j].ID
	})
	if len(out) > 8 {
		return out[:8]
	}
	return out
}

func issue(id string, severity IssueSeverity, category IssueCategory, title, description, rec string, action *Action) Issue {
	return Issue{
		ID:             id,
		Severity:       severity,
		Category:       category,
		Title:          title,
		Description:    description,
		Recommendation: rec,
		Action:         action,
	}
}

func action(tripID uuid.UUID, label string, tab string) *Action {
	return &Action{
		Label: label,
		Href:  fmt.Sprintf("/trips/%s?tab=%s", tripID.String(), tab),
	}
}

func recommendation(id, label, description string, tripID uuid.UUID, tab string, priority RecommendationPriority) Recommendation {
	return Recommendation{
		ID:          id,
		Label:       label,
		Description: description,
		Href:        fmt.Sprintf("/trips/%s?tab=%s", tripID.String(), tab),
		Priority:    priority,
	}
}

func severityForOverBudget(overPercent float64) IssueSeverity {
	switch {
	case overPercent > 0.25:
		return SeverityCritical
	case overPercent > 0.10:
		return SeverityHigh
	default:
		return SeverityWarning
	}
}

func countRecords(records []costRecord, match func(costRecord) bool) int {
	count := 0
	for _, record := range records {
		if match(record) {
			count++
		}
	}
	return count
}

func dedupeIssues(issues []Issue) []Issue {
	seen := map[string]struct{}{}
	out := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		if _, ok := seen[issue.ID]; ok {
			continue
		}
		seen[issue.ID] = struct{}{}
		out = append(out, issue)
	}
	return out
}

func priorityRank(priority RecommendationPriority) int {
	switch priority {
	case PriorityHigh:
		return 3
	case PriorityMedium:
		return 2
	case PriorityLow:
		return 1
	default:
		return 0
	}
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func coverageCurrency(in Input, records []costRecord) string {
	if strings.TrimSpace(in.Currency) != "" {
		return currencyOrDefault(in.Currency, "EUR")
	}
	for _, record := range records {
		if record.Amount != nil && record.Amount.Currency != "" {
			return record.Amount.Currency
		}
	}
	return "EUR"
}
