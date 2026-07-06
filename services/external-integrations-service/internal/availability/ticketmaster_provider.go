package availability

import (
	"context"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
)

// ticketmasterProvider is the real availability adapter for the Ticketmaster
// Discovery API. It gates unsupported item types before any HTTP call, searches
// for events, scores each candidate deterministically, and returns a canonical
// AvailabilitySearchResult. It never claims a booking guarantee — results are
// labelled provider availability and always carry a verify-on-provider warning.
type ticketmasterProvider struct {
	client                 *ticketmasterClient
	minMatchConfidence     float64
	lowConfidenceThreshold float64
	maxOptions             int
	defaultCurrency        string
	log                    *zap.Logger
}

// newTicketmasterProvider builds the provider. A missing API key is reported as
// an auth/config ProviderError so the selector can fall back to mock (local) or
// fail startup (production), matching the weather/place provider convention.
func newTicketmasterProvider(cfg config.AvailabilityConfig, log *zap.Logger) (AvailabilityProvider, error) {
	if log == nil {
		log = zap.NewNop()
	}
	apiKey := strings.TrimSpace(cfg.TicketmasterAPIKey)
	if apiKey == "" {
		return nil, &ProviderError{Provider: ticketmasterProviderName, Kind: providerErrorAuthConfig}
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.TicketmasterBaseURL), "/")
	if baseURL == "" {
		baseURL = "https://app.ticketmaster.com/discovery/v2"
	}
	timeout := time.Duration(cfg.TicketmasterTimeoutSeconds) * time.Second

	maxOptions := cfg.MaxOptions
	if maxOptions <= 0 {
		maxOptions = 10
	}
	return &ticketmasterProvider{
		client:                 newTicketmasterClient(apiKey, baseURL, timeout, log),
		minMatchConfidence:     cfg.MinMatchConfidence,
		lowConfidenceThreshold: cfg.LowConfidenceThreshold,
		maxOptions:             maxOptions,
		defaultCurrency:        normalizeCurrency(cfg.DefaultCurrency),
		log:                    log,
	}, nil
}

func (p *ticketmasterProvider) Name() string { return ticketmasterProviderName }

func (p *ticketmasterProvider) DisplayName() string { return ticketmasterDisplayName }

// SupportsItem lets the service short-circuit unsupported item types before the
// cache/quota chain, so a museum/tour/rest never consumes Ticketmaster quota.
func (p *ticketmasterProvider) SupportsItem(item AvailabilityItem) bool {
	return ticketmasterSupportsItem(item)
}

func (p *ticketmasterProvider) SearchAvailability(ctx context.Context, req AvailabilitySearchRequest) (*AvailabilitySearchResult, error) {
	currency := normalizeCurrency(req.Currency)
	if currency == "" {
		currency = p.defaultCurrency
	}

	// Unsupported item types never reach the network (task: do not consume quota).
	if !ticketmasterSupportsItem(req.Item) {
		return unsupportedItemResult(ticketmasterProviderName, ticketmasterDisplayName), nil
	}

	payload, err := p.client.searchEvents(ctx, p.buildSearchParams(req))
	if err != nil {
		return nil, err
	}

	events := payload.Embedded.Events
	if len(events) == 0 {
		return p.noEventsResult(), nil
	}

	options := p.buildOptions(req, currency, events)
	if len(options) == 0 {
		return p.noEventsResult(), nil
	}

	// Highest-confidence option leads and drives the top-level match.
	sort.SliceStable(options, func(i, j int) bool {
		return options[i].MatchConfidence > options[j].MatchConfidence
	})
	if len(options) > p.maxOptions {
		options = options[:p.maxOptions]
	}

	best := options[0]
	match := AvailabilityMatch{
		Confidence:       best.MatchConfidence,
		MatchedName:      best.Title,
		ProviderEntityID: best.ProviderEntityID,
		ProviderURL:      best.BookingURL,
	}

	result := &AvailabilitySearchResult{
		Provider:            ticketmasterProviderName,
		ProviderDisplayName: ticketmasterDisplayName,
		CheckedAt:           time.Now().UTC(),
		Options:             options,
		Warnings:            []string{ticketmasterVerifyWarning},
		Metadata:            map[string]any{"eventCount": len(events)},
	}

	if best.MatchConfidence < p.minMatchConfidence {
		// Conservative behaviour: never mark low-confidence matches as available.
		match.Matched = false
		result.Status = StatusUnknown
		result.Result = ProviderResultNoMatch
		result.Warnings = append([]string{
			"Possible match only. Verify this is the correct event before applying its price.",
		}, result.Warnings...)
	} else {
		match.Matched = true
		result.Status = best.Availability
		if best.MatchConfidence < p.lowConfidenceThreshold {
			result.Warnings = append([]string{
				"Match confidence is medium. Confirm the event details before applying its price.",
			}, result.Warnings...)
		}
	}
	result.Match = match
	return result, nil
}

