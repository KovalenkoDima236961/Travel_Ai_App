package analytics

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type TripInput struct {
	Trip               *entity.Trip
	Itinerary          aggregate.Itinerary
	BudgetSummary      budget.Summary
	Currency           string
	GeneratedAt        time.Time
	Converter          budget.CurrencyConverter
	ConversionEnabled  bool
	ConversionFailOpen bool
}

type WorkspaceInput struct {
	WorkspaceID uuid.UUID
	Currency    string
	GeneratedAt time.Time
	From        *time.Time
	To          *time.Time
	Trips       []WorkspaceTripInput
}

type WorkspaceTripInput struct {
	Trip      entity.Trip
	Analytics TripCostAnalytics
}

type costRecord struct {
	TripID          uuid.UUID
	TripTitle       string
	Destination     string
	DayNumber       int
	ItemIndex       int
	Name            string
	Type            string
	Category        string
	Source          string
	Confidence      string
	Amount          float64
	Currency        string
	ConvertedAmount *float64
}

type amountCount struct {
	amount float64
	count  int
}

type currencyTotal struct {
	amount       float64
	converted    float64
	hasConverted bool
}

// ResolveTripCurrency applies the analytics target-currency fallback chain for
// a single trip.
func ResolveTripCurrency(requested string, trip *entity.Trip, itinerary aggregate.Itinerary) string {
	if c := normalizeCurrency(requested); c != "" {
		return c
	}
	if trip != nil {
		if c := normalizeCurrency(trip.BudgetCurrency); c != "" {
			return c
		}
		if trip.Accommodation != nil && trip.Accommodation.EstimatedCost != nil {
			if c := normalizeCurrency(trip.Accommodation.EstimatedCost.Currency); c != "" {
				return c
			}
		}
	}
	if c := normalizeCurrency(itinerary.Currency); c != "" {
		return c
	}
	for _, day := range itinerary.Days {
		for i := range day.Items {
			if day.Items[i].EstimatedCost == nil {
				continue
			}
			if c := normalizeCurrency(day.Items[i].EstimatedCost.Currency); c != "" {
				return c
			}
		}
	}
	return budget.DefaultCurrency
}

func CalculateTripCost(ctx context.Context, in TripInput) (TripCostAnalytics, error) {
	if in.Trip == nil {
		return TripCostAnalytics{}, fmt.Errorf("trip is required")
	}
	generatedAt := in.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	currency := normalizeCurrency(in.Currency)
	if currency == "" {
		currency = ResolveTripCurrency("", in.Trip, in.Itinerary)
	}

	records, missingByDay, uncertainCount, priceMissing := collectCostRecords(ctx, in, currency)
	if priceMissing.err != nil {
		return TripCostAnalytics{}, priceMissing.err
	}

	budgetAmount, budgetWarnings, budgetConversion, err := convertTripBudget(ctx, in, currency)
	if err != nil {
		return TripCostAnalytics{}, err
	}
	exchangeRateInfo := cloneExchangeRateInfo(in.BudgetSummary.ExchangeRateInfo)
	mergeExchangeRateInfo(&exchangeRateInfo, budgetConversion)

	summary := buildSummary(in.BudgetSummary, budgetAmount, uncertainCount)
	if in.Trip.BudgetAmount == nil || budgetAmount == nil {
		summary.IncompleteBudgetDataCount = 1
	}

	byDay := buildByDay(in.Trip, in.BudgetSummary, records, missingByDay, budgetAmount)
	byCategory := buildBreakdown(records, func(record costRecord) string { return record.Category }, "category")
	bySource := buildBreakdown(records, func(record costRecord) string { return record.Source }, "source")
	byConfidence := buildBreakdown(records, func(record costRecord) string { return record.Confidence }, "confidence")
	originalTotals := buildOriginalCurrencyTotals(records)
	expensiveItems := topExpensiveItems(records, 10, in.BudgetSummary.EstimatedTotal)
	insights := buildTripInsights(in.Trip.ID, currency, summary, byDay, expensiveItems, priceMissing)
	warnings := buildWarnings(in.BudgetSummary.ConversionWarnings, budgetWarnings)

	return TripCostAnalytics{
		TripID:                 in.Trip.ID,
		WorkspaceID:            in.Trip.WorkspaceID,
		Currency:               currency,
		GeneratedAt:            generatedAt,
		Summary:                summary,
		ByDay:                  byDay,
		ByCategory:             byCategory,
		BySource:               bySource,
		ByConfidence:           byConfidence,
		OriginalCurrencyTotals: originalTotals,
		ExpensiveItems:         expensiveItems,
		Insights:               insights,
		Warnings:               warnings,
		ExchangeRateInfo:       exchangeRateInfo,
	}, nil
}

