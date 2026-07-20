package knowledge

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge/provider"
)

// Source and destination resolution for provider ingestion. Provider sources
// are registered from the adapter's own license metadata, so a source row can
// never claim provenance the adapter did not declare.

// providerSourceKey is the stable source_key for a provider adapter.
func providerSourceKey(providerName string) string {
	return "provider_" + strings.ToLower(strings.TrimSpace(providerName))
}

// sourceTypeForTrust maps a trust level onto the source_type vocabulary already
// defined by the 000042 CHECK constraint.
func sourceTypeForTrust(trustLevel string) string {
	switch trustLevel {
	case "mock":
		return "mock_test_data"
	case "public_open_data":
		return "open_data"
	case TrustLevelCurated:
		return SourceTypeManualCurated
	default:
		return "provider_place"
	}
}

// EnsureProviderSource registers or refreshes the knowledge source row for a
// provider adapter. A non-curated source without license metadata is rejected
// here rather than being stored with unknown provenance.
func (s *Store) EnsureProviderSource(
	ctx context.Context,
	providerName string,
	trustLevel string,
	license provider.LicenseInfo,
	refreshSupported bool,
) (uuid.UUID, error) {
	if s == nil || s.db == nil {
		return uuid.Nil, fmt.Errorf("knowledge store is required")
	}
	if strings.TrimSpace(license.Name) == "" {
		return uuid.Nil, fmt.Errorf("%w: provider %s declares no license", ErrLicenseMissing, providerName)
	}

	sourceKey := providerSourceKey(providerName)
	var id uuid.UUID
	err := s.db.QueryRow(ctx, `INSERT INTO travel_knowledge_sources
    (source_key, source_type, display_name, provider_name, license_name, license_url, attribution,
     terms_url, trust_level, enabled, refresh_supported, rate_limit_category)
    VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,TRUE,$10,$11)
    ON CONFLICT (source_key) DO UPDATE SET
      source_type=EXCLUDED.source_type, display_name=EXCLUDED.display_name,
      provider_name=EXCLUDED.provider_name, license_name=EXCLUDED.license_name,
      license_url=EXCLUDED.license_url, attribution=EXCLUDED.attribution, terms_url=EXCLUDED.terms_url,
      trust_level=EXCLUDED.trust_level, refresh_supported=EXCLUDED.refresh_supported,
      rate_limit_category=EXCLUDED.rate_limit_category, updated_at=NOW()
    RETURNING id`,
		sourceKey, sourceTypeForTrust(trustLevel), providerDisplayName(providerName), providerName,
		license.Name, nullText(license.URL), nullText(license.Attribution), nullText(license.TermsURL),
		trustLevel, refreshSupported, nullText("places_"+strings.ToLower(providerName))).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("ensure provider source %q: %w", sourceKey, err)
	}
	return id, nil
}

// ResolveDestination finds an existing destination or creates one. Provider
// ingestion should normally target a destination that already exists; creating
// one is allowed but marked with lower confidence, because a destination
// invented from a provider search string has not been reviewed.
func (s *Store) ResolveDestination(ctx context.Context, name, countryCode string, knownID *uuid.UUID) (uuid.UUID, error) {
	if s == nil || s.db == nil {
		return uuid.Nil, fmt.Errorf("knowledge store is required")
	}
	if knownID != nil && *knownID != uuid.Nil {
		var id uuid.UUID
		if err := s.db.QueryRow(ctx, `SELECT id FROM travel_destinations WHERE id=$1`, *knownID).Scan(&id); err != nil {
			return uuid.Nil, fmt.Errorf("destination %s not found: %w", knownID, err)
		}
		return id, nil
	}

	trimmedName := strings.TrimSpace(name)
	normalizedCountry := strings.ToUpper(strings.TrimSpace(countryCode))

	var id uuid.UUID
	err := s.db.QueryRow(ctx, `SELECT id FROM travel_destinations
    WHERE (lower(canonical_name) = lower($1) OR aliases @> to_jsonb($1::text))
      AND ($2::text IS NULL OR country_code = $2)
    ORDER BY confidence DESC, canonical_name
    LIMIT 1`, trimmedName, nullText(normalizedCountry)).Scan(&id)
	if err == nil {
		return id, nil
	}

	if normalizedCountry == "" {
		return uuid.Nil, fmt.Errorf("destination %q is unknown and countryCode is required to create it", trimmedName)
	}
	err = s.db.QueryRow(ctx, `INSERT INTO travel_destinations
    (canonical_name, country_code, country_name, confidence)
    VALUES ($1,$2,$3,0.6)
    ON CONFLICT (canonical_name, country_code) DO UPDATE SET updated_at=NOW()
    RETURNING id`, trimmedName, normalizedCountry, normalizedCountry).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create destination %q: %w", trimmedName, err)
	}
	return id, nil
}

// DestinationName returns the canonical name and country code for a
// destination, used when a refresh run re-queries a provider.
func (s *Store) DestinationName(ctx context.Context, destinationID uuid.UUID) (string, string, error) {
	if s == nil || s.db == nil {
		return "", "", fmt.Errorf("knowledge store is required")
	}
	var (
		name        string
		countryCode *string
	)
	if err := s.db.QueryRow(ctx, `SELECT canonical_name, country_code FROM travel_destinations WHERE id=$1`,
		destinationID).Scan(&name, &countryCode); err != nil {
		return "", "", fmt.Errorf("load destination %s: %w", destinationID, err)
	}
	if countryCode == nil {
		return name, "", nil
	}
	return name, *countryCode, nil
}

func providerDisplayName(providerName string) string {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case provider.ProviderMock:
		return "Mock knowledge provider (synthetic)"
	case provider.ProviderFoursquare:
		return "Foursquare Places"
	case provider.ProviderOpenTripMap:
		return "OpenTripMap"
	case provider.ProviderWikidata:
		return "Wikidata"
	default:
		return providerName
	}
}
