package budgetconfidence

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type categoryCoverage struct {
	ExpectedCount int
	CostCount     int
	QualityTotal  int
	QualityCount  int
	ActualCount   int
	Score         *int
}

type scoreBreakdown struct {
	OverallCoverageScore      int
	AverageSourceQualityScore int
	PlannedActualScore        int
	ConversionReliability     int
	BudgetLimitSafety         int
	BaseScore                 int
	FinalScore                int
}

func buildCoverage(tripDays int32, records []costRecord) (Coverage, map[CoverageCategory]categoryCoverage) {
	byCategory := map[CoverageCategory]categoryCoverage{}
	for _, record := range records {
		group := coverageCategoryFor(record.Category)
		stats := byCategory[group]
		if record.IsEstimate {
			stats.ExpectedCount++
			if record.Amount != nil && !record.Missing && !record.ConversionFailed {
				stats.CostCount++
			}
			stats.QualityTotal += record.QualityScore
			stats.QualityCount++
		}
		if record.IsActual {
			stats.ActualCount++
			stats.QualityTotal += record.QualityScore
			stats.QualityCount++
			if stats.ExpectedCount == 0 {
				stats.ExpectedCount = 1
			}
			if record.Amount != nil && !record.ConversionFailed {
				stats.CostCount = maxInt(stats.CostCount, 1)
			}
		}
		byCategory[group] = stats
	}

	for category, stats := range byCategory {
		if stats.ExpectedCount <= 0 {
			continue
		}
		costCoverage := float64(stats.CostCount) / float64(stats.ExpectedCount) * 100
		quality := 0.0
		if stats.QualityCount > 0 {
			quality = float64(stats.QualityTotal) / float64(stats.QualityCount)
		}
		score := intScore(0.6*costCoverage + 0.4*quality)
		stats.Score = &score
		byCategory[category] = stats
	}

	overall := weightedCoverageOverall(tripDays, byCategory, records)
	coverage := Coverage{
		Overall:          overall,
		Transport:        coverageScorePtr(byCategory, CoverageTransport),
		Accommodation:    coverageScorePtr(byCategory, CoverageAccommodation),
		Activities:       coverageScorePtr(byCategory, CoverageActivities),
		Food:             coverageScorePtr(byCategory, CoverageFood),
		Shopping:         coverageScorePtr(byCategory, CoverageShopping),
		FuelParkingTolls: coverageScorePtr(byCategory, CoverageFuelParkingTolls),
		Other:            coverageScorePtr(byCategory, CoverageOther),
	}
	return coverage, byCategory
}

func weightedCoverageOverall(tripDays int32, byCategory map[CoverageCategory]categoryCoverage, records []costRecord) int {
	weights := map[CoverageCategory]float64{}
	if hasCoverageSignal(byCategory, CoverageTransport) {
		weights[CoverageTransport] = 20
	}
	if tripDays > 1 || hasCoverageSignal(byCategory, CoverageAccommodation) {
		weights[CoverageAccommodation] = 20
	}
	if hasCoverageSignal(byCategory, CoverageActivities) {
		weights[CoverageActivities] = 20
	}
	if tripDays > 1 || hasCoverageSignal(byCategory, CoverageFood) {
		weights[CoverageFood] = 15
	}
	if hasCoverageSignal(byCategory, CoverageFuelParkingTolls) || hasCarRouteCost(records) {
		weights[CoverageFuelParkingTolls] = 10
	}
	if hasCoverageSignal(byCategory, CoverageShopping) {
		weights[CoverageShopping] = 7.5
	}
	if hasCoverageSignal(byCategory, CoverageOther) {
		weights[CoverageOther] = 7.5
	}

	if len(weights) == 0 {
		return 60
	}
	totalWeight := 0.0
	weighted := 0.0
	for category, weight := range weights {
		score := 0
		if stats, ok := byCategory[category]; ok && stats.Score != nil {
			score = *stats.Score
		}
		weighted += weight * float64(score)
		totalWeight += weight
	}
	if totalWeight == 0 {
		return 60
	}
	return intScore(weighted / totalWeight)
}

