package aimodel

import (
	"fmt"
	"math"
	"time"
)

type GuardrailStatus string

const (
	GuardrailInsufficientData GuardrailStatus = "insufficient_data"
	GuardrailPassing          GuardrailStatus = "passing"
	GuardrailWarning          GuardrailStatus = "warning"
	GuardrailFailing          GuardrailStatus = "failing"
	GuardrailCritical         GuardrailStatus = "critical"
)

type GuardrailConfig struct {
	MinSampleCount                   int
	WindowDuration                   time.Duration
	MaxCandidateFailureRate          float64
	MaxParseFailureRate              float64
	MaxHallucinationRegression       float64
	MaxDestinationMismatchRegression float64
	MaxRepairRateIncrease            float64
	MaxP95LatencyIncreasePercent     float64
	MaxP95LatencyMS                  int
	MinGroundedPlaceRate             float64
	MinOverallQualityDelta           float64
	MaxLanguageScoreDrop             float64
}

type RolloutMetrics struct {
	SampleCount                   int
	CandidateFailureRate          float64
	CandidateParseFailureRate     float64
	HallucinationRegression       float64
	DestinationMismatchRegression float64
	RepairRateIncrease            float64
	P95LatencyIncreasePercent     float64
	CandidateP95LatencyMS         int
	CandidateGroundedPlaceRate    float64
	OverallQualityDelta           float64
	MaxLanguageScoreDrop          float64
}

type GuardrailBreach struct {
	Name      string          `json:"name"`
	Status    GuardrailStatus `json:"status"`
	Threshold float64         `json:"threshold"`
	Value     float64         `json:"value"`
}

type GuardrailEvaluation struct {
	Status      GuardrailStatus   `json:"status"`
	SampleCount int               `json:"sampleCount"`
	Breaches    []GuardrailBreach `json:"breaches,omitempty"`
	ShouldPause bool              `json:"shouldPause"`
}

func DefaultGuardrailConfig() GuardrailConfig {
	return GuardrailConfig{
		MinSampleCount:                   50,
		WindowDuration:                   time.Hour,
		MaxCandidateFailureRate:          0.05,
		MaxParseFailureRate:              0.01,
		MaxHallucinationRegression:       0,
		MaxDestinationMismatchRegression: 0,
		MaxRepairRateIncrease:            0.05,
		MaxP95LatencyIncreasePercent:     25,
		MaxP95LatencyMS:                  0,
		MinGroundedPlaceRate:             0,
		MinOverallQualityDelta:           0,
		MaxLanguageScoreDrop:             0.05,
	}
}

func NormalizeGuardrailConfig(cfg GuardrailConfig) GuardrailConfig {
	defaults := DefaultGuardrailConfig()
	if cfg.MinSampleCount <= 0 {
		cfg.MinSampleCount = defaults.MinSampleCount
	}
	if cfg.WindowDuration <= 0 {
		cfg.WindowDuration = defaults.WindowDuration
	}
	if cfg.MaxCandidateFailureRate <= 0 {
		cfg.MaxCandidateFailureRate = defaults.MaxCandidateFailureRate
	}
	if cfg.MaxParseFailureRate <= 0 {
		cfg.MaxParseFailureRate = defaults.MaxParseFailureRate
	}
	if cfg.MaxRepairRateIncrease <= 0 {
		cfg.MaxRepairRateIncrease = defaults.MaxRepairRateIncrease
	}
	if cfg.MaxP95LatencyIncreasePercent <= 0 {
		cfg.MaxP95LatencyIncreasePercent = defaults.MaxP95LatencyIncreasePercent
	}
	if cfg.MaxLanguageScoreDrop <= 0 {
		cfg.MaxLanguageScoreDrop = defaults.MaxLanguageScoreDrop
	}
	return cfg
}

func ValidateGuardrailConfig(cfg GuardrailConfig) error {
	cfg = NormalizeGuardrailConfig(cfg)
	if cfg.MinSampleCount < 1 {
		return fmt.Errorf("guardrail minimum sample count must be positive")
	}
	for name, value := range map[string]float64{
		"maximum candidate failure rate":          cfg.MaxCandidateFailureRate,
		"maximum parse failure rate":              cfg.MaxParseFailureRate,
		"maximum hallucination regression":        cfg.MaxHallucinationRegression,
		"maximum destination mismatch regression": cfg.MaxDestinationMismatchRegression,
		"maximum repair rate increase":            cfg.MaxRepairRateIncrease,
		"maximum language score drop":             cfg.MaxLanguageScoreDrop,
	} {
		if value < 0 || value > 1 || math.IsNaN(value) {
			return fmt.Errorf("%s must be between 0 and 1", name)
		}
	}
	if cfg.MaxP95LatencyIncreasePercent < 0 || math.IsNaN(cfg.MaxP95LatencyIncreasePercent) {
		return fmt.Errorf("maximum p95 latency increase percent must be non-negative")
	}
	if cfg.MinGroundedPlaceRate < 0 || cfg.MinGroundedPlaceRate > 1 || math.IsNaN(cfg.MinGroundedPlaceRate) {
		return fmt.Errorf("minimum grounded-place rate must be between 0 and 1")
	}
	if cfg.MinOverallQualityDelta < -1 || cfg.MinOverallQualityDelta > 1 || math.IsNaN(cfg.MinOverallQualityDelta) {
		return fmt.Errorf("minimum overall quality delta must be between -1 and 1")
	}
	return nil
}

