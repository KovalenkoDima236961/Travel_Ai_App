package knowledge

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge/provider"
)

// Normalization maps heterogeneous provider vocabulary onto the taxonomy that
// already exists in this codebase: the travel_places category CHECK constraint
// and allowedCategories in model.go. Provider-specific labels such as
// "historical_site" or "nightlife" are folded into an existing category and
// preserved as subcategory or tags, so retrieval, validation, and the Python
// GroundingPlace schema keep working unchanged.

// providerCategoryAliases maps common provider category vocabulary onto the app
// taxonomy. Anything unmapped becomes "other" and is flagged for review rather
// than silently guessed.
var providerCategoryAliases = map[string]string{
	"landmark":           "landmark",
	"monument":           "landmark",
	"historic":           "landmark",
	"historic_site":      "landmark",
	"historical_site":    "landmark",
	"heritage":           "landmark",
	"castle":             "landmark",
	"palace":             "landmark",
	"church":             "landmark",
	"religious_site":     "landmark",
	"cathedral":          "landmark",
	"temple":             "landmark",
	"attraction":         "landmark",
	"tourist_attraction": "landmark",
	"museum":             "museum",
	"gallery":            "museum",
	"art_gallery":        "museum",
	"exhibition":         "museum",
	"park":               "park",
	"garden":             "park",
	"playground":         "park",
	"neighborhood":       "neighborhood",
	"neighbourhood":      "neighborhood",
	"district":           "neighborhood",
	"quarter":            "neighborhood",
	"viewpoint":          "viewpoint",
	"observation_deck":   "viewpoint",
	"panorama":           "viewpoint",
	"lookout":            "viewpoint",
	"market":             "market",
	"marketplace":        "market",
	"bazaar":             "market",
	"shopping":           "market",
	"mall":               "market",
	"restaurant":         "restaurant",
	"food":               "restaurant",
	"bistro":             "restaurant",
	"eatery":             "restaurant",
	"cafe":               "cafe",
	"coffee":             "cafe",
	"coffee_shop":        "cafe",
	"bakery":             "cafe",
	"tearoom":            "cafe",
	"activity":           "activity",
	"experience":         "activity",
	"tour":               "activity",
	"entertainment":      "activity",
	"nightlife":          "activity",
	"bar":                "activity",
	"outdoor_activity":   "activity",
	"sports":             "activity",
	"nature":             "nature",
	"beach":              "nature",
	"lake":               "nature",
	"forest":             "nature",
	"mountain":           "nature",
	"trail":              "nature",
	"transport":          "transport",
	"station":            "transport",
	"train_station":      "transport",
	"airport":            "transport",
	"bus_station":        "transport",
	"metro":              "transport",
	// Accommodation is intentionally not a planning category in this codebase:
	// it maps to "other" and is held out of strong grounding.
	"accommodation": "other",
	"hotel":         "other",
	"hostel":        "other",
	"other":         "other",
}

// nameSuffixNoise are trailing qualifiers that providers append to place names.
// They are stripped for matching purposes only; the display name keeps them.
var nameSuffixNoise = []string{
	"official site", "official website", "ticket office", "tickets",
	"visitor centre", "visitor center", "entrance", "main entrance",
}

// priceLevelAliases folds provider price vocabulary into the app's levels.
var priceLevelAliases = map[string]string{
	"1": "budget", "2": "moderate", "3": "premium", "4": "luxury",
	"$": "budget", "$$": "moderate", "$$$": "premium", "$$$$": "luxury",
	"cheap": "budget", "inexpensive": "budget", "budget": "budget", "free": "budget",
	"moderate": "moderate", "mid": "moderate", "medium": "moderate",
	"expensive": "premium", "premium": "premium", "pricey": "premium",
	"very_expensive": "luxury", "luxury": "luxury",
}

