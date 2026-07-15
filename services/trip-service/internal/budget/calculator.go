package budget

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

// TripBudget is the trip-level budget context the calculator needs. Days is the
// trip's planned day count, used to derive the optional daily budget share.
type TripBudget struct {
	Amount        *float64
	Currency      string
	Days          int
	Accommodation *aggregate.Accommodation
	Route         *aggregate.TripRoute
}

// CurrencyConverter is the exchange-rate client port used by converted budget
// summaries. It is intentionally small so tests can substitute deterministic
// converters.
type CurrencyConverter interface {
	Convert(ctx context.Context, amount float64, from string, to string) (*CurrencyConversionResult, error)
}

type ConversionOptions struct {
	Enabled  bool
	FailOpen bool
}

// freeItemTypes are itinerary item types that usually carry no cost, so a
// missing estimate on them is not flagged. Matching is case-insensitive on a
// normalized type string.
var freeItemTypes = toSet([]string{
	"walk", "walking", "viewpoint", "rest", "break",
	"free_time", "freetime", "note", "check_in", "checkin",
	"check_out", "checkout", "photo", "scenic",
})

// paidItemTypes are itinerary item types that usually carry a cost, so a missing
// estimate on them is flagged as missingEstimateCount.
var paidItemTypes = toSet([]string{
	"food", "restaurant", "cafe", "dining", "meal", "lunch", "dinner", "breakfast",
	"ticket", "museum", "attraction", "landmark", "sight", "sightseeing",
	"activity", "tour", "experience", "show",
	"transport", "transfer", "taxi", "train", "bus", "flight",
	"accommodation", "hotel", "hostel", "lodging", "stay",
	"shopping", "market",
})

// CalculateBudgetSummary computes the on-demand budget summary for a trip from
// its budget and itinerary. It parses leniently: malformed or other-currency
// estimates are skipped (and counted) rather than failing, so a summary is
// always returned.
func CalculateBudgetSummary(trip TripBudget, itinerary aggregate.Itinerary) Summary {
	summary, _ := CalculateBudgetSummaryWithConversion(
		context.Background(),
		trip,
		itinerary,
		nil,
		ConversionOptions{},
	)
	return summary
}

// CalculateBudgetSummaryWithConversion computes the on-demand budget summary
// with optional approximate currency conversion. When conversion is disabled or
// no converter is supplied it preserves the old behavior: same-currency costs
// are included and foreign currencies are counted as unsupported/unconverted.
func CalculateBudgetSummaryWithConversion(
	ctx context.Context,
	trip TripBudget,
	itinerary aggregate.Itinerary,
	converter CurrencyConverter,
	options ConversionOptions,
) (Summary, error) {
	currency := resolveSummaryCurrency(trip, itinerary)
	conversionEnabled := options.Enabled && converter != nil

	summary := Summary{
		Currency:               currency,
		OriginalCurrencyTotals: make([]OriginalCurrencyTotal, 0),
		ConversionWarnings:     make([]ConversionWarning, 0),
		ByDay:                  make([]DaySummary, 0, len(itinerary.Days)),
		ByCategory:             make([]CategorySummary, 0),
	}

	categoryTotals := make(map[string]float64)
	categoryCounts := make(map[string]int)
	originalTotals := make(map[string]float64)

	days := append([]aggregate.ItineraryDay(nil), itinerary.Days...)
	sort.SliceStable(days, func(i, j int) bool { return days[i].Day < days[j].Day })

	for _, day := range days {
		dayTotal := 0.0
		dayMissing := 0
		dayOriginalTotals := make(map[string]float64)

		for i := range day.Items {
			item := day.Items[i]
			cost := item.EstimatedCost

			if !hasUsableAmount(cost) {
				if itemNeedsCost(item.Type) {
					dayMissing++
					summary.MissingEstimateCount++
				}
				continue
			}

			originalCurrency := costCurrency(cost.Currency, currency)
			amount := *cost.Amount
			addOriginalTotal(originalTotals, originalCurrency, amount)
			addOriginalTotal(dayOriginalTotals, originalCurrency, amount)

			convertedAmount, converted, ok, reason, err := convertAmount(ctx, converter, conversionEnabled, options.FailOpen, amount, originalCurrency, currency)
			if err != nil {
				return Summary{}, err
			}
			if !ok {
				summary.UnsupportedCurrencyCount++
				summary.UnconvertedItemCount++
				summary.ConversionWarnings = append(summary.ConversionWarnings, ConversionWarning{
					Currency: originalCurrency,
					Amount:   floatPtr(amount),
					Reason:   reason,
				})
				continue
			}
			if converted != nil {
				summary.ConvertedItemCount++
				mergeExchangeRateInfo(&summary, converted)
			}

			dayTotal += convertedAmount
			summary.EstimatedTotal += convertedAmount
			summary.EstimatedItemCount++

			category := itemCategory(cost, item.Type)
			categoryTotals[category] += convertedAmount
			categoryCounts[category]++
		}

		daySummary := DaySummary{
			DayNumber:              day.Day,
			EstimatedTotal:         round2(dayTotal),
			MissingEstimateCount:   dayMissing,
			OriginalCurrencyTotals: buildOriginalCurrencyTotals(dayOriginalTotals),
		}
		summary.ByDay = append(summary.ByDay, daySummary)
	}

	if err := addAccommodationCost(ctx, &summary, trip, currency, converter, conversionEnabled, options.FailOpen, originalTotals, categoryTotals, categoryCounts); err != nil {
		return Summary{}, err
	}
	if err := addRouteTransportCosts(ctx, &summary, trip, itinerary, currency, converter, conversionEnabled, options.FailOpen, originalTotals, categoryTotals, categoryCounts); err != nil {
		return Summary{}, err
	}

	summary.EstimatedTotal = round2(summary.EstimatedTotal)
	summary.OriginalCurrencyTotals = buildOriginalCurrencyTotals(originalTotals)
	summary.ByCategory = buildCategorySummaries(categoryTotals, categoryCounts)

	applyBudget(&summary, trip)

	return summary, nil
}

