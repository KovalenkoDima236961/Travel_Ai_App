package copilot

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/approvalrisk"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetconfidence"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/groupreadiness"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/personalization"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/triphealth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

// BuildSafeContext rebuilds context server-side. ClientContext is only used to
// focus an already-authorized route leg/day; it never supplies trip facts.
func BuildSafeContext(
	ctx context.Context,
	svc *service.Service,
	tripID uuid.UUID,
	client ClientContext,
) (SafeContext, service.TripAccess, error) {
	trip, access, err := svc.GetWithAccess(ctx, tripID)
	if err != nil {
		return SafeContext{}, access, err
	}

	out := SafeContext{
		Trip: map[string]any{
			"title":       safeText(trip.Destination, 160),
			"destination": safeText(trip.Destination, 160),
			"days":        trip.Days,
			"tripType":    safeText(trip.TripType, 64),
			"accessRole":  access.Role(),
		},
		Unavailable: []string{},
	}
	if trip.StartDate != nil {
		out.Trip["startDate"] = trip.StartDate.Format("2006-01-02")
	}

	if summary, loadErr := svc.GetCommandCenterSummary(ctx, tripID); loadErr == nil {
		out.CommandCenter = map[string]any{
			"health": map[string]any{
				"score":              summaryValue(summary.Health, func(value *service.CommandCenterHealthSummary) int { return value.Score }),
				"level":              summaryString(summary.Health, func(value *service.CommandCenterHealthSummary) string { return string(value.Level) }),
				"criticalIssueCount": summaryValue(summary.Health, func(value *service.CommandCenterHealthSummary) int { return value.CriticalIssueCount }),
				"highIssueCount":     summaryValue(summary.Health, func(value *service.CommandCenterHealthSummary) int { return value.HighIssueCount }),
			},
			"budget": map[string]any{
				"score":          summaryValue(summary.Budget, func(value *service.CommandCenterBudgetSummary) int { return value.ConfidenceScore }),
				"level":          summaryString(summary.Budget, func(value *service.CommandCenterBudgetSummary) string { return string(value.ConfidenceLevel) }),
				"riskLevel":      summaryString(summary.Budget, func(value *service.CommandCenterBudgetSummary) string { return string(value.RiskLevel) }),
				"budgetExceeded": summaryBool(summary.Budget, func(value *service.CommandCenterBudgetSummary) bool { return value.BudgetExceeded }),
				"missingCount":   summaryValue(summary.Budget, func(value *service.CommandCenterBudgetSummary) int { return value.MissingCount }),
			},
			"group": map[string]any{
				"score":                   summaryValue(summary.GroupReadiness, func(value *service.CommandCenterGroupSummary) int { return value.Score }),
				"level":                   summaryString(summary.GroupReadiness, func(value *service.CommandCenterGroupSummary) string { return string(value.Level) }),
				"membersNeedingAttention": summaryValue(summary.GroupReadiness, func(value *service.CommandCenterGroupSummary) int { return value.MembersNeedingAttention }),
			},
			"realWorldReadiness": map[string]any{
				"score":         summaryValue(summary.RealWorldReadiness, func(value *service.CommandCenterVerificationSummary) int { return value.Score }),
				"level":         summaryString(summary.RealWorldReadiness, func(value *service.CommandCenterVerificationSummary) string { return string(value.Level) }),
				"topIssueCount": summaryValue(summary.RealWorldReadiness, func(value *service.CommandCenterVerificationSummary) int { return value.TopIssueCount }),
				"staleCount":    summaryValue(summary.RealWorldReadiness, func(value *service.CommandCenterVerificationSummary) int { return value.StaleCount }),
				"missingCount":  summaryValue(summary.RealWorldReadiness, func(value *service.CommandCenterVerificationSummary) int { return value.MissingCount }),
			},
			"activity": map[string]any{
				"recentCount": summaryValue(summary.Activity, func(value *service.CommandCenterActivitySummary) int { return value.RecentCount }),
				"hasRecent":   summary.Activity != nil && summary.Activity.RecentCount > 0,
			},
			"route": map[string]any{
				"stopCount":                 summary.Route.StopCount,
				"legCount":                  summary.Route.LegCount,
				"selectedTransportCoverage": summary.Route.SelectedTransportCoverage,
				"missingTransportCount":     summary.Route.MissingTransportCount,
			},
		}
		if summary.Checklist != nil {
			out.Checklist = map[string]any{
				"completedCount":    summary.Checklist.CompletedCount,
				"totalCount":        summary.Checklist.TotalCount,
				"overdueCount":      summary.Checklist.OverdueCount,
				"highPriorityCount": summary.Checklist.HighPriorityCount,
			}
		}
		if summary.Reminders != nil {
			out.Reminders = map[string]any{
				"totalCount":   summary.Reminders.TotalCount,
				"overdueCount": summary.Reminders.OverdueCount,
				"dueSoonCount": summary.Reminders.DueSoonCount,
			}
		}
		if summary.Expenses != nil {
			out.Expenses = map[string]any{
				"expenseCount":           summary.Expenses.ExpenseCount,
				"actualTotal":            summary.Expenses.ActualTotal,
				"pendingSettlementCount": summary.Expenses.PendingSettlementCount,
			}
		}
		for _, sectionErr := range summary.SectionErrors {
			out.Unavailable = append(out.Unavailable, safeText(sectionErr.Code, 80))
		}
	} else {
		out.Unavailable = append(out.Unavailable, "command_center_unavailable")
	}
	if readiness, loadErr := svc.GetTripVerification(ctx, tripID); loadErr == nil {
		out.Verification = map[string]any{
			"score":         readiness.Score,
			"level":         string(readiness.Level),
			"topIssueCount": len(readiness.TopIssues),
			"staleCount":    readiness.Summary.StaleCount,
			"missingCount":  readiness.Summary.MissingCount,
		}
	} else {
		out.Unavailable = append(out.Unavailable, "verification_unavailable")
	}

	if health, loadErr := svc.GetTripHealth(ctx, tripID, triphealth.Options{}); loadErr == nil {
		out.Health = safeHealth(health)
	} else {
		out.Unavailable = append(out.Unavailable, "trip_health_unavailable")
	}
	if budget, loadErr := svc.GetBudgetConfidence(ctx, tripID, budgetconfidence.Options{Currency: trip.BudgetCurrency}); loadErr == nil {
		out.Budget = safeBudget(budget)
	} else {
		out.Unavailable = append(out.Unavailable, "budget_confidence_unavailable")
	}
	if readiness, loadErr := svc.GetGroupReadiness(ctx, tripID, groupreadiness.Options{}); loadErr == nil {
		out.Group = safeGroup(readiness)
	} else {
		out.Unavailable = append(out.Unavailable, "group_readiness_unavailable")
	}

	out.Route = safeRoute(trip, client)
	out.Itinerary = safeItinerary(trip.Itinerary, client)
	if client.CurrentTab == "travel_day" {
		if travelDay, loadErr := svc.GetTravelDay(ctx, tripID, client.Date); loadErr == nil {
			out.TravelDay = safeTravelDay(travelDay)
		} else {
			out.Unavailable = append(out.Unavailable, "travel_day_unavailable")
		}
	}

	if approval, loadErr := svc.GetTripApproval(ctx, tripID); loadErr == nil {
		out.Approval = map[string]any{
			"status":            safeText(approval.Status, 80),
			"canSubmit":         approval.CanSubmit,
			"canApprove":        approval.CanApprove,
			"canRequestChanges": approval.CanRequestChanges,
		}
	} else {
		out.Unavailable = append(out.Unavailable, "approval_status_unavailable")
	}
	if policy, loadErr := svc.EvaluateTripPolicy(ctx, tripID); loadErr == nil {
		out.Policy = map[string]any{
			"status":          safeText(string(policy.Status), 80),
			"warningCount":    policy.Summary.WarningCount,
			"blockingCount":   policy.Summary.BlockingCount,
			"notApplicable":   policy.NotApplicableReason != nil,
			"topResultTitles": safePolicyTitles(policy.Results),
		}
	} else {
		out.Unavailable = append(out.Unavailable, "policy_evaluation_unavailable")
	}
	if risk, loadErr := svc.GetTripApprovalRisk(ctx, tripID); loadErr == nil {
		appendApprovalRisk(out.Approval, risk)
	} else {
		out.Unavailable = append(out.Unavailable, "approval_risk_unavailable")
	}
	if generation := safeGenerationQuality(trip.CreationMetadata); generation != nil {
		out.Generation = generation
	}
	if profile, loadErr := svc.GetPersonalizationContext(ctx, personalization.SourceCommandCenter, &tripID); loadErr == nil {
		summary := profile.PlanningSummary()
		out.Personalization = map[string]any{
			"completenessScore": summary.CompletenessScore,
			"travelStyles":      safeTextSlice(summary.TravelStyles, 6, 48),
			"transportBias":     safeTextSlice(summary.TransportBias, 5, 48),
			"activityBias":      safeTextSlice(summary.ActivityBias, 5, 48),
			"budgetComfort":     safeText(summary.BudgetComfort, 48),
			"walkingTolerance":  safeText(summary.WalkingTolerance, 48),
		}
	} else {
		out.Unavailable = append(out.Unavailable, "personalization_unavailable")
	}

	out.Unavailable = uniqueStrings(out.Unavailable)
	return out, access, nil
}