// NormalizedObservation is a provider record after normalization, ready to be
// persisted as a travel_provider_place_observations row. It carries the
// warnings that scoring and review use, so no downstream stage has to re-derive
// why a record is weak.
type NormalizedObservation struct {
	Provider        string
	ProviderPlaceID string
	RawName         string
	NormalizedName  string
	DisplayName     string
	Aliases         []string
	Category        string
	Subcategory     string
	Latitude        *float64
	Longitude       *float64
	Address         string
	Website         string
	OpeningHours    []provider.OpeningHoursPeriod
	Rating          *float64
	RatingCount     *int
	PriceLevel      string
	Tags            []string
	SourceURL       string
	LicenseName     string
	Attribution     string
	ObservedAt      time.Time
	ExpiresAt       *time.Time
	RawPayload      map[string]any
	Warnings        []string
}

// NormalizeProviderRecord converts a provider record into the app taxonomy.
// It is pure and deterministic: the same input always yields the same output,
// which is what makes ingestion idempotent and CI reproducible.
func NormalizeProviderRecord(record provider.PlaceRecord, policy provider.SourcePolicy) (NormalizedObservation, error) {
	displayName := collapseSpaces(record.Name)
	if displayName == "" {
		return NormalizedObservation{}, fmt.Errorf("provider record %q: name is required", record.ProviderPlaceID)
	}
	if strings.TrimSpace(record.ProviderPlaceID) == "" {
		return NormalizedObservation{}, fmt.Errorf("provider record %q: providerPlaceId is required", displayName)
	}
	if policy.RequireLicense && !record.License.Valid() {
		return NormalizedObservation{}, fmt.Errorf("provider record %q: %w", displayName, provider.ErrLicenseMissing)
	}

	warnings := make([]string, 0, 4)
	category, categoryWarning := NormalizeCategory(record.Category)
	if categoryWarning != "" {
		warnings = append(warnings, categoryWarning)
	}

	latitude, longitude := NormalizeCoordinates(record.Latitude, record.Longitude)
	if latitude == nil || longitude == nil {
		warnings = append(warnings, "missing or invalid coordinates")
	}
	if len(record.OpeningHours) == 0 {
		warnings = append(warnings, "no opening hours reported")
	}

	observation := NormalizedObservation{
		Provider:        strings.ToLower(strings.TrimSpace(record.Provider)),
		ProviderPlaceID: strings.TrimSpace(record.ProviderPlaceID),
		RawName:         collapseSpaces(record.Name),
		NormalizedName:  NormalizeMatchName(record.Name),
		DisplayName:     displayName,
		Aliases:         NormalizeAliases(displayName, record.Aliases),
		Category:        category,
		Subcategory:     normalizeSubcategory(record.Category, record.Subcategory, category),
		Latitude:        latitude,
		Longitude:       longitude,
		Address:         collapseSpaces(record.Address),
		Website:         NormalizeWebsite(record.Website),
		OpeningHours:    NormalizeOpeningHours(record.OpeningHours),
		Rating:          normalizeRating(record.Rating),
		RatingCount:     record.RatingCount,
		PriceLevel:      NormalizePriceLevel(record.PriceLevel),
		Tags:            normalizeStrings(record.Tags),
		SourceURL:       strings.TrimSpace(record.SourceURL),
		LicenseName:     strings.TrimSpace(record.License.Name),
		Attribution:     strings.TrimSpace(firstNonEmpty(record.Attribution, record.License.Attribution)),
		ObservedAt:      record.ObservedAt.UTC(),
		ExpiresAt:       record.ExpiresAt,
		Warnings:        warnings,
	}
	if observation.ObservedAt.IsZero() {
		return NormalizedObservation{}, fmt.Errorf("provider record %q: observedAt is required", displayName)
	}
	// Raw payload is retained only when the license and the run policy both
	// allow it. Provider secrets never reach this path: adapters build payloads
	// from response bodies, not from configuration.
	if policy.AllowRawPayload && record.License.AllowsRawPayload {
		observation.RawPayload = record.RawPayload
	}
	return observation, nil
}

