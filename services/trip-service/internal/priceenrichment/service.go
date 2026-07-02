package priceenrichment

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/priceclient"
)

type PriceEstimator interface {
	EstimatePrice(ctx context.Context, input priceclient.PriceEstimateInput) (*priceclient.PriceEstimateResult, error)
}

type Service struct {
	estimator PriceEstimator
	cfg       Config
	now       func() time.Time
}

func New(estimator PriceEstimator, cfg Config) *Service {
	return &Service{estimator: estimator, cfg: cfg.normalized(), now: time.Now}
}

func (s *Service) EnrichItinerary(ctx context.Context, input EnrichItineraryInput) (*EnrichItineraryResult, error) {
	if !s.cfg.Enabled {
		return &EnrichItineraryResult{Itinerary: copyItinerary(input.Itinerary)}, nil
	}
	if s.estimator == nil {
		return nil, fmt.Errorf("price estimator is not configured")
	}

	out := copyItinerary(input.Itinerary)
	result := &EnrichItineraryResult{Itinerary: out}
	destination := strings.TrimSpace(input.Destination)
	if destination == "" {
		destination = strings.TrimSpace(input.Itinerary.Destination)
	}
	currency := resolveCurrency(input.BudgetCurrency, input.UserPreferredCurrency, input.Itinerary.Currency, s.cfg.DefaultCurrency)

	for dayIndex := range result.Itinerary.Days {
		day := &result.Itinerary.Days[dayIndex]
		for itemIndex := range day.Items {
			item := &day.Items[itemIndex]
			if result.Stats.Candidates >= s.cfg.MaxItems {
				result.Stats.Skipped++
				continue
			}
			if !IsCandidateItem(*item) {
				result.Stats.Skipped++
				continue
			}
			result.Stats.Candidates++

			if !s.canOverwrite(item.EstimatedCost) {
				result.Stats.NotOverwrittenExistingCost++
				item.PriceEnrichment = &aggregate.PriceEnrichmentMeta{
					Status:       StatusSkipped,
					ReviewStatus: ReviewStatusPending,
					UpdatedAt:    s.now().UTC().Format(time.RFC3339),
					Reason:       "existing_cost_preserved",
				}
				continue
			}

			req := buildEstimateInput(destination, currency, itemDate(input.StartDate, day.Day), *item)
			price, err := s.estimator.EstimatePrice(ctx, req)
			if err != nil {
				result.Stats.Failed++
				item.PriceEnrichment = &aggregate.PriceEnrichmentMeta{
					Status:       StatusFailed,
					ReviewStatus: ReviewStatusPending,
					UpdatedAt:    s.now().UTC().Format(time.RFC3339),
					Reason:       "provider_failed",
				}
				if !s.cfg.FailOpen {
					return nil, fmt.Errorf("price estimate failed: %w", err)
				}
				continue
			}
			if price == nil || !price.Matched || price.EstimatedCost == nil || price.MatchConfidence < s.cfg.MinMatchConfidence {
				result.Stats.NoMatch++
				item.PriceEnrichment = &aggregate.PriceEnrichmentMeta{
					Status:          StatusNoMatch,
					Provider:        providerName(price),
					MatchConfidence: matchConfidence(price),
					ReviewStatus:    ReviewStatusPending,
					UpdatedAt:       s.now().UTC().Format(time.RFC3339),
					Reason:          resultReason(price),
				}
				continue
			}

			cost := *price.EstimatedCost
			if err := budget.NormalizeEstimatedCost(&cost, budget.SourceProvider); err != nil {
				result.Stats.Failed++
				item.PriceEnrichment = &aggregate.PriceEnrichmentMeta{
					Status:       StatusFailed,
					Provider:     price.Provider,
					ReviewStatus: ReviewStatusPending,
					UpdatedAt:    s.now().UTC().Format(time.RFC3339),
					Reason:       "invalid_provider_cost",
				}
				if !s.cfg.FailOpen {
					return nil, fmt.Errorf("price estimate invalid: %w", err)
				}
				continue
			}
			if item.EstimatedCost != nil {
				result.Stats.Overwritten++
			}
			item.EstimatedCost = &cost
			item.PriceEnrichment = &aggregate.PriceEnrichmentMeta{
				Status:          StatusMatched,
				Provider:        price.Provider,
				MatchConfidence: price.MatchConfidence,
				PriceType:       derefString(price.PriceType),
				ReviewStatus:    ReviewStatusPending,
				UpdatedAt:       s.now().UTC().Format(time.RFC3339),
				Reason:          resultReason(price),
			}
			result.Stats.Matched++
		}
	}
	return result, nil
}