// buildSearchParams builds the Discovery API query. City is the primary geo
// filter; coordinates are used only for scoring to avoid over-filtering with the
// deprecated latlong parameter. Date filters use the required UTC "...Z" format.
func (p *ticketmasterProvider) buildSearchParams(req AvailabilitySearchRequest) url.Values {
	params := url.Values{}
	if keyword := strings.TrimSpace(firstNonEmpty(req.Item.Name, req.Item.PlaceName())); keyword != "" {
		params.Set("keyword", keyword)
	}
	if city := strings.TrimSpace(req.Destination); city != "" {
		params.Set("city", city)
	}
	if date := strings.TrimSpace(req.Date); date != "" {
		// The Discovery API filters on a UTC window (must be "...Z"). A local
		// evening event in a UTC-negative city (e.g. New York, 20:00 local) is the
		// next UTC day, so a same-day-only window would silently miss it. Widen the
		// window by ±1 day and let the exact-date match score (dateScore) rank the
		// requested day highest.
		if parsed, err := time.Parse("2006-01-02", date); err == nil {
			params.Set("startDateTime", parsed.AddDate(0, 0, -1).Format("2006-01-02")+"T00:00:00Z")
			params.Set("endDateTime", parsed.AddDate(0, 0, 1).Format("2006-01-02")+"T23:59:59Z")
		}
	}
	if segment := ticketmasterClassification(req.Item.Type); segment != "" {
		params.Set("classificationName", segment)
	}
	size := p.maxOptions
	if size < 1 {
		size = 1
	}
	if size > 50 {
		size = 50
	}
	params.Set("size", strconv.Itoa(size))
	params.Set("sort", "relevance,desc")
	return params
}

// buildOptions scores and maps events, deduplicating by provider entity id.
func (p *ticketmasterProvider) buildOptions(req AvailabilitySearchRequest, currency string, events []tmEvent) []AvailabilityOption {
	seen := make(map[string]struct{}, len(events))
	options := make([]AvailabilityOption, 0, len(events))
	for _, event := range events {
		id := strings.TrimSpace(event.ID)
		if id != "" {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
		}
		score := scoreTicketmasterEvent(req, event)
		option, ok := mapTicketmasterEvent(event, currency, score.total)
		if !ok {
			continue
		}
		options = append(options, option)
	}
	return options
}

func (p *ticketmasterProvider) noEventsResult() *AvailabilitySearchResult {
	return &AvailabilitySearchResult{
		Status:              StatusUnknown,
		Result:              ProviderResultNoMatch,
		Provider:            ticketmasterProviderName,
		ProviderDisplayName: ticketmasterDisplayName,
		CheckedAt:           time.Now().UTC(),
		Match:               AvailabilityMatch{Matched: false, Confidence: 0},
		Options:             []AvailabilityOption{},
		Warnings: []string{
			"No matching events were found on Ticketmaster for this date.",
			ticketmasterVerifyWarning,
		},
	}
}

// unsupportedItemResult is the canonical response for item types a provider does
// not serve. It never consumes provider quota (the provider returns before any
// network call) and is safe to share across real providers.
func unsupportedItemResult(provider, displayName string) *AvailabilitySearchResult {
	return &AvailabilitySearchResult{
		Status:              StatusUnknown,
		Result:              ProviderResultNoMatch,
		Provider:            provider,
		ProviderDisplayName: displayName,
		CheckedAt:           time.Now().UTC(),
		Match:               AvailabilityMatch{Matched: false, Confidence: 0},
		Options:             []AvailabilityOption{},
		Warnings: []string{
			"Availability provider does not support this item type.",
		},
		Metadata: map[string]any{"reason": "unsupported_item_type"},
	}
}