// NormalizeCategory folds provider vocabulary into the app taxonomy defined in
// model.go. The second return value is a warning when the mapping was a
// fallback rather than a known alias.
func NormalizeCategory(raw string) (string, string) {
	key := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(raw)), " ", "_")
	key = strings.ReplaceAll(key, "-", "_")
	if key == "" {
		return "other", "provider reported no category"
	}
	if mapped, ok := providerCategoryAliases[key]; ok {
		return mapped, ""
	}
	// A provider category that already matches the app taxonomy is accepted
	// directly; anything else is "other" and gets reviewed.
	if _, ok := allowedCategories[key]; ok {
		return key, ""
	}
	return "other", fmt.Sprintf("unmapped provider category %q", raw)
}

// NormalizeMatchName produces the comparison key used by deduplication. It
// folds case, accents, punctuation, and parenthetical/suffix noise so that
// "Colosseo (Anfiteatro Flavio)" and "Colosseum" compare on equal footing.
func NormalizeMatchName(value string) string {
	lowered := strings.ToLower(strings.TrimSpace(value))
	lowered = stripParentheticals(lowered)
	folded := FoldAccents(lowered)
	cleaned := NormalizeName(folded)
	for _, suffix := range nameSuffixNoise {
		cleaned = strings.TrimSpace(strings.TrimSuffix(cleaned, suffix))
	}
	return cleaned
}

// FoldAccents maps Latin-script diacritics onto their base letters. A small
// explicit table avoids pulling in a transliteration dependency for the few
// scripts this codebase's destinations actually use.
func FoldAccents(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range value {
		if folded, ok := accentFolding[r]; ok {
			builder.WriteString(folded)
			continue
		}
		builder.WriteRune(r)
	}
	return builder.String()
}

var accentFolding = map[rune]string{
	'á': "a", 'à': "a", 'â': "a", 'ä': "a", 'ã': "a", 'å': "a", 'ā': "a", 'ă': "a", 'ą': "a",
	'é': "e", 'è': "e", 'ê': "e", 'ë': "e", 'ē': "e", 'ĕ': "e", 'ę': "e", 'ě': "e",
	'í': "i", 'ì': "i", 'î': "i", 'ï': "i", 'ī': "i", 'į': "i",
	'ó': "o", 'ò': "o", 'ô': "o", 'ö': "o", 'õ': "o", 'ø': "o", 'ō': "o", 'ő': "o",
	'ú': "u", 'ù': "u", 'û': "u", 'ü': "u", 'ū': "u", 'ů': "u", 'ű': "u", 'ų': "u",
	'ý': "y", 'ÿ': "y",
	'ñ': "n", 'ń': "n", 'ň': "n",
	'ç': "c", 'ć': "c", 'č': "c",
	'š': "s", 'ś': "s", 'ş': "s",
	'ž': "z", 'ź': "z", 'ż': "z",
	'ť': "t", 'ţ': "t",
	'ď': "d", 'đ': "d",
	'ľ': "l", 'ĺ': "l", 'ł': "l",
	'ř': "r", 'ŕ': "r",
	'ğ': "g",
	'ß': "ss", 'æ': "ae", 'œ': "oe",
}

