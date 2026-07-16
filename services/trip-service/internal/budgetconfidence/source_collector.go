package budgetconfidence

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type collectionResult struct {
	Records                  []costRecord
	Warnings                 []string
	ConversionFailureCount   int
	ConversionApproxCount    int
	ConversionFailureAmount  float64
	ConversionAttemptedCount int
}

func collectCostRecords(ctx context.Context, in Input, currency string) collectionResult {
	result := collectionResult{
		Records:  make([]costRecord, 0),
		Warnings: make([]string, 0),
	}
	records := make([]costRecord, 0)
	records = append(records, collectItineraryCosts(in.Itinerary, currency)...)
	if in.Trip != nil {
		records = append(records, collectAccommodationCosts(in.Trip, currency)...)
		records = append(records, collectRouteCosts(in.Trip.Route, currency)...)
	}
	records = append(records, collectExpenseCosts(in.Expenses, in.Receipts, in.ReceiptOCR, currency)...)

	for i := range records {
		record := records[i]
		converted, warning, attempted := convertRecord(ctx, in, record, currency)
		if attempted {
			result.ConversionAttemptedCount++
		}
		if warning != "" {
			result.Warnings = append(result.Warnings, warning)
		}
		if converted.ConversionFailed {
			result.ConversionFailureCount++
			if converted.OriginalAmount != nil {
				result.ConversionFailureAmount += converted.OriginalAmount.Amount
			}
		}
		if converted.ConversionApprox {
			result.ConversionApproxCount++
		}
		result.Records = append(result.Records, converted)
	}

	return result
}

func collectItineraryCosts(itinerary aggregate.Itinerary, fallbackCurrency string) []costRecord {
	records := make([]costRecord, 0)
	for _, day := range itinerary.Days {
		dayNumber := day.Day
		for itemIndex, item := range day.Items {
			category := normalizeCostCategory(budget.ItemCategory(item.EstimatedCost, item.Type))
			id := fmt.Sprintf("itinerary:%d:%d", dayNumber, itemIndex)
			if hasUsableCost(item.EstimatedCost) {
				source, quality := sourceQualityForEstimatedCost(item.EstimatedCost, item.PriceEnrichment)
				amount := moneyFromEstimatedCost(item.EstimatedCost, fallbackCurrency)
				records = append(records, costRecord{
					ID:             id,
					EntityType:     EntityItineraryItem,
					Category:       category,
					Amount:         amount,
					OriginalAmount: amount,
					Source:         source,
					Confidence:     normalizeToken(item.EstimatedCost.Confidence),
					QualityScore:   quality,
					IsEstimate:     true,
					DayNumber:      intPtr(dayNumber),
					ItemIndex:      intPtr(itemIndex),
					Metadata: map[string]any{
						"name": item.Name,
						"type": item.Type,
					},
				})
				continue
			}
			if budget.ItemNeedsCost(item.Type) {
				records = append(records, missingRecord(id, EntityItineraryItem, category, map[string]any{
					"name": item.Name,
					"type": item.Type,
				}, intPtr(dayNumber), intPtr(itemIndex)))
			}
		}
	}
	return records
}

func collectAccommodationCosts(trip *entity.Trip, fallbackCurrency string) []costRecord {
	if trip == nil {
		return nil
	}
	if trip.Accommodation != nil && hasUsableCost(trip.Accommodation.EstimatedCost) {
		source, quality := sourceQualityForEstimatedCost(trip.Accommodation.EstimatedCost, nil)
		amount := moneyFromEstimatedCost(trip.Accommodation.EstimatedCost, fallbackCurrency)
		return []costRecord{{
			ID:             "accommodation",
			EntityType:     EntityAccommodation,
			Category:       CategoryAccommodation,
			Amount:         amount,
			OriginalAmount: amount,
			Source:         source,
			Confidence:     normalizeToken(trip.Accommodation.EstimatedCost.Confidence),
			QualityScore:   quality,
			IsEstimate:     true,
			Metadata: map[string]any{
				"name": trip.Accommodation.Name,
				"type": trip.Accommodation.Type,
			},
		}}
	}
	if trip.Days > 1 {
		return []costRecord{missingRecord("accommodation_missing", EntityAccommodation, CategoryAccommodation, nil, nil, nil)}
	}
	return nil
}

