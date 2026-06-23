package placeenrichment

import (
	"math"
	"sort"
	"strings"
	"unicode"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

// PlaceMatchScore is the deterministic score for one item/place candidate pair.
type PlaceMatchScore struct {
	Confidence float64
	Reason     string
}

// ScorePlace scores how well a provider result matches an itinerary item.
func ScorePlace(item aggregate.ItineraryItem, destination string, place aggregate.PlaceRef) PlaceMatchScore {
	itemName := normalizeText(item.Name)
	placeName := normalizeText(place.Name)
	if itemName == "" || placeName == "" {
		return PlaceMatchScore{Confidence: 0, Reason: "low_confidence"}
	}

	score := 0.0
	reason := "low_confidence"
	switch {
	case itemName == placeName:
		score += 0.65
		reason = "exact_name_match"
	case strings.Contains(placeName, itemName) || strings.Contains(itemName, placeName):
		score += 0.45
		reason = "name_contains_query"
	default:
		overlap := tokenOverlapRatio(itemName, placeName)
		if overlap > 0 {
			score += overlap * 0.35
			reason = "token_overlap"
		}
	}

	if destinationMatches(destination, place) {
		score += 0.10
	}
	if categoryMatches(item.Type, place.Category) {
		score += 0.10
	}
	if hasValidCoordinates(place) {
		score += 0.10
		if reason == "token_overlap" {
			reason = "token_overlap_with_coordinates"
		}
	}
	if place.Rating != nil {
		score += 0.05
	}

	return PlaceMatchScore{
		Confidence: math.Min(score, 1),
		Reason:     reason,
	}
}

func normalizeText(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	var builder strings.Builder
	previousSpace := false
	for _, r := range value {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			previousSpace = false
		case unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r):
			if !previousSpace {
				builder.WriteByte(' ')
				previousSpace = true
			}
		}
	}
	return strings.Join(strings.Fields(builder.String()), " ")
}

func tokenOverlapRatio(a, b string) float64 {
	aTokens := uniqueTokens(a)
	bTokens := uniqueTokens(b)
	if len(aTokens) == 0 || len(bTokens) == 0 {
		return 0
	}

	overlap := 0
	for token := range aTokens {
		if _, ok := bTokens[token]; ok {
			overlap++
		}
	}
	denominator := len(aTokens)
	if len(bTokens) < denominator {
		denominator = len(bTokens)
	}
	return float64(overlap) / float64(denominator)
}

func uniqueTokens(value string) map[string]struct{} {
	tokens := strings.Fields(value)
	out := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		if len(token) < 2 {
			continue
		}
		out[token] = struct{}{}
	}
	return out
}

func destinationMatches(destination string, place aggregate.PlaceRef) bool {
	tokens := sortedTokenList(uniqueTokens(normalizeText(destination)))
	if len(tokens) == 0 {
		return false
	}
	haystack := normalizeText(strings.Join([]string{
		place.Address,
		place.ProviderPlaceID,
		place.MapURL,
	}, " "))
	for _, token := range tokens {
		if len(token) >= 3 && strings.Contains(haystack, token) {
			return true
		}
	}
	return false
}

func sortedTokenList(tokens map[string]struct{}) []string {
	out := make([]string, 0, len(tokens))
	for token := range tokens {
		out = append(out, token)
	}
	sort.Strings(out)
	return out
}

func categoryMatches(itemType string, category string) bool {
	itemType = normalizeText(itemType)
	category = normalizeText(category)
	if itemType == "" || category == "" {
		return false
	}

	switch itemType {
	case "food", "restaurant", "cafe":
		return containsAny(category, "restaurant", "cafe", "food", "trattoria", "bistro")
	case "museum":
		return strings.Contains(category, "museum")
	case "market":
		return strings.Contains(category, "market")
	case "park":
		return containsAny(category, "park", "garden")
	case "landmark", "place", "activity", "attraction", "viewpoint":
		return containsAny(category, "landmark", "attraction", "historic", "site", "castle", "palace", "viewpoint", "church", "cathedral")
	default:
		return strings.Contains(category, itemType)
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

func hasValidCoordinates(place aggregate.PlaceRef) bool {
	if place.Latitude == nil || place.Longitude == nil {
		return false
	}
	return *place.Latitude >= -90 && *place.Latitude <= 90 &&
		*place.Longitude >= -180 && *place.Longitude <= 180
}
