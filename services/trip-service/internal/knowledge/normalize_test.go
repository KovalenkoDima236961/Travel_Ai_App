package knowledge

import (
	"errors"
	"testing"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge/provider"
)

func testLicense() provider.LicenseInfo {
	return provider.LicenseInfo{Name: "Test License", Attribution: "Test Attribution", AllowsStorage: true, AllowsRawPayload: true}
}

func TestNormalizeCategoryMapsProviderVocabularyIntoAppTaxonomy(t *testing.T) {
	cases := []struct {
		raw          string
		want         string
		wantsWarning bool
	}{
		{"historical_site", "landmark", false},
		{"Religious Site", "landmark", false},
		{"art_gallery", "museum", false},
		{"nightlife", "activity", false},
		{"shopping", "market", false},
		{"observation-deck", "viewpoint", false},
		{"accommodation", "other", false},
		{"museum", "museum", false},
		{"", "other", true},
		{"time_travel_agency", "other", true},
	}
	for _, testCase := range cases {
		got, warning := NormalizeCategory(testCase.raw)
		if got != testCase.want {
			t.Errorf("NormalizeCategory(%q) = %q, want %q", testCase.raw, got, testCase.want)
		}
		if (warning != "") != testCase.wantsWarning {
			t.Errorf("NormalizeCategory(%q) warning = %q, wantsWarning = %v", testCase.raw, warning, testCase.wantsWarning)
		}
	}
}

// Every mapped category must be storable: the DB CHECK constraint and
// allowedCategories are the contract this normalization has to respect.
func TestNormalizeCategoryAlwaysProducesAllowedCategory(t *testing.T) {
	for raw := range providerCategoryAliases {
		mapped, _ := NormalizeCategory(raw)
		if _, ok := allowedCategories[mapped]; !ok {
			t.Fatalf("NormalizeCategory(%q) = %q which is not an allowed travel_places category", raw, mapped)
		}
	}
}

func TestNormalizeMatchNameFoldsAccentsAndParentheticals(t *testing.T) {
	cases := []struct{ raw, want string }{
		{"Colosseo (Anfiteatro Flavio)", "colosseo"},
		{"Musée d'Orsay", "musee d orsay"},
		{"Sad Janka Kráľa", "sad janka krala"},
		{"St. Martin's Cathedral", "st martin s cathedral"},
		{"  Schönbrunn  Palace  ", "schonbrunn palace"},
	}
	for _, testCase := range cases {
		if got := NormalizeMatchName(testCase.raw); got != testCase.want {
			t.Errorf("NormalizeMatchName(%q) = %q, want %q", testCase.raw, got, testCase.want)
		}
	}
}

func TestNormalizeCoordinatesRejectsPlaceholders(t *testing.T) {
	zero := 0.0
	if latitude, longitude := NormalizeCoordinates(&zero, &zero); latitude != nil || longitude != nil {
		t.Fatal("null island coordinates must be treated as missing")
	}
	outOfRange := 91.0
	valid := 12.0
	if latitude, _ := NormalizeCoordinates(&outOfRange, &valid); latitude != nil {
		t.Fatal("out-of-range latitude must be dropped")
	}
	latitude, longitude := 41.890211111, 12.492211111
	gotLat, gotLng := NormalizeCoordinates(&latitude, &longitude)
	if gotLat == nil || gotLng == nil || *gotLat != 41.890211 || *gotLng != 12.492211 {
		t.Fatalf("coordinates were not rounded to 6dp: %v %v", gotLat, gotLng)
	}
}

func TestNormalizeWebsiteStripsQueryAndRejectsNonHTTP(t *testing.T) {
	if got := NormalizeWebsite("https://example.invalid/place?token=secret#top"); got != "https://example.invalid/place" {
		t.Fatalf("NormalizeWebsite() = %q, query and fragment must be stripped", got)
	}
	if got := NormalizeWebsite("javascript:alert(1)"); got != "" {
		t.Fatalf("NormalizeWebsite() = %q, non-http scheme must be rejected", got)
	}
}

