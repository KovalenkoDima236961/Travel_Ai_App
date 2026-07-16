package budgetconfidence

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
)

func Compute(ctx context.Context, in Input) Response {
	cfg := normalizedConfig(in.Config)
	in.Config = cfg
	now := in.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	in.Now = now
	currency := resolveCurrency(in)
	in.Currency = currency

	tripID := uuid.Nil
	if in.Trip != nil {
		tripID = in.Trip.ID
	}

	collection := collectCostRecords(ctx, in, currency)
	warnings := dedupeStrings(append(append([]string{}, in.AdditionalWarnings...), collection.Warnings...))
	coverage, coverageDebug := buildCoverage(tripDays(in), collection.Records)
	sourceQuality := buildSourceQuality(collection.Records, currency)
	planned := buildPlannedVsActual(collection.Records, currency)
	estimatedTotal, actualTotal := totals(collection.Records)
	tripBudget := tripBudgetMoney(in.Trip, currency)

	sourceScore := averageSourceQuality(collection.Records)
	plannedScore := plannedActualScore(planned, actualTotal)
	conversionScore := conversionReliabilityScore(collection)
	budgetSafety := budgetLimitSafetyScore(tripBudget, estimatedTotal, actualTotal)

	preliminary := calculateScore(coverage, sourceScore, plannedScore, conversionScore, budgetSafety, nil)
	issues := detectIssues(
		tripID,
		in,
		collection.Records,
		coverage,
		planned,
		estimatedTotal,
		actualTotal,
		collection,
		preliminary.FinalScore,
	)
	breakdown := calculateScore(coverage, sourceScore, plannedScore, conversionScore, budgetSafety, issues)
	level := levelFromScore(breakdown.FinalScore)
	risk := riskFromIssues(breakdown.FinalScore, issues)

	response := Response{
		TripID:          tripID,
		Score:           breakdown.FinalScore,
		Level:           level,
		RiskLevel:       risk,
		Summary:         summaryFor(level, coverage, issues),
		Currency:        currency,
		EstimatedTotal:  Money{Amount: estimatedTotal, Currency: currency},
		ActualTotal:     Money{Amount: actualTotal, Currency: currency},
		TripBudget:      tripBudget,
		Coverage:        coverage,
		SourceQuality:   sourceQuality,
		PlannedVsActual: planned,
		Issues:          issues,
		Recommendations: buildRecommendations(tripID, issues),
		Warnings:        warnings,
		ComputedAt:      now,
	}
	if in.IncludeDebug {
		response.Debug = map[string]any{
			"score": map[string]any{
				"overallCoverage":       breakdown.OverallCoverageScore,
				"averageSourceQuality":  breakdown.AverageSourceQualityScore,
				"plannedActual":         breakdown.PlannedActualScore,
				"conversionReliability": breakdown.ConversionReliability,
				"budgetLimitSafety":     breakdown.BudgetLimitSafety,
				"baseScore":             breakdown.BaseScore,
				"finalScore":            breakdown.FinalScore,
			},
			"coverage":                 coverageDebugMap(coverageDebug),
			"recordCount":              len(collection.Records),
			"conversionFailureCount":   collection.ConversionFailureCount,
			"conversionApproxCount":    collection.ConversionApproxCount,
			"conversionAttemptedCount": collection.ConversionAttemptedCount,
		}
	}
	return response
}

func normalizedConfig(cfg Config) Config {
	defaults := DefaultConfig()
	if cfg == (Config{}) {
		return defaults
	}
	if cfg.LargeExpenseReceiptThreshold <= 0 {
		cfg.LargeExpenseReceiptThreshold = defaults.LargeExpenseReceiptThreshold
	}
	if cfg.ActualSpendHighThresholdPercent <= 0 {
		cfg.ActualSpendHighThresholdPercent = defaults.ActualSpendHighThresholdPercent
	}
	if cfg.PlannedActualGapWarningPercent <= 0 {
		cfg.PlannedActualGapWarningPercent = defaults.PlannedActualGapWarningPercent
	}
	if cfg.PlannedActualGapHighPercent <= 0 {
		cfg.PlannedActualGapHighPercent = defaults.PlannedActualGapHighPercent
	}
	return cfg
}

func resolveCurrency(in Input) string {
	if strings.TrimSpace(in.Currency) != "" {
		return currencyOrDefault(in.Currency, budget.DefaultCurrency)
	}
	if in.BudgetSummary != nil && strings.TrimSpace(in.BudgetSummary.Currency) != "" {
		return currencyOrDefault(in.BudgetSummary.Currency, budget.DefaultCurrency)
	}
	if in.Trip != nil {
		if strings.TrimSpace(in.Trip.BudgetCurrency) != "" {
			return currencyOrDefault(in.Trip.BudgetCurrency, budget.DefaultCurrency)
		}
		if in.Trip.Accommodation != nil && in.Trip.Accommodation.EstimatedCost != nil {
			if strings.TrimSpace(in.Trip.Accommodation.EstimatedCost.Currency) != "" {
				return currencyOrDefault(in.Trip.Accommodation.EstimatedCost.Currency, budget.DefaultCurrency)
			}
		}
	}
	if strings.TrimSpace(in.Itinerary.Currency) != "" {
		return currencyOrDefault(in.Itinerary.Currency, budget.DefaultCurrency)
	}
	return budget.DefaultCurrency
}

func tripDays(in Input) int32 {
	if in.Trip != nil {
		return in.Trip.Days
	}
	return int32(len(in.Itinerary.Days))
}

func dedupeStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func coverageDebugMap(stats map[CoverageCategory]categoryCoverage) map[string]any {
	out := map[string]any{}
	for category, value := range stats {
		score := any(nil)
		if value.Score != nil {
			score = *value.Score
		}
		out[string(category)] = map[string]any{
			"expectedCount": value.ExpectedCount,
			"costCount":     value.CostCount,
			"qualityCount":  value.QualityCount,
			"actualCount":   value.ActualCount,
			"score":         score,
		}
	}
	return out
}
