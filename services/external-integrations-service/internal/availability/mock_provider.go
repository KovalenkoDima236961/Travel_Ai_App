package availability

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"net/url"
	"strings"
)

const mockProviderName = "mock"

type MockAvailabilityProvider struct{}

func NewMockAvailabilityProvider() *MockAvailabilityProvider {
	return &MockAvailabilityProvider{}
}

func (p *MockAvailabilityProvider) Name() string { return mockProviderName }

func (p *MockAvailabilityProvider) DisplayName() string { return "Mock Tickets" }

func (p *MockAvailabilityProvider) SearchAvailability(_ context.Context, req AvailabilitySearchRequest) (*AvailabilitySearchResult, error) {
	currency := normalizeCurrency(req.Currency)
	if currency == "" {
		currency = "EUR"
	}
	if !currencyPattern.MatchString(currency) || !supportedMockCurrency(currency) {
		return nil, ErrUnsupportedCurrency
	}

	itemType := normalizeKey(req.Item.Type)
	itemName := normalizeText(req.Item.Name)
	description := normalizeText(req.Item.Description)
	placeName := ""
	placeAddress := ""
	if req.Item.Place != nil {
		placeName = normalizeText(req.Item.Place.Name)
		placeAddress = normalizeText(req.Item.Place.Address)
	}
	searchText := strings.TrimSpace(strings.Join([]string{itemType, itemName, description, placeName, placeAddress}, " "))
	displayName := firstNonEmpty(req.Item.PlaceName(), req.Item.Name)

	if containsAny(searchText, "sold out", "unavailable") {
		return &AvailabilitySearchResult{
			Status:              StatusUnavailable,
			Result:              ProviderResultUnavailable,
			Provider:            p.Name(),
			ProviderDisplayName: p.DisplayName(),
			Match: AvailabilityMatch{
				Matched:     true,
				Confidence:  0.78,
				MatchedName: displayName,
			},
			Options: []AvailabilityOption{},
			Warnings: []string{
				"Availability and prices can change on the provider website.",
			},
			Metadata: map[string]any{"reason": "deterministic unavailable mock item"},
		}, nil
	}

	if isFreeOrNoBooking(itemType, searchText) {
		return &AvailabilitySearchResult{
			Status:              StatusUnknown,
			Result:              ProviderResultNoMatch,
			Provider:            p.Name(),
			ProviderDisplayName: p.DisplayName(),
			Match: AvailabilityMatch{
				Matched:    false,
				Confidence: 0.25,
			},
			Options: []AvailabilityOption{},
			Warnings: []string{
				"No paid booking option was found for this item.",
				"Availability and prices can change on the provider website.",
			},
			Metadata: map[string]any{"reason": "likely public or non-ticketed item"},
		}, nil
	}

	band, matched := mockAvailabilityBand(itemType, searchText)
	if !matched {
		return &AvailabilitySearchResult{
			Status:              StatusUnknown,
			Result:              ProviderResultNoMatch,
			Provider:            p.Name(),
			ProviderDisplayName: p.DisplayName(),
			Match: AvailabilityMatch{
				Matched:    false,
				Confidence: 0.35,
			},
			Options: []AvailabilityOption{},
			Warnings: []string{
				"No matching bookable option was found.",
				"Availability and prices can change on the provider website.",
			},
			Metadata: map[string]any{"reason": "item did not match a known mock bookable category"},
		}, nil
	}

	key := strings.Join([]string{normalizeText(req.Destination), req.Date, itemName, itemType}, "|")
	status := StatusAvailable
	if deterministicInt(key, 0, 9) <= 1 {
		status = StatusLimited
	}
	amountEUR := deterministicAmount(band.minEUR, band.maxEUR, key)
	amount := convertMockAmount(amountEUR, currency)
	duration := deterministicInt(key+":duration", 90, 180)
	optionID := "mock-" + slug(displayName) + "-entry"
	option := AvailabilityOption{
		ID:           optionID,
		Title:        mockOptionTitle(displayName, band.category),
		Description:  "External booking option for the selected date.",
		Availability: status,
		Price: &AvailabilityPrice{
			Amount:   amount,
			Currency: currency,
		},
		PriceType:          PriceTypePerPerson,
		StartTimes:         mockStartTimes(key, req.Item.StartTime),
		DurationMinutes:    &duration,
		BookingURL:         mockBookingURL(displayName, req.Date),
		ProviderName:       p.DisplayName(),
		CancellationPolicy: "unknown",
		Metadata: map[string]any{
			"providerOptionId": optionID,
			"mockCategory":     band.category,
		},
	}

	return &AvailabilitySearchResult{
		Status:              status,
		Result:              ProviderResultSuccess,
		Provider:            p.Name(),
		ProviderDisplayName: p.DisplayName(),
		Match: AvailabilityMatch{
			Matched:     true,
			Confidence:  0.82,
			MatchedName: option.Title,
		},
		Options: []AvailabilityOption{option},
		Warnings: []string{
			"Availability and prices can change on the provider website.",
		},
		Metadata: map[string]any{"reason": "Known mock bookable category"},
	}, nil
}

