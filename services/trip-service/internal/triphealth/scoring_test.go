package triphealth

import "testing"

func TestScoreIssuesAppliesSeverityPenaltiesCategoryCapsAndIgnoresResolved(t *testing.T) {
	issues := []Issue{
		{ID: "policy_blocking_violation", Category: CategoryPolicy, Severity: SeverityCritical, Status: StatusOpen},
		{ID: "policy_warning_violation", Category: CategoryPolicy, Severity: SeverityHigh, Status: StatusOpen},
		{ID: "route_missing", Category: CategoryRoute, Severity: SeverityHigh, Status: StatusOpen},
		{ID: "route_resolved", Category: CategoryRoute, Severity: SeverityCritical, Status: StatusResolved},
		{ID: "checklist_missing", Category: CategoryChecklist, Severity: SeverityWarning, Status: StatusOpen},
	}

	score, categories := ScoreIssues(issues)

	if score != 57 {
		t.Fatalf("score = %d, want 57", score)
	}
	if len(categories) != 3 {
		t.Fatalf("categories length = %d, want 3", len(categories))
	}

	assertCategorySummary(t, categories[0], CategoryPolicy, 75, 2, SeverityCritical)
	assertCategorySummary(t, categories[1], CategoryRoute, 88, 1, SeverityHigh)
	assertCategorySummary(t, categories[2], CategoryChecklist, 94, 1, SeverityWarning)

	if got := ReadinessLevel(score, issues); got != LevelNotReady {
		t.Fatalf("readiness = %s, want %s", got, LevelNotReady)
	}
}

func TestTopFixesPrioritizesSeverityThenCategory(t *testing.T) {
	issues := []Issue{
		{
			ID:       "checklist",
			Category: CategoryChecklist,
			Severity: SeverityHigh,
			Status:   StatusOpen,
			Title:    "Checklist",
			Action:   &Action{Href: "/trips/1#checklist", Label: "Fix checklist"},
		},
		{
			ID:       "policy",
			Category: CategoryPolicy,
			Severity: SeverityHigh,
			Status:   StatusOpen,
			Title:    "Policy",
			Action:   &Action{Href: "/trips/1#policy", Label: "Fix policy"},
		},
		{
			ID:       "route",
			Category: CategoryRoute,
			Severity: SeverityCritical,
			Status:   StatusOpen,
			Title:    "Route",
			Action:   &Action{Href: "/trips/1#route", Label: "Fix route"},
		},
		{
			ID:       "hidden",
			Category: CategoryBudget,
			Severity: SeverityCritical,
			Status:   StatusResolved,
			Action:   &Action{Href: "/trips/1#budget", Label: "Hidden"},
		},
	}

	fixes := TopFixes(issues, 2)

	if len(fixes) != 2 {
		t.Fatalf("top fixes length = %d, want 2", len(fixes))
	}
	if fixes[0].IssueID != "route" || fixes[1].IssueID != "policy" {
		t.Fatalf("top fixes order = %#v, want route then policy", fixes)
	}
}

func assertCategorySummary(t *testing.T, got CategorySummary, category Category, score int, count int, severity Severity) {
	t.Helper()
	if got.Category != category {
		t.Fatalf("category = %s, want %s", got.Category, category)
	}
	if got.Score != score {
		t.Fatalf("%s score = %d, want %d", category, got.Score, score)
	}
	if got.OpenIssueCount != count {
		t.Fatalf("%s open count = %d, want %d", category, got.OpenIssueCount, count)
	}
	if got.HighestSeverity != severity {
		t.Fatalf("%s severity = %s, want %s", category, got.HighestSeverity, severity)
	}
}
