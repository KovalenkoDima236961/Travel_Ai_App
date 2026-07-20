package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

// MockReferenceTime anchors every mock observation so ingestion, freshness
// scoring, and evaluation runs are byte-for-byte reproducible in CI. Callers
// that need relative ages use WithClock.
var MockReferenceTime = time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)

// mockLicense is a synthetic license. It is valid on purpose so that ingestion
// paths are exercised, and it is clearly labelled so mock data can never be
// mistaken for a real licensed source in Ops or in exports.
var mockLicense = LicenseInfo{
	Name:             "Mock test data (synthetic, not for production grounding)",
	URL:              "https://example.invalid/mock-license",
	Attribution:      "Travel Planner synthetic fixtures",
	TermsURL:         "https://example.invalid/mock-terms",
	AllowsStorage:    true,
	AllowsRawPayload: true,
}

// MockKnowledgeProvider serves fixed fixtures for Rome, Paris, Vienna, and
// Bratislava. The fixtures deliberately contain duplicates, missing
// coordinates, stale observations, and cross-provider disagreement so that
// deduplication, quality scoring, and review flows have real cases to run
// against without a network dependency.
type MockKnowledgeProvider struct {
	now func() time.Time
}

func NewMockKnowledgeProvider() *MockKnowledgeProvider {
	return &MockKnowledgeProvider{now: func() time.Time { return MockReferenceTime }}
}

// WithClock overrides the fixture clock. Tests that assert on staleness use it;
// the default keeps CI deterministic.
func (m *MockKnowledgeProvider) WithClock(now func() time.Time) *MockKnowledgeProvider {
	if now != nil {
		m.now = now
	}
	return m
}

func (m *MockKnowledgeProvider) ProviderName() string     { return ProviderMock }
func (m *MockKnowledgeProvider) SupportsRefresh() bool    { return true }
func (m *MockKnowledgeProvider) LicenseInfo() LicenseInfo { return mockLicense }

func (m *MockKnowledgeProvider) SearchPlaces(ctx context.Context, request SearchRequest) ([]PlaceRecord, ProviderMetadata, error) {
	metadata := ProviderMetadata{Provider: ProviderMock, Requests: 1, License: mockLicense}
	if err := ctx.Err(); err != nil {
		return nil, metadata, err
	}
	key := destinationKey(request.DestinationName)
	fixtures, ok := mockFixtures()[key]
	if !ok {
		return nil, metadata, nil
	}

	records := make([]PlaceRecord, 0, len(fixtures))
	categories := lowerSet(request.Categories)
	for _, fixture := range fixtures {
		record := m.materialize(fixture)
		if len(categories) > 0 {
			if _, wanted := categories[strings.ToLower(record.Category)]; !wanted {
				continue
			}
		}
		if request.SourcePolicy.RequireCoords && (record.Latitude == nil || record.Longitude == nil) {
			continue
		}
		if !request.SourcePolicy.AllowRawPayload {
			record.RawPayload = nil
		}
		records = append(records, record)
	}

	// Stable order keeps ingestion idempotent across runs.
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].ProviderPlaceID < records[j].ProviderPlaceID
	})
	if request.Limit > 0 && len(records) > request.Limit {
		records = records[:request.Limit]
		metadata.Truncated = true
	}
	metadata.ResultCount = len(records)
	return records, metadata, nil
}

func (m *MockKnowledgeProvider) GetPlaceDetails(ctx context.Context, providerPlaceID string) (PlaceRecord, ProviderMetadata, error) {
	metadata := ProviderMetadata{Provider: ProviderMock, Requests: 1, License: mockLicense}
	if err := ctx.Err(); err != nil {
		return PlaceRecord{}, metadata, err
	}
	for _, fixtures := range mockFixtures() {
		for _, fixture := range fixtures {
			if fixture.providerPlaceID == providerPlaceID {
				metadata.ResultCount = 1
				return m.materialize(fixture), metadata, nil
			}
		}
	}
	return PlaceRecord{}, metadata, fmt.Errorf("%w: %s", ErrPlaceNotFound, providerPlaceID)
}

