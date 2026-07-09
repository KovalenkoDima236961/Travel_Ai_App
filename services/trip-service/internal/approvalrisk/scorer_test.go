package approvalrisk

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/approvals"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

func TestScoreNoIssuesLow(t *testing.T) {
	response := Score(baseInput())
	if response.Score == nil || *response.Score != 0 {
		t.Fatalf("expected score 0, got %+v", response.Score)
	}
	if response.Status != RiskLevelLow {
		t.Fatalf("expected low, got %s", response.Status)
	}
	if len(response.Factors) != 0 {
		t.Fatalf("expected no factors, got %+v", response.Factors)
	}
}

func TestScorePolicyWarningsAndBlockingThresholds(t *testing.T) {
	in := baseInput()
	in.PolicyEvaluation = &workspacepolicies.Evaluation{
		TripID:      in.TripID,
		WorkspaceID: in.WorkspaceID,
		Status:      workspacepolicies.EvaluationWarning,
		Results: []workspacepolicies.EvaluationResult{
			{
				RuleKey:  "maxTripBudget",
				Status:   workspacepolicies.ResultViolation,
				Severity: workspacepolicies.SeverityWarning,
				Title:    "Budget warning",
				Message:  "Budget warning",
			},
		},
	}
	response := Score(in)
	if response.Score == nil || *response.Score != 10 || response.Status != RiskLevelLow {
		t.Fatalf("expected one warning to score 10/low, got score=%v status=%s", response.Score, response.Status)
	}

	in = baseInput()
	in.PolicyEvaluation = &workspacepolicies.Evaluation{
		TripID:      in.TripID,
		WorkspaceID: in.WorkspaceID,
		Status:      workspacepolicies.EvaluationBlocking,
		Results: []workspacepolicies.EvaluationResult{
			{RuleKey: "a", Status: workspacepolicies.ResultViolation, Severity: workspacepolicies.SeverityBlocking},
		},
	}
	response = Score(in)
	if response.Score == nil || *response.Score < 50 || response.Status != RiskLevelHigh {
		t.Fatalf("expected one blocker to force high risk, got score=%v status=%s", response.Score, response.Status)
	}

	in.PolicyEvaluation.Results = append(in.PolicyEvaluation.Results,
		workspacepolicies.EvaluationResult{
			RuleKey:  "b",
			Status:   workspacepolicies.ResultViolation,
			Severity: workspacepolicies.SeverityBlocking,
		},
	)
	response = Score(in)
	if response.Score == nil || *response.Score < 75 || response.Status != RiskLevelCritical {
		t.Fatalf("expected multiple blockers to force critical risk, got score=%v status=%s", response.Score, response.Status)
	}
}

func TestScoreBudgetThresholds(t *testing.T) {
	cases := []struct {
		name           string
		estimatedTotal float64
		expectedPoints int
	}{
		{name: "up to ten percent", estimatedTotal: 108, expectedPoints: 10},
		{name: "ten to thirty percent", estimatedTotal: 125, expectedPoints: 18},
		{name: "over thirty percent", estimatedTotal: 140, expectedPoints: 25},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := baseInput()
			in.ChecklistInput.EstimatedTotal = tc.estimatedTotal
			response := Score(in)
			factor := findFactor(response.Factors, "trip_budget_exceeded")
			if factor == nil || factor.Points != tc.expectedPoints {
				t.Fatalf("expected trip budget points %d, got %+v", tc.expectedPoints, factor)
			}
		})
	}
}

func TestScoreCostSplittingAvailabilityAIAndSchedule(t *testing.T) {
	in := baseInput()
	in.ChecklistInput.EstimatedTotal = 120
	in.ChecklistInput.TravelerCount = 0
	in.ChecklistInput.InvalidSplitCount = 2
	in.ChecklistInput.MissingEstimateCount = 6
	in.ChecklistInput.AvailabilityUncheckedCount = 6
	in.ChecklistInput.AvailabilityUnavailableCount = 1
	in.Metadata.TemplateFallbackUsed = true
	distance := 20.0
	in.Itinerary = aggregate.Itinerary{
		Days: []aggregate.ItineraryDay{
			{
				Day: 1,
				Items: []aggregate.ItineraryItem{
					{Time: "09:00", EndTime: "10:00", Name: "Museum", WalkingDistanceKm: &distance},
				},
			},
		},
	}

	response := Score(in)
	for _, factorType := range []string{
		"cost_splitting_no_travelers",
		"cost_splitting_invalid_rules",
		"missing_cost_estimates",
		"availability_unchecked",
		"availability_unavailable",
		"ai_fallback_used",
		"walking_distance_high",
	} {
		if findFactor(response.Factors, factorType) == nil {
			t.Fatalf("expected factor %s in %+v", factorType, response.Factors)
		}
	}
}

func TestScoreCappedTopReasonsAndActionDedupe(t *testing.T) {
	in := baseInput()
	in.ChecklistInput.EstimatedTotal = 500
	in.ChecklistInput.MissingEstimateCount = 10
	in.ChecklistInput.InvalidSplitCount = 10
	in.ChecklistInput.UnassignedCostCount = 10
	in.ChecklistInput.AvailabilityUncheckedCount = 10
	in.ChecklistInput.AvailabilityUnavailableCount = 10
	in.PolicyEvaluation = &workspacepolicies.Evaluation{
		TripID:      in.TripID,
		WorkspaceID: in.WorkspaceID,
		Status:      workspacepolicies.EvaluationBlocking,
		Results: []workspacepolicies.EvaluationResult{
			{RuleKey: "a", Status: workspacepolicies.ResultViolation, Severity: workspacepolicies.SeverityBlocking},
			{RuleKey: "b", Status: workspacepolicies.ResultViolation, Severity: workspacepolicies.SeverityBlocking},
			{RuleKey: "c", Status: workspacepolicies.ResultViolation, Severity: workspacepolicies.SeverityWarning},
		},
	}

	response := Score(in)
	if response.Score == nil || *response.Score != 100 {
		t.Fatalf("expected capped score 100, got %+v", response.Score)
	}
	if len(response.TopReasons) > maxTopReasons {
		t.Fatalf("expected top reasons cap, got %d", len(response.TopReasons))
	}
	if len(response.SuggestedActions) > maxSuggestedActions {
		t.Fatalf("expected suggested actions cap, got %d", len(response.SuggestedActions))
	}
	seen := map[string]struct{}{}
	for _, action := range response.SuggestedActions {
		key := actionKey(action)
		if _, ok := seen[key]; ok {
			t.Fatalf("duplicate suggested action %s", key)
		}
		seen[key] = struct{}{}
	}
}

func baseInput() Input {
	tripID := uuid.New()
	workspaceID := uuid.New()
	return Input{
		TripID:      tripID,
		WorkspaceID: &workspaceID,
		GeneratedAt: time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC),
		Trip: TripContext{
			BudgetAmount:   floatPtr(100),
			BudgetCurrency: "EUR",
			Days:           1,
		},
		ChecklistInput: approvals.ChecklistInput{
			HasTripBudget:    true,
			TripBudgetAmount: 100,
			EstimatedTotal:   50,
			TravelerCount:    1,
		},
		Itinerary: aggregate.Itinerary{Days: []aggregate.ItineraryDay{}},
	}
}

func findFactor(factors []Factor, factorType string) *Factor {
	for i := range factors {
		if factors[i].Type == factorType {
			return &factors[i]
		}
	}
	return nil
}

func floatPtr(value float64) *float64 {
	return &value
}