func collectRouteCosts(route *aggregate.TripRoute, fallbackCurrency string) []costRecord {
	if route == nil {
		return nil
	}
	records := make([]costRecord, 0, len(route.Legs))
	for index, leg := range route.Legs {
		legID := strings.TrimSpace(leg.ID)
		if legID == "" {
			legID = fmt.Sprintf("%d", index)
		}
		if leg.SelectedTransportOption != nil {
			option := leg.SelectedTransportOption
			if option.EstimatedPrice != nil && option.EstimatedPrice.Amount >= 0 {
				amount := &Money{
					Amount:   round2(option.EstimatedPrice.Amount),
					Currency: currencyOrDefault(option.EstimatedPrice.Currency, fallbackCurrency),
				}
				source, quality := sourceQualityForSelectedTransport(option)
				records = append(records, costRecord{
					ID:             "selected_transport:" + legID,
					EntityType:     EntitySelectedTransportOption,
					Category:       CategoryTransport,
					Amount:         amount,
					OriginalAmount: amount,
					Source:         source,
					Confidence:     normalizeToken(option.Confidence),
					QualityScore:   quality,
					IsEstimate:     true,
					RouteLegID:     legID,
					Metadata: map[string]any{
						"provider": option.Provider,
						"mode":     option.Mode,
					},
				})
				continue
			}
			if routeLegNeedsBudgetPrice(leg.Mode) {
				records = append(records, missingRecord("selected_transport_price_missing:"+legID, EntitySelectedTransportOption, CategoryTransport, map[string]any{
					"provider": option.Provider,
					"mode":     option.Mode,
				}, nil, nil))
			}
			continue
		}
		if hasUsableCost(leg.EstimatedCost) {
			source, quality := sourceQualityForEstimatedCost(leg.EstimatedCost, nil)
			amount := moneyFromEstimatedCost(leg.EstimatedCost, fallbackCurrency)
			records = append(records, costRecord{
				ID:             "route_leg:" + legID,
				EntityType:     EntityRouteLeg,
				Category:       CategoryTransport,
				Amount:         amount,
				OriginalAmount: amount,
				Source:         source,
				Confidence:     normalizeToken(leg.EstimatedCost.Confidence),
				QualityScore:   quality,
				IsEstimate:     true,
				RouteLegID:     legID,
				Metadata: map[string]any{
					"mode": leg.Mode,
				},
			})
			continue
		}
		if routeLegNeedsBudgetPrice(leg.Mode) {
			records = append(records, missingRecord("route_leg_price_missing:"+legID, EntityRouteLeg, CategoryTransport, map[string]any{
				"mode": leg.Mode,
			}, nil, nil))
		}
	}
	return records
}

func collectExpenseCosts(
	expenses []entity.TripExpense,
	receipts []entity.TripExpenseReceipt,
	ocr map[uuid.UUID]*entity.ReceiptOCRResult,
	fallbackCurrency string,
) []costRecord {
	receiptIDsByExpense := map[uuid.UUID][]uuid.UUID{}
	for _, receipt := range receipts {
		if receipt.DeletedAt != nil || receipt.ExpenseID == nil {
			continue
		}
		receiptIDsByExpense[*receipt.ExpenseID] = append(receiptIDsByExpense[*receipt.ExpenseID], receipt.ID)
	}

	records := make([]costRecord, 0, len(expenses))
	for _, expense := range expenses {
		if expense.Status == entity.ExpenseStatusDeleted || expense.Amount < 0 {
			continue
		}
		receiptIDs := receiptIDsByExpense[expense.ID]
		receiptBacked := len(receiptIDs) > 0
		source := SourceActualManualExpense
		quality := 90
		if receiptBacked {
			source = SourceActualReceiptExpense
			quality = receiptQuality(receiptIDs, ocr)
		}
		amount := &Money{
			Amount:   round2(expense.Amount),
			Currency: currencyOrDefault(expense.Currency, fallbackCurrency),
		}
		records = append(records, costRecord{
			ID:             "expense:" + expense.ID.String(),
			EntityType:     entityTypeForExpense(receiptBacked),
			Category:       normalizeExpenseCategory(expense.Category),
			Amount:         amount,
			OriginalAmount: amount,
			Source:         source,
			QualityScore:   quality,
			IsActual:       true,
			ExpenseID:      expense.ID,
			ReceiptBacked:  receiptBacked,
			Metadata: map[string]any{
				"linkedAccommodation": expense.LinkedAccommodation,
				"linkedRouteLegId":    stringPtrValue(expense.LinkedRouteLegID),
			},
		})
	}
	return records
}

