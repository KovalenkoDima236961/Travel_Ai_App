package placeenrichment

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

const (
	StatusMatched = "matched"
	StatusNoMatch = "no_match"
	StatusSkipped = "skipped"
	StatusFailed  = "failed"

	ReviewStatusPending  = "pending"
	ReviewStatusAccepted = "accepted"
	ReviewStatusChanged  = "changed"
	ReviewStatusRemoved  = "removed"

	maxSearchQueryLength = 200
)

// PlaceSearcher is the External Integrations Service search port consumed by
// automatic enrichment.
type PlaceSearcher interface {
	SearchPlaces(ctx context.Context, query string, destination string) ([]aggregate.PlaceRef, error)
}

// Service enriches generated itinerary items with normalized place metadata.
type Service struct {
	searcher PlaceSearcher
	cfg      Config
}

// EnrichItineraryInput is the request for automatic place enrichment.
type EnrichItineraryInput struct {
	Destination string
	Itinerary   aggregate.Itinerary
}

// EnrichItineraryResult contains the enriched itinerary and summary stats.
type EnrichItineraryResult struct {
	Itinerary aggregate.Itinerary
	Stats     PlaceEnrichmentStats
}

// PlaceEnrichmentStats summarizes one enrichment run.
type PlaceEnrichmentStats struct {
	Attempted int
	Matched   int
	NoMatch   int
	Skipped   int
	Failed    int
}

// New constructs a place enrichment service.
func New(searcher PlaceSearcher, cfg Config) *Service {
	return &Service{searcher: searcher, cfg: cfg.normalized()}
}

// EnrichItinerary attempts to attach places to suitable itinerary items. It
// returns a copied itinerary and does not mutate the input itinerary.
func (s *Service) EnrichItinerary(ctx context.Context, input EnrichItineraryInput) (*EnrichItineraryResult, error) {
	if s.searcher == nil {
		return nil, fmt.Errorf("place search client is not configured")
	}

	destination := strings.TrimSpace(input.Destination)
	if destination == "" {
		destination = strings.TrimSpace(input.Itinerary.Destination)
	}

	out := copyItinerary(input.Itinerary)
	result := &EnrichItineraryResult{Itinerary: out}

	for dayIndex := range result.Itinerary.Days {
		day := &result.Itinerary.Days[dayIndex]
		for itemIndex := range day.Items {
			item := &day.Items[itemIndex]
			if !s.shouldAttempt(*item, result.Stats.Attempted) {
				result.Stats.Skipped++
				continue
			}

			query := buildSearchQuery(*item)
			result.Stats.Attempted++
			places, err := s.searcher.SearchPlaces(ctx, query, destination)
			if err != nil {
				result.Stats.Failed++
				item.PlaceEnrichment = &aggregate.PlaceEnrichmentMeta{
					Status:       StatusFailed,
					ReviewStatus: ReviewStatusPending,
					Query:        query,
					Reason:       "search_failed",
				}
				if !s.cfg.FailOpen {
					return nil, fmt.Errorf("place search failed: %w", err)
				}
				continue
			}

			bestPlace, bestScore, ok := bestMatch(*item, destination, places)
			if !ok || bestScore.Confidence < s.cfg.MinConfidence {
				result.Stats.NoMatch++
				item.PlaceEnrichment = &aggregate.PlaceEnrichmentMeta{
					Status:       StatusNoMatch,
					ReviewStatus: ReviewStatusPending,
					Confidence:   bestScore.Confidence,
					Query:        query,
					Reason:       bestScore.Reason,
				}
				continue
			}

			place := copyPlaceRef(bestPlace)
			item.Place = &place
			item.PlaceEnrichment = &aggregate.PlaceEnrichmentMeta{
				Status:       StatusMatched,
				ReviewStatus: ReviewStatusPending,
				Confidence:   bestScore.Confidence,
				Query:        query,
				Provider:     bestPlace.Provider,
				MatchedAt:    time.Now().UTC().Format(time.RFC3339),
				Reason:       bestScore.Reason,
			}
			result.Stats.Matched++
		}
	}

	return result, nil
}

func (s *Service) shouldAttempt(item aggregate.ItineraryItem, attempted int) bool {
	if attempted >= s.cfg.MaxItems {
		return false
	}
	if strings.TrimSpace(item.Name) == "" {
		return false
	}
	if item.Place != nil && !s.cfg.OverwriteExisting {
		return false
	}
	return isCandidateType(item.Type)
}

func isCandidateType(itemType string) bool {
	switch strings.ToLower(strings.TrimSpace(itemType)) {
	case "place",
		"food",
		"activity",
		"museum",
		"landmark",
		"restaurant",
		"cafe",
		"market",
		"park",
		"attraction",
		"viewpoint":
		return true
	default:
		return false
	}
}

func buildSearchQuery(item aggregate.ItineraryItem) string {
	query := strings.Join(strings.Fields(strings.TrimSpace(item.Name)), " ")
	return truncateRunes(query, maxSearchQueryLength)
}

func truncateRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return strings.TrimSpace(string(runes[:max]))
}

func bestMatch(item aggregate.ItineraryItem, destination string, places []aggregate.PlaceRef) (aggregate.PlaceRef, PlaceMatchScore, bool) {
	var bestPlace aggregate.PlaceRef
	bestScore := PlaceMatchScore{Confidence: 0, Reason: "low_confidence"}
	found := false
	for _, place := range places {
		score := ScorePlace(item, destination, place)
		if !found || score.Confidence > bestScore.Confidence {
			bestPlace = place
			bestScore = score
			found = true
		}
	}
	return bestPlace, bestScore, found
}

func copyItinerary(in aggregate.Itinerary) aggregate.Itinerary {
	out := in
	out.Days = make([]aggregate.ItineraryDay, len(in.Days))
	for dayIndex := range in.Days {
		out.Days[dayIndex] = in.Days[dayIndex]
		out.Days[dayIndex].Items = make([]aggregate.ItineraryItem, len(in.Days[dayIndex].Items))
		for itemIndex := range in.Days[dayIndex].Items {
			item := in.Days[dayIndex].Items[itemIndex]
			if item.EstimatedCost != nil {
				value := *item.EstimatedCost
				item.EstimatedCost = &value
			}
			if item.Place != nil {
				place := copyPlaceRef(*item.Place)
				item.Place = &place
			}
			if item.PlaceEnrichment != nil {
				meta := *item.PlaceEnrichment
				item.PlaceEnrichment = &meta
			}
			out.Days[dayIndex].Items[itemIndex] = item
		}
	}
	return out
}

func copyPlaceRef(in aggregate.PlaceRef) aggregate.PlaceRef {
	out := in
	if in.Latitude != nil {
		value := *in.Latitude
		out.Latitude = &value
	}
	if in.Longitude != nil {
		value := *in.Longitude
		out.Longitude = &value
	}
	if in.Rating != nil {
		value := *in.Rating
		out.Rating = &value
	}
	if in.RatingCount != nil {
		value := *in.RatingCount
		out.RatingCount = &value
	}
	if in.OpeningHours != nil {
		out.OpeningHours = append([]aggregate.OpeningHoursInterval(nil), in.OpeningHours...)
	}
	return out
}