func addAccommodationCost(
	ctx context.Context,
	summary *Summary,
	trip TripBudget,
	currency string,
	converter CurrencyConverter,
	conversionEnabled bool,
	failOpen bool,
	originalTotals map[string]float64,
	categoryTotals map[string]float64,
	categoryCounts map[string]int,
) error {
	if trip.Accommodation == nil || !hasUsableAmount(trip.Accommodation.EstimatedCost) {
		return nil
	}
	cost := trip.Accommodation.EstimatedCost
	originalCurrency := costCurrency(cost.Currency, currency)
	amount := *cost.Amount
	addOriginalTotal(originalTotals, originalCurrency, amount)

	convertedAmount, converted, ok, reason, err := convertAmount(ctx, converter, conversionEnabled, failOpen, amount, originalCurrency, currency)
	if err != nil {
		return err
	}
	if !ok {
		summary.UnsupportedCurrencyCount++
		summary.UnconvertedItemCount++
		summary.ConversionWarnings = append(summary.ConversionWarnings, ConversionWarning{
			Currency: originalCurrency,
			Amount:   floatPtr(amount),
			Reason:   reason,
		})
		return nil
	}
	if converted != nil {
		summary.ConvertedItemCount++
		mergeExchangeRateInfo(summary, converted)
	}

	summary.EstimatedTotal += convertedAmount
	summary.EstimatedItemCount++
	categoryTotals[CategoryAccommodation] += convertedAmount
	categoryCounts[CategoryAccommodation]++

	total := round2(convertedAmount)
	summary.AccommodationTotal = &total
	return nil
}

func addRouteTransportCosts(
	ctx context.Context,
	summary *Summary,
	trip TripBudget,
	itinerary aggregate.Itinerary,
	currency string,
	converter CurrencyConverter,
	conversionEnabled bool,
	failOpen bool,
	originalTotals map[string]float64,
	categoryTotals map[string]float64,
	categoryCounts map[string]int,
) error {
	if trip.Route == nil {
		return nil
	}
	itineraryLegCosts := itineraryTransportCostLegIDs(itinerary)
	for i := range trip.Route.Legs {
		if _, exists := itineraryLegCosts[trip.Route.Legs[i].ID]; exists {
			continue
		}
		cost := routeLegTransportCost(trip.Route.Legs[i], currency)
		if !hasUsableAmount(cost) {
			continue
		}
		originalCurrency := costCurrency(cost.Currency, currency)
		amount := *cost.Amount
		addOriginalTotal(originalTotals, originalCurrency, amount)

		convertedAmount, converted, ok, reason, err := convertAmount(ctx, converter, conversionEnabled, failOpen, amount, originalCurrency, currency)
		if err != nil {
			return err
		}
		if !ok {
			summary.UnsupportedCurrencyCount++
			summary.UnconvertedItemCount++
			summary.ConversionWarnings = append(summary.ConversionWarnings, ConversionWarning{
				Currency: originalCurrency,
				Amount:   floatPtr(amount),
				Reason:   reason,
			})
			continue
		}
		if converted != nil {
			summary.ConvertedItemCount++
			mergeExchangeRateInfo(summary, converted)
		}

		summary.EstimatedTotal += convertedAmount
		summary.EstimatedItemCount++
		categoryTotals[CategoryTransport] += convertedAmount
		categoryCounts[CategoryTransport]++
	}
	return nil
}

