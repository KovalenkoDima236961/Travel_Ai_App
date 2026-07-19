// Package knowledge contains the shared, privacy-safe contract for curated
// travel grounding records. It intentionally contains no provider credentials,
// raw user content, or prompt data.
package knowledge

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"
)

const (
	SourceTypeManualCurated = "manual_curated"
	TrustLevelCurated       = "trusted_curated"
	StatusActive            = "active"
)

var allowedCategories = map[string]struct{}{
	"landmark": {}, "museum": {}, "park": {}, "neighborhood": {}, "viewpoint": {},
	"market": {}, "restaurant": {}, "cafe": {}, "activity": {}, "nature": {},
	"transport": {}, "other": {},
}

// Source describes the provenance of normalized knowledge records.
type Source struct {
	SourceKey   string `json:"sourceKey"`
	SourceType  string `json:"sourceType"`
	DisplayName string `json:"displayName"`
	LicenseName string `json:"licenseName,omitempty"`
	LicenseURL  string `json:"licenseUrl,omitempty"`
	Attribution string `json:"attribution,omitempty"`
	TrustLevel  string `json:"trustLevel"`
	Enabled     bool   `json:"enabled"`
}

// DestinationKnowledge is the curated-file shape before persistence.
type DestinationKnowledge struct {
	CanonicalName string           `json:"canonicalName"`
	CountryCode   string           `json:"countryCode"`
	CountryName   string           `json:"countryName"`
	RegionName    string           `json:"regionName,omitempty"`
	Latitude      *float64         `json:"lat,omitempty"`
	Longitude     *float64         `json:"lng,omitempty"`
	Aliases       []string         `json:"aliases"`
	Tags          []string         `json:"tags"`
	Places        []PlaceKnowledge `json:"places"`
}

// PlaceKnowledge is compact enough to safely send to an AI prompt after
// retrieval; it is not a booking or availability claim.
type PlaceKnowledge struct {
	Name                   string   `json:"name"`
	Category               string   `json:"category"`
	Subcategory            string   `json:"subcategory,omitempty"`
	Latitude               *float64 `json:"lat,omitempty"`
	Longitude              *float64 `json:"lng,omitempty"`
	Address                string   `json:"address,omitempty"`
	Neighborhood           string   `json:"neighborhood,omitempty"`
	Aliases                []string `json:"aliases,omitempty"`
	Tags                   []string `json:"tags,omitempty"`
	TypicalDurationMinutes *int     `json:"typicalDurationMinutes,omitempty"`
	PriceLevel             string   `json:"priceLevel,omitempty"`
	Outdoor                *bool    `json:"outdoor,omitempty"`
	RainFriendly           *bool    `json:"rainFriendly,omitempty"`
	FamilyFriendly         *bool    `json:"familyFriendly,omitempty"`
	BestTimeOfDay          []string `json:"bestTimeOfDay,omitempty"`
	AvoidIf                []string `json:"avoidIf,omitempty"`
	SourceKey              string   `json:"sourceKey"`
	SourceURL              string   `json:"sourceUrl,omitempty"`
	LicenseName            string   `json:"licenseName,omitempty"`
	Attribution            string   `json:"attribution,omitempty"`
	Confidence             float64  `json:"confidence"`
}

// KnowledgeDocument and KnowledgeChunk are normalized ingestion units. Their
// content is limited to approved, original/public source material.
type KnowledgeDocument struct {
	Title       string
	Content     string
	ContentType string
	Language    string
	SourceKey   string
	Checksum    string
}

type KnowledgeChunk struct {
	ChunkIndex int
	Content    string
	Checksum   string
}

// FeedbackSignal is intentionally metadata-only. ItemSnapshot is supplied by
// higher layers after sanitization, never from comments, receipts, calendars,
// email, or raw prompts.
type FeedbackSignal struct {
	SignalType         string
	SignalValue        string
	ConsentForTraining bool
}

