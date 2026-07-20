// Package provider defines the trusted travel knowledge provider contract and
// the deterministic mock adapter used by local development and CI.
//
// It lives in the Trip Service module because Trip Service owns the normalized
// knowledge store and Worker Service already reuses that module for ingestion.
// Real network-backed adapters belong in External Integrations Service, behind
// its existing quota, cache, and rate-limit guards; this package never holds
// provider credentials and never performs a network call in CI.
package provider

import (
	"context"
	"errors"
	"time"
)

// Provider name constants mirror the KNOWLEDGE_PROVIDER configuration values.
const (
	ProviderMock        = "mock"
	ProviderFoursquare  = "foursquare"
	ProviderOpenTripMap = "opentripmap"
	ProviderWikidata    = "wikidata"
)

// Controlled provider failures. Callers map these onto the knowledge_provider_*
// error codes instead of leaking adapter internals.
var (
	ErrProviderUnavailable = errors.New("knowledge provider unavailable")
	ErrProviderRateLimited = errors.New("knowledge provider rate limited")
	ErrLicenseMissing      = errors.New("knowledge provider license metadata missing")
	ErrPlaceNotFound       = errors.New("knowledge provider place not found")
)

// LicenseInfo is mandatory for every non-mock adapter. A record whose adapter
// cannot state its license is rejected at ingestion rather than stored with an
// unknown provenance.
type LicenseInfo struct {
	Name          string `json:"name"`
	URL           string `json:"url,omitempty"`
	Attribution   string `json:"attribution,omitempty"`
	TermsURL      string `json:"termsUrl,omitempty"`
	AllowsStorage bool   `json:"allowsStorage"`
	// AllowsRawPayload is false when provider terms discourage retaining the
	// original response; ingestion then stores normalized fields only.
	AllowsRawPayload bool `json:"allowsRawPayload"`
}

// Valid reports whether the license metadata is complete enough to persist
// records from this adapter.
func (l LicenseInfo) Valid() bool {
	return l.Name != "" && l.AllowsStorage
}

// SourcePolicy constrains what an ingestion run may retain, independent of what
// the adapter is technically able to return.
type SourcePolicy struct {
	AllowRawPayload  bool `json:"allowRawPayload"`
	RequireLicense   bool `json:"requireLicense"`
	RequireCoords    bool `json:"requireCoords"`
	MaxRawPayloadKiB int  `json:"maxRawPayloadKiB"`
}

// DefaultSourcePolicy is conservative: license required, raw payload withheld.
func DefaultSourcePolicy() SourcePolicy {
	return SourcePolicy{
		AllowRawPayload:  false,
		RequireLicense:   true,
		RequireCoords:    false,
		MaxRawPayloadKiB: 16,
	}
}

// SearchRequest asks an adapter for candidate places in one destination.
type SearchRequest struct {
	DestinationName string       `json:"destinationName"`
	CountryCode     string       `json:"countryCode,omitempty"`
	Latitude        *float64     `json:"lat,omitempty"`
	Longitude       *float64     `json:"lng,omitempty"`
	RadiusKm        float64      `json:"radiusKm,omitempty"`
	Categories      []string     `json:"categories,omitempty"`
	Limit           int          `json:"limit,omitempty"`
	Language        string       `json:"language,omitempty"`
	SourcePolicy    SourcePolicy `json:"sourcePolicy"`
}

// ProviderMetadata reports what an adapter actually did, so ingestion can log
// and meter provider behaviour without inspecting adapter internals.
type ProviderMetadata struct {
	Provider     string        `json:"provider"`
	Requests     int           `json:"requests"`
	ResultCount  int           `json:"resultCount"`
	Truncated    bool          `json:"truncated"`
	RateLimited  bool          `json:"rateLimited"`
	FromCache    bool          `json:"fromCache"`
	FallbackUsed bool          `json:"fallbackUsed"`
	Duration     time.Duration `json:"-"`
	License      LicenseInfo   `json:"license"`
}

// OpeningHoursPeriod is a normalized weekly opening window. Weekday follows
// time.Weekday (0 = Sunday) and times are local "HH:MM" strings.
type OpeningHoursPeriod struct {
	Weekday int    `json:"weekday"`
	Opens   string `json:"opens"`
	Closes  string `json:"closes"`
}

// PlaceRecord is one provider observation before normalization into the app
// taxonomy. It carries provenance, never credentials or user data.
type PlaceRecord struct {
	Provider        string               `json:"provider"`
	ProviderPlaceID string               `json:"providerPlaceId"`
	Name            string               `json:"name"`
	Aliases         []string             `json:"aliases,omitempty"`
	Category        string               `json:"category,omitempty"`
	Subcategory     string               `json:"subcategory,omitempty"`
	Latitude        *float64             `json:"lat,omitempty"`
	Longitude       *float64             `json:"lng,omitempty"`
	Address         string               `json:"address,omitempty"`
	Website         string               `json:"website,omitempty"`
	OpeningHours    []OpeningHoursPeriod `json:"openingHours,omitempty"`
	Rating          *float64             `json:"rating,omitempty"`
	RatingCount     *int                 `json:"ratingCount,omitempty"`
	PriceLevel      string               `json:"priceLevel,omitempty"`
	Tags            []string             `json:"tags,omitempty"`
	SourceURL       string               `json:"sourceUrl,omitempty"`
	License         LicenseInfo          `json:"license"`
	Attribution     string               `json:"attribution,omitempty"`
	ObservedAt      time.Time            `json:"observedAt"`
	ExpiresAt       *time.Time           `json:"expiresAt,omitempty"`
	// RawPayload is retained only when both the adapter license and the run's
	// SourcePolicy permit it. It must already be free of secrets.
	RawPayload map[string]any `json:"rawPayload,omitempty"`
}

// TravelKnowledgeProvider is the ingestion-side abstraction. It is deliberately
// narrower than the existing service.PlaceProvider used for live trip flows:
// knowledge ingestion needs provenance and refresh semantics, not booking or
// availability data.
type TravelKnowledgeProvider interface {
	SearchPlaces(ctx context.Context, request SearchRequest) ([]PlaceRecord, ProviderMetadata, error)
	GetPlaceDetails(ctx context.Context, providerPlaceID string) (PlaceRecord, ProviderMetadata, error)
	SupportsRefresh() bool
	ProviderName() string
	LicenseInfo() LicenseInfo
}