func itineraryTransportCostLegIDs(itinerary aggregate.Itinerary) map[string]struct{} {
	out := map[string]struct{}{}
	for _, day := range itinerary.Days {
		for _, item := range day.Items {
			if item.Transfer == nil || strings.TrimSpace(item.Transfer.LegID) == "" || !hasUsableAmount(item.EstimatedCost) {
				continue
			}
			category := itemCategory(item.EstimatedCost, item.Type)
			if category != CategoryTransport {
				continue
			}
			out[strings.TrimSpace(item.Transfer.LegID)] = struct{}{}
		}
	}
	return out
}

func routeLegTransportCost(leg aggregate.RouteLeg, summaryCurrency string) *aggregate.EstimatedCost {
	if leg.SelectedTransportOption != nil && leg.SelectedTransportOption.EstimatedPrice != nil {
		amount := leg.SelectedTransportOption.EstimatedPrice.Amount
		return &aggregate.EstimatedCost{
			Amount:     &amount,
			Currency:   costCurrency(leg.SelectedTransportOption.EstimatedPrice.Currency, summaryCurrency),
			Category:   CategoryTransport,
			Confidence: leg.SelectedTransportOption.Confidence,
			Source:     SourceProvider,
		}
	}
	return leg.EstimatedCost
}

// applyBudget fills the budget-relative fields. When the trip has no budget,
// tripBudget/remaining/overBudgetBy stay nil and the per-day share is omitted.
func applyBudget(summary *Summary, trip TripBudget) {
	if trip.Amount == nil {
		return
	}

	budgetAmount := round2(*trip.Amount)
	summary.TripBudget = &budgetAmount

	remaining := round2(budgetAmount - summary.EstimatedTotal)
	summary.Remaining = &remaining

	over := round2(math.Max(0, summary.EstimatedTotal-budgetAmount))
	summary.OverBudgetBy = &over

	if trip.Days <= 0 {
		return
	}
	share := round2(budgetAmount / float64(trip.Days))
	for i := range summary.ByDay {
		dayShare := share
		summary.ByDay[i].DailyBudgetShare = &dayShare
		overDay := round2(math.Max(0, summary.ByDay[i].EstimatedTotal-share))
		summary.ByDay[i].OverDailyBudgetBy = &overDay
	}
}

func buildCategorySummaries(totals map[string]float64, counts map[string]int) []CategorySummary {
	out := make([]CategorySummary, 0, len(totals))
	for _, category := range categoryOrder {
		if counts[category] == 0 {
			continue
		}
		out = append(out, CategorySummary{
			Category:       category,
			EstimatedTotal: round2(totals[category]),
			ItemCount:      counts[category],
		})
	}
	return out
}

// resolveSummaryCurrency uses the trip budget currency, else the first item
// estimate currency found, else the default currency.
func resolveSummaryCurrency(trip TripBudget, itinerary aggregate.Itinerary) string {
	if c := strings.ToUpper(strings.TrimSpace(trip.Currency)); c != "" {
		return c
	}
	if trip.Accommodation != nil && trip.Accommodation.EstimatedCost != nil {
		if c := strings.ToUpper(strings.TrimSpace(trip.Accommodation.EstimatedCost.Currency)); c != "" {
			return c
		}
	}
	if trip.Route != nil {
		for _, leg := range trip.Route.Legs {
			cost := routeLegTransportCost(leg, "")
			if cost == nil {
				continue
			}
			if c := strings.ToUpper(strings.TrimSpace(cost.Currency)); c != "" {
				return c
			}
		}
	}
	for _, day := range itinerary.Days {
		for i := range day.Items {
			cost := day.Items[i].EstimatedCost
			if cost == nil {
				continue
			}
			if c := strings.ToUpper(strings.TrimSpace(cost.Currency)); c != "" {
				return c
			}
		}
	}
	if c := strings.ToUpper(strings.TrimSpace(itinerary.Currency)); c != "" {
		return c
	}
	return DefaultCurrency
}

func hasUsableAmount(cost *aggregate.EstimatedCost) bool {
	return cost != nil && cost.Amount != nil && *cost.Amount >= 0
}

// currencyMatches reports whether an item cost currency belongs to the summary.
// An empty item currency is assumed to be the summary currency (no conversion).
func currencyMatches(itemCurrency, summaryCurrency string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(itemCurrency))
	return normalized == "" || normalized == summaryCurrency
}

func costCurrency(itemCurrency, summaryCurrency string) string {
	normalized := strings.ToUpper(strings.TrimSpace(itemCurrency))
	if normalized == "" {
		return summaryCurrency
	}
	return normalized
}

