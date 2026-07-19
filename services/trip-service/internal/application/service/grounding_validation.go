package service

import (
	"strings"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

const defaultGroundingMinConfidence = 0.65

// applyGroundingValidation is deliberately deterministic and provider-free.
// Provider enrichment remains additional evidence; a model's self-confidence
// never converts an ungrounded name into a verified place.
func applyGroundingValidation(itinerary aggregate.Itinerary, metadata map[string]any) (aggregate.Itinerary, map[string]any) {
	if metadata == nil {
		metadata = map[string]any{}
	}
	seen := map[string]struct{}{}
	grounded, unverified, bad := 0, 0, 0
	status := "unavailable"
	for dayIndex := range itinerary.Days {
		for itemIndex := range itinerary.Days[dayIndex].Items {
			item := &itinerary.Days[dayIndex].Items[itemIndex]
			key := strings.ToLower(strings.TrimSpace(item.Name))
			if key != "" {
				if _, exists := seen[key]; exists {
					item.GroundingValidationStatus = "duplicate_place"
					item.NeedsPlaceReview = true
					item.GroundingWarnings = appendUniqueGroundingWarning(item.GroundingWarnings, "Duplicate named place in itinerary.")
					bad++
					continue
				}
				seen[key] = struct{}{}
			}
			switch item.GroundingSource {
			case "grounded":
				if item.GroundingPlaceID != "" && item.GroundingConfidence != nil && *item.GroundingConfidence >= defaultGroundingMinConfidence {
					item.GroundingValidationStatus = "valid_grounded"
					item.NeedsPlaceReview = false
					grounded++
					status = "available"
				} else {
					item.GroundingValidationStatus = "low_confidence_match"
					item.NeedsPlaceReview = true
					bad++
				}
			case "provider":
				item.GroundingValidationStatus = "valid_provider_match"
				grounded++
				status = "available"
			case "generic":
				item.GroundingValidationStatus = "generic_needs_review"
				item.NeedsPlaceReview = true
				unverified++
				if status == "unavailable" {
					status = "partial"
				}
			default:
				item.GroundingValidationStatus = "unverified_model_suggestion"
				item.NeedsPlaceReview = true
				unverified++
				bad++
				if status == "unavailable" {
					status = "partial"
				}
			}
		}
	}
	metadata["groundingStatus"] = status
	metadata["groundedItemCount"] = grounded
	metadata["unverifiedItemCount"] = unverified
	metadata["badPlaceCount"] = bad
	return itinerary, metadata
}

func appendUniqueGroundingWarning(warnings []string, warning string) []string {
	for _, existing := range warnings {
		if existing == warning {
			return warnings
		}
	}
	return append(warnings, warning)
}