func CalculateWorkspaceCost(in WorkspaceInput) WorkspaceCostAnalytics {
	generatedAt := in.GeneratedAt
	if generatedAt.IsZero() {
		generatedAt = time.Now().UTC()
	}
	currency := normalizeCurrency(in.Currency)
	if currency == "" {
		currency = budget.DefaultCurrency
	}

	byTrip := make([]TripCostSummary, 0, len(in.Trips))
	allItems := make([]ExpensiveCostItem, 0)
	warnings := make(map[string]struct{})
	categoryTotals := make(map[string]amountCount)
	sourceTotals := make(map[string]amountCount)
	monthTotals := make(map[string]amountCount)

	summary := WorkspaceAnalyticsSummary{TripCount: len(in.Trips)}
	budgetTotal := 0.0
	hasBudget := false

	for _, item := range in.Trips {
		trip := item.Trip
		tripAnalytics := item.Analytics
		tripSummary := buildTripCostSummary(trip, tripAnalytics)
		byTrip = append(byTrip, tripSummary)

		summary.EstimatedTotal += tripAnalytics.Summary.EstimatedTotal
		summary.MissingEstimateCount += tripAnalytics.Summary.MissingEstimateCount
		summary.UncertainEstimateCount += tripAnalytics.Summary.UncertainEstimateCount
		summary.ConvertedItemCount += tripAnalytics.Summary.ConvertedItemCount
		summary.UnconvertedItemCount += tripAnalytics.Summary.UnconvertedItemCount
		if tripAnalytics.Summary.BudgetAmount == nil {
			summary.IncompleteBudgetTripCount++
		} else {
			hasBudget = true
			budgetTotal += *tripAnalytics.Summary.BudgetAmount
		}
		if tripAnalytics.Summary.OverBudgetAmount != nil && *tripAnalytics.Summary.OverBudgetAmount > 0 {
			summary.OverBudgetTripCount++
		}

		for _, entry := range tripAnalytics.ByCategory {
			current := categoryTotals[entry.Category]
			current.amount += entry.Amount
			current.count += entry.ItemCount
			categoryTotals[entry.Category] = current
		}
		for _, entry := range tripAnalytics.BySource {
			current := sourceTotals[entry.Source]
			current.amount += entry.Amount
			current.count += entry.ItemCount
			sourceTotals[entry.Source] = current
		}

		month := tripMonth(trip.StartDate)
		currentMonth := monthTotals[month]
		currentMonth.amount += tripAnalytics.Summary.EstimatedTotal
		currentMonth.count++
		monthTotals[month] = currentMonth

		for _, warning := range tripAnalytics.Warnings {
			warnings[warning] = struct{}{}
		}
		for _, expensive := range tripAnalytics.ExpensiveItems {
			copyItem := expensive
			tripID := trip.ID
			copyItem.TripID = &tripID
			copyItem.TripTitle = trip.Destination
			copyItem.Destination = trip.Destination
			allItems = append(allItems, copyItem)
		}
	}

	summary.EstimatedTotal = round2(summary.EstimatedTotal)
	if hasBudget {
		value := round2(budgetTotal)
		summary.BudgetTotal = &value
	}

	sort.SliceStable(byTrip, func(i, j int) bool {
		if byTrip[i].StartDate == nil || byTrip[j].StartDate == nil {
			return byTrip[i].Destination < byTrip[j].Destination
		}
		return *byTrip[i].StartDate < *byTrip[j].StartDate
	})

	expensiveTrips := append([]TripCostSummary(nil), byTrip...)
	sort.SliceStable(expensiveTrips, func(i, j int) bool {
		return expensiveTrips[i].EstimatedTotal > expensiveTrips[j].EstimatedTotal
	})
	if len(expensiveTrips) > 10 {
		expensiveTrips = expensiveTrips[:10]
	}

	sort.SliceStable(allItems, func(i, j int) bool {
		return amountValue(allItems[i].ConvertedAmount) > amountValue(allItems[j].ConvertedAmount)
	})
	if len(allItems) > 20 {
		allItems = allItems[:20]
	}

	warningList := sortedWarningList(warnings)
	if len(warningList) == 0 {
		warningList = []string{PlanningDisclaimer}
	}

	return WorkspaceCostAnalytics{
		WorkspaceID:    in.WorkspaceID,
		Currency:       currency,
		GeneratedAt:    generatedAt,
		DateRange:      dateRange(in.From, in.To),
		Summary:        summary,
		ByTrip:         byTrip,
		ByCategory:     breakdownFromTotals(categoryTotals, "category"),
		BySource:       breakdownFromTotals(sourceTotals, "source"),
		ByMonth:        byMonthFromTotals(monthTotals),
		ExpensiveTrips: expensiveTrips,
		ExpensiveItems: allItems,
		Insights:       buildWorkspaceInsights(in.WorkspaceID, summary, expensiveTrips),
		Warnings:       warningList,
	}
}