func EvaluateGuardrails(cfg GuardrailConfig, metrics RolloutMetrics, userVisible bool) GuardrailEvaluation {
	cfg = NormalizeGuardrailConfig(cfg)
	result := GuardrailEvaluation{
		Status:      GuardrailPassing,
		SampleCount: metrics.SampleCount,
	}
	if metrics.SampleCount < cfg.MinSampleCount {
		result.Status = GuardrailInsufficientData
		return result
	}

	checkMax(&result, "candidate_failure_rate", metrics.CandidateFailureRate, cfg.MaxCandidateFailureRate, true)
	checkMax(&result, "parse_failure_rate", metrics.CandidateParseFailureRate, cfg.MaxParseFailureRate, true)
	checkMax(&result, "hallucination_regression", metrics.HallucinationRegression, cfg.MaxHallucinationRegression, true)
	checkMax(&result, "destination_mismatch_regression", metrics.DestinationMismatchRegression, cfg.MaxDestinationMismatchRegression, true)
	checkMax(&result, "repair_rate_increase", metrics.RepairRateIncrease, cfg.MaxRepairRateIncrease, false)
	checkMax(&result, "p95_latency_increase_percent", metrics.P95LatencyIncreasePercent, cfg.MaxP95LatencyIncreasePercent, false)
	if cfg.MaxP95LatencyMS > 0 {
		checkMax(&result, "p95_latency_ms", float64(metrics.CandidateP95LatencyMS), float64(cfg.MaxP95LatencyMS), true)
	}
	if cfg.MinGroundedPlaceRate > 0 {
		checkMin(&result, "grounded_place_rate", metrics.CandidateGroundedPlaceRate, cfg.MinGroundedPlaceRate, true)
	}
	checkMin(&result, "overall_quality_delta", metrics.OverallQualityDelta, cfg.MinOverallQualityDelta, true)
	checkMax(&result, "language_score_drop", metrics.MaxLanguageScoreDrop, cfg.MaxLanguageScoreDrop, false)

	for _, breach := range result.Breaches {
		if breach.Status == GuardrailCritical {
			result.Status = GuardrailCritical
			break
		}
		if breach.Status == GuardrailFailing && result.Status != GuardrailCritical {
			result.Status = GuardrailFailing
		}
		if breach.Status == GuardrailWarning && result.Status == GuardrailPassing {
			result.Status = GuardrailWarning
		}
	}
	result.ShouldPause = userVisible && result.Status == GuardrailCritical
	return result
}

func checkMax(result *GuardrailEvaluation, name string, value, threshold float64, critical bool) {
	if math.IsNaN(value) {
		value = 0
	}
	if value <= threshold {
		if threshold > 0 && value >= threshold*0.8 {
			result.Breaches = append(result.Breaches, GuardrailBreach{Name: name, Status: GuardrailWarning, Value: value, Threshold: threshold})
		}
		return
	}
	status := GuardrailFailing
	if critical {
		status = GuardrailCritical
	}
	result.Breaches = append(result.Breaches, GuardrailBreach{Name: name, Status: status, Value: value, Threshold: threshold})
}

func checkMin(result *GuardrailEvaluation, name string, value, threshold float64, critical bool) {
	if math.IsNaN(value) {
		value = 0
	}
	if value >= threshold {
		return
	}
	status := GuardrailFailing
	if critical {
		status = GuardrailCritical
	}
	result.Breaches = append(result.Breaches, GuardrailBreach{Name: name, Status: status, Value: value, Threshold: threshold})
}

func CalculateMetricDeltas(baseline, candidate map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(candidate))
	for key, candidateValue := range candidate {
		out[key] = candidateValue - baseline[key]
	}
	return out
}

func LatencyIncreasePercent(baselineMS, candidateMS int) float64 {
	if baselineMS <= 0 || candidateMS <= baselineMS {
		return 0
	}
	return (float64(candidateMS-baselineMS) / float64(baselineMS)) * 100
}