func hasCoverageSignal(byCategory map[CoverageCategory]categoryCoverage, category CoverageCategory) bool {
	stats, ok := byCategory[category]
	return ok && stats.ExpectedCount > 0
}

func hasCarRouteCost(records []costRecord) bool {
	for _, record := range records {
		mode, _ := record.Metadata["mode"].(string)
		mode = normalizeToken(mode)
		if mode == "car" || mode == "rental_car" {
			return true
		}
	}
	return false
}

func coverageScorePtr(byCategory map[CoverageCategory]categoryCoverage, category CoverageCategory) *int {
	if stats, ok := byCategory[category]; ok {
		return stats.Score
	}
	return nil
}

func buildSourceQuality(records []costRecord, currency string) []SourceQuality {
	type accumulator struct {
		count        int
		total        float64
		qualityTotal int
	}
	bySource := map[Source]accumulator{}
	for _, record := range records {
		acc := bySource[record.Source]
		acc.count++
		if record.Amount != nil {
			acc.total += record.Amount.Amount
		}
		acc.qualityTotal += record.QualityScore
		bySource[record.Source] = acc
	}
	sources := make([]Source, 0, len(bySource))
	for source := range bySource {
		sources = append(sources, source)
	}
	sort.Slice(sources, func(i, j int) bool { return sourceRank(sources[i]) < sourceRank(sources[j]) })
	out := make([]SourceQuality, 0, len(sources))
	for _, source := range sources {
		acc := bySource[source]
		quality := 0
		if acc.count > 0 {
			quality = intScore(float64(acc.qualityTotal) / float64(acc.count))
		}
		out = append(out, SourceQuality{
			Source:       source,
			ItemCount:    acc.count,
			TotalAmount:  Money{Amount: round2(acc.total), Currency: currency},
			QualityScore: quality,
			Reason:       sourceReason(source),
		})
	}
	return out
}

func buildPlannedVsActual(records []costRecord, currency string) PlannedVsActual {
	estimatedByCategory := map[Category]float64{}
	actualByCategory := map[Category]float64{}
	for _, record := range records {
		if record.Amount == nil || record.ConversionFailed {
			continue
		}
		if record.IsActual {
			actualByCategory[record.Category] += record.Amount.Amount
			continue
		}
		if record.IsEstimate && !record.Missing {
			estimatedByCategory[record.Category] += record.Amount.Amount
		}
	}

	categories := make([]Category, 0)
	seen := map[Category]struct{}{}
	for category := range estimatedByCategory {
		seen[category] = struct{}{}
	}
	for category := range actualByCategory {
		seen[category] = struct{}{}
	}
	for category := range seen {
		categories = append(categories, category)
	}
	sort.Slice(categories, func(i, j int) bool { return categoryRank(categories[i]) < categoryRank(categories[j]) })

	totalEstimated := 0.0
	totalActual := 0.0
	out := make([]PlannedVsActualByCategory, 0, len(categories))
	for _, category := range categories {
		estimated := round2(estimatedByCategory[category])
		actual := round2(actualByCategory[category])
		totalEstimated += estimated
		totalActual += actual
		status := "on_track"
		var diffPercent *float64
		switch {
		case estimated == 0 && actual > 0:
			status = "actual_without_estimate"
			diffPercent = roundPercent(100)
		case estimated > 0:
			diff := ((actual - estimated) / estimated) * 100
			diffPercent = roundPercent(diff)
			if diff > 10 {
				status = "over_estimate"
			} else if diff < -10 {
				status = "under_estimate"
			}
		}
		out = append(out, PlannedVsActualByCategory{
			Category:          category,
			Estimated:         Money{Amount: estimated, Currency: currency},
			Actual:            Money{Amount: actual, Currency: currency},
			DifferencePercent: diffPercent,
			Status:            status,
		})
	}
	overallDifference := round2(totalEstimated - totalActual)
	var overallPercent *float64
	if totalEstimated > 0 {
		overallPercent = roundPercent((overallDifference / totalEstimated) * 100)
	}
	return PlannedVsActual{
		OverallDifference:        Money{Amount: overallDifference, Currency: currency},
		OverallDifferencePercent: overallPercent,
		Categories:               out,
	}
}