type priceMissingSummary struct {
	count     int
	dayNumber int
	itemIndex int
	err       error
}

func collectCostRecords(ctx context.Context, in TripInput, targetCurrency string) ([]costRecord, map[int]int, int, priceMissingSummary) {
	records := make([]costRecord, 0)
	missingByDay := make(map[int]int)
	uncertainCount := 0
	priceMissing := priceMissingSummary{}

	days := append([]aggregate.ItineraryDay(nil), in.Itinerary.Days...)
	sort.SliceStable(days, func(i, j int) bool { return days[i].Day < days[j].Day })

	for _, day := range days {
		for itemIndex := range day.Items {
			item := day.Items[itemIndex]
			if !hasUsableAmount(item.EstimatedCost) {
				if budget.ItemNeedsCost(item.Type) {
					missingByDay[day.Day]++
				}
				if providerPriceLikelyUseful(item) {
					priceMissing.count++
					if priceMissing.dayNumber == 0 {
						priceMissing.dayNumber = day.Day
						priceMissing.itemIndex = itemIndex
					}
				}
				continue
			}

			record, err := buildRecord(ctx, in, targetCurrency, day.Day, itemIndex, item)
			if err != nil {
				priceMissing.err = err
				return records, missingByDay, uncertainCount, priceMissing
			}
			if record.Confidence == budget.ConfidenceLow || record.Confidence == ConfidenceUnknown {
				uncertainCount++
			}
			if providerPriceLikelyUseful(item) && record.Source != budget.SourceProvider && record.Source != budget.SourceAvailability {
				priceMissing.count++
				if priceMissing.dayNumber == 0 {
					priceMissing.dayNumber = day.Day
					priceMissing.itemIndex = itemIndex
				}
			}
			records = append(records, record)
		}
	}

	if in.Trip.Accommodation != nil && hasUsableAmount(in.Trip.Accommodation.EstimatedCost) {
		accommodation := in.Trip.Accommodation
		record, err := buildAccommodationRecord(ctx, in, targetCurrency, accommodation)
		if err != nil {
			priceMissing.err = err
			return records, missingByDay, uncertainCount, priceMissing
		}
		if record.Confidence == budget.ConfidenceLow || record.Confidence == ConfidenceUnknown {
			uncertainCount++
		}
		records = append(records, record)
	}

	return records, missingByDay, uncertainCount, priceMissing
}

func buildRecord(
	ctx context.Context,
	in TripInput,
	targetCurrency string,
	dayNumber int,
	itemIndex int,
	item aggregate.ItineraryItem,
) (costRecord, error) {
	cost := item.EstimatedCost
	amount := *cost.Amount
	sourceCurrency := costCurrency(cost.Currency, targetCurrency)
	converted, err := convertAmount(ctx, in, amount, sourceCurrency, targetCurrency)
	if err != nil {
		return costRecord{}, err
	}
	return costRecord{
		TripID:          in.Trip.ID,
		TripTitle:       in.Trip.Destination,
		Destination:     in.Trip.Destination,
		DayNumber:       dayNumber,
		ItemIndex:       itemIndex,
		Name:            item.Name,
		Type:            item.Type,
		Category:        budget.ItemCategory(cost, item.Type),
		Source:          costSource(cost),
		Confidence:      costConfidence(cost),
		Amount:          round2(amount),
		Currency:        sourceCurrency,
		ConvertedAmount: converted,
	}, nil
}

func buildAccommodationRecord(
	ctx context.Context,
	in TripInput,
	targetCurrency string,
	accommodation *aggregate.Accommodation,
) (costRecord, error) {
	cost := accommodation.EstimatedCost
	amount := *cost.Amount
	sourceCurrency := costCurrency(cost.Currency, targetCurrency)
	converted, err := convertAmount(ctx, in, amount, sourceCurrency, targetCurrency)
	if err != nil {
		return costRecord{}, err
	}
	return costRecord{
		TripID:          in.Trip.ID,
		TripTitle:       in.Trip.Destination,
		Destination:     in.Trip.Destination,
		Name:            accommodation.Name,
		Type:            string(accommodation.Type),
		Category:        budget.CategoryAccommodation,
		Source:          costSource(cost),
		Confidence:      costConfidence(cost),
		Amount:          round2(amount),
		Currency:        sourceCurrency,
		ConvertedAmount: converted,
	}, nil
}