func (m *MockKnowledgeProvider) materialize(fixture mockFixture) PlaceRecord {
	observedAt := m.now().Add(-time.Duration(fixture.observedDaysAgo) * 24 * time.Hour)
	record := PlaceRecord{
		Provider:        ProviderMock,
		ProviderPlaceID: fixture.providerPlaceID,
		Name:            fixture.name,
		Aliases:         fixture.aliases,
		Category:        fixture.category,
		Subcategory:     fixture.subcategory,
		Latitude:        fixture.latitude,
		Longitude:       fixture.longitude,
		Address:         fixture.address,
		Website:         fixture.website,
		OpeningHours:    fixture.openingHours,
		Rating:          fixture.rating,
		RatingCount:     fixture.ratingCount,
		PriceLevel:      fixture.priceLevel,
		Tags:            fixture.tags,
		SourceURL:       fixture.sourceURL,
		License:         mockLicense,
		Attribution:     mockLicense.Attribution,
		ObservedAt:      observedAt,
		RawPayload: map[string]any{
			"mockFixture": fixture.providerPlaceID,
			"rawName":     fixture.name,
		},
	}
	if fixture.expiresInDays > 0 {
		expires := observedAt.Add(time.Duration(fixture.expiresInDays) * 24 * time.Hour)
		record.ExpiresAt = &expires
	}
	return record
}

type mockFixture struct {
	providerPlaceID string
	name            string
	aliases         []string
	category        string
	subcategory     string
	latitude        *float64
	longitude       *float64
	address         string
	website         string
	openingHours    []OpeningHoursPeriod
	rating          *float64
	ratingCount     *int
	priceLevel      string
	tags            []string
	sourceURL       string
	observedDaysAgo int
	expiresInDays   int
}

func floatPtr(value float64) *float64 { return &value }
func intPtr(value int) *int           { return &value }

// weekdayHours expands one open/close window across the given weekdays.
func weekdayHours(opens, closes string, weekdays ...int) []OpeningHoursPeriod {
	periods := make([]OpeningHoursPeriod, 0, len(weekdays))
	for _, weekday := range weekdays {
		periods = append(periods, OpeningHoursPeriod{Weekday: weekday, Opens: opens, Closes: closes})
	}
	return periods
}

var everyDay = []int{0, 1, 2, 3, 4, 5, 6}