func convertRecord(ctx context.Context, in Input, record costRecord, targetCurrency string) (costRecord, string, bool) {
	if record.Amount == nil || record.Missing {
		return record, "", false
	}
	from := currencyOrDefault(record.Amount.Currency, targetCurrency)
	record.Amount.Currency = from
	record.OriginalAmount = &Money{Amount: record.Amount.Amount, Currency: from}
	if strings.EqualFold(from, targetCurrency) {
		record.Amount.Currency = targetCurrency
		return record, "", false
	}
	if !in.ConversionEnabled || in.Converter == nil {
		record.Amount = nil
		record.ConversionFailed = true
		return record, fmt.Sprintf("Currency conversion unavailable for %s costs.", from), false
	}
	converted, err := in.Converter.Convert(ctx, record.OriginalAmount.Amount, from, targetCurrency)
	if err != nil {
		record.Amount = nil
		record.ConversionFailed = true
		return record, fmt.Sprintf("Currency conversion unavailable for %s costs.", from), true
	}
	record.Amount = &Money{Amount: round2(converted.ConvertedAmount), Currency: targetCurrency}
	if converted.FallbackUsed {
		record.ConversionApprox = true
		return record, "Currency conversion is approximate.", true
	}
	return record, "", true
}

func missingRecord(id string, entityType EntityType, category Category, metadata map[string]any, dayNumber *int, itemIndex *int) costRecord {
	if metadata == nil {
		metadata = map[string]any{}
	}
	return costRecord{
		ID:           id,
		EntityType:   entityType,
		Category:     category,
		Source:       SourceMissingCost,
		QualityScore: 0,
		IsEstimate:   true,
		Missing:      true,
		DayNumber:    dayNumber,
		ItemIndex:    itemIndex,
		Metadata:     metadata,
	}
}

func sourceQualityForEstimatedCost(cost *aggregate.EstimatedCost, priceMeta *aggregate.PriceEnrichmentMeta) (Source, int) {
	if cost == nil {
		return SourceMissingCost, 0
	}
	source := normalizeToken(cost.Source)
	confidence := normalizeToken(cost.Confidence)
	if source == "mock" || strings.Contains(strings.ToLower(cost.Note), "mock") {
		return SourceMockEstimate, 25
	}
	switch source {
	case budget.SourceProvider, budget.SourceAvailability:
		return SourceProviderPrice, providerQuality(priceMeta, confidence)
	case budget.SourceManual:
		return SourceManualEstimate, 65
	case budget.SourceAI:
		switch confidence {
		case budget.ConfidenceHigh:
			return SourceAIEstimateHighConfidence, 55
		case budget.ConfidenceMedium:
			return SourceAIEstimateMediumConfidence, 45
		case budget.ConfidenceLow:
			return SourceAIEstimateLowConfidence, 30
		default:
			return SourceAIEstimateLowConfidence, 35
		}
	default:
		return SourceUnknown, 20
	}
}

func providerQuality(meta *aggregate.PriceEnrichmentMeta, confidence string) int {
	if meta == nil {
		if confidence == budget.ConfidenceHigh {
			return 85
		}
		return 80
	}
	match := meta.MatchConfidence
	priceType := normalizeToken(meta.PriceType)
	switch {
	case match >= 0.85 && (priceType == "exact" || priceType == "admission" || priceType == "ticket"):
		return 90
	case match >= 0.75:
		return 80
	case match >= 0.55:
		return 70
	case match > 0:
		return 50
	default:
		return 75
	}
}

func sourceQualityForSelectedTransport(option *aggregate.SelectedTransportOption) (Source, int) {
	if option == nil {
		return SourceMissingCost, 0
	}
	provider := normalizeToken(option.Provider)
	confidence := normalizeToken(option.Confidence)
	if provider == "mock" {
		if confidence == budget.ConfidenceLow {
			return SourceMockEstimate, 25
		}
		return SourceMockEstimate, 35
	}
	switch confidence {
	case budget.ConfidenceHigh:
		return SourceSelectedTransportOptionHighConfidence, 85
	case budget.ConfidenceMedium:
		return SourceSelectedTransportOptionMediumConfidence, 75
	case budget.ConfidenceLow:
		return SourceSelectedTransportOptionLowConfidence, 45
	default:
		return SourceSelectedTransportOptionMediumConfidence, 65
	}
}

