package availability

import (
	"context"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"

	extobs "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/observability"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/observability"
)

const availabilityOperation = "availability_search"

type Service struct {
	provider AvailabilityProvider
	log      *zap.Logger
	enabled  bool
}

func NewService(provider AvailabilityProvider, log *zap.Logger, enabled bool) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	return &Service{provider: provider, log: log, enabled: enabled}
}

func (s *Service) SearchAvailability(ctx context.Context, input AvailabilitySearchRequest) (*AvailabilitySearchResult, error) {
	if !s.enabled {
		return nil, &ProviderError{Provider: s.provider.Name(), Kind: providerErrorUnavailable}
	}

	start := time.Now()
	provider := s.provider.Name()
	fields := []zap.Field{
		zap.String("provider", provider),
		zap.String("operation", availabilityOperation),
		zap.String("destination", input.Destination),
	}
	fields = append(fields, observability.RequestIDFields(ctx)...)
	s.log.Info("availability_search_started", fields...)

	// Gate unsupported item types before the cache/quota/rate-limit chain so they
	// never consume provider quota (task: do not spend quota on item types a
	// provider cannot serve). providerSupportsItem forwards through every decorator
	// to the real adapter; mock/unconfigured providers support everything.
	var result *AvailabilitySearchResult
	var err error
	if !providerSupportsItem(s.provider, input.Item) {
		result = unsupportedItemResult(provider, providerDisplayName(s.provider))
	} else {
		result, err = s.provider.SearchAvailability(ctx, input)
	}
	duration := time.Since(start)
	if err != nil {
		code := providerErrorKind(err)
		if code == "unknown" {
			code = ErrorProviderUnavailable
		}
		extobs.RecordProviderRequest(provider, availabilityOperation, string(ProviderResultProviderError), duration)
		extobs.RecordProviderFailure(provider, availabilityOperation, code)
		recordAvailabilityRequest(provider, string(ProviderResultProviderError), duration)
		recordAvailabilityError(provider, code)
		failFields := []zap.Field{
			zap.String("provider", provider),
			zap.String("operation", availabilityOperation),
			zap.String("result", string(ProviderResultProviderError)),
			zap.Float64("durationMs", float64(duration.Microseconds())/1000),
			zap.String("errorCode", code),
			zap.Error(err),
		}
		failFields = append(failFields, observability.RequestIDFields(ctx)...)
		s.log.Warn("availability_search_failed", failFields...)
		return nil, err
	}
	if result == nil {
		result = noOptions(provider, providerDisplayName(s.provider), "No matching availability options were found.")
	}
	if err := normalizeResult(result); err != nil {
		recordAvailabilityError(provider, ErrorMalformedResponse)
		return nil, err
	}
	if result.CheckedAt.IsZero() {
		result.CheckedAt = time.Now().UTC()
	}
	if result.Provider == "" {
		result.Provider = provider
	}
	if result.ProviderDisplayName == "" {
		result.ProviderDisplayName = providerDisplayName(s.provider)
	}
	if result.Result == "" {
		result.Result = resultForStatus(result.Status, len(result.Options))
	}
	result.Warnings = ensureBaseWarning(result.Warnings)

	resultLabel := string(result.Result)
	if result.FallbackUsed {
		resultLabel = string(ProviderResultFallback)
		extobs.RecordProviderFallback(provider, availabilityOperation, mockProviderName)
		recordAvailabilityFallback(provider, "provider_fallback")
	}
	extobs.RecordProviderRequest(provider, availabilityOperation, resultLabel, duration)
	recordAvailabilityRequest(provider, resultLabel, duration)
	recordAvailabilityOptions(provider, len(result.Options))
	recordAvailabilityMatchConfidence(result.Provider, availabilityConfidenceBucket(result.Match))

	completeFields := []zap.Field{
		zap.String("provider", result.Provider),
		zap.String("operation", availabilityOperation),
		zap.String("result", resultLabel),
		zap.Float64("durationMs", float64(duration.Microseconds())/1000),
		zap.Bool("fallbackUsed", result.FallbackUsed),
		zap.Bool("cached", result.Cached),
		zap.Int("optionCount", len(result.Options)),
	}
	completeFields = append(completeFields, observability.RequestIDFields(ctx)...)
	s.log.Info("availability_search_completed", completeFields...)
	return result, nil
}

