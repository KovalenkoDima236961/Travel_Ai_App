package triphealth

import (
	"fmt"
	"sort"
)

var severityPenalty = map[Severity]int{
	SeverityCritical: 20,
	SeverityHigh:     12,
	SeverityWarning:  6,
	SeverityInfo:     2,
}

var categoryPenaltyCap = map[Category]int{
	CategoryItinerary:     25,
	CategoryRoute:         20,
	CategoryTransport:     20,
	CategoryBudget:        20,
	CategoryAvailability:  15,
	CategoryCollaboration: 15,
	CategoryChecklist:     12,
	CategoryReminders:     10,
	CategoryAccommodation: 10,
	CategoryExpenses:      10,
	CategoryPolicy:        25,
	CategoryApproval:      20,
	CategoryOffline:       8,
	CategoryDataQuality:   12,
	CategoryPublicShare:   8,
	CategoryOther:         8,
}

var categoryPriority = map[Category]int{
	CategoryPolicy:        1,
	CategoryItinerary:     2,
	CategoryRoute:         3,
	CategoryTransport:     4,
	CategoryBudget:        5,
	CategoryAvailability:  6,
	CategoryAccommodation: 7,
	CategoryChecklist:     8,
	CategoryReminders:     9,
	CategoryCollaboration: 10,
	CategoryExpenses:      11,
	CategoryApproval:      12,
	CategoryOffline:       13,
	CategoryDataQuality:   14,
	CategoryPublicShare:   15,
	CategoryOther:         16,
}

func ScoreIssues(issues []Issue) (int, []CategorySummary) {
	byCategory := map[Category]int{}
	counts := map[Category]int{}
	highest := map[Category]Severity{}
	for _, issue := range issues {
		if issue.Status != "" && issue.Status != StatusOpen {
			continue
		}
		penalty := severityPenalty[issue.Severity]
		if penalty == 0 {
			penalty = severityPenalty[SeverityWarning]
		}
		byCategory[issue.Category] += penalty
		counts[issue.Category]++
		if severityRank(issue.Severity) > severityRank(highest[issue.Category]) {
			highest[issue.Category] = issue.Severity
		}
	}

	totalPenalty := 0
	categories := make([]CategorySummary, 0, len(byCategory))
	for category, penalty := range byCategory {
		capped := minInt(penalty, categoryPenaltyCap[category])
		totalPenalty += capped
		categories = append(categories, CategorySummary{
			Category:        category,
			Score:           clamp(100-capped, 0, 100),
			OpenIssueCount:  counts[category],
			HighestSeverity: highest[category],
		})
	}
	sort.SliceStable(categories, func(i, j int) bool {
		return categorySortRank(categories[i].Category) < categorySortRank(categories[j].Category)
	})
	return clamp(100-totalPenalty, 0, 100), categories
}

func ReadinessLevel(score int, issues []Issue) Level {
	if score < 50 || hasBlockingPolicyViolation(issues) {
		return LevelNotReady
	}
	hasCritical := false
	hasHigh := false
	for _, issue := range issues {
		if issue.Status != "" && issue.Status != StatusOpen {
			continue
		}
		if issue.Severity == SeverityCritical {
			hasCritical = true
		}
		if issue.Severity == SeverityHigh {
			hasHigh = true
		}
	}
	switch {
	case score >= 90 && !hasCritical && !hasHigh:
		return LevelReady
	case score >= 75 && !hasCritical:
		return LevelAlmostReady
	default:
		return LevelNeedsAttention
	}
}

func TopFixes(issues []Issue, limit int) []TopFix {
	if limit <= 0 {
		return nil
	}
	candidates := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		if issue.Status != "" && issue.Status != StatusOpen {
			continue
		}
		if issue.Action == nil || issue.Action.Href == "" {
			continue
		}
		candidates = append(candidates, issue)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if severityRank(left.Severity) != severityRank(right.Severity) {
			return severityRank(left.Severity) > severityRank(right.Severity)
		}
		if categorySortRank(left.Category) != categorySortRank(right.Category) {
			return categorySortRank(left.Category) < categorySortRank(right.Category)
		}
		if (left.Action != nil) != (right.Action != nil) {
			return left.Action != nil
		}
		return left.ID < right.ID
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	out := make([]TopFix, 0, len(candidates))
	for _, issue := range candidates {
		label := issue.Action.Label
		if label == "" {
			label = issue.Title
		}
		out = append(out, TopFix{
			IssueID: issue.ID,
			Label:   label,
			Href:    issue.Action.Href,
		})
	}
	return out
}

func Summary(level Level, issues []Issue) string {
	if len(issues) == 0 {
		return "This trip looks ready."
	}
	highest := highestSeverity(issues)
	top := make([]Issue, 0, len(issues))
	for _, issue := range issues {
		if issue.Status != "" && issue.Status != StatusOpen {
			continue
		}
		if issue.Severity == highest {
			top = append(top, issue)
		}
	}
	sort.SliceStable(top, func(i, j int) bool {
		if categorySortRank(top[i].Category) != categorySortRank(top[j].Category) {
			return categorySortRank(top[i].Category) < categorySortRank(top[j].Category)
		}
		return top[i].ID < top[j].ID
	})
	labels := make([]string, 0, minInt(len(top), 3))
	for _, issue := range top {
		if len(labels) == 3 {
			break
		}
		labels = append(labels, issue.Title)
	}
	prefix := map[Level]string{
		LevelReady:          "Ready.",
		LevelAlmostReady:    "Almost ready.",
		LevelNeedsAttention: "Needs attention.",
		LevelNotReady:       "Not ready.",
	}[level]
	if len(labels) == 0 {
		return prefix
	}
	return fmt.Sprintf("%s Fix %s.", prefix, joinHuman(labels))
}

func severityRank(severity Severity) int {
	switch severity {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityWarning:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}

func highestSeverity(issues []Issue) Severity {
	highest := SeverityInfo
	for _, issue := range issues {
		if issue.Status != "" && issue.Status != StatusOpen {
			continue
		}
		if severityRank(issue.Severity) > severityRank(highest) {
			highest = issue.Severity
		}
	}
	return highest
}

func categorySortRank(category Category) int {
	if rank, ok := categoryPriority[category]; ok {
		return rank
	}
	return 100
}

func hasBlockingPolicyViolation(issues []Issue) bool {
	for _, issue := range issues {
		if issue.Status != "" && issue.Status != StatusOpen {
			continue
		}
		if issue.ID == "policy_blocking_violation" || issue.Metadata["blockingPolicyViolation"] == true {
			return true
		}
	}
	return false
}

func joinHuman(values []string) string {
	switch len(values) {
	case 0:
		return ""
	case 1:
		return values[0]
	case 2:
		return values[0] + " and " + values[1]
	default:
		return values[0] + ", " + values[1] + ", and " + values[2]
	}
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
