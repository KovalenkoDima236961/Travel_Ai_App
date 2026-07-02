package prices

import (
	"context"
	"hash/fnv"
	"math"
	"strings"
)

const mockProviderName = "mock"

type MockPriceProvider struct{}

func NewMockPriceProvider() *MockPriceProvider {
	return &MockPriceProvider{}
}

func (p *MockPriceProvider) EstimatePrice(_ context.Context, input PriceEstimateInput) (*PriceEstimateResult, error) {
	currency := normalizeCurrency(input.Currency)
	if currency == "" {
		currency = "EUR"
	}
	if !currencyPattern.MatchString(currency) {
		return nil, ErrUnsupportedCurrency
	}
	if !supportedMockCurrency(currency) {
		return nil, ErrUnsupportedCurrency
	}

	category := normalizedCategory(input.Place.Category)
	itemType := ""
	itemName := ""
	description := ""
	if input.ItemContext != nil {
		itemType = normalizedCategory(input.ItemContext.Type)
		itemName = normalizeText(input.ItemContext.Name)
		description = normalizeText(input.ItemContext.Description)
	}
	placeName := normalizeText(input.Place.Name)
	searchText := strings.TrimSpace(strings.Join([]string{placeName, itemName, description}, " "))

	if isLikelyFree(category, itemType, searchText) {
		return noMatch("Likely free public place or non-ticket item", 0.2), nil
	}

	priceType, minEUR, maxEUR, matched := mockPriceBand(category, itemType, searchText)
	if !matched {
		return noMatch("No likely paid ticket price found", 0.2), nil
	}

	base := deterministicAmount(minEUR, maxEUR, strings.Join([]string{
		normalizeText(input.Place.Name),
		category,
		itemType,
		normalizeText(input.Destination),
	}, "|"))
	amount := convertMockAmount(base, currency)
	confidence, matchConfidence := mockConfidence(category, itemType, searchText)
	note := "Estimated entry ticket"
	if priceType == "activity" {
		note = "Estimated activity price"
	}

	return &PriceEstimateResult{
		EstimatedCost: &EstimatedCost{
			Amount:     &amount,
			Currency:   currency,
			Category:   priceType,
			Confidence: confidence,
			Source:     "provider",
			Note:       note,
		},
		Provider:        mockProviderName,
		FallbackUsed:    false,
		PriceType:       stringPtr(priceType),
		Matched:         true,
		MatchConfidence: matchConfidence,
		Metadata: map[string]any{
			"reason": "Known mock attraction category",
		},
	}, nil
}

func noMatch(reason string, confidence float64) *PriceEstimateResult {
	return &PriceEstimateResult{
		EstimatedCost:   nil,
		Provider:        mockProviderName,
		FallbackUsed:    false,
		PriceType:       nil,
		Matched:         false,
		MatchConfidence: confidence,
		Metadata: map[string]any{
			"reason": reason,
		},
	}
}

func mockPriceBand(category, itemType, text string) (string, int, int, bool) {
	kind := firstNonEmpty(category, itemType)
	if containsAny(kind, "theme_park", "amusement_park") || containsAny(text, "theme park", "amusement park") {
		return "ticket", 40, 90, true
	}
	if containsAny(kind, "tour") || containsAny(text, "tour", "guided tour", "class", "workshop") {
		return "activity", 25, 70, true
	}
	if containsAny(kind, "activity") && containsAny(text, "tour", "class", "experience", "ticket") {
		return "activity", 25, 70, true
	}
	if containsAny(kind, "viewpoint", "tower") || containsAny(text, "tower", "viewpoint", "observation") {
		return "ticket", 8, 20, true
	}
	if containsAny(kind, "museum", "gallery") || containsAny(text, "museum", "gallery") {
		return "ticket", 10, 25, true
	}
	if containsAny(kind, "palace", "castle", "aquarium", "zoo", "historical_site", "historic_site") ||
		containsAny(text, "palace", "castle", "aquarium", "zoo") {
		return "ticket", 12, 30, true
	}
	if containsAny(kind, "landmark", "attraction", "ticket") ||
		containsAny(text, "attraction", "landmark", "ticket", "colosseum", "louvre", "eiffel") {
		return "ticket", 12, 30, true
	}
	return "", 0, 0, false
}

func mockConfidence(category, itemType, text string) (string, float64) {
	if category != "" && containsAny(category, "museum", "gallery", "landmark", "attraction", "theme_park", "palace", "castle", "aquarium", "zoo", "viewpoint", "tour", "historical_site", "historic_site") {
		return "high", 0.82
	}
	if itemType != "" && containsAny(itemType, "ticket", "museum", "attraction", "tour", "activity") {
		return "medium", 0.68
	}
	if containsAny(text, "museum", "ticket", "tour", "castle", "palace", "zoo", "aquarium") {
		return "medium", 0.64
	}
	return "low", 0.55
}

func deterministicAmount(minValue, maxValue int, key string) float64 {
	if maxValue <= minValue {
		return float64(minValue)
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	span := maxValue - minValue + 1
	return float64(minValue + int(h.Sum32()%uint32(span)))
}

func convertMockAmount(eurAmount float64, currency string) float64 {
	multiplier := map[string]float64{
		"EUR": 1,
		"USD": 1.09,
		"GBP": 0.86,
		"CZK": 25,
		"JPY": 170,
	}[currency]
	if multiplier == 0 {
		multiplier = 1
	}
	amount := eurAmount * multiplier
	if currency == "JPY" || currency == "CZK" {
		return math.Round(amount)
	}
	return math.Round(amount*100) / 100
}

func supportedMockCurrency(currency string) bool {
	switch currency {
	case "EUR", "USD", "GBP", "CZK", "JPY":
		return true
	default:
		return false
	}
}

func isLikelyFree(category, itemType, text string) bool {
	if strings.Contains(text, "free") {
		return true
	}
	kind := firstNonEmpty(category, itemType)
	return containsAny(kind,
		"park",
		"public_square",
		"square",
		"street",
		"neighborhood",
		"walk",
		"rest",
		"break",
		"free_time",
		"note",
		"cafe",
		"restaurant",
		"transport",
	)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func containsAny(value string, needles ...string) bool {
	value = strings.ToLower(value)
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func stringPtr(value string) *string {
	return &value
}