func (s *Service) canOverwrite(cost *aggregate.EstimatedCost) bool {
	if cost == nil {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(cost.Source)) {
	case "", budget.SourceAI:
		return s.cfg.OverwriteAICosts
	case budget.SourceManual:
		return s.cfg.OverwriteManualCosts
	case budget.SourceProvider:
		return true
	default:
		return false
	}
}

func buildEstimateInput(destination, currency, date string, item aggregate.ItineraryItem) priceclient.PriceEstimateInput {
	place := priceclient.PricePlace{
		Name:     item.Name,
		Category: item.Type,
	}
	if item.Place != nil {
		place = priceclient.PricePlace{
			Provider:        item.Place.Provider,
			ProviderPlaceID: item.Place.ProviderPlaceID,
			Name:            item.Place.Name,
			Address:         item.Place.Address,
			Category:        item.Place.Category,
			Latitude:        copyFloat(item.Place.Latitude),
			Longitude:       copyFloat(item.Place.Longitude),
			Rating:          copyFloat(item.Place.Rating),
		}
	}
	return priceclient.PriceEstimateInput{
		Destination: destination,
		Currency:    currency,
		Date:        date,
		Place:       place,
		ItemContext: &priceclient.PriceItemContext{
			Name:        item.Name,
			Type:        item.Type,
			Description: item.Note,
		},
	}
}

func itemDate(startDate *time.Time, dayNumber int) string {
	if startDate == nil || dayNumber < 1 {
		return ""
	}
	return startDate.AddDate(0, 0, dayNumber-1).Format("2006-01-02")
}

func resolveCurrency(values ...string) string {
	for _, value := range values {
		normalized := strings.ToUpper(strings.TrimSpace(value))
		if normalized != "" {
			return normalized
		}
	}
	return "EUR"
}

func providerName(result *priceclient.PriceEstimateResult) string {
	if result == nil {
		return ""
	}
	return strings.TrimSpace(result.Provider)
}

func matchConfidence(result *priceclient.PriceEstimateResult) float64 {
	if result == nil {
		return 0
	}
	return result.MatchConfidence
}

func resultReason(result *priceclient.PriceEstimateResult) string {
	if result == nil || result.Metadata == nil {
		return ""
	}
	if reason, ok := result.Metadata["reason"].(string); ok {
		return reason
	}
	return ""
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func copyFloat(value *float64) *float64 {
	if value == nil {
		return nil
	}
	out := *value
	return &out
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
				if value.Amount != nil {
					amount := *value.Amount
					value.Amount = &amount
				}
				item.EstimatedCost = &value
			}
			if item.Place != nil {
				place := *item.Place
				place.Latitude = copyFloat(place.Latitude)
				place.Longitude = copyFloat(place.Longitude)
				place.Rating = copyFloat(place.Rating)
				if place.RatingCount != nil {
					ratingCount := *place.RatingCount
					place.RatingCount = &ratingCount
				}
				if place.OpeningHours != nil {
					place.OpeningHours = append([]aggregate.OpeningHoursInterval(nil), place.OpeningHours...)
				}
				item.Place = &place
			}
			if item.PlaceEnrichment != nil {
				meta := *item.PlaceEnrichment
				item.PlaceEnrichment = &meta
			}
			if item.PriceEnrichment != nil {
				meta := *item.PriceEnrichment
				item.PriceEnrichment = &meta
			}
			out.Days[dayIndex].Items[itemIndex] = item
		}
	}
	return out
}
