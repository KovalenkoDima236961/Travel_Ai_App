package workspacepolicies

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/analytics"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func TestEvaluateBlockingBudgetAndItemRules(t *testing.T) {
	workspaceID := uuid.New()
	trip := &entity.Trip{
		ID:             uuid.New(),
		WorkspaceID:    &workspaceID,
		BudgetCurrency: "EUR",
		Days:           1,
	}
	cost := 180.0
	policy := policyWithRules(func(rules *Rules) {
		rules.RequireTripBudget = Rule{Enabled: true, Severity: SeverityWarning}
		rules.MaxTripBudget = MoneyRule{
			Rule: Rule{Enabled: true, Severity: SeverityBlocking}, Amount: 100, Currency: "EUR",
		}
		rules.MaxDailyBudget = MoneyRule{
			Rule: Rule{Enabled: true, Severity: SeverityWarning}, Amount: 90, Currency: "EUR",
		}
		rules.MaxItemCost = ItemCostRule{
			MoneyRule: MoneyRule{
				Rule:   Rule{Enabled: true, Severity: SeverityWarning},
				Amount: 100, Currency: "EUR",
			},
			Categories: []string{"activity"},
		}
	})
	evaluation := Evaluate(context.Background(), EvaluationInput{
		Trip:   trip,
		Policy: policy,
		Itinerary: aggregate.Itinerary{
			Currency: "EUR",
			Days: []aggregate.ItineraryDay{{
				Day: 1,
				Items: []aggregate.ItineraryItem{{
					Time: "10:00", Type: "activity", Name: "Private tour",
					EstimatedCost: &aggregate.EstimatedCost{
						Amount: &cost, Currency: "EUR", Category: "activity",
					},
				}},
			}},
		},
		AnalyticsByCurrency: map[string]analytics.TripCostAnalytics{
			"EUR": {
				Currency: "EUR",
				Summary:  analytics.CostAnalyticsSummary{EstimatedTotal: 180},
				ByDay:    []analytics.CostByDay{{DayNumber: 1, EstimatedTotal: 180}},
			},
		},
	})

	if evaluation.Status != EvaluationBlocking {
		t.Fatalf("status = %s, want blocking", evaluation.Status)
	}
	if evaluation.Summary.BlockingCount != 1 || evaluation.Summary.WarningCount != 3 {
		t.Fatalf("unexpected summary: %#v", evaluation.Summary)
	}
	assertResult(t, evaluation, "requireTripBudget", ResultViolation)
	assertResult(t, evaluation, "maxTripBudget", ResultViolation)
	assertResult(t, evaluation, "maxDailyBudget", ResultViolation)
	assertResult(t, evaluation, "maxItemCost", ResultViolation)
}

func TestEvaluateScheduleAvailabilityTransportAndDisallowedRules(t *testing.T) {
	workspaceID := uuid.New()
	trip := &entity.Trip{ID: uuid.New(), WorkspaceID: &workspaceID, BudgetCurrency: "EUR"}
	duration := 30
	policy := policyWithRules(func(rules *Rules) {
		rules.RequireAvailabilityForTicketedItems = Rule{Enabled: true, Severity: SeverityWarning}
		rules.NoLateActivitiesAfter = LateActivityRule{
			Rule: Rule{Enabled: true, Severity: SeverityWarning}, Time: "22:00",
		}
		rules.RequiredRestTimePerDay = RestTimeRule{
			Rule: Rule{Enabled: true, Severity: SeverityWarning}, Minutes: 60,
		}
		rules.PreferredTransportModes = TransportRule{
			Rule: Rule{Enabled: true, Severity: SeverityInfo}, Modes: []string{"walking"},
		}
		rules.DisallowedActivityTypes = ActivityTypesRule{
			Rule: Rule{Enabled: true, Severity: SeverityBlocking}, Types: []string{"gambling"},
		}
	})
	evaluation := Evaluate(context.Background(), EvaluationInput{
		Trip:   trip,
		Policy: policy,
		Itinerary: aggregate.Itinerary{Days: []aggregate.ItineraryDay{{
			Day: 1,
			Items: []aggregate.ItineraryItem{
				{Time: "23:00", Type: "gambling", Name: "Casino", Category: "gambling"},
				{Time: "10:00", Type: "museum", Name: "Museum"},
				{Time: "11:00", Type: "transport", TransportMode: "taxi", Name: "Taxi"},
				{
					Time: "14:00", EndTime: "14:30", Type: "rest", Name: "Break",
					DurationMinutes: &duration,
				},
			},
		}}},
	})

	if evaluation.Status != EvaluationBlocking {
		t.Fatalf("status = %s, want blocking", evaluation.Status)
	}
	for _, key := range []string{
		"requireAvailabilityForTicketedItems",
		"noLateActivitiesAfter",
		"requiredRestTimePerDay",
		"preferredTransportModes",
		"disallowedActivityTypes",
	} {
		assertResult(t, evaluation, key, ResultViolation)
	}
}