func convertTripBudget(
	ctx context.Context,
	in TripInput,
	targetCurrency string,
) (*float64, []string, *budget.CurrencyConversionResult, error) {
	if in.Trip == nil || in.Trip.BudgetAmount == nil {
		return nil, nil, nil, nil
	}
	from := costCurrency(in.Trip.BudgetCurrency, targetCurrency)
	amount := *in.Trip.BudgetAmount
	if from == targetCurrency {
		return floatPtr(amount), nil, nil, nil
	}
	if !in.ConversionEnabled || in.Converter == nil {
		return nil, []string{fmt.Sprintf("Trip budget in %s could not be converted to %s.", from, targetCurrency)}, nil, nil
	}
	result, err := in.Converter.Convert(ctx, amount, from, targetCurrency)
	if err != nil {
		if in.ConversionFailOpen {
			return nil, []string{fmt.Sprintf("Trip budget in %s could not be converted to %s.", from, targetCurrency)}, nil, nil
		}
		return nil, nil, nil, err
	}
	return floatPtr(result.ConvertedAmount), nil, result, nil
}

func convertAmount(
	ctx context.Context,
	in TripInput,
	amount float64,
	from string,
	to string,
) (*float64, error) {
	if from == to {
		return floatPtr(amount), nil
	}
	if !in.ConversionEnabled || in.Converter == nil {
		return nil, nil
	}
	result, err := in.Converter.Convert(ctx, amount, from, to)
	if err != nil {
		if in.ConversionFailOpen {
			return nil, nil
		}
		return nil, err
	}
	return floatPtr(result.ConvertedAmount), nil
}

func buildSummary(summary budget.Summary, budgetAmount *float64, uncertainCount int) CostAnalyticsSummary {
	accommodationTotal := cloneFloat(summary.AccommodationTotal)
	itemTotal := summary.EstimatedTotal
	if accommodationTotal != nil {
		itemTotal -= *accommodationTotal
	}
	itemTotal = math.Max(0, itemTotal)

	out := CostAnalyticsSummary{
		BudgetAmount:           cloneFloat(budgetAmount),
		EstimatedTotal:         round2(summary.EstimatedTotal),
		ItemEstimatedTotal:     round2(itemTotal),
		AccommodationTotal:     accommodationTotal,
		MissingEstimateCount:   summary.MissingEstimateCount,
		UncertainEstimateCount: uncertainCount,
		ConvertedItemCount:     summary.ConvertedItemCount,
		UnconvertedItemCount:   summary.UnconvertedItemCount,
	}
	if budgetAmount == nil {
		return out
	}
	remaining := round2(*budgetAmount - summary.EstimatedTotal)
	over := round2(math.Max(0, summary.EstimatedTotal-*budgetAmount))
	out.RemainingAmount = &remaining
	out.OverBudgetAmount = &over
	if *budgetAmount > 0 {
		utilization := round2(summary.EstimatedTotal / *budgetAmount * 100)
		out.BudgetUtilizationPercent = &utilization
	}
	return out
}

func buildByDay(
	trip *entity.Trip,
	summary budget.Summary,
	records []costRecord,
	missingByDay map[int]int,
	budgetAmount *float64,
) []CostByDay {
	itemsByDay := make(map[int][]ExpensiveCostItem)
	for _, record := range records {
		if record.DayNumber <= 0 || record.ConvertedAmount == nil {
			continue
		}
		itemsByDay[record.DayNumber] = append(itemsByDay[record.DayNumber], expensiveItem(record, summary.EstimatedTotal))
	}
	for dayNumber := range itemsByDay {
		sort.SliceStable(itemsByDay[dayNumber], func(i, j int) bool {
			return amountValue(itemsByDay[dayNumber][i].ConvertedAmount) > amountValue(itemsByDay[dayNumber][j].ConvertedAmount)
		})
		if len(itemsByDay[dayNumber]) > 3 {
			itemsByDay[dayNumber] = itemsByDay[dayNumber][:3]
		}
	}

	dayCount := len(summary.ByDay)
	budgetShare := (*float64)(nil)
	if budgetAmount != nil && dayCount > 0 {
		share := round2(*budgetAmount / float64(dayCount))
		budgetShare = &share
	}

	out := make([]CostByDay, 0, len(summary.ByDay))
	for _, day := range summary.ByDay {
		var share *float64
		var over *float64
		if budgetShare != nil {
			shareValue := *budgetShare
			share = &shareValue
			overValue := round2(math.Max(0, day.EstimatedTotal-shareValue))
			over = &overValue
		}
		out = append(out, CostByDay{
			DayNumber:            day.DayNumber,
			Date:                 dayDate(trip.StartDate, day.DayNumber),
			EstimatedTotal:       round2(day.EstimatedTotal),
			BudgetShare:          share,
			OverBudgetAmount:     over,
			MissingEstimateCount: missingByDay[day.DayNumber],
			TopItems:             nonNilItems(itemsByDay[day.DayNumber]),
		})
	}
	return out
}