type availabilityBand struct {
	category string
	minEUR   int
	maxEUR   int
}

func mockAvailabilityBand(itemType, text string) (availabilityBand, bool) {
	kind := firstNonEmpty(itemType, text)
	switch {
	case containsAny(kind, "museum", "gallery") || containsAny(text, "museum", "gallery"):
		return availabilityBand{category: "ticket", minEUR: 10, maxEUR: 25}, true
	case containsAny(kind, "landmark", "attraction", "palace", "castle", "historical", "historic", "ticket") ||
		containsAny(text, "landmark", "attraction", "palace", "castle", "colosseum", "louvre", "eiffel", "ticket"):
		return availabilityBand{category: "ticket", minEUR: 12, maxEUR: 35}, true
	case containsAny(kind, "tour", "activity") || containsAny(text, "tour", "guided tour", "activity", "experience", "workshop"):
		return availabilityBand{category: "activity", minEUR: 25, maxEUR: 80}, true
	case containsAny(kind, "zoo", "aquarium", "theme_park", "amusement") ||
		containsAny(text, "zoo", "aquarium", "theme park", "amusement park"):
		return availabilityBand{category: "ticket", minEUR: 20, maxEUR: 90}, true
	default:
		return availabilityBand{}, false
	}
}

func isFreeOrNoBooking(itemType, text string) bool {
	if containsAny(text, "free", "free walk", "public park", "walk through") {
		return true
	}
	return containsAny(itemType, "park", "walk", "walking", "rest", "break", "transport", "accommodation", "hotel", "note", "food", "restaurant", "cafe") ||
		containsAny(text, "public garden", "public square", "neighborhood walk")
}

func mockOptionTitle(name, category string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Selected attraction"
	}
	if category == "activity" {
		return name + " guided experience"
	}
	return name + " entry ticket"
}

func mockStartTimes(key, preferred string) []string {
	base := []string{"09:00", "10:00", "10:30", "12:00", "14:00", "15:30", "17:00"}
	start := deterministicInt(key+":times", 0, len(base)-3)
	times := []string{base[start], base[start+1], base[start+2]}
	preferred = strings.TrimSpace(preferred)
	if preferred != "" && preferred != times[0] {
		return append([]string{preferred}, times[:2]...)
	}
	return times
}

func mockBookingURL(name, date string) string {
	u := url.URL{
		Scheme: "https",
		Host:   "example.com",
		Path:   "/book/" + slug(name),
	}
	q := u.Query()
	if strings.TrimSpace(date) != "" {
		q.Set("date", date)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func deterministicAmount(minValue, maxValue int, key string) float64 {
	if maxValue <= minValue {
		return float64(minValue)
	}
	return float64(deterministicInt(key, minValue, maxValue))
}

func deterministicInt(key string, minValue, maxValue int) int {
	if maxValue <= minValue {
		return minValue
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	span := maxValue - minValue + 1
	return minValue + int(h.Sum32()%uint32(span))
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

func slug(value string) string {
	value = normalizeText(value)
	if value == "" {
		return "item"
	}
	return strings.ReplaceAll(value, " ", "-")
}

func (i AvailabilityItem) PlaceName() string {
	if i.Place != nil && strings.TrimSpace(i.Place.Name) != "" {
		return i.Place.Name
	}
	return i.Name
}

func stringPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

func debugMockKey(req AvailabilitySearchRequest) string {
	return fmt.Sprintf("%s|%s|%s|%s", req.Destination, req.Date, req.Item.Name, req.Item.Type)
}