func normalizeResult(result *AvailabilitySearchResult) error {
	result.Provider = strings.ToLower(strings.TrimSpace(result.Provider))
	result.ProviderDisplayName = strings.TrimSpace(result.ProviderDisplayName)
	if result.Status == "" {
		result.Status = statusFromOptions(result.Options)
	}
	for index := range result.Options {
		option := &result.Options[index]
		option.ID = strings.TrimSpace(option.ID)
		option.Title = strings.TrimSpace(option.Title)
		option.BookingURL = strings.TrimSpace(option.BookingURL)
		option.ProviderName = strings.TrimSpace(option.ProviderName)
		if option.Availability == "" {
			option.Availability = result.Status
		}
		if option.PriceType == "" {
			option.PriceType = PriceTypeUnknown
		}
		if option.ProviderName == "" {
			option.ProviderName = result.ProviderDisplayName
		}
		if option.BookingURL != "" && !isSafeHTTPURL(option.BookingURL) {
			return &ProviderError{Provider: result.Provider, Kind: providerErrorMalformed}
		}
		if option.Price != nil {
			option.Price.Currency = normalizeCurrency(option.Price.Currency)
			if !currencyPattern.MatchString(option.Price.Currency) || option.Price.Amount < 0 {
				return &ProviderError{Provider: result.Provider, Kind: providerErrorMalformed}
			}
			if option.Price.Qualifier == "" {
				option.Price.Qualifier = PriceQualifierUnknown
			}
		}
	}
	return nil
}

// availabilityConfidenceBucket buckets a match confidence for the low-cardinality
// metric. Unmatched results always bucket as "none" regardless of raw score.
func availabilityConfidenceBucket(match AvailabilityMatch) string {
	if !match.Matched {
		return ConfidenceBucketNone
	}
	switch {
	case match.Confidence >= 0.80:
		return ConfidenceBucketHigh
	case match.Confidence >= 0.55:
		return ConfidenceBucketMedium
	case match.Confidence > 0:
		return ConfidenceBucketLow
	default:
		return ConfidenceBucketNone
	}
}

func statusFromOptions(options []AvailabilityOption) AvailabilityStatus {
	hasUnknown := false
	for _, option := range options {
		switch option.Availability {
		case StatusAvailable:
			return StatusAvailable
		case StatusLimited:
			return StatusLimited
		case StatusUnknown:
			hasUnknown = true
		}
	}
	if hasUnknown {
		return StatusUnknown
	}
	if len(options) > 0 {
		return StatusUnavailable
	}
	return StatusUnknown
}

func resultForStatus(status AvailabilityStatus, optionCount int) ProviderResult {
	if optionCount == 0 {
		if status == StatusUnavailable {
			return ProviderResultUnavailable
		}
		return ProviderResultNoMatch
	}
	if status == StatusUnavailable {
		return ProviderResultUnavailable
	}
	return ProviderResultSuccess
}

func isSafeHTTPURL(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func ensureBaseWarning(warnings []string) []string {
	const base = "Availability and prices can change on the provider website."
	for _, warning := range warnings {
		if warning == base {
			return warnings
		}
	}
	return append(warnings, base)
}

func noOptions(provider, displayName, reason string) *AvailabilitySearchResult {
	return &AvailabilitySearchResult{
		Status:              StatusUnknown,
		Result:              ProviderResultNoMatch,
		Provider:            provider,
		ProviderDisplayName: displayName,
		Match: AvailabilityMatch{
			Matched:    false,
			Confidence: 0.2,
		},
		Options:  []AvailabilityOption{},
		Warnings: []string{reason, "Availability and prices can change on the provider website."},
	}
}
