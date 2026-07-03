package availability

import "time"

type AvailabilityStatus string

const (
	StatusAvailable   AvailabilityStatus = "available"
	StatusLimited     AvailabilityStatus = "limited"
	StatusUnavailable AvailabilityStatus = "unavailable"
	StatusUnknown     AvailabilityStatus = "unknown"
)

type ProviderResult string

const (
	ProviderResultSuccess       ProviderResult = "success"
	ProviderResultNoMatch       ProviderResult = "no_match"
	ProviderResultUnavailable   ProviderResult = "unavailable"
	ProviderResultProviderError ProviderResult = "provider_error"
	ProviderResultRateLimited   ProviderResult = "rate_limited"
	ProviderResultQuotaExceeded ProviderResult = "quota_exceeded"
	ProviderResultFallback      ProviderResult = "fallback"
)

type PriceType string

const (
	PriceTypePerPerson PriceType = "per_person"
	PriceTypePerGroup  PriceType = "per_group"
	PriceTypeTotal     PriceType = "total"
	PriceTypeUnknown   PriceType = "unknown"
)

type AvailabilitySearchRequest struct {
	Destination string                `json:"destination"`
	Date        string                `json:"date"`
	Currency    string                `json:"currency,omitempty"`
	Item        AvailabilityItem      `json:"item"`
	Travelers   AvailabilityTravelers `json:"travelers,omitempty"`
}

type AvailabilityItem struct {
	Name          string                     `json:"name"`
	Type          string                     `json:"type,omitempty"`
	Description   string                     `json:"description,omitempty"`
	StartTime     string                     `json:"startTime,omitempty"`
	Place         *AvailabilityPlace         `json:"place,omitempty"`
	EstimatedCost *AvailabilityEstimatedCost `json:"estimatedCost,omitempty"`
}

type AvailabilityPlace struct {
	Name            string   `json:"name,omitempty"`
	Address         string   `json:"address,omitempty"`
	Latitude        *float64 `json:"lat,omitempty"`
	Longitude       *float64 `json:"lng,omitempty"`
	Provider        string   `json:"provider,omitempty"`
	ProviderPlaceID string   `json:"providerPlaceId,omitempty"`
}

type AvailabilityEstimatedCost struct {
	Amount     *float64 `json:"amount,omitempty"`
	Currency   string   `json:"currency,omitempty"`
	Category   string   `json:"category,omitempty"`
	Source     string   `json:"source,omitempty"`
	Confidence string   `json:"confidence,omitempty"`
	Note       string   `json:"note,omitempty"`
}

type AvailabilityTravelers struct {
	Adults   int `json:"adults,omitempty"`
	Children int `json:"children,omitempty"`
}

type AvailabilitySearchResult struct {
	Status              AvailabilityStatus   `json:"status"`
	Result              ProviderResult       `json:"result"`
	Provider            string               `json:"provider"`
	ProviderDisplayName string               `json:"providerDisplayName"`
	FallbackUsed        bool                 `json:"fallbackUsed"`
	Cached              bool                 `json:"cached"`
	CheckedAt           time.Time            `json:"checkedAt"`
	CacheExpiresAt      *time.Time           `json:"cacheExpiresAt,omitempty"`
	Match               AvailabilityMatch    `json:"match"`
	Options             []AvailabilityOption `json:"options"`
	Warnings            []string             `json:"warnings,omitempty"`
	Metadata            map[string]any       `json:"metadata,omitempty"`
}

type AvailabilityMatch struct {
	Matched     bool    `json:"matched"`
	Confidence  float64 `json:"confidence"`
	MatchedName string  `json:"matchedName,omitempty"`
}

type AvailabilityOption struct {
	ID                  string             `json:"id"`
	Title               string             `json:"title"`
	Description         string             `json:"description,omitempty"`
	Availability        AvailabilityStatus `json:"availability"`
	Price               *AvailabilityPrice `json:"price,omitempty"`
	PriceType           PriceType          `json:"priceType"`
	StartTimes          []string           `json:"startTimes,omitempty"`
	DurationMinutes     *int               `json:"durationMinutes,omitempty"`
	BookingURL          string             `json:"bookingUrl,omitempty"`
	ProviderName        string             `json:"providerName"`
	CancellationPolicy  string             `json:"cancellationPolicy,omitempty"`
	InstantConfirmation *bool              `json:"instantConfirmation,omitempty"`
	Metadata            map[string]any     `json:"metadata,omitempty"`
}

type AvailabilityPrice struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}