func TestNormalizePriceLevelFoldsVocabulary(t *testing.T) {
	cases := map[string]string{
		"$$": "moderate", "1": "budget", "very_expensive": "luxury",
		"CHEAP": "budget", "unknown-scale": "",
	}
	for raw, want := range cases {
		if got := NormalizePriceLevel(raw); got != want {
			t.Errorf("NormalizePriceLevel(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestNormalizeOpeningHoursSortsAndDropsMalformed(t *testing.T) {
	periods := []provider.OpeningHoursPeriod{
		{Weekday: 3, Opens: "09:00", Closes: "17:00"},
		{Weekday: 1, Opens: "09:00", Closes: "17:00"},
		{Weekday: 1, Opens: "09:00", Closes: "17:00"},
		{Weekday: 9, Opens: "09:00", Closes: "17:00"},
		{Weekday: 2, Opens: "not-a-time", Closes: "17:00"},
	}
	normalized := NormalizeOpeningHours(periods)
	if len(normalized) != 2 {
		t.Fatalf("expected 2 valid deduplicated periods, got %d: %+v", len(normalized), normalized)
	}
	if normalized[0].Weekday != 1 || normalized[1].Weekday != 3 {
		t.Fatalf("periods were not sorted by weekday: %+v", normalized)
	}
}

func TestNormalizeProviderRecordRequiresLicenseUnderPolicy(t *testing.T) {
	record := provider.PlaceRecord{
		Provider: "mock", ProviderPlaceID: "x-1", Name: "Somewhere",
		ObservedAt: time.Now(),
		License:    provider.LicenseInfo{}, // no license
	}
	_, err := NormalizeProviderRecord(record, provider.DefaultSourcePolicy())
	if !errors.Is(err, provider.ErrLicenseMissing) {
		t.Fatalf("expected ErrLicenseMissing, got %v", err)
	}
}

func TestNormalizeProviderRecordWithholdsRawPayloadByDefault(t *testing.T) {
	record := provider.PlaceRecord{
		Provider: "mock", ProviderPlaceID: "x-1", Name: "Somewhere",
		ObservedAt: time.Now(), License: testLicense(),
		RawPayload: map[string]any{"anything": "value"},
	}
	observation, err := NormalizeProviderRecord(record, provider.DefaultSourcePolicy())
	if err != nil {
		t.Fatalf("NormalizeProviderRecord() error = %v", err)
	}
	if observation.RawPayload != nil {
		t.Fatal("raw payload must be withheld unless the policy opts in")
	}

	policy := provider.DefaultSourcePolicy()
	policy.AllowRawPayload = true
	observation, err = NormalizeProviderRecord(record, policy)
	if err != nil {
		t.Fatalf("NormalizeProviderRecord() error = %v", err)
	}
	if observation.RawPayload == nil {
		t.Fatal("raw payload must be retained when policy and license allow it")
	}
}

// A license that forbids raw payload retention must win over the run policy.
func TestNormalizeProviderRecordRespectsLicenseRawPayloadRestriction(t *testing.T) {
	license := testLicense()
	license.AllowsRawPayload = false
	record := provider.PlaceRecord{
		Provider: "osm", ProviderPlaceID: "x-2", Name: "Somewhere",
		ObservedAt: time.Now(), License: license,
		RawPayload: map[string]any{"anything": "value"},
	}
	policy := provider.DefaultSourcePolicy()
	policy.AllowRawPayload = true
	observation, err := NormalizeProviderRecord(record, policy)
	if err != nil {
		t.Fatalf("NormalizeProviderRecord() error = %v", err)
	}
	if observation.RawPayload != nil {
		t.Fatal("license restriction on raw payload must override the run policy")
	}
}

func TestNormalizeProviderRecordCollectsWarnings(t *testing.T) {
	record := provider.PlaceRecord{
		Provider: "mock", ProviderPlaceID: "x-3", Name: "Trastevere",
		Category: "unmapped_thing", ObservedAt: time.Now(), License: testLicense(),
	}
	observation, err := NormalizeProviderRecord(record, provider.DefaultSourcePolicy())
	if err != nil {
		t.Fatalf("NormalizeProviderRecord() error = %v", err)
	}
	if len(observation.Warnings) != 3 {
		t.Fatalf("expected warnings for category, coordinates, and opening hours; got %v", observation.Warnings)
	}
	if observation.Category != "other" {
		t.Fatalf("unmapped category should fall back to other, got %q", observation.Category)
	}
}

func TestNormalizeAliasesDropsDisplayNameDuplicates(t *testing.T) {
	aliases := NormalizeAliases("Colosseum", []string{"colosseum", "Colosseo", "Colosseo", ""})
	if len(aliases) != 1 || aliases[0] != "Colosseo" {
		t.Fatalf("NormalizeAliases() = %v, want [Colosseo]", aliases)
	}
}