func safeTravelDay(summary service.TravelDaySummary) map[string]any {
	items := make([]map[string]any, 0, min(3, len(summary.Timeline)))
	for _, item := range summary.Timeline {
		if len(items) == 3 {
			break
		}
		items = append(items, map[string]any{
			"time":   safeText(item.StartTime, 16),
			"title":  safeText(item.Title, 160),
			"type":   safeText(item.Type, 48),
			"status": safeText(item.TravelStatus.Status, 24),
		})
	}
	return map[string]any{
		"date":              summary.Date,
		"dayNumber":         summary.DayNumber,
		"mode":              safeText(summary.Mode, 32),
		"todayTitle":        safeText(summary.Today.Title, 160),
		"currentItem":       safeTravelDayItem(summary.NowNext.CurrentItem),
		"nextItem":          safeTravelDayItem(summary.NowNext.NextItem),
		"upcomingItems":     items,
		"warningCount":      len(summary.Verification.TopWarnings) + len(summary.Weather.Warnings),
		"checklistDueCount": len(summary.Checklist.DueToday) + len(summary.Checklist.Overdue),
		"reminderDueCount":  len(summary.Reminders.DueToday) + len(summary.Reminders.Overdue),
	}
}

func safeTravelDayItem(item *service.TravelDayTimelineItem) map[string]any {
	if item == nil {
		return nil
	}
	return map[string]any{
		"time":   safeText(item.StartTime, 16),
		"title":  safeText(item.Title, 160),
		"type":   safeText(item.Type, 48),
		"status": safeText(item.TravelStatus.Status, 24),
	}
}

