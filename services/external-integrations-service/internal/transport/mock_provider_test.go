package transport

import (
	"context"
	"reflect"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/cache"
)

func TestMockProviderDeterministicOptions(t *testing.T) {
	provider := NewMockProvider(3)
	req := sampleTransportSearchRequest([]string{ModeTrain, ModeBus, ModeCar})

	first, err := provider.SearchTransportOptions(context.Background(), req)
	if err != nil {
		t.Fatalf("first search failed: %v", err)
	}
	second, err := provider.SearchTransportOptions(context.Background(), req)
	if err != nil {
		t.Fatalf("second search failed: %v", err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("expected deterministic response\nfirst=%+v\nsecond=%+v", first, second)
	}
	if len(first.Options) == 0 {
		t.Fatal("expected options")
	}
	seen := map[string]bool{}
	for _, option := range first.Options {
		seen[option.Mode] = true
		if option.Provider != ProviderMock {
			t.Fatalf("expected mock provider, got %q", option.Provider)
		}
		if option.DurationMinutes <= 0 {
			t.Fatalf("expected positive duration: %+v", option)
		}
		if len(option.Warnings) == 0 {
			t.Fatalf("expected warnings: %+v", option)
		}
	}
	for _, mode := range []string{ModeTrain, ModeBus, ModeCar} {
		if !seen[mode] {
			t.Fatalf("expected %s option in %+v", mode, seen)
		}
	}
}

func TestMockProviderFlightAndFerryRequested(t *testing.T) {
	provider := NewMockProvider(3)
	req := sampleTransportSearchRequest([]string{ModeFlight, ModeFerry})

	result, err := provider.SearchTransportOptions(context.Background(), req)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	seen := map[string]bool{}
	for _, option := range result.Options {
		seen[option.Mode] = true
	}
	if !seen[ModeFlight] {
		t.Fatalf("expected explicitly requested flight option")
	}
	if !seen[ModeFerry] {
		t.Fatalf("expected explicitly requested ferry option")
	}
}

func TestCachingProviderMarksRepeatedResponseCached(t *testing.T) {
	provider := newCachingProvider(ProviderMock, NewMockProvider(3), cache.New(16), time.Hour, zap.NewNop())
	req := sampleTransportSearchRequest([]string{ModeTrain})

	first, err := provider.SearchTransportOptions(context.Background(), req)
	if err != nil {
		t.Fatalf("first search failed: %v", err)
	}
	if first.Summary.Cached {
		t.Fatalf("first response should not be cached")
	}
	second, err := provider.SearchTransportOptions(context.Background(), req)
	if err != nil {
		t.Fatalf("second search failed: %v", err)
	}
	if !second.Summary.Cached {
		t.Fatalf("second response should be cached")
	}
}

func sampleTransportSearchRequest(modes []string) TransportSearchRequest {
	return TransportSearchRequest{
		Origin: Location{
			Name:    "Bratislava",
			Country: "Slovakia",
			Lat:     floatPtr(48.1486),
			Lng:     floatPtr(17.1077),
		},
		Destination: Location{
			Name:    "Vienna",
			Country: "Austria",
			Lat:     floatPtr(48.2082),
			Lng:     floatPtr(16.3738),
		},
		Date:           "2026-09-10",
		Time:           "09:00",
		TimePreference: TimePreferenceDepartAfter,
		Travelers:      2,
		Modes:          modes,
		Currency:       "EUR",
	}
}

func floatPtr(value float64) *float64 {
	return &value
}
