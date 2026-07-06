package availability

import (
	"strconv"
	"strings"
)

const ticketmasterVerifyWarning = "Verify availability and final price on Ticketmaster."

// mapTicketmasterEvent converts one discovered event plus its computed match
// confidence into a canonical AvailabilityOption. It returns ok=false when the
// event lacks the minimum data to be a usable option (no id or title).
func mapTicketmasterEvent(event tmEvent, preferredCurrency string, confidence float64) (AvailabilityOption, bool) {
	id := strings.TrimSpace(event.ID)
	title := strings.TrimSpace(event.Name)
	if id == "" || title == "" {
		return AvailabilityOption{}, false
	}

	option := AvailabilityOption{
		ID:               "ticketmaster-" + id,
		Title:            title,
		Availability:     ticketmasterStatusFromCode(event.Dates.Status.Code),
		PriceType:        PriceTypePerPerson,
		Date:             strings.TrimSpace(event.Dates.Start.LocalDate),
		BookingURL:       ticketmasterBookingURL(event.URL),
		ProviderName:     ticketmasterDisplayName,
		ProviderEntityID: id,
		MatchConfidence:  confidence,
		Warnings:         []string{ticketmasterVerifyWarning},
	}

	if startTime := parseLocalTime(event.Dates.Start.LocalTime); startTime != "" {
		option.StartTimes = []string{startTime}
	}
	if price := ticketmasterPrice(event.PriceRanges, preferredCurrency); price != nil {
		option.Price = price
	}
	if location := ticketmasterLocation(event); location != nil {
		option.Location = location
	}
	return option, true
}

// ticketmasterStatusFromCode maps a Discovery API date status code to a canonical
// availability status. "limited" is deliberately not inferred here because the
// Discovery API does not expose inventory levels.
func ticketmasterStatusFromCode(code string) AvailabilityStatus {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "onsale":
		return StatusAvailable
	case "offsale", "canceled", "cancelled":
		return StatusUnavailable
	case "postponed", "rescheduled":
		return StatusUnknown
	default:
		return StatusUnknown
	}
}

// ticketmasterPrice picks a price range, preferring one whose currency matches
// the requested currency. min becomes the amount; a min<max range is labelled
// "from", an exact single value "exact". Prices are never invented.
func ticketmasterPrice(ranges []tmPriceRange, preferredCurrency string) *AvailabilityPrice {
	if len(ranges) == 0 {
		return nil
	}
	preferredCurrency = normalizeCurrency(preferredCurrency)
	selected := ranges[0]
	for _, candidate := range ranges {
		if normalizeCurrency(candidate.Currency) == preferredCurrency && preferredCurrency != "" {
			selected = candidate
			break
		}
	}

	currency := normalizeCurrency(selected.Currency)
	if !currencyPattern.MatchString(currency) {
		return nil
	}
	amount := selected.Min
	qualifier := PriceQualifierFrom
	if selected.Min <= 0 && selected.Max > 0 {
		amount = selected.Max
		qualifier = PriceQualifierEstimate
	}
	if selected.Max > 0 && selected.Max == selected.Min {
		qualifier = PriceQualifierExact
	}
	if amount < 0 {
		return nil
	}
	return &AvailabilityPrice{Amount: amount, Currency: currency, Qualifier: qualifier}
}

func ticketmasterLocation(event tmEvent) *AvailabilityLocation {
	venue := firstVenue(event)
	if venue == nil {
		return nil
	}
	location := &AvailabilityLocation{
		Name:    strings.TrimSpace(venue.Name),
		Address: strings.TrimSpace(venue.Address.Line1),
	}
	if lat, ok := parseCoordinate(venue.Location.Latitude); ok {
		location.Latitude = &lat
	}
	if lng, ok := parseCoordinate(venue.Location.Longitude); ok {
		location.Longitude = &lng
	}
	if location.Name == "" && location.Address == "" && location.Latitude == nil && location.Longitude == nil {
		return nil
	}
	return location
}

// ticketmasterBookingURL returns the event URL only when it is a safe http(s)
// URL; the normalization layer rejects unsafe URLs, so filtering here keeps a
// single bad event from failing the whole response.
func ticketmasterBookingURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || !isSafeHTTPURL(raw) {
		return ""
	}
	return raw
}

// parseLocalTime normalizes an "HH:MM:SS" (or "HH:MM") local time to "HH:MM".
func parseLocalTime(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ":")
	if len(parts) < 2 {
		return ""
	}
	return parts[0] + ":" + parts[1]
}

func parseCoordinate(value string) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}
