package aimodel

import "testing"

func TestEvaluateGuardrailsInsufficientData(t *testing.T) {
	result := EvaluateGuardrails(DefaultGuardrailConfig(), RolloutMetrics{SampleCount: 10}, true)
	if result.Status != GuardrailInsufficientData {
		t.Fatalf("expected insufficient data, got %s", result.Status)
	}
	if result.ShouldPause {
		t.Fatal("insufficient data must not pause rollout")
	}
}

func TestEvaluateGuardrailsCriticalFailurePausesUserVisibleCandidate(t *testing.T) {
	cfg := DefaultGuardrailConfig()
	result := EvaluateGuardrails(cfg, RolloutMetrics{
		SampleCount:               cfg.MinSampleCount,
		CandidateFailureRate:      cfg.MaxCandidateFailureRate + 0.01,
		CandidateParseFailureRate: 0,
		OverallQualityDelta:       0,
	}, true)

	if result.Status != GuardrailCritical {
		t.Fatalf("expected critical guardrail status, got %s", result.Status)
	}
	if !result.ShouldPause {
		t.Fatal("critical user-visible guardrail failure must pause candidate")
	}
}

func TestEvaluateGuardrailsShadowFailureDoesNotPauseForQualityOnly(t *testing.T) {
	cfg := DefaultGuardrailConfig()
	result := EvaluateGuardrails(cfg, RolloutMetrics{
		SampleCount:             cfg.MinSampleCount,
		HallucinationRegression: 0.01,
		OverallQualityDelta:     0,
	}, false)

	if result.Status != GuardrailCritical {
		t.Fatalf("expected critical status for hallucination regression, got %s", result.Status)
	}
	if result.ShouldPause {
		t.Fatal("shadow quality regression should be visible but not pause user traffic")
	}
}

func TestCalculateMetricDeltas(t *testing.T) {
	got := CalculateMetricDeltas(
		map[string]float64{"overallQualityScore": 0.82, "hallucinatedPlaceCount": 1},
		map[string]float64{"overallQualityScore": 0.85, "hallucinatedPlaceCount": 0},
	)

	if got["overallQualityScore"] < 0.029 || got["overallQualityScore"] > 0.031 {
		t.Fatalf("unexpected quality delta: %v", got["overallQualityScore"])
	}
	if got["hallucinatedPlaceCount"] != -1 {
		t.Fatalf("expected hallucination count delta -1, got %v", got["hallucinatedPlaceCount"])
	}
}

func TestLatencyIncreasePercent(t *testing.T) {
	if got := LatencyIncreasePercent(1000, 1250); got != 25 {
		t.Fatalf("expected 25 percent latency increase, got %v", got)
	}
	if got := LatencyIncreasePercent(0, 1250); got != 0 {
		t.Fatalf("zero baseline must not divide by zero, got %v", got)
	}
}