func receiptQuality(receiptIDs []uuid.UUID, ocr map[uuid.UUID]*entity.ReceiptOCRResult) int {
	if len(receiptIDs) == 0 {
		return 90
	}
	best := 90
	for _, receiptID := range receiptIDs {
		result := ocr[receiptID]
		if result == nil {
			if best < 95 {
				best = 95
			}
			continue
		}
		switch result.Confidence {
		case entity.ReceiptOCRConfidenceHigh:
			return 100
		case entity.ReceiptOCRConfidenceMedium:
			if best < 95 {
				best = 95
			}
		default:
			if best < 90 {
				best = 90
			}
		}
	}
	return best
}

func hasUsableCost(cost *aggregate.EstimatedCost) bool {
	return cost != nil && cost.Amount != nil && *cost.Amount >= 0
}

func moneyFromEstimatedCost(cost *aggregate.EstimatedCost, fallbackCurrency string) *Money {
	if !hasUsableCost(cost) {
		return nil
	}
	return &Money{
		Amount:   round2(*cost.Amount),
		Currency: currencyOrDefault(cost.Currency, fallbackCurrency),
	}
}

func normalizeCostCategory(value string) Category {
	switch normalizeToken(value) {
	case "transport":
		return CategoryTransport
	case "accommodation":
		return CategoryAccommodation
	case "activity", "activities", "tour", "experience":
		return CategoryActivities
	case "ticket", "tickets", "museum", "attraction":
		return CategoryTickets
	case "food", "restaurant", "meal", "dining", "cafe":
		return CategoryFood
	case "shopping":
		return CategoryShopping
	case "fuel":
		return CategoryFuel
	case "parking":
		return CategoryParking
	case "tolls", "toll":
		return CategoryTolls
	case "groceries", "grocery":
		return CategoryGroceries
	case "camping":
		return CategoryCamping
	case "health_safety":
		return CategoryHealthSafety
	default:
		return CategoryOther
	}
}

func normalizeExpenseCategory(value entity.ExpenseCategory) Category {
	return normalizeCostCategory(string(value))
}

func coverageCategoryFor(category Category) CoverageCategory {
	switch category {
	case CategoryTransport:
		return CoverageTransport
	case CategoryAccommodation:
		return CoverageAccommodation
	case CategoryActivities, CategoryTickets:
		return CoverageActivities
	case CategoryFood, CategoryGroceries:
		return CoverageFood
	case CategoryShopping:
		return CoverageShopping
	case CategoryFuel, CategoryParking, CategoryTolls:
		return CoverageFuelParkingTolls
	default:
		return CoverageOther
	}
}

func routeLegNeedsBudgetPrice(mode string) bool {
	switch aggregate.NormalizeRouteToken(mode) {
	case aggregate.TransportModeWalk, aggregate.TransportModeBike, aggregate.TransportModeHiking:
		return false
	default:
		return true
	}
}

func entityTypeForExpense(receiptBacked bool) EntityType {
	if receiptBacked {
		return EntityReceiptExpense
	}
	return EntityExpense
}

func currencyOrDefault(value, fallback string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	if value != "" {
		return value
	}
	fallback = strings.ToUpper(strings.TrimSpace(fallback))
	if fallback != "" {
		return fallback
	}
	return budget.DefaultCurrency
}

func normalizeToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func intPtr(value int) *int {
	return &value
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func roundPercent(value float64) *float64 {
	rounded := round2(value)
	return &rounded
}

func intScore(value float64) int {
	return clampInt(int(math.Round(value)), 0, 100)
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func sortedCoverageCategories(m map[CoverageCategory]categoryCoverage) []CoverageCategory {
	out := make([]CoverageCategory, 0, len(m))
	for category := range m {
		out = append(out, category)
	}
	sort.Slice(out, func(i, j int) bool { return coverageSortRank(out[i]) < coverageSortRank(out[j]) })
	return out
}

func coverageSortRank(category CoverageCategory) int {
	switch category {
	case CoverageTransport:
		return 1
	case CoverageAccommodation:
		return 2
	case CoverageActivities:
		return 3
	case CoverageFood:
		return 4
	case CoverageFuelParkingTolls:
		return 5
	case CoverageShopping:
		return 6
	case CoverageOther:
		return 7
	default:
		return 100
	}
}