func buildBreakdown(records []costRecord, keyFn func(costRecord) string, field string) []CostAmountBreakdown {
	totals := make(map[string]amountCount)
	for _, record := range records {
		if record.ConvertedAmount == nil {
			continue
		}
		key := keyFn(record)
		if key == "" {
			key = SourceUnknown
		}
		current := totals[key]
		current.amount += *record.ConvertedAmount
		current.count++
		totals[key] = current
	}
	return breakdownFromTotals(totals, field)
}

func breakdownFromTotals(totals map[string]amountCount, field string) []CostAmountBreakdown {
	totalAmount := 0.0
	for _, entry := range totals {
		totalAmount += entry.amount
	}
	keys := orderedBreakdownKeys(totals, field)
	out := make([]CostAmountBreakdown, 0, len(keys))
	for _, key := range keys {
		entry := totals[key]
		breakdown := CostAmountBreakdown{
			Name:       key,
			Amount:     round2(entry.amount),
			Percentage: percentage(entry.amount, totalAmount),
			ItemCount:  entry.count,
		}
		switch field {
		case "category":
			breakdown.Category = key
		case "source":
			breakdown.Source = key
		case "confidence":
			breakdown.Confidence = key
		}
		out = append(out, breakdown)
	}
	return out
}

func orderedBreakdownKeys(totals map[string]amountCount, field string) []string {
	preferred := []string{}
	switch field {
	case "category":
		preferred = []string{
			budget.CategoryFood,
			budget.CategoryTransport,
			budget.CategoryTicket,
			budget.CategoryActivity,
			budget.CategoryAccommodation,
			budget.CategoryShopping,
			budget.CategoryOther,
			CategoryUnknown,
		}
	case "source":
		preferred = []string{
			budget.SourceProvider,
			budget.SourceAvailability,
			budget.SourceManual,
			budget.SourceAI,
			SourceUnknown,
		}
	case "confidence":
		preferred = []string{
			budget.ConfidenceHigh,
			budget.ConfidenceMedium,
			budget.ConfidenceLow,
			ConfidenceUnknown,
		}
	}
	seen := make(map[string]struct{}, len(totals))
	out := make([]string, 0, len(totals))
	for _, key := range preferred {
		if totals[key].count == 0 {
			continue
		}
		out = append(out, key)
		seen[key] = struct{}{}
	}
	rest := make([]string, 0)
	for key, entry := range totals {
		if entry.count == 0 {
			continue
		}
		if _, ok := seen[key]; !ok {
			rest = append(rest, key)
		}
	}
	sort.Strings(rest)
	return append(out, rest...)
}