func totals(records []costRecord) (estimated float64, actual float64) {
	for _, record := range records {
		if record.Amount == nil || record.ConversionFailed {
			continue
		}
		if record.IsActual {
			actual += record.Amount.Amount
		} else if record.IsEstimate && !record.Missing {
			estimated += record.Amount.Amount
		}
	}
	return round2(estimated), round2(actual)
}

func averageSourceQuality(records []costRecord) int {
	totalWeight := 0.0
	weighted := 0.0
	for _, record := range records {
		weight := 1.0
		if record.Amount != nil {
			weight = math.Max(record.Amount.Amount, 1)
		} else if record.Missing {
			weight = 50
		}
		weighted += float64(record.QualityScore) * weight
		totalWeight += weight
	}
	if totalWeight == 0 {
		return 60
	}
	return intScore(weighted / totalWeight)
}

func plannedActualScore(planned PlannedVsActual, actualTotal float64) int {
	if actualTotal <= 0 {
		return 60
	}
	if planned.OverallDifferencePercent == nil {
		return 45
	}
	abs := math.Abs(*planned.OverallDifferencePercent)
	switch {
	case abs <= 10:
		return 100
	case abs <= 20:
		return 75
	case abs <= 40:
		return 45
	default:
		return 20
	}
}

func conversionReliabilityScore(collection collectionResult) int {
	if collection.ConversionFailureCount > 0 {
		if collection.ConversionFailureCount >= 3 || collection.ConversionFailureAmount >= 250 {
			return 10
		}
		return 40
	}
	if collection.ConversionApproxCount > 0 {
		return 75
	}
	return 100
}

func budgetLimitSafetyScore(tripBudget *Money, estimatedTotal, actualTotal float64) int {
	if tripBudget == nil || tripBudget.Amount <= 0 {
		return 60
	}
	used := math.Max(estimatedTotal, actualTotal)
	ratio := used / tripBudget.Amount
	switch {
	case ratio <= 0.9:
		return 100
	case ratio <= 1:
		return 70
	case ratio <= 1.1:
		return 40
	default:
		return 10
	}
}

func calculateScore(
	coverage Coverage,
	sourceQuality int,
	plannedActual int,
	conversionReliability int,
	budgetSafety int,
	issues []Issue,
) scoreBreakdown {
	base := intScore(
		0.45*float64(coverage.Overall) +
			0.25*float64(sourceQuality) +
			0.15*float64(plannedActual) +
			0.10*float64(conversionReliability) +
			0.05*float64(budgetSafety),
	)
	penalty := 0
	for _, issue := range issues {
		switch issue.Severity {
		case SeverityCritical:
			penalty += 20
		case SeverityHigh:
			penalty += 12
		case SeverityWarning:
			penalty += 5
		case SeverityInfo:
			penalty++
		}
	}
	final := clampInt(base-penalty, 0, 100)
	return scoreBreakdown{
		OverallCoverageScore:      coverage.Overall,
		AverageSourceQualityScore: sourceQuality,
		PlannedActualScore:        plannedActual,
		ConversionReliability:     conversionReliability,
		BudgetLimitSafety:         budgetSafety,
		BaseScore:                 base,
		FinalScore:                final,
	}
}

func levelFromScore(score int) ConfidenceLevel {
	switch {
	case score >= 90:
		return LevelVeryHigh
	case score >= 75:
		return LevelHigh
	case score >= 55:
		return LevelMedium
	case score >= 30:
		return LevelLow
	default:
		return LevelVeryLow
	}
}

