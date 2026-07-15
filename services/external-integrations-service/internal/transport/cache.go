package transport

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/cache"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/observability"
)

const transportCacheMaxEntries = 2048

type cachingProvider struct {
	providerName string
	next         TransportProvider
	cache        *cache.TTLCache
	ttl          time.Duration
	log          *zap.Logger
}

func newCachingProvider(providerName string, next TransportProvider, store *cache.TTLCache, ttl time.Duration, log *zap.Logger) TransportProvider {
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

func (p *cachingProvider) SearchTransportOptions(ctx context.Context, req TransportSearchRequest) (TransportSearchResponse, error) {
	key := transportCacheKey(p.providerName, req)
	if cached, ok := p.cache.Get(key); ok {
		if result, ok := cached.(TransportSearchResponse); ok {
			observability.RecordProviderCacheHit(p.providerName, "transport_search")
			p.log.Info("transport cache lookup",
				zap.String("endpoint", "transport"),
				zap.String("provider", p.providerName),
				zap.Bool("cacheHit", true),
			)
			return result.markCached(), nil
		}
	}

	result, err := p.next.SearchTransportOptions(ctx, req)
	if err != nil {
		return TransportSearchResponse{}, err
	}
	observability.RecordProviderCacheMiss(p.providerName, "transport_search")
	p.log.Info("transport cache lookup",
		zap.String("endpoint", "transport"),
		zap.String("provider", p.providerName),
		zap.Bool("cacheHit", false),
	)
	if !result.Summary.FallbackUsed {
		p.cache.Set(key, cloneResponse(result), p.ttl)
	}
	return result, nil
}

func transportCacheKey(provider string, req TransportSearchRequest) string {
	return fmt.Sprintf(
		"transport:%s:%s:%s:%s:%s:%s:%s:%d:%s",
		normalizeKey(provider),
		locationCachePart(req.Origin),
		locationCachePart(req.Destination),
		strings.TrimSpace(req.Date),
		strings.TrimSpace(req.Time),
		normalizeKey(req.TimePreference),
		strings.Join(normalizedCacheModes(req.Modes), ","),
		req.Travelers,
		normalizeCurrency(req.Currency),
	)
}

func locationCachePart(location Location) string {
	lat := "na"
	lng := "na"
	if location.Lat != nil {
		lat = fmt.Sprintf("%.5f", *location.Lat)
	}
	if location.Lng != nil {
		lng = fmt.Sprintf("%.5f", *location.Lng)
	}
	return fmt.Sprintf("%s:%s:%s:%s", normalizeText(location.Name), lat, lng, normalizeText(location.Country))
}