func buildOriginalCurrencyTotals(records []costRecord) []OriginalCurrencyTotal {
	totals := make(map[string]currencyTotal)
	for _, record := range records {
		current := totals[record.Currency]
		current.amount += record.Amount
		if record.ConvertedAmount != nil {
			current.converted += *record.ConvertedAmount
			current.hasConverted = true
		}
		totals[record.Currency] = current
	}
	currencies := make([]string, 0, len(totals))
	for currency := range totals {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	out := make([]OriginalCurrencyTotal, 0, len(currencies))
	for _, currency := range currencies {
		total := totals[currency]
		var converted *float64
		if total.hasConverted {
			converted = floatPtr(total.converted)
		}
		out = append(out, OriginalCurrencyTotal{
			Currency:        currency,
			Amount:          round2(total.amount),
			ConvertedAmount: converted,
		})
	}
	return out
}

func topExpensiveItems(records []costRecord, limit int, tripTotal float64) []ExpensiveCostItem {
	items := make([]ExpensiveCostItem, 0, len(records))
	for _, record := range records {
		if record.ConvertedAmount == nil {
			continue
		}
		items = append(items, expensiveItem(record, tripTotal))
	}
	sort.SliceStable(items, func(i, j int) bool {
		return amountValue(items[i].ConvertedAmount) > amountValue(items[j].ConvertedAmount)
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items
}

func expensiveItem(record costRecord, tripTotal float64) ExpensiveCostItem {
	return ExpensiveCostItem{
		DayNumber:        record.DayNumber,
		ItemIndex:        record.ItemIndex,
		Name:             fallbackString(record.Name, "Accommodation"),
		Type:             record.Type,
		Category:         record.Category,
		Amount:           record.Amount,
		Currency:         record.Currency,
		ConvertedAmount:  cloneFloat(record.ConvertedAmount),
		Source:           record.Source,
		Confidence:       record.Confidence,
		PercentageOfTrip: percentage(amountValue(record.ConvertedAmount), tripTotal),
	}
}

func buildTripInsights(tripID uuid.UUID, currency string, summary CostAnalyticsSummary, byDay []CostByDay, expensiveItems []ExpensiveCostItem, priceMissing priceMissingSummary) []CostInsight {
	insights := make([]CostInsight, 0)
	if summary.BudgetAmount == nil {
		insights = append(insights, CostInsight{
			Type:     "budget_missing",
			Severity: InsightSeverityInfo,
			Title:    "Budget is not set",
			Message:  "Add a trip budget to compare estimated costs against a planning target.",
		})
	} else if summary.OverBudgetAmount != nil && *summary.OverBudgetAmount > 0 {
		severity := InsightSeverityWarning
		if summary.BudgetUtilizationPercent != nil && *summary.BudgetUtilizationPercent >= 120 {
			severity = InsightSeverityCritical
		}
		insights = append(insights, CostInsight{
			Type:     "trip_over_budget",
			Severity: severity,
			Title:    "Trip is over budget",
			Message:  fmt.Sprintf("Estimated costs are %.2f %s above the trip budget.", *summary.OverBudgetAmount, currency),
			Action:   &CostInsightAction{Type: ActionOptimizeBudget, TripID: &tripID},
		})
	}

	if day := mostOverBudgetDay(byDay); day != nil && day.OverBudgetAmount != nil && *day.OverBudgetAmount > 0 {
		dayNumber := day.DayNumber
		insights = append(insights, CostInsight{
			Type:     "day_over_budget",
			Severity: InsightSeverityWarning,
			Title:    fmt.Sprintf("Day %d is above its budget share", day.DayNumber),
			Message:  fmt.Sprintf("Day %d is estimated %.2f %s above its daily budget share.", day.DayNumber, *day.OverBudgetAmount, currency),
			Action:   &CostInsightAction{Type: ActionOptimizeBudget, TripID: &tripID, DayNumber: &dayNumber},
		})
	}
	if summary.MissingEstimateCount > 0 {
		insights = append(insights, CostInsight{
			Type:     "missing_estimates",
			Severity: InsightSeverityWarning,
			Title:    "Some cost estimates are missing",
			Message:  fmt.Sprintf("%d cost-relevant item(s) likely need an estimate.", summary.MissingEstimateCount),
			Action:   &CostInsightAction{Type: ActionUpdatePrice, TripID: &tripID},
		})
	}
	if summary.UncertainEstimateCount > 0 {
		insights = append(insights, CostInsight{
			Type:     "uncertain_estimates",
			Severity: InsightSeverityInfo,
			Title:    "Some estimates are uncertain",
			Message:  fmt.Sprintf("%d estimate(s) have low or unknown confidence.", summary.UncertainEstimateCount),
		})
	}
	if summary.UnconvertedItemCount > 0 {
		insights = append(insights, CostInsight{
			Type:     "conversion_warnings",
			Severity: InsightSeverityInfo,
			Title:    "Some costs were not converted",
			Message:  fmt.Sprintf("%d cost(s) could not be converted into the selected currency.", summary.UnconvertedItemCount),
		})
	}
	if len(expensiveItems) > 0 && expensiveItems[0].PercentageOfTrip >= 15 {
		item := expensiveItems[0]
		dayNumber := item.DayNumber
		itemIndex := item.ItemIndex
		insights = append(insights, CostInsight{
			Type:     "expensive_item",
			Severity: InsightSeverityInfo,
			Title:    "One item dominates the estimate",
			Message:  fmt.Sprintf("%s is %.2f%% of the current trip estimate.", item.Name, item.PercentageOfTrip),
			Action:   &CostInsightAction{Type: ActionOpenItem, TripID: &tripID, DayNumber: &dayNumber, ItemIndex: &itemIndex},
		})
	}
	if summary.AccommodationTotal != nil && summary.EstimatedTotal > 0 && *summary.AccommodationTotal/summary.EstimatedTotal >= 0.35 {
		insights = append(insights, CostInsight{
			Type:     "accommodation_cost_high",
			Severity: InsightSeverityInfo,
			Title:    "Accommodation is a major cost",
			Message:  "The accommodation estimate is a large share of this planning total.",
		})
	}
	if priceMissing.count > 0 {
		dayNumber := priceMissing.dayNumber
		itemIndex := priceMissing.itemIndex
		insights = append(insights, CostInsight{
			Type:     "provider_prices_missing",
			Severity: InsightSeverityInfo,
			Title:    "Provider prices could improve this estimate",
			Message:  fmt.Sprintf("%d ticket or activity item(s) may benefit from provider price checks.", priceMissing.count),
			Action:   &CostInsightAction{Type: ActionCheckAvailability, TripID: &tripID, DayNumber: &dayNumber, ItemIndex: &itemIndex},
		})
	}
	return insights
}

func buildWorkspaceInsights(_ uuid.UUID, summary WorkspaceAnalyticsSummary, expensiveTrips []TripCostSummary) []CostInsight {
	insights := make([]CostInsight, 0)
	if summary.OverBudgetTripCount > 0 {
		insights = append(insights, CostInsight{
			Type:     "over_budget_trips",
			Severity: InsightSeverityWarning,
			Title:    "Some trips are over budget",
			Message:  fmt.Sprintf("%d workspace trip(s) are estimated above their budgets.", summary.OverBudgetTripCount),
			Action:   &CostInsightAction{Type: ActionOpenTrip},
		})
	}
	if len(expensiveTrips) > 0 && expensiveTrips[0].EstimatedTotal > 0 {
		tripID := expensiveTrips[0].TripID
		insights = append(insights, CostInsight{
			Type:     "expensive_trip",
			Severity: InsightSeverityInfo,
			Title:    "Top estimated trip",
			Message:  fmt.Sprintf("%s has the highest estimated workspace cost.", expensiveTrips[0].Title),
			Action:   &CostInsightAction{Type: ActionOpenTrip, TripID: &tripID},
		})
	}
	if summary.MissingEstimateCount > 0 {
		insights = append(insights, CostInsight{
			Type:     "missing_estimates_across_workspace",
			Severity: InsightSeverityWarning,
			Title:    "Workspace estimates are incomplete",
			Message:  fmt.Sprintf("%d cost-relevant item(s) across workspace trips likely need estimates.", summary.MissingEstimateCount),
		})
	}
	if summary.UnconvertedItemCount > 0 {
		insights = append(insights, CostInsight{
			Type:     "conversion_warnings",
			Severity: InsightSeverityInfo,
			Title:    "Some costs were not converted",
			Message:  fmt.Sprintf("%d cost(s) could not be converted into the selected currency.", summary.UnconvertedItemCount),
		})
	}
	if summary.IncompleteBudgetTripCount > 0 {
		insights = append(insights, CostInsight{
			Type:     "incomplete_trip_budgets",
			Severity: InsightSeverityInfo,
			Title:    "Some trips have incomplete budget data",
			Message:  fmt.Sprintf("%d trip(s) have no comparable budget in the selected currency.", summary.IncompleteBudgetTripCount),
		})
	}
	if len(insights) == 0 {
		insights = append(insights, CostInsight{
			Type:     "export_report",
			Severity: InsightSeverityInfo,
			Title:    "Workspace cost report is ready",
			Message:  "Export the current workspace estimate for planning review.",
			Action:   &CostInsightAction{Type: ActionExportReport},
		})
	}
	return insights
}

func buildWarnings(conversionWarnings []budget.ConversionWarning, extra []string) []string {
	warnings := map[string]struct{}{
		PlanningDisclaimer: {},
	}
	for _, warning := range conversionWarnings {
		message := fmt.Sprintf("%s costs could not be converted", warning.Currency)
		if warning.Amount != nil {
			message = fmt.Sprintf("%.2f %s could not be converted", *warning.Amount, warning.Currency)
		}
		if warning.Reason != "" {
			message = fmt.Sprintf("%s (%s)", message, strings.ReplaceAll(warning.Reason, "_", " "))
		}
		warnings[message+"."] = struct{}{}
	}
	for _, warning := range extra {
		if strings.TrimSpace(warning) != "" {
			warnings[strings.TrimSpace(warning)] = struct{}{}
		}
	}
	return sortedWarningList(warnings)
}

func sortedWarningList(warnings map[string]struct{}) []string {
	out := make([]string, 0, len(warnings))
	for warning := range warnings {
		out = append(out, warning)
	}
	sort.Strings(out)
	return out
}

func buildTripCostSummary(trip entity.Trip, analytics TripCostAnalytics) TripCostSummary {
	startDate := dateString(trip.StartDate)
	endDate := endDateString(trip.StartDate, int(trip.Days))
	workspaceID := uuid.Nil
	if trip.WorkspaceID != nil {
		workspaceID = *trip.WorkspaceID
	}
	return TripCostSummary{
		TripID:               trip.ID,
		Title:                trip.Destination,
		Destination:          trip.Destination,
		StartDate:            startDate,
		EndDate:              endDate,
		BudgetAmount:         cloneFloat(analytics.Summary.BudgetAmount),
		EstimatedTotal:       analytics.Summary.EstimatedTotal,
		OverBudgetAmount:     cloneFloat(analytics.Summary.OverBudgetAmount),
		MissingEstimateCount: analytics.Summary.MissingEstimateCount,
		WorkspaceID:          workspaceID,
	}
}

func byMonthFromTotals(totals map[string]amountCount) []CostByMonth {
	months := make([]string, 0, len(totals))
	for month := range totals {
		months = append(months, month)
	}
	sort.Strings(months)
	out := make([]CostByMonth, 0, len(months))
	for _, month := range months {
		entry := totals[month]
		out = append(out, CostByMonth{
			Month:          month,
			EstimatedTotal: round2(entry.amount),
			TripCount:      entry.count,
		})
	}
	return out
}

func mostOverBudgetDay(days []CostByDay) *CostByDay {
	var selected *CostByDay
	for i := range days {
		day := &days[i]
		if day.OverBudgetAmount == nil || *day.OverBudgetAmount <= 0 {
			continue
		}
		if selected == nil || amountValue(day.OverBudgetAmount) > amountValue(selected.OverBudgetAmount) {
			selected = day
		}
	}
	return selected
}

func providerPriceLikelyUseful(item aggregate.ItineraryItem) bool {
	text := strings.ToLower(item.Type + " " + item.Name)
	likely := strings.Contains(text, "ticket") ||
		strings.Contains(text, "museum") ||
		strings.Contains(text, "attraction") ||
		strings.Contains(text, "tour") ||
		strings.Contains(text, "activity") ||
		strings.Contains(text, "event")
	if !likely {
		return false
	}
	if item.PriceEnrichment == nil {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(item.PriceEnrichment.Status)) {
	case "matched":
		return false
	default:
		return true
	}
}

func hasUsableAmount(cost *aggregate.EstimatedCost) bool {
	return cost != nil && cost.Amount != nil && *cost.Amount >= 0
}

func costCurrency(itemCurrency, fallback string) string {
	if c := normalizeCurrency(itemCurrency); c != "" {
		return c
	}
	if c := normalizeCurrency(fallback); c != "" {
		return c
	}
	return budget.DefaultCurrency
}

func costSource(cost *aggregate.EstimatedCost) string {
	if cost == nil {
		return SourceUnknown
	}
	switch strings.ToLower(strings.TrimSpace(cost.Source)) {
	case budget.SourceAI:
		return budget.SourceAI
	case budget.SourceManual:
		return budget.SourceManual
	case budget.SourceProvider:
		return budget.SourceProvider
	case budget.SourceAvailability:
		return budget.SourceAvailability
	default:
		return SourceUnknown
	}
}

func costConfidence(cost *aggregate.EstimatedCost) string {
	if cost == nil {
		return ConfidenceUnknown
	}
	switch strings.ToLower(strings.TrimSpace(cost.Confidence)) {
	case budget.ConfidenceLow:
		return budget.ConfidenceLow
	case budget.ConfidenceMedium:
		return budget.ConfidenceMedium
	case budget.ConfidenceHigh:
		return budget.ConfidenceHigh
	default:
		return ConfidenceUnknown
	}
}

func normalizeCurrency(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) != 3 {
		return ""
	}
	for _, r := range value {
		if r < 'A' || r > 'Z' {
			return ""
		}
	}
	return value
}

func percentage(amount, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return round2(amount / total * 100)
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func floatPtr(value float64) *float64 {
	v := round2(value)
	return &v
}

func cloneFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	return floatPtr(*value)
}

func amountValue(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func nonNilItems(items []ExpensiveCostItem) []ExpensiveCostItem {
	if items == nil {
		return []ExpensiveCostItem{}
	}
	return items
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func dayDate(startDate *time.Time, dayNumber int) *string {
	if startDate == nil || dayNumber <= 0 {
		return nil
	}
	date := startDate.AddDate(0, 0, dayNumber-1).Format("2006-01-02")
	return &date
}

func dateString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	date := value.Format("2006-01-02")
	return &date
}

func endDateString(startDate *time.Time, days int) *string {
	if startDate == nil || days <= 0 {
		return nil
	}
	date := startDate.AddDate(0, 0, days-1).Format("2006-01-02")
	return &date
}

func tripMonth(startDate *time.Time) string {
	if startDate == nil {
		return "unknown"
	}
	return startDate.Format("2006-01")
}

func dateRange(from, to *time.Time) DateRange {
	return DateRange{From: dateString(from), To: dateString(to)}
}

func cloneExchangeRateInfo(info *budget.ExchangeRateInfo) *budget.ExchangeRateInfo {
	if info == nil {
		return nil
	}
	copyInfo := *info
	return &copyInfo
}

func mergeExchangeRateInfo(info **budget.ExchangeRateInfo, conversion *budget.CurrencyConversionResult) {
	if conversion == nil {
		return
	}
	if *info == nil {
		*info = &budget.ExchangeRateInfo{
			Provider:     conversion.Provider,
			AsOf:         conversion.AsOf,
			FallbackUsed: conversion.FallbackUsed,
		}
		return
	}
	if (*info).Provider == "" {
		(*info).Provider = conversion.Provider
	}
	if (*info).AsOf.IsZero() || conversion.AsOf.After((*info).AsOf) {
		(*info).AsOf = conversion.AsOf
	}
	(*info).FallbackUsed = (*info).FallbackUsed || conversion.FallbackUsed
}
