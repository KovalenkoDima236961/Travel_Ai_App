package budget

import (
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
	currency := resolveSummaryCurrency(trip, itinerary)

	summary := Summary{
		Currency:   currency,
		ByDay:      make([]DaySummary, 0, len(itinerary.Days)),
		ByCategory: make([]CategorySummary, 0),
	}

	categoryTotals := make(map[string]float64)
	categoryCounts := make(map[string]int)

	days := append([]aggregate.ItineraryDay(nil), itinerary.Days...)
	sort.SliceStable(days, func(i, j int) bool { return days[i].Day < days[j].Day })

	for _, day := range days {
		dayTotal := 0.0
		dayMissing := 0

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

			if !currencyMatches(cost.Currency, currency) {
				summary.UnsupportedCurrencyCount++
				continue
			}

			amount := *cost.Amount
			dayTotal += amount
			summary.EstimatedTotal += amount
			summary.EstimatedItemCount++

			category := itemCategory(cost, item.Type)
			categoryTotals[category] += amount
			categoryCounts[category]++
		}

		daySummary := DaySummary{
			DayNumber:            day.Day,
			EstimatedTotal:       round2(dayTotal),
			MissingEstimateCount: dayMissing,
		}
		summary.ByDay = append(summary.ByDay, daySummary)
	}

	addAccommodationCost(&summary, trip, currency, categoryTotals, categoryCounts)

	summary.EstimatedTotal = round2(summary.EstimatedTotal)
	summary.ByCategory = buildCategorySummaries(categoryTotals, categoryCounts)

	applyBudget(&summary, trip)

	return summary
}

func addAccommodationCost(
	summary *Summary,
	trip TripBudget,
	currency string,
	categoryTotals map[string]float64,
	categoryCounts map[string]int,
) {
	if trip.Accommodation == nil || !hasUsableAmount(trip.Accommodation.EstimatedCost) {
		return
	}
	cost := trip.Accommodation.EstimatedCost
	if !currencyMatches(cost.Currency, currency) {
		summary.UnsupportedCurrencyCount++
		return
	}

	amount := *cost.Amount
	summary.EstimatedTotal += amount
	summary.EstimatedItemCount++
	categoryTotals[CategoryAccommodation] += amount
	categoryCounts[CategoryAccommodation]++

	total := round2(amount)
	summary.AccommodationTotal = &total
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

func normalizeType(itemType string) string {
	t := strings.ToLower(strings.TrimSpace(itemType))
	t = strings.ReplaceAll(t, " ", "_")
	t = strings.ReplaceAll(t, "-", "_")
	return t
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