// mockFixtures returns the deterministic corpus keyed by normalized
// destination name. Fixture intent is documented inline because these cases
// are the test surface for dedup, review, and scoring.
func mockFixtures() map[string][]mockFixture {
	return map[string][]mockFixture{
		"rome": {
			{
				providerPlaceID: "mock-rome-001",
				name:            "Colosseum",
				aliases:         []string{"Colosseo", "Flavian Amphitheatre"},
				category:        "landmark",
				subcategory:     "historical_site",
				latitude:        floatPtr(41.8902),
				longitude:       floatPtr(12.4922),
				address:         "Piazza del Colosseo, 1, Rome",
				website:         "https://example.invalid/colosseum",
				openingHours:    weekdayHours("09:00", "19:00", everyDay...),
				rating:          floatPtr(4.7),
				ratingCount:     intPtr(210000),
				tags:            []string{"unesco", "ancient", "iconic"},
				sourceURL:       "https://example.invalid/rome/colosseum",
				observedDaysAgo: 2,
				expiresInDays:   90,
			},
			{
				// Duplicate of mock-rome-001 under a local-language name with
				// slightly different coordinates: exercises alias + proximity
				// matching and duplicate group creation.
				providerPlaceID: "mock-rome-002",
				name:            "Colosseo (Anfiteatro Flavio)",
				aliases:         []string{"Colosseum"},
				category:        "landmark",
				latitude:        floatPtr(41.8905),
				longitude:       floatPtr(12.4924),
				address:         "Piazza del Colosseo, Roma",
				rating:          floatPtr(4.6),
				ratingCount:     intPtr(1850),
				tags:            []string{"ancient"},
				sourceURL:       "https://example.invalid/rome/colosseo",
				observedDaysAgo: 5,
				expiresInDays:   90,
			},
			{
				providerPlaceID: "mock-rome-003",
				name:            "Pantheon",
				aliases:         []string{"Basilica di Santa Maria ad Martyres"},
				category:        "landmark",
				subcategory:     "religious_site",
				latitude:        floatPtr(41.8986),
				longitude:       floatPtr(12.4769),
				address:         "Piazza della Rotonda, Rome",
				openingHours:    weekdayHours("09:00", "18:30", 1, 2, 3, 4, 5, 6),
				rating:          floatPtr(4.8),
				ratingCount:     intPtr(150000),
				tags:            []string{"ancient", "free-entry"},
				sourceURL:       "https://example.invalid/rome/pantheon",
				observedDaysAgo: 3,
				expiresInDays:   180,
			},
			{
				// Missing coordinates: must be penalized by coordinate
				// completeness and land in needs_review rather than be dropped.
				providerPlaceID: "mock-rome-004",
				name:            "Trastevere",
				category:        "neighborhood",
				tags:            []string{"nightlife", "dining"},
				sourceURL:       "https://example.invalid/rome/trastevere",
				observedDaysAgo: 10,
			},
			{
				// Stale observation past its provider TTL: exercises the
				// refresh-stale-places job and freshness scoring.
				providerPlaceID: "mock-rome-005",
				name:            "Mercato di Testaccio",
				aliases:         []string{"Testaccio Market"},
				category:        "market",
				latitude:        floatPtr(41.8767),
				longitude:       floatPtr(12.4753),
				openingHours:    weekdayHours("07:00", "15:30", 1, 2, 3, 4, 5, 6),
				priceLevel:      "budget",
				tags:            []string{"food", "local"},
				sourceURL:       "https://example.invalid/rome/testaccio",
				observedDaysAgo: 240,
				expiresInDays:   30,
			},
		},
		"paris": {
			{
				providerPlaceID: "mock-paris-001",
				name:            "Louvre Museum",
				aliases:         []string{"Musée du Louvre"},
				category:        "museum",
				latitude:        floatPtr(48.8606),
				longitude:       floatPtr(2.3376),
				address:         "Rue de Rivoli, 75001 Paris",
				website:         "https://example.invalid/louvre",
				openingHours:    weekdayHours("09:00", "18:00", 0, 1, 3, 4, 5, 6),
				rating:          floatPtr(4.7),
				ratingCount:     intPtr(320000),
				priceLevel:      "moderate",
				tags:            []string{"art", "indoor"},
				sourceURL:       "https://example.invalid/paris/louvre",
				observedDaysAgo: 1,
				expiresInDays:   120,
			},
			{
				providerPlaceID: "mock-paris-002",
				name:            "Eiffel Tower",
				aliases:         []string{"Tour Eiffel"},
				category:        "landmark",
				latitude:        floatPtr(48.8584),
				longitude:       floatPtr(2.2945),
				address:         "Champ de Mars, 75007 Paris",
				openingHours:    weekdayHours("09:30", "23:45", everyDay...),
				rating:          floatPtr(4.6),
				ratingCount:     intPtr(410000),
				priceLevel:      "moderate",
				tags:            []string{"iconic", "viewpoint"},
				sourceURL:       "https://example.invalid/paris/eiffel",
				observedDaysAgo: 4,
				expiresInDays:   120,
			},
			{
				// Provider disagreement: same place as mock-paris-002 with a
				// conflicting category and a coarse coordinate. Merge logic must
				// prefer the higher-trust, more complete record.
				providerPlaceID: "mock-paris-003",
				name:            "Tour Eiffel",
				aliases:         []string{"Eiffel Tower"},
				category:        "viewpoint",
				latitude:        floatPtr(48.8600),
				longitude:       floatPtr(2.2950),
				rating:          floatPtr(4.2),
				ratingCount:     intPtr(90),
				tags:            []string{"viewpoint"},
				sourceURL:       "https://example.invalid/paris/tour-eiffel",
				observedDaysAgo: 45,
				expiresInDays:   60,
			},
			{
				providerPlaceID: "mock-paris-004",
				name:            "Luxembourg Gardens",
				aliases:         []string{"Jardin du Luxembourg"},
				category:        "park",
				latitude:        floatPtr(48.8462),
				longitude:       floatPtr(2.3372),
				openingHours:    weekdayHours("07:30", "20:30", everyDay...),
				tags:            []string{"outdoor", "family"},
				sourceURL:       "https://example.invalid/paris/luxembourg",
				observedDaysAgo: 12,
				expiresInDays:   180,
			},
		},
		"vienna": {
			{
				providerPlaceID: "mock-vienna-001",
				name:            "Schönbrunn Palace",
				aliases:         []string{"Schloss Schönbrunn"},
				category:        "landmark",
				latitude:        floatPtr(48.1845),
				longitude:       floatPtr(16.3122),
				address:         "Schönbrunner Schloßstraße 47, Vienna",
				openingHours:    weekdayHours("08:00", "17:30", everyDay...),
				rating:          floatPtr(4.7),
				ratingCount:     intPtr(180000),
				priceLevel:      "moderate",
				tags:            []string{"unesco", "palace"},
				sourceURL:       "https://example.invalid/vienna/schonbrunn",
				observedDaysAgo: 6,
				expiresInDays:   150,
			},
			{
				providerPlaceID: "mock-vienna-002",
				name:            "Kunsthistorisches Museum",
				aliases:         []string{"Museum of Art History"},
				category:        "museum",
				latitude:        floatPtr(48.2038),
				longitude:       floatPtr(16.3617),
				openingHours:    weekdayHours("10:00", "18:00", 2, 3, 4, 5, 6, 0),
				rating:          floatPtr(4.6),
				ratingCount:     intPtr(42000),
				priceLevel:      "moderate",
				tags:            []string{"art", "indoor"},
				sourceURL:       "https://example.invalid/vienna/khm",
				observedDaysAgo: 8,
				expiresInDays:   150,
			},
			{
				providerPlaceID: "mock-vienna-003",
				name:            "Naschmarkt",
				category:        "market",
				latitude:        floatPtr(48.1985),
				longitude:       floatPtr(16.3625),
				openingHours:    weekdayHours("06:00", "19:30", 1, 2, 3, 4, 5, 6),
				priceLevel:      "budget",
				tags:            []string{"food", "outdoor"},
				sourceURL:       "https://example.invalid/vienna/naschmarkt",
				observedDaysAgo: 20,
				expiresInDays:   45,
			},
		},
		"bratislava": {
			{
				providerPlaceID: "mock-bratislava-001",
				name:            "Bratislava Castle",
				aliases:         []string{"Bratislavský hrad"},
				category:        "landmark",
				latitude:        floatPtr(48.1420),
				longitude:       floatPtr(17.1000),
				openingHours:    weekdayHours("09:00", "18:00", 2, 3, 4, 5, 6, 0),
				rating:          floatPtr(4.5),
				ratingCount:     intPtr(31000),
				tags:            []string{"castle", "viewpoint"},
				sourceURL:       "https://example.invalid/bratislava/castle",
				observedDaysAgo: 7,
				expiresInDays:   150,
			},
			{
				providerPlaceID: "mock-bratislava-002",
				name:            "UFO Observation Deck",
				aliases:         []string{"UFO Bratislava", "Most SNP"},
				category:        "viewpoint",
				latitude:        floatPtr(48.1378),
				longitude:       floatPtr(17.1050),
				openingHours:    weekdayHours("10:00", "22:00", everyDay...),
				priceLevel:      "moderate",
				tags:            []string{"viewpoint", "panorama"},
				sourceURL:       "https://example.invalid/bratislava/ufo",
				observedDaysAgo: 15,
				expiresInDays:   120,
			},
			{
				// Low-signal record with a generic name: quality scoring should
				// hold this below the strong-grounding threshold.
				providerPlaceID: "mock-bratislava-003",
				name:            "Cafe",
				category:        "cafe",
				tags:            []string{},
				sourceURL:       "https://example.invalid/bratislava/cafe",
				observedDaysAgo: 120,
				expiresInDays:   30,
			},
		},
	}
}

func destinationKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func lowerSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if trimmed != "" {
			result[trimmed] = struct{}{}
		}
	}
	return result
}
