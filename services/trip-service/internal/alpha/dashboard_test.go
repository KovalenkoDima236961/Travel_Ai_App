package alpha

import (
	"strings"
	"testing"
	"time"
)

func TestDeriveAlertsCoversAlphaRiskSignals(t *testing.T) {
	alerts := deriveAlerts(Dashboard{
		Users: UserMetrics{Invited: 12, Retained: 2},
		AI: AIMetrics{
			Generations:      10,
			SuccessRate:      0.6,
			FallbackRate:     0.3,
			AverageLatencyMS: 45000,
		},
		Feedback: FeedbackMetrics{BugReports: 5},
		Costs:    CostMetrics{EstimatedOpenAICost: 30},
	})

	want := map[string]bool{
		"ai_failure_rate_spike":       false,
		"fallback_rate_spike":         false,
		"bug_reports_spike":           false,
		"openai_cost_spike":           false,
		"retention_drop":              false,
		"generation_latency_increase": false,
	}
	for _, alert := range alerts {
		if _, ok := want[alert.Type]; ok {
			want[alert.Type] = true
		}
	}
	for alertType, seen := range want {
		if !seen {
			t.Fatalf("deriveAlerts() missing %s in %#v", alertType, alerts)
		}
	}
}

func TestBuildWeeklyMarkdownIncludesAlphaDecisionInputs(t *testing.T) {
	start := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 7)
	report := buildWeeklyMarkdown(start, end, Dashboard{
		Users:    UserMetrics{Invited: 20, Active: 8, Retained: 5},
		Trips:    TripMetrics{Created: 9},
		AI:       AIMetrics{Generations: 12, SuccessRate: 0.75, RepairRate: 0.25, FallbackRate: 0.1, AverageLatencyMS: 18000},
		Usage:    UsageMetrics{DAU: 3, WAU: 8, MAU: 11},
		Costs:    CostMetrics{EstimatedOpenAICost: 4.25},
		Feedback: FeedbackMetrics{BugReports: 2, FeatureRequests: 1},
		Alerts:   []AlphaAlert{{Type: "ai_failure_rate_spike", Message: "AI failure rate is above 20%."}},
	}, []string{"Share modal copy failure"}, []string{"Calendar export"}, []string{"Museum cafe"}, []string{"Lisbon"})

	for _, fragment := range []string{
		"# Weekly Alpha Report",
		"Period: 2026-07-20 to 2026-07-26",
		"AI generations: 12, success rate: 75%",
		"Top bugs",
		"Calendar export",
		"Most removed AI places",
		"Retention rate: 25%",
		"ai_failure_rate_spike",
	} {
		if !strings.Contains(report, fragment) {
			t.Fatalf("weekly report missing %q:\n%s", fragment, report)
		}
	}
}