func (d *DestinationKnowledge) NormalizeAndValidate(sources map[string]Source) error {
	d.CanonicalName = strings.TrimSpace(d.CanonicalName)
	d.CountryCode = strings.ToUpper(strings.TrimSpace(d.CountryCode))
	d.CountryName = strings.TrimSpace(d.CountryName)
	d.RegionName = strings.TrimSpace(d.RegionName)
	d.Aliases = normalizeStrings(d.Aliases)
	d.Tags = normalizeStrings(d.Tags)
	if d.CanonicalName == "" {
		return fmt.Errorf("destination canonicalName is required")
	}
	if len(d.CountryCode) != 2 {
		return fmt.Errorf("destination %q countryCode must be ISO alpha-2", d.CanonicalName)
	}
	if d.CountryName == "" {
		return fmt.Errorf("destination %q countryName is required", d.CanonicalName)
	}
	if err := validateCoordinates(d.Latitude, d.Longitude); err != nil {
		return fmt.Errorf("destination %q: %w", d.CanonicalName, err)
	}
	if len(d.Places) == 0 {
		return fmt.Errorf("destination %q has no places", d.CanonicalName)
	}
	seen := make(map[string]struct{}, len(d.Places))
	for index := range d.Places {
		place := &d.Places[index]
		if err := place.NormalizeAndValidate(sources); err != nil {
			return fmt.Errorf("destination %q place %d: %w", d.CanonicalName, index, err)
		}
		key := NormalizeName(place.Name)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("duplicate normalized place %q", place.Name)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func (p *PlaceKnowledge) NormalizeAndValidate(sources map[string]Source) error {
	p.Name = strings.TrimSpace(p.Name)
	p.Category = strings.ToLower(strings.TrimSpace(p.Category))
	p.Subcategory = strings.TrimSpace(p.Subcategory)
	p.Address = strings.TrimSpace(p.Address)
	p.Neighborhood = strings.TrimSpace(p.Neighborhood)
	p.SourceKey = strings.TrimSpace(p.SourceKey)
	p.SourceURL = strings.TrimSpace(p.SourceURL)
	p.LicenseName = strings.TrimSpace(p.LicenseName)
	p.Attribution = strings.TrimSpace(p.Attribution)
	p.Aliases = normalizeStrings(p.Aliases)
	p.Tags = normalizeStrings(p.Tags)
	p.BestTimeOfDay = normalizeStrings(p.BestTimeOfDay)
	p.AvoidIf = normalizeStrings(p.AvoidIf)
	if p.Name == "" {
		return fmt.Errorf("name is required")
	}
	if _, ok := allowedCategories[p.Category]; !ok {
		return fmt.Errorf("invalid category %q", p.Category)
	}
	if p.Confidence < 0 || p.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	if err := validateCoordinates(p.Latitude, p.Longitude); err != nil {
		return err
	}
	if p.TypicalDurationMinutes != nil && (*p.TypicalDurationMinutes < 5 || *p.TypicalDurationMinutes > 720) {
		return fmt.Errorf("typicalDurationMinutes must be between 5 and 720")
	}
	source, ok := sources[p.SourceKey]
	if !ok || !source.Enabled {
		return fmt.Errorf("unknown or disabled source %q", p.SourceKey)
	}
	if source.SourceType != SourceTypeManualCurated && p.Attribution == "" {
		return fmt.Errorf("non-curated source %q requires attribution", p.SourceKey)
	}
	return nil
}

func NormalizeName(value string) string {
	var builder strings.Builder
	lastSpace := false
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			builder.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(builder.String())
}

func Checksum(content string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(content)))
	return hex.EncodeToString(sum[:])
}

func validateCoordinates(latitude, longitude *float64) error {
	if latitude != nil && (*latitude < -90 || *latitude > 90) {
		return fmt.Errorf("latitude must be between -90 and 90")
	}
	if longitude != nil && (*longitude < -180 || *longitude > 180) {
		return fmt.Errorf("longitude must be between -180 and 180")
	}
	return nil
}

func normalizeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		key := strings.ToLower(trimmed)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