func safeHealth(health triphealth.Response) map[string]any {
	issues := make([]map[string]any, 0, min(5, len(health.Issues)))
	for _, issue := range health.Issues {
		if issue.Status != triphealth.StatusOpen || len(issues) == 5 {
			continue
		}
		issues = append(issues, map[string]any{
			"category":       string(issue.Category),
			"severity":       string(issue.Severity),
			"title":          safeText(issue.Title, 180),
			"recommendation": safeText(issue.Recommendation, 240),
		})
	}
	fixes := make([]map[string]any, 0, min(3, len(health.TopFixes)))
	for _, fix := range health.TopFixes {
		if len(fixes) == 3 {
			break
		}
		fixes = append(fixes, map[string]any{"label": safeText(fix.Label, 160)})
	}
	return map[string]any{
		"score":     health.Score,
		"level":     string(health.Level),
		"summary":   safeText(health.Summary, 300),
		"topIssues": issues,
		"topFixes":  fixes,
	}
}

func summaryValue[T any](value *T, read func(*T) int) int {
	if value == nil {
		return 0
	}
	return read(value)
}

func summaryString[T any](value *T, read func(*T) string) string {
	if value == nil {
		return "unknown"
	}
	return safeText(read(value), 80)
}

func summaryBool[T any](value *T, read func(*T) bool) bool {
	return value != nil && read(value)
}

func safeBudget(budget budgetconfidence.Response) map[string]any {
	issues := make([]map[string]any, 0, min(5, len(budget.Issues)))
	for _, issue := range budget.Issues {
		if len(issues) == 5 {
			break
		}
		issues = append(issues, map[string]any{
			"severity":       string(issue.Severity),
			"category":       string(issue.Category),
			"title":          safeText(issue.Title, 180),
			"recommendation": safeText(issue.Recommendation, 240),
		})
	}
	recommendations := make([]string, 0, min(3, len(budget.Recommendations)))
	for _, recommendation := range budget.Recommendations {
		if len(recommendations) == 3 {
			break
		}
		recommendations = append(recommendations, safeText(recommendation.Label, 180))
	}
	return map[string]any{
		"score":           budget.Score,
		"level":           string(budget.Level),
		"riskLevel":       string(budget.RiskLevel),
		"summary":         safeText(budget.Summary, 300),
		"currency":        safeText(budget.Currency, 12),
		"estimatedTotal":  budget.EstimatedTotal,
		"actualTotal":     budget.ActualTotal,
		"coverage":        budget.Coverage.Overall,
		"issues":          issues,
		"recommendations": recommendations,
	}
}

func safeGroup(readiness groupreadiness.Response) map[string]any {
	attention := 0
	for _, member := range readiness.Members {
		if member.Level != groupreadiness.LevelReady {
			attention++
		}
	}
	categories := make([]string, 0, 4)
	for _, summary := range readiness.CategorySummary {
		if summary.OpenIssueCount > 0 && len(categories) < 4 {
			categories = append(categories, string(summary.Category))
		}
	}
	return map[string]any{
		"score":                   readiness.Score,
		"level":                   string(readiness.Level),
		"summary":                 safeText(readiness.Summary, 300),
		"memberCount":             len(readiness.Members),
		"membersNeedingAttention": attention,
		"attentionCategories":     categories,
	}
}