// NormalizeAliases deduplicates aliases and drops any that collapse to the
// display name under match normalization.
func NormalizeAliases(displayName string, aliases []string) []string {
	primary := NormalizeMatchName(displayName)
	seen := map[string]struct{}{primary: {}}
	result := make([]string, 0, len(aliases))
	for _, alias := range aliases {
		trimmed := collapseSpaces(alias)
		if trimmed == "" {
			continue
		}
		key := NormalizeMatchName(trimmed)
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

// NormalizeCoordinates drops out-of-range or clearly placeholder coordinates.
// Null Island (0,0) is treated as missing because providers use it as a
// default far more often than a place genuinely sits there.
func NormalizeCoordinates(latitude, longitude *float64) (*float64, *float64) {
	if latitude == nil || longitude == nil {
		return nil, nil
	}
	if *latitude < -90 || *latitude > 90 || *longitude < -180 || *longitude > 180 {
		return nil, nil
	}
	if *latitude == 0 && *longitude == 0 {
		return nil, nil
	}
	lat := roundTo(*latitude, 6)
	lng := roundTo(*longitude, 6)
	return &lat, &lng
}

// NormalizeOpeningHours sorts periods, drops malformed windows, and merges
// exact duplicates. It makes no availability guarantee: hours are planning
// hints and are re-verified on refresh.
func NormalizeOpeningHours(periods []provider.OpeningHoursPeriod) []provider.OpeningHoursPeriod {
	if len(periods) == 0 {
		return nil
	}
	seen := make(map[provider.OpeningHoursPeriod]struct{}, len(periods))
	result := make([]provider.OpeningHoursPeriod, 0, len(periods))
	for _, period := range periods {
		if period.Weekday < 0 || period.Weekday > 6 {
			continue
		}
		opens, opensOK := normalizeClockTime(period.Opens)
		closes, closesOK := normalizeClockTime(period.Closes)
		if !opensOK || !closesOK {
			continue
		}
		normalized := provider.OpeningHoursPeriod{Weekday: period.Weekday, Opens: opens, Closes: closes}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	if len(result) == 0 {
		return nil
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Weekday != result[j].Weekday {
			return result[i].Weekday < result[j].Weekday
		}
		return result[i].Opens < result[j].Opens
	})
	return result
}

// NormalizePriceLevel folds provider price vocabulary into the app's levels.
// An unrecognized value yields an empty string rather than a guess.
func NormalizePriceLevel(raw string) string {
	key := strings.ToLower(strings.TrimSpace(raw))
	key = strings.ReplaceAll(key, " ", "_")
	if key == "" {
		return ""
	}
	if mapped, ok := priceLevelAliases[key]; ok {
		return mapped
	}
	return ""
}

// NormalizeWebsite keeps only http(s) URLs and strips query strings and
// fragments, which is where providers put tracking and occasionally
// credential-like tokens.
func NormalizeWebsite(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	if parsed.Host == "" {
		return ""
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.User = nil
	return strings.TrimSuffix(parsed.String(), "/")
}

func normalizeSubcategory(rawCategory, rawSubcategory, mappedCategory string) string {
	subcategory := strings.ToLower(collapseSpaces(rawSubcategory))
	if subcategory != "" {
		return subcategory
	}
	// When a specific provider category was folded into a broader app category,
	// retain the original as subcategory so detail is not lost.
	raw := strings.ReplaceAll(strings.ToLower(collapseSpaces(rawCategory)), " ", "_")
	if raw != "" && raw != mappedCategory {
		if _, known := providerCategoryAliases[raw]; known {
			return raw
		}
	}
	return ""
}

func normalizeRating(rating *float64) *float64 {
	if rating == nil {
		return nil
	}
	if *rating < 0 || *rating > 5 {
		return nil
	}
	value := roundTo(*rating, 2)
	return &value
}

func normalizeClockTime(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	parsed, err := time.Parse("15:04", trimmed)
	if err != nil {
		return "", false
	}
	return parsed.Format("15:04"), true
}

func stripParentheticals(value string) string {
	var builder strings.Builder
	depth := 0
	for _, r := range value {
		switch r {
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				builder.WriteRune(r)
			}
		}
	}
	return strings.TrimSpace(builder.String())
}

func collapseSpaces(value string) string {
	return strings.Join(strings.FieldsFunc(strings.TrimSpace(value), unicode.IsSpace), " ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func roundTo(value float64, decimals int) float64 {
	factor := 1.0
	for i := 0; i < decimals; i++ {
		factor *= 10
	}
	rounded := float64(int64(value*factor+copySign(0.5, value))) / factor
	return rounded
}

func copySign(magnitude, sign float64) float64 {
	if sign < 0 {
		return -magnitude
	}
	return magnitude
}
