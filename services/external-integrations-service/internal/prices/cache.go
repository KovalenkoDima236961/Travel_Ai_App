package prices

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/cache"
	extobs "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/observability"
)

const priceCacheMaxEntries = 4096

type cachingProvider struct {
	providerName string
	next         PriceProvider
	cache        *cache.TTLCache
	ttl          time.Duration
	log          *zap.Logger
}

func newCachingProvider(providerName string, next PriceProvider, store *cache.TTLCache, ttl time.Duration, log *zap.Logger) PriceProvider {
	if log == nil {
		log = zap.NewNop()
	}
	return &cachingProvider{
		providerName: strings.ToLower(strings.TrimSpace(providerName)),
		next:         next,
		cache:        store,
		ttl:          ttl,
		log:          log,
	}
}

func (p *cachingProvider) EstimatePrice(ctx context.Context, input PriceEstimateInput) (*PriceEstimateResult, error) {
	key := priceCacheKey(p.providerName, input)
	if cached, ok := p.cache.Get(key); ok {
		if result, ok := cached.(PriceEstimateResult); ok {
			extobs.RecordProviderCacheHit(p.providerName, "price_estimate")
			p.log.Info("price cache lookup",
				zap.String("endpoint", "price"),
				zap.String("provider", p.providerName),
				zap.Bool("cacheHit", true),
			)
			out := copyResult(result)
			return &out, nil
		}
	}

	result, err := p.next.EstimatePrice(ctx, input)
	if err != nil {
		return nil, err
	}
	extobs.RecordProviderCacheMiss(p.providerName, "price_estimate")
	if result == nil {
		result = noMatch("No likely paid ticket price found", 0.2)
	}

	p.log.Info("price cache lookup",
		zap.String("endpoint", "price"),
		zap.String("provider", p.providerName),
		zap.Bool("cacheHit", false),
	)
	if !result.FallbackUsed {
		p.cache.Set(key, copyResult(*result), p.ttl)
	}
	return result, nil
}

func priceCacheKey(provider string, input PriceEstimateInput) string {
	placeProvider := ""
	placeID := ""
	placeName := ""
	category := ""
	if input.Place != nil {
		placeProvider = normalizeKey(input.Place.Provider)
		placeID = normalizeKey(input.Place.ProviderPlaceID)
		placeName = normalizeKey(input.Place.Name)
		category = normalizeKey(input.Place.Category)
	}
	return fmt.Sprintf(
		"price:%s:%s:%s:%s:%s:%s:%s:%s",
		normalizeKey(provider),
		normalizeKey(input.Destination),
		placeProvider,
		placeID,
		placeName,
		category,
		normalizeCurrency(input.Currency),
		strings.TrimSpace(input.Date),
	)
}

func copyResult(in PriceEstimateResult) PriceEstimateResult {
	out := in
	if in.EstimatedCost != nil {
		cost := *in.EstimatedCost
		if cost.Amount != nil {
			amount := *cost.Amount
			cost.Amount = &amount
		}
		out.EstimatedCost = &cost
	}
	if in.PriceType != nil {
		priceType := *in.PriceType
		out.PriceType = &priceType
	}
	if in.Metadata != nil {
		out.Metadata = make(map[string]any, len(in.Metadata))
		for key, value := range in.Metadata {
			out.Metadata[key] = value
		}
	}
	return out
}