func safeRoute(trip *entity.Trip, client ClientContext) map[string]any {
	if trip == nil || trip.Route == nil {
		return map[string]any{"stopCount": 0, "legCount": 0, "missingTransportCount": 0}
	}
	legs := make([]map[string]any, 0, min(8, len(trip.Route.Legs)))
	missing := 0
	for _, leg := range trip.Route.Legs {
		if leg.SelectedTransportOption == nil {
			missing++
		}
		if len(legs) == 8 {
			continue
		}
		item := map[string]any{
			"id":                   safeText(leg.ID, 128),
			"from":                 safeText(leg.FromName, 120),
			"to":                   safeText(leg.ToName, 120),
			"mode":                 safeText(leg.Mode, 40),
			"hasSelectedTransport": leg.SelectedTransportOption != nil,
		}
		if leg.SelectedTransportOption != nil {
			item["selectedTransport"] = map[string]any{
				"mode":       safeText(leg.SelectedTransportOption.Mode, 40),
				"provider":   safeText(leg.SelectedTransportOption.Provider, 80),
				"status":     safeText(leg.SelectedTransportOption.Status, 80),
				"confidence": safeText(leg.SelectedTransportOption.Confidence, 40),
			}
		}
		legs = append(legs, item)
	}
	out := map[string]any{
		"stopCount":             len(trip.Route.Stops),
		"legCount":              len(trip.Route.Legs),
		"missingTransportCount": missing,
		"legs":                  legs,
	}
	if safeIdentifier(client.SelectedRouteLegID) {
		for _, leg := range legs {
			if leg["id"] == client.SelectedRouteLegID {
				out["selectedLeg"] = leg
				break
			}
		}
	}
	return out
}

func safeItinerary(raw json.RawMessage, client ClientContext) map[string]any {
	var payload struct {
		Days []struct {
			Day   int               `json:"day"`
			Items []json.RawMessage `json:"items"`
		} `json:"days"`
	}
	if len(raw) == 0 || json.Unmarshal(raw, &payload) != nil {
		return map[string]any{"dayCount": 0, "itemCount": 0}
	}
	itemCount := 0
	selectedCount := 0
	for _, day := range payload.Days {
		itemCount += len(day.Items)
		if client.SelectedDayNumber != nil && day.Day == *client.SelectedDayNumber {
			selectedCount = len(day.Items)
		}
	}
	out := map[string]any{"dayCount": len(payload.Days), "itemCount": itemCount}
	if client.SelectedDayNumber != nil && *client.SelectedDayNumber > 0 {
		out["selectedDay"] = map[string]any{
			"dayNumber": *client.SelectedDayNumber,
			"itemCount": selectedCount,
		}
	}
	return out
}

func appendApprovalRisk(approval map[string]any, risk approvalrisk.Response) {
	if approval == nil {
		return
	}
	approval["riskLevel"] = string(risk.Status)
	if risk.Score != nil {
		approval["riskScore"] = *risk.Score
	}
	reasons := make([]string, 0, min(3, len(risk.TopReasons)))
	for _, reason := range risk.TopReasons {
		if len(reasons) == 3 {
			break
		}
		reasons = append(reasons, safeText(reason, 220))
	}
	approval["topReasons"] = reasons
}

func safePolicyTitles(results []workspacepolicies.EvaluationResult) []string {
	titles := make([]string, 0, min(4, len(results)))
	for _, result := range results {
		if len(titles) == 4 {
			break
		}
		if result.Status == workspacepolicies.ResultViolation || result.Status == workspacepolicies.ResultWarningUnknown {
			titles = append(titles, safeText(result.Title, 180))
		}
	}
	return titles
}

func safeGenerationQuality(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}
	raw, ok := metadata["generationQuality"]
	if !ok || raw == nil {
		return nil
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var value map[string]any
	if json.Unmarshal(encoded, &value) != nil {
		return nil
	}
	out := map[string]any{}
	if status, ok := value["status"].(string); ok {
		out["status"] = safeText(status, 80)
	}
	if attempts, ok := safeNumber(value["repairAttempts"]); ok {
		out["repairAttempts"] = attempts
	}
	if issues, ok := value["remainingIssues"].([]any); ok {
		out["remainingIssueCount"] = len(issues)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func safeNumber(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	default:
		return 0, false
	}
}

func safeText(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit > 0 && len(value) > limit {
		return value[:limit] + "…"
	}
	return value
}

func safeTextSlice(values []string, maxItems, maxChars int) []string {
	out := make([]string, 0, min(maxItems, len(values)))
	for _, value := range values {
		value = safeText(value, maxChars)
		if value == "" {
			continue
		}
		out = append(out, value)
		if len(out) == maxItems {
			break
		}
	}
	return out
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
