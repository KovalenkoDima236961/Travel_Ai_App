package budgetoptimization

import (
	"math"
	"sort"
	"strings"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/weathercontext"
)

func BuildOptimizeDayInput(
	trip entity.Trip,
	itinerary aggregate.Itinerary,
	dayNumber int,
	summary budget.Summary,
	payload JobPayload,
	instruction string,
	userProfile *usercontext.UserProfile,
	userPreferences *usercontext.UserPreferences,
	weatherForecast *weathercontext.WeatherForecast,
) (OptimizeDayInput, error) {
	day, err := findDay(itinerary, dayNumber)
	if err != nil {
		return OptimizeDayInput{}, err
	}
	currency := strings.ToUpper(strings.TrimSpace(payload.Currency))
	if currency == "" {
		currency = strings.ToUpper(strings.TrimSpace(summary.Currency))
	}
	if currency == "" {
		currency = strings.ToUpper(strings.TrimSpace(trip.BudgetCurrency))
	}
	if currency == "" {
		currency = "EUR"
	}

	daySummary := findDaySummary(summary, dayNumber)
	dayTotal := totalForDay(*day, currency)
	if daySummary != nil {
		dayTotal = daySummary.EstimatedTotal
	}
	target := defaultTargetReduction(dayTotal, daySummary, payload.TargetReductionAmount)

	constraints := defaultConstraints(payload.Constraints)
	return OptimizeDayInput{
		Trip:             trip,
		CurrentItinerary: itinerary,
		DayNumber:        dayNumber,
		CurrentDay:       *day,
		BudgetSummary:    summary,
		BudgetContext: BudgetContext{
			Currency:              currency,
			TripBudget:            summary.TripBudget,
			TripEstimatedTotal:    summary.EstimatedTotal,
			DayEstimatedTotal:     dayTotal,
			DailyBudgetShare:      dailyBudgetShare(daySummary),
			TargetReductionAmount: target,
			ExpensiveItems:        expensiveItems(*day, dayTotal, currency),
		},
		Constraints:     constraints,
		Instruction:     strings.TrimSpace(instruction),
		UserProfile:     userProfile,
		UserPreferences: userPreferences,
		WeatherForecast: weatherForecast,
		Accommodation:   trip.Accommodation,
	}, nil
}

func findDay(itinerary aggregate.Itinerary, dayNumber int) (*aggregate.ItineraryDay, error) {
	for index := range itinerary.Days {
		if itinerary.Days[index].Day == dayNumber {
			return &itinerary.Days[index], nil
		}
	}
	return nil, apperrs.NewInvalidInput("current itinerary is invalid")
}

func findDaySummary(summary budget.Summary, dayNumber int) *budget.DaySummary {
	for index := range summary.ByDay {
		if summary.ByDay[index].DayNumber == dayNumber {
			return &summary.ByDay[index]
		}
	}
	return nil
}

func dailyBudgetShare(day *budget.DaySummary) *float64 {
	if day == nil {
		return nil
	}
	return day.DailyBudgetShare
}

func defaultTargetReduction(dayTotal float64, daySummary *budget.DaySummary, requested *float64) float64 {
	if requested != nil {
		return round2(math.Max(0, *requested))
	}
	if daySummary != nil && daySummary.OverDailyBudgetBy != nil && *daySummary.OverDailyBudgetBy > 0 {
		return round2(*daySummary.OverDailyBudgetBy)
	}
	if dayTotal <= 0 {
		return 0
	}
	return round2(dayTotal * 0.15)
}

func defaultConstraints(in *Constraints) Constraints {
	if in == nil {
		return Constraints{
			PreserveMustSeeItems:      true,
			KeepMealCount:             true,
			AvoidReplacingManualCosts: true,
		}
	}
	out := *in
	return out
}

func expensiveItems(day aggregate.ItineraryDay, dayTotal float64, fallbackCurrency string) []ExpensiveItem {
	items := make([]ExpensiveItem, 0)
	for index, item := range day.Items {
		if item.EstimatedCost == nil || item.EstimatedCost.Amount == nil || *item.EstimatedCost.Amount < 0 {
			continue
		}
		amount := *item.EstimatedCost.Amount
		share := 0.0
		if dayTotal > 0 {
			share = round2(amount / dayTotal)
		}
		if amount < 25 && share < 0.25 {
			continue
		}
		currency := strings.ToUpper(strings.TrimSpace(item.EstimatedCost.Currency))
		if currency == "" {
			currency = fallbackCurrency
		}
		items = append(items, ExpensiveItem{
			ItemIndex:       index,
			ItemName:        item.Name,
			ItemType:        item.Type,
			Amount:          round2(amount),
			Currency:        currency,
			Category:        item.EstimatedCost.Category,
			Source:          item.EstimatedCost.Source,
			Confidence:      item.EstimatedCost.Confidence,
			ShareOfDayTotal: share,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Amount > items[j].Amount
	})
	return items
}

func totalForDay(day aggregate.ItineraryDay, fallbackCurrency string) float64 {
	total := 0.0
	for _, item := range day.Items {
		if item.EstimatedCost == nil || item.EstimatedCost.Amount == nil || *item.EstimatedCost.Amount < 0 {
			continue
		}
		currency := strings.ToUpper(strings.TrimSpace(item.EstimatedCost.Currency))
		if currency == "" {
			currency = fallbackCurrency
		}
		if currency != fallbackCurrency {
			continue
		}
		total += *item.EstimatedCost.Amount
	}
	return round2(total)
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
