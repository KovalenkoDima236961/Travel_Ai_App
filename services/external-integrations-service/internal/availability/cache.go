package availability

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/cache"
	extobs "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/observability"
)

const availabilityCacheMaxEntries = 4096

type cachedResult struct {
	Result    AvailabilitySearchResult
	ExpiresAt time.Time
}

type cachingProvider struct {
	providerName string
	next         AvailabilityProvider
	cache        *cache.TTLCache
	ttl          time.Duration
	log          *zap.Logger
}

func newCachingProvider(providerName string, next AvailabilityProvider, store *cache.TTLCache, ttl time.Duration, log *zap.Logger) AvailabilityProvider {
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

func (p *cachingProvider) Name() string { return p.next.Name() }

func (p *cachingProvider) DisplayName() string { return providerDisplayName(p.next) }

func (p *cachingProvider) SupportsItem(item AvailabilityItem) bool {
	return providerSupportsItem(p.next, item)
}

func (p *cachingProvider) SearchAvailability(ctx context.Context, req AvailabilitySearchRequest) (*AvailabilitySearchResult, error) {
	key := availabilityCacheKey(p.providerName, req)
	if cached, ok := p.cache.Get(key); ok {
		if entry, ok := cached.(cachedResult); ok {
			extobs.RecordProviderCacheHit(p.providerName, availabilityOperation)
			recordAvailabilityCacheHit(p.providerName)
			p.log.Info("availability cache lookup",
				zap.String("provider", p.providerName),
				zap.Bool("cacheHit", true),
			)
			out := copyResult(entry.Result)
			out.Cached = true
			out.CacheExpiresAt = cloneTime(entry.ExpiresAt)
			return &out, nil
		}
	}

	extobs.RecordProviderCacheMiss(p.providerName, availabilityOperation)
	recordAvailabilityCacheMiss(p.providerName)
	result, err := p.next.SearchAvailability(ctx, req)
	if err != nil {
		return nil, err
	}
	if result == nil {
		result = noOptions(p.providerName, providerDisplayName(p.next), "No matching availability options were found.")
	}

	now := time.Now().UTC()
	if result.CheckedAt.IsZero() {
		result.CheckedAt = now
	}
	expiresAt := now.Add(p.ttl)
	result.Cached = false
	result.CacheExpiresAt = &expiresAt

	p.log.Info("availability cache lookup",
		zap.String("provider", p.providerName),
		zap.Bool("cacheHit", false),
	)
	p.cache.Set(key, cachedResult{Result: copyResult(*result), ExpiresAt: expiresAt}, p.ttl)
	return result, nil
}

func availabilityCacheKey(provider string, req AvailabilitySearchRequest) string {
	lat := ""
	lng := ""
	if req.Item.Place != nil {
		if req.Item.Place.Latitude != nil {
			lat = fmt.Sprintf("%.3f", *req.Item.Place.Latitude)
		}
		if req.Item.Place.Longitude != nil {
			lng = fmt.Sprintf("%.3f", *req.Item.Place.Longitude)
		}
	}
	return fmt.Sprintf(
		"availability:%s:%s:%s:%s:%s:%s:%s:%d:%d",
		normalizeKey(provider),
		normalizeKey(req.Destination),
		strings.TrimSpace(req.Date),
		normalizeCurrency(req.Currency),
		normalizeKey(req.Item.Name),
		lat,
		lng,
		req.Travelers.Adults,
		req.Travelers.Children,
	)
}

func copyResult(in AvailabilitySearchResult) AvailabilitySearchResult {
	out := in
	out.Options = make([]AvailabilityOption, len(in.Options))
	for i, option := range in.Options {
		out.Options[i] = option
		if option.Price != nil {
			price := *option.Price
			out.Options[i].Price = &price
		}
		if option.DurationMinutes != nil {
			duration := *option.DurationMinutes
			out.Options[i].DurationMinutes = &duration
		}
		if option.InstantConfirmation != nil {
			instant := *option.InstantConfirmation
			out.Options[i].InstantConfirmation = &instant
		}
		if option.StartTimes != nil {
			out.Options[i].StartTimes = append([]string(nil), option.StartTimes...)
		}
		if option.Warnings != nil {
			out.Options[i].Warnings = append([]string(nil), option.Warnings...)
		}
		if option.Location != nil {
			location := *option.Location
			if option.Location.Latitude != nil {
				lat := *option.Location.Latitude
				location.Latitude = &lat
			}
			if option.Location.Longitude != nil {
				lng := *option.Location.Longitude
				location.Longitude = &lng
			}
			out.Options[i].Location = &location
		}
		if option.Metadata != nil {
			out.Options[i].Metadata = make(map[string]any, len(option.Metadata))
			for key, value := range option.Metadata {
				out.Options[i].Metadata[key] = value
			}
		}
	}
	if in.Warnings != nil {
		out.Warnings = append([]string(nil), in.Warnings...)
	}
	if in.CacheExpiresAt != nil {
		cacheExpiresAt := *in.CacheExpiresAt
		out.CacheExpiresAt = &cacheExpiresAt
	}
	if in.Metadata != nil {
		out.Metadata = make(map[string]any, len(in.Metadata))
		for key, value := range in.Metadata {
			out.Metadata[key] = value
		}
	}
	return out
}

func cloneTime(value time.Time) *time.Time {
	cloned := value
	return &cloned
}
