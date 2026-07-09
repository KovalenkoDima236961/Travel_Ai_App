package triprepair

import (
	"strconv"
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/approvalrisk"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

func BuildIssues(
	policy workspacepolicies.Evaluation,
	risk approvalrisk.Response,
	payload JobPayload,
) []Issue {
	selectedPolicy := set(payload.SelectedIssueTypes)
	selectedRisk := set(payload.SelectedRiskFactorTypes)
	out := make([]Issue, 0)

	for _, result := range policy.Results {
		if result.Status == workspacepolicies.ResultPassed {
			continue
		}
		if len(selectedPolicy) > 0 && !selectedPolicy[strings.ToLower(result.RuleKey)] {
			continue
		}
		if !repairablePolicyIssue(result.RuleKey) {
			continue
		}
		if !includePolicyForMode(payload.RepairMode, result.RuleKey) && len(selectedPolicy) == 0 {
			continue
		}
		if len(result.AffectedItems) == 0 {
			out = append(out, Issue{
				Type:     result.RuleKey,
				Severity: string(result.Severity),
				Message:  result.Message,
			})
			continue
		}
		for _, affected := range result.AffectedItems {
			out = append(out, Issue{
				Type:     result.RuleKey,
				Severity: string(result.Severity),
				Message:  result.Message,
				Affected: &IssueAffected{
					DayNumber: affected.DayNumber,
					ItemIndex: affected.ItemIndex,
					Name:      affected.Name,
					Amount:    affected.Amount,
					Currency:  affected.Currency,
				},
			})
		}
	}

	for _, factor := range risk.Factors {
		if len(selectedRisk) > 0 && !selectedRisk[strings.ToLower(factor.Type)] {
			continue
		}
		if !repairableRiskFactor(factor.Type) {
			continue
		}
		if !includeRiskForMode(payload.RepairMode, factor.Type) && len(selectedRisk) == 0 {
			continue
		}
		if factor.Affected == nil || len(factor.Affected.AffectedItems) == 0 {
			out = append(out, Issue{
				Type:     factor.Type,
				Severity: string(factor.Severity),
				Message:  factor.Message,
			})
			continue
		}
		for _, affected := range factor.Affected.AffectedItems {
			out = append(out, Issue{
				Type:     factor.Type,
				Severity: string(factor.Severity),
				Message:  factor.Message,
				Affected: &IssueAffected{
					DayNumber: affected.DayNumber,
					ItemIndex: affected.ItemIndex,
					Name:      affected.Name,
					Amount:    affected.Amount,
					Currency:  affected.Currency,
				},
			})
		}
	}

	return dedupeIssues(out)
}

func repairablePolicyIssue(rule string) bool {
	switch strings.ToLower(strings.TrimSpace(rule)) {
	case "maxtripbudget",
		"maxdailybudget",
		"maxitemcost",
		"maxaccommodationtotal",
		"maxaccommodationpernight",
		"maxwalkingkmperday",
		"nolateactivitiesafter",
		"requiredresttimeperday",
		"preferredtransportmodes",
		"disallowedactivitytypes":
		return true
	default:
		return false
	}
}

func repairableRiskFactor(factor string) bool {
	switch strings.ToLower(strings.TrimSpace(factor)) {
	case "trip_budget_exceeded",
		"workspace_budget_exceeded",
		"daily_budget_exceeded",
		"high_item_cost",
		"late_activity",
		"dense_schedule",
		"rest_time_missing",
		"walking_distance_high",
		"accommodation_cost_high",
		"workspace_policy_blocking",
		"workspace_policy_warning":
		return true
	default:
		return strings.Contains(strings.ToLower(factor), "budget") ||
			strings.Contains(strings.ToLower(factor), "policy") ||
			strings.Contains(strings.ToLower(factor), "walking") ||
			strings.Contains(strings.ToLower(factor), "schedule")
	}
}

func includePolicyForMode(mode RepairMode, rule string) bool {
	mode = NormalizeRepairMode(mode)
	key := strings.ToLower(rule)
	switch mode {
	case RepairModePolicyCompliance:
		return true
	case RepairModeReduceBudgetRisk:
		return strings.Contains(key, "budget") || strings.Contains(key, "cost")
	case RepairModeFixScheduleRisk:
		return strings.Contains(key, "late")
	case RepairModeReduceWalking:
		return strings.Contains(key, "walking")
	case RepairModeAddRestTime:
		return strings.Contains(key, "rest")
	case RepairModeReplaceDisallowedItems:
		return strings.Contains(key, "disallowed") || strings.Contains(key, "transport")
	case RepairModeSelectedIssues:
		return false
	default:
		return true
	}
}

func includeRiskForMode(mode RepairMode, factor string) bool {
	mode = NormalizeRepairMode(mode)
	key := strings.ToLower(factor)
	switch mode {
	case RepairModePolicyCompliance:
		return true
	case RepairModeReduceBudgetRisk:
		return strings.Contains(key, "budget") || strings.Contains(key, "cost")
	case RepairModeFixScheduleRisk:
		return strings.Contains(key, "late") || strings.Contains(key, "schedule")
	case RepairModeReduceWalking:
		return strings.Contains(key, "walking")
	case RepairModeAddRestTime:
		return strings.Contains(key, "rest") || strings.Contains(key, "dense")
	case RepairModeReplaceDisallowedItems:
		return strings.Contains(key, "disallowed")
	case RepairModeSelectedIssues:
		return false
	default:
		return true
	}
}

func dedupeIssues(in []Issue) []Issue {
	out := make([]Issue, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, issue := range in {
		key := strings.ToLower(issue.Type) + "|" + issue.Message
		if issue.Affected != nil {
			key += "|"
			if issue.Affected.DayNumber != nil {
				key += strconv.Itoa(*issue.Affected.DayNumber)
			}
			key += ":"
			if issue.Affected.ItemIndex != nil {
				key += strconv.Itoa(*issue.Affected.ItemIndex)
			}
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, issue)
	}
	return out
}

func set(values []string) map[string]bool {
	out := map[string]bool{}
	for _, value := range values {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if trimmed != "" {
			out[trimmed] = true
		}
	}
	return out
}
