package transport

import (
	"regexp"
	"sort"
	"strings"
	"time"
)

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

func normalizeSearchRequest(req *TransportSearchRequest, defaultCurrency string) {
	req.Origin = normalizeLocation(req.Origin)
	req.Destination = normalizeLocation(req.Destination)
	req.Date = strings.TrimSpace(req.Date)
	req.Time = strings.TrimSpace(req.Time)
	req.TimePreference = normalizeKey(req.TimePreference)
	if req.TimePreference == "" {
		req.TimePreference = TimePreferenceDepartAfter
	}
	req.Currency = normalizeCurrency(req.Currency)
	if req.Currency == "" {
		req.Currency = normalizeCurrency(defaultCurrency)
	}
	if req.Currency == "" {
		req.Currency = "EUR"
	}
	req.Locale = strings.TrimSpace(req.Locale)
	if req.Locale == "" {
		req.Locale = "en"
	}
	if req.Travelers == 0 {
		req.Travelers = 1
	}
	req.Modes = normalizeModes(req.Modes)
	if len(req.Modes) == 0 {
		req.Modes = []string{ModeTrain, ModeBus, ModeCar}
	}
	req.Constraints.PreferredModes = normalizeModes(req.Constraints.PreferredModes)
	if req.Constraints.AccessibilityNotes != nil {
		trimmed := strings.TrimSpace(*req.Constraints.AccessibilityNotes)
		if trimmed == "" {
			req.Constraints.AccessibilityNotes = nil
		} else {
			req.Constraints.AccessibilityNotes = &trimmed
		}
	}
}

func validateSearchRequest(req TransportSearchRequest) (string, bool) {
	if !locationHasIdentity(req.Origin) {
		return "origin.name is required when origin coordinates are missing", false
	}
	if !locationHasIdentity(req.Destination) {
		return "destination.name is required when destination coordinates are missing", false
	}
	if req.Date == "" {
		return "date is required", false
	}
	if _, err := time.Parse("2006-01-02", req.Date); err != nil {
		return "date must be YYYY-MM-DD", false
	}
	if req.Time != "" {
		if _, err := time.Parse("15:04", req.Time); err != nil {
			return "time must be HH:mm", false
		}
	}
	switch req.TimePreference {
	case TimePreferenceDepartAfter, TimePreferenceArriveBefore, TimePreferenceFlexible:
	default:
		return "timePreference is unsupported", false
	}
	if req.Travelers < 1 || req.Travelers > 50 {
		return "travelers must be between 1 and 50", false
	}
	if !currencyPattern.MatchString(req.Currency) {
		return "currency must be a 3-letter uppercase code", false
	}
	if len(req.Modes) == 0 || len(req.Modes) > 8 {
		return "modes must contain between 1 and 8 values", false
	}
	for _, mode := range req.Modes {
		if _, ok := supportedModes[mode]; !ok {
			return "unsupported mode", false
		}
	}
	if req.Origin.Lat != nil && req.Origin.Lng != nil && !validLatLng(*req.Origin.Lat, *req.Origin.Lng) {
		return "origin coordinates must contain valid lat/lng", false
	}
	if req.Destination.Lat != nil && req.Destination.Lng != nil && !validLatLng(*req.Destination.Lat, *req.Destination.Lng) {
		return "destination coordinates must contain valid lat/lng", false
	}
	if req.Constraints.MaxDurationMinutes != nil && *req.Constraints.MaxDurationMinutes <= 0 {
		return "constraints.maxDurationMinutes must be greater than 0", false
	}
	if req.Constraints.MaxPriceAmount != nil && *req.Constraints.MaxPriceAmount < 0 {
		return "constraints.maxPriceAmount must be >= 0", false
	}
	return "", true
}

func normalizeLocation(in Location) Location {
	in.Name = strings.TrimSpace(in.Name)
	in.Country = strings.TrimSpace(in.Country)
	in.StopID = strings.TrimSpace(in.StopID)
	return in
}

func locationHasIdentity(location Location) bool {
	if location.Lat != nil && location.Lng != nil {
		return true
	}
	return strings.TrimSpace(location.Name) != ""
}

func normalizeModes(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, raw := range values {
		mode := normalizeMode(raw)
		if mode == "" {
			continue
		}
		if _, ok := seen[mode]; ok {
			continue
		}
		seen[mode] = struct{}{}
		out = append(out, mode)
	}
	return out
}

func normalizeMode(value string) string {
	mode := normalizeKey(value)
	switch mode {
	case "walking":
		return ModeWalk
	case "driving":
		return ModeCar
	case "cycling":
		return ModeBike
	case "public_transportation", "transit":
		return ModePublicTransport
	default:
		return mode
	}
}

func normalizeCurrency(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func normalizeKey(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func normalizeText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.ReplaceAll(value, "-", " ")
	return strings.Join(strings.Fields(value), " ")
}

func normalizedCacheModes(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}