func TestEvaluateUnknownBlockingRuleDoesNotBlock(t *testing.T) {
	workspaceID := uuid.New()
	policy := policyWithRules(func(rules *Rules) {
		rules.MaxWalkingKmPerDay = WalkingRule{
			Rule: Rule{Enabled: true, Severity: SeverityBlocking}, Km: 10,
		}
	})
	evaluation := Evaluate(context.Background(), EvaluationInput{
		Trip:      &entity.Trip{ID: uuid.New(), WorkspaceID: &workspaceID},
		Policy:    policy,
		Itinerary: aggregate.Itinerary{Days: []aggregate.ItineraryDay{{Day: 1}}},
	})
	if evaluation.Status != EvaluationWarning {
		t.Fatalf("status = %s, want warning", evaluation.Status)
	}
	result := assertResult(t, evaluation, "maxWalkingKmPerDay", ResultWarningUnknown)
	if result.Severity != SeverityWarning {
		t.Fatalf("unknown severity = %s, want warning", result.Severity)
	}
}

func TestEvaluatePersonalTripNotApplicable(t *testing.T) {
	evaluation := Evaluate(context.Background(), EvaluationInput{
		Trip: &entity.Trip{ID: uuid.New()},
	})
	if evaluation.Status != EvaluationNotApplicable ||
		evaluation.NotApplicableReason == nil ||
		*evaluation.NotApplicableReason != "personal_trip" {
		t.Fatalf("unexpected evaluation: %#v", evaluation)
	}
}

func TestBuildAIConstraintsExcludesDisabledRules(t *testing.T) {
	policy := policyWithRules(func(rules *Rules) {
		rules.MaxTripBudget = MoneyRule{
			Rule: Rule{Enabled: true, Severity: SeverityBlocking}, Amount: 1500, Currency: "EUR",
		}
		rules.NoLateActivitiesAfter = LateActivityRule{
			Rule: Rule{Enabled: true, Severity: SeverityWarning}, Time: "22:00",
		}
		rules.MaxDailyBudget = MoneyRule{
			Rule: Rule{Enabled: false, Severity: SeverityWarning}, Amount: 250, Currency: "EUR",
		}
	})
	constraints := BuildAIConstraints(policy)
	if constraints == nil {
		t.Fatal("expected constraints")
	}
	if !strings.Contains(constraints.Summary, "1500.00 EUR") ||
		!strings.Contains(constraints.Summary, "22:00") {
		t.Fatalf("summary missing enabled rules: %s", constraints.Summary)
	}
	if strings.Contains(constraints.Summary, "250.00 EUR") {
		t.Fatalf("summary contains disabled rule: %s", constraints.Summary)
	}
}

func policyWithRules(update func(*Rules)) *Policy {
	document := DefaultRules()
	// Default recommendations include enabled rules; tests opt in explicitly.
	document.Rules = Rules{}
	update(&document.Rules)
	return &Policy{
		ID: uuid.New(), WorkspaceID: uuid.New(), Rules: document,
		Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

func assertResult(
	t *testing.T,
	evaluation Evaluation,
	key string,
	status RuleResultStatus,
) EvaluationResult {
	t.Helper()
	for _, result := range evaluation.Results {
		if result.RuleKey == key {
			if result.Status != status {
				t.Fatalf("%s status = %s, want %s", key, result.Status, status)
			}
			return result
		}
	}
	t.Fatalf("result %s not found", key)
	return EvaluationResult{}
}