func riskFromIssues(score int, issues []Issue) RiskLevel {
	highest := SeverityInfo
	for _, issue := range issues {
		if severityRank(issue.Severity) > severityRank(highest) {
			highest = issue.Severity
		}
	}
	switch {
	case highest == SeverityCritical || score < 30:
		return RiskCritical
	case highest == SeverityHigh || score < 55:
		return RiskHigh
	case highest == SeverityWarning || score < 75:
		return RiskMedium
	default:
		return RiskLow
	}
}

func summaryFor(level ConfidenceLevel, coverage Coverage, issues []Issue) string {
	label := strings.ReplaceAll(string(level), "_", " ")
	if len(issues) == 0 {
		return fmt.Sprintf("Budget confidence is %s with %d%% coverage.", label, coverage.Overall)
	}
	top := make([]Issue, len(issues))
	copy(top, issues)
	sort.SliceStable(top, func(i, j int) bool {
		if severityRank(top[i].Severity) != severityRank(top[j].Severity) {
			return severityRank(top[i].Severity) > severityRank(top[j].Severity)
		}
		return top[i].ID < top[j].ID
	})
	parts := make([]string, 0, 2)
	for _, issue := range top {
		if len(parts) == 2 {
			break
		}
		parts = append(parts, strings.ToLower(issue.Title))
	}
	return fmt.Sprintf("Budget confidence is %s. %s need review.", label, strings.Join(parts, " and "))
}

func tripBudgetMoney(trip *entity.Trip, currency string) *Money {
	if trip == nil || trip.BudgetAmount == nil {
		return nil
	}
	return &Money{Amount: round2(*trip.BudgetAmount), Currency: currencyOrDefault(trip.BudgetCurrency, currency)}
}

func isTripInProgressOrFuture(trip *entity.Trip, now time.Time) bool {
	if trip == nil || trip.Status != entity.StatusCompleted {
		return true
	}
	if trip.StartDate == nil || trip.Days <= 0 {
		return true
	}
	end := trip.StartDate.AddDate(0, 0, int(trip.Days))
	return !now.After(end)
}

func sourceRank(source Source) int {
	switch source {
	case SourceActualReceiptExpense:
		return 1
	case SourceActualManualExpense:
		return 2
	case SourceProviderPrice:
		return 3
	case SourceSelectedTransportOptionHighConfidence, SourceSelectedTransportOptionMediumConfidence, SourceSelectedTransportOptionLowConfidence:
		return 4
	case SourceManualEstimate:
		return 5
	case SourceAIEstimateHighConfidence, SourceAIEstimateMediumConfidence, SourceAIEstimateLowConfidence:
		return 6
	case SourceMockEstimate:
		return 7
	case SourceMissingCost:
		return 8
	default:
		return 9
	}
}

func sourceReason(source Source) string {
	switch source {
	case SourceActualReceiptExpense:
		return "confirmed actual expense with receipt"
	case SourceActualManualExpense:
		return "confirmed actual expense manually entered"
	case SourceProviderPrice:
		return "provider-backed ticket or price estimate"
	case SourceSelectedTransportOptionHighConfidence, SourceSelectedTransportOptionMediumConfidence, SourceSelectedTransportOptionLowConfidence:
		return "selected transport option estimate"
	case SourceManualEstimate:
		return "user-entered estimate but not actual"
	case SourceAIEstimateHighConfidence, SourceAIEstimateMediumConfidence, SourceAIEstimateLowConfidence:
		return "AI-generated estimate"
	case SourceMockEstimate:
		return "mock or fallback estimate"
	case SourceMissingCost:
		return "missing cost"
	default:
		return "unknown source"
	}
}

func categoryRank(category Category) int {
	switch category {
	case CategoryTransport:
		return 1
	case CategoryAccommodation:
		return 2
	case CategoryActivities:
		return 3
	case CategoryTickets:
		return 4
	case CategoryFood:
		return 5
	case CategoryGroceries:
		return 6
	case CategoryFuel:
		return 7
	case CategoryParking:
		return 8
	case CategoryTolls:
		return 9
	case CategoryShopping:
		return 10
	default:
		return 50
	}
}

func severityRank(severity IssueSeverity) int {
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