func convertAmount(
	ctx context.Context,
	converter CurrencyConverter,
	conversionEnabled bool,
	failOpen bool,
	amount float64,
	from string,
	to string,
) (float64, *CurrencyConversionResult, bool, string, error) {
	if currencyMatches(from, to) {
		return amount, nil, true, "", nil
	}
	if !conversionEnabled {
		return 0, nil, false, "conversion_disabled", nil
	}
	result, err := converter.Convert(ctx, amount, from, to)
	if err != nil {
		if failOpen {
			return 0, nil, false, conversionReason(err), nil
		}
		return 0, nil, false, conversionReason(err), err
	}
	return result.ConvertedAmount, result, true, "", nil
}

func addOriginalTotal(totals map[string]float64, currency string, amount float64) {
	totals[currency] += amount
}

func buildOriginalCurrencyTotals(totals map[string]float64) []OriginalCurrencyTotal {
	currencies := make([]string, 0, len(totals))
	for currency := range totals {
		currencies = append(currencies, currency)
	}
	sort.Strings(currencies)
	out := make([]OriginalCurrencyTotal, 0, len(currencies))
	for _, currency := range currencies {
		out = append(out, OriginalCurrencyTotal{
			Currency: currency,
			Amount:   round2(totals[currency]),
		})
	}
	return out
}

func mergeExchangeRateInfo(summary *Summary, conversion *CurrencyConversionResult) {
	if conversion == nil {
		return
	}
	if summary.ExchangeRateInfo == nil {
		summary.ExchangeRateInfo = &ExchangeRateInfo{
			Provider:     conversion.Provider,
			AsOf:         conversion.AsOf,
			FallbackUsed: conversion.FallbackUsed,
		}
		return
	}
	if summary.ExchangeRateInfo.Provider == "" {
		summary.ExchangeRateInfo.Provider = conversion.Provider
	}
	if summary.ExchangeRateInfo.AsOf.IsZero() || conversion.AsOf.After(summary.ExchangeRateInfo.AsOf) {
		summary.ExchangeRateInfo.AsOf = conversion.AsOf
	}
	summary.ExchangeRateInfo.FallbackUsed = summary.ExchangeRateInfo.FallbackUsed || conversion.FallbackUsed
}

type reasonedError interface {
	Reason() string
}

func conversionReason(err error) string {
	if err == nil {
		return "conversion_unavailable"
	}
	if reasoned, ok := err.(reasonedError); ok {
		reason := strings.TrimSpace(reasoned.Reason())
		if reason != "" {
			return reason
		}
	}
	return "conversion_unavailable"
}

func floatPtr(value float64) *float64 {
	v := round2(value)
	return &v
}

// itemCategory uses the explicit estimate category when present, otherwise it is
// inferred from the item type.
func itemCategory(cost *aggregate.EstimatedCost, itemType string) string {
	if cost != nil {
		if c := strings.ToLower(strings.TrimSpace(cost.Category)); validCategories[c] {
			return c
		}
	}
	return inferCategory(itemType)
}

// ItemCategory returns the category used by the budget calculator for an
// itinerary item. It is exposed for read-only analytics that need to group the
// same costs without reimplementing category inference.
func ItemCategory(cost *aggregate.EstimatedCost, itemType string) string {
	return itemCategory(cost, itemType)
}

func inferCategory(itemType string) string {
	switch normalizeType(itemType) {
	case "food", "restaurant", "cafe", "dining", "meal", "lunch", "dinner", "breakfast":
		return CategoryFood
	case "transport", "transfer", "taxi", "train", "bus", "flight":
		return CategoryTransport
	case "ticket", "museum", "attraction", "landmark", "sight", "sightseeing":
		return CategoryTicket
	case "activity", "tour", "experience", "show":
		return CategoryActivity
	case "accommodation", "hotel", "hostel", "lodging", "stay":
		return CategoryAccommodation
	case "shopping", "market":
		return CategoryShopping
	default:
		return CategoryOther
	}
}

// itemNeedsCost reports whether a missing estimate on this item type should be
// flagged. Unknown types (e.g. the generic "place") are not flagged, to avoid
// noisy warnings.
func itemNeedsCost(itemType string) bool {
	t := normalizeType(itemType)
	if freeItemTypes[t] {
		return false
	}
	return paidItemTypes[t]
}

// ItemNeedsCost reports whether the budget calculator treats a missing estimate
// on this item type as actionable.
func ItemNeedsCost(itemType string) bool {
	return itemNeedsCost(itemType)
}

func normalizeType(itemType string) string {
	t := strings.ToLower(strings.TrimSpace(itemType))
	t = strings.ReplaceAll(t, " ", "_")
	t = strings.ReplaceAll(t, "-", "_")
	return t
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
