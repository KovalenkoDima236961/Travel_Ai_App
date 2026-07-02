package priceenrichment

import (
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

var paidTerms = []string{
	"museum",
	"gallery",
	"landmark",
	"attraction",
	"activity",
	"tour",
	"viewpoint",
	"theme_park",
	"historical_site",
	"historic_site",
	"palace",
	"castle",
	"aquarium",
	"zoo",
	"ticket",
	"tower",
}

var skipTerms = []string{
	"walk",
	"rest",
	"break",
	"free_time",
	"note",
	"check_in",
	"checkout",
	"check_out",
	"accommodation",
	"transport",
	"food",
	"restaurant",
	"cafe",
	"shopping",
}

func IsCandidateItem(item aggregate.ItineraryItem) bool {
	if strings.TrimSpace(item.Name) == "" {
		return false
	}
	itemType := normalize(item.Type)
	placeCategory := ""
	if item.Place != nil {
		placeCategory = normalize(item.Place.Category)
	}
	name := normalize(item.Name)

	if containsAny(itemType, skipTerms...) || containsAny(placeCategory, skipTerms...) {
		return false
	}
	if item.Place != nil && strings.TrimSpace(item.Place.Name) != "" {
		return containsAny(itemType, paidTerms...) ||
			containsAny(placeCategory, paidTerms...) ||
			containsAny(name, paidTerms...)
	}
	return containsAny(itemType, paidTerms...) || containsAny(name, paidTerms...)
}

func normalize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("-", "_", " ", "_")
	return strings.Join(strings.Fields(replacer.Replace(value)), "_")
}

func containsAny(value string, terms ...string) bool {
	if value == "" {
		return false
	}
	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}
