package places

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
)

func TestNewMockProviderSelectsMock(t *testing.T) {
	provider, err := New(&config.Config{PlaceProvider: config.PlaceProviderConfig{Provider: "mock"}}, zap.NewNop())
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	items, err := provider.SearchPlaces(context.Background(), "Colosseum", "Rome")
	if err != nil {
		t.Fatalf("search places: %v", err)
	}
	if len(items) == 0 || items[0].Provider != "mock" {
		t.Fatalf("expected mock provider result, got %+v", items)
	}
}

func TestNewUnsupportedProviderReturnsError(t *testing.T) {
	_, err := New(&config.Config{PlaceProvider: config.PlaceProviderConfig{Provider: "google"}}, zap.NewNop())
	if err == nil {
		t.Fatal("expected unsupported provider error")
	}
	if !strings.Contains(err.Error(), "unsupported PLACE_PROVIDER") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewFoursquareWithoutAPIKeyFallsBackToMockWhenEnabled(t *testing.T) {
	provider, err := New(&config.Config{
		Env: "development",
		PlaceProvider: config.PlaceProviderConfig{
			Provider:       config.PlaceProviderFoursquare,
			FallbackToMock: true,
		},
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("expected fallback provider, got error: %v", err)
	}

	items, err := provider.SearchPlaces(context.Background(), "Colosseum", "Rome")
	if err != nil {
		t.Fatalf("search fallback places: %v", err)
	}
	if len(items) == 0 || items[0].Provider != "mock" {
		t.Fatalf("expected mock fallback result, got %+v", items)
	}
}

func TestNewFoursquareWithoutAPIKeyFailsWhenFallbackDisabled(t *testing.T) {
	_, err := New(&config.Config{
		Env: "development",
		PlaceProvider: config.PlaceProviderConfig{
			Provider:       config.PlaceProviderFoursquare,
			FallbackToMock: false,
		},
	}, zap.NewNop())
	assertProviderErrorKind(t, err, providerErrorAuthConfig)
}

func TestFoursquareFallbackToMockWhenRealProviderErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusInternalServerError)
	}))
	defer server.Close()

	provider, err := New(&config.Config{
		PlaceProvider: config.PlaceProviderConfig{
			Provider:                 config.PlaceProviderFoursquare,
			FallbackToMock:           true,
			FoursquareAPIKey:         "test-key",
			FoursquareBaseURL:        server.URL,
			FoursquareTimeoutSeconds: 1,
		},
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	items, err := provider.SearchPlaces(context.Background(), "Colosseum", "Rome")
	if err != nil {
		t.Fatalf("expected mock fallback result, got error: %v", err)
	}
	if len(items) == 0 || items[0].Provider != "mock" {
		t.Fatalf("expected mock fallback provider, got %+v", items)
	}
}

func TestFoursquareNoFallbackReturnsRealProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusInternalServerError)
	}))
	defer server.Close()

	provider, err := New(&config.Config{
		PlaceProvider: config.PlaceProviderConfig{
			Provider:                 config.PlaceProviderFoursquare,
			FallbackToMock:           false,
			FoursquareAPIKey:         "test-key",
			FoursquareBaseURL:        server.URL,
			FoursquareTimeoutSeconds: 1,
		},
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	_, err = provider.SearchPlaces(context.Background(), "Colosseum", "Rome")
	assertProviderErrorKind(t, err, providerErrorUnavailable)
}

func TestFoursquareSearchPlacesCallsEndpointAndNormalizesResponse(t *testing.T) {
	var called bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/places/search" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "test-key" {
			t.Fatalf("expected Authorization header, got %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Fatalf("expected Accept application/json, got %q", got)
		}
		if got := r.URL.Query().Get("query"); got != "Colosseum" {
			t.Fatalf("unexpected query: %q", got)
		}
		if got := r.URL.Query().Get("near"); got != "Rome" {
			t.Fatalf("unexpected near: %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Fatalf("unexpected limit: %q", got)
		}
		if fields := r.URL.Query().Get("fields"); !strings.Contains(fields, "rating") || !strings.Contains(fields, "stats") {
			t.Fatalf("expected requested rating/stats fields, got %q", fields)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"results": [
				{
					"fsq_id": "abc123",
					"name": "Colosseum",
					"location": {"formatted_address": "Piazza del Colosseo, Rome"},
					"geocodes": {"main": {"latitude": 41.8902, "longitude": 12.4922}},
					"categories": [{"name": "Historic Site"}],
					"rating": 9.4,
					"stats": {"total_ratings": 120000},
					"website": "https://example.com/colosseum"
				}
			]
		}`))
	}))
	defer server.Close()

	provider := newTestFoursquareProvider(t, server.URL)
	items, err := provider.SearchPlaces(context.Background(), "Colosseum", "Rome")
	if err != nil {
		t.Fatalf("search places: %v", err)
	}
	if !called {
		t.Fatal("expected server to be called")
	}
	if len(items) != 1 {
		t.Fatalf("expected one place, got %+v", items)
	}

	place := items[0]
	if place.Provider != "foursquare" || place.ProviderPlaceID != "abc123" || place.Name != "Colosseum" {
		t.Fatalf("unexpected normalized identity: %+v", place)
	}
	if place.Address != "Piazza del Colosseo, Rome" {
		t.Fatalf("unexpected address: %q", place.Address)
	}
	if place.Latitude == nil || *place.Latitude != 41.8902 || place.Longitude == nil || *place.Longitude != 12.4922 {
		t.Fatalf("unexpected coordinates: %+v", place)
	}
	if place.Category != "Historic Site" {
		t.Fatalf("unexpected category: %q", place.Category)
	}
	if place.Rating == nil || *place.Rating != 4.7 {
		t.Fatalf("expected 0-5 normalized rating 4.7, got %+v", place.Rating)
	}
	if place.RatingCount == nil || *place.RatingCount != 120000 {
		t.Fatalf("unexpected rating count: %+v", place.RatingCount)
	}
	if place.MapURL == "" || !strings.Contains(place.MapURL, "google.com/maps/search") {
		t.Fatalf("expected generated map URL, got %q", place.MapURL)
	}
}

func TestFoursquareGetPlaceDetailsNormalizesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/places/fsq-details-id" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "test-key" {
			t.Fatalf("expected Authorization header, got %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"fsq_id": "fsq-details-id",
			"name": "Test Museum",
			"location": {
				"address": "1 Museum Way",
				"locality": "Rome",
				"region": "Lazio",
				"postcode": "00100",
				"country": "IT"
			},
			"geocodes": {"main": {"latitude": 41.91, "longitude": 12.48}},
			"categories": [{"name": "Museum"}],
			"rating": 4.8,
			"stats": {"total_ratings": 42},
			"website": "https://example.com/museum",
			"link": "https://foursquare.com/v/test-museum"
		}`))
	}))
	defer server.Close()

	provider := newTestFoursquareProvider(t, server.URL)
	place, err := provider.GetPlaceDetails(context.Background(), "fsq-details-id")
	if err != nil {
		t.Fatalf("get details: %v", err)
	}
	if place == nil {
		t.Fatal("expected place details")
	}
	if place.Provider != "foursquare" || place.ProviderPlaceID != "fsq-details-id" {
		t.Fatalf("unexpected place identity: %+v", place)
	}
	if place.Address != "1 Museum Way, Rome, Lazio, 00100, IT" {
		t.Fatalf("unexpected address fallback: %q", place.Address)
	}
	if place.Rating == nil || *place.Rating != 4.8 {
		t.Fatalf("unexpected rating: %+v", place.Rating)
	}
	if place.MapURL != "https://foursquare.com/v/test-museum" {
		t.Fatalf("expected provider URL map link, got %q", place.MapURL)
	}
	if len(place.OpeningHours) != 0 {
		t.Fatalf("expected opening hours to be empty in v1, got %+v", place.OpeningHours)
	}
}

func TestFoursquareSearchErrorsAreClassified(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantKind   string
	}{
		{name: "unauthorized", statusCode: http.StatusUnauthorized, wantKind: providerErrorAuthConfig},
		{name: "forbidden", statusCode: http.StatusForbidden, wantKind: providerErrorAuthConfig},
		{name: "rate_limit", statusCode: http.StatusTooManyRequests, wantKind: providerErrorRateLimit},
		{name: "unavailable", statusCode: http.StatusBadGateway, wantKind: providerErrorUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "provider error", tt.statusCode)
			}))
			defer server.Close()

			provider := newTestFoursquareProvider(t, server.URL)
			_, err := provider.SearchPlaces(context.Background(), "Colosseum", "Rome")
			assertProviderErrorKind(t, err, tt.wantKind)
		})
	}
}

func TestFoursquareMalformedJSONReturnsResponseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results": [`))
	}))
	defer server.Close()

	provider := newTestFoursquareProvider(t, server.URL)
	_, err := provider.SearchPlaces(context.Background(), "Colosseum", "Rome")
	assertProviderErrorKind(t, err, providerErrorResponse)
}

func TestMockPlaceOpeningHoursUseValidConvention(t *testing.T) {
	timeFormat := regexp.MustCompile(`^(?:[01][0-9]|2[0-3]):[0-5][0-9]$`)

	for _, item := range mockPlaces() {
		if len(item.OpeningHours) == 0 {
			t.Fatalf("expected mock place %s to include opening hours", item.ProviderPlaceID)
		}
		for _, interval := range item.OpeningHours {
			if interval.DayOfWeek < 1 || interval.DayOfWeek > 7 {
				t.Fatalf("mock place %s has invalid dayOfWeek: %+v", item.ProviderPlaceID, interval)
			}
			if !timeFormat.MatchString(interval.Open) || !timeFormat.MatchString(interval.Close) {
				t.Fatalf("mock place %s has invalid HH:mm interval: %+v", item.ProviderPlaceID, interval)
			}
			if interval.Open >= interval.Close {
				t.Fatalf("mock place %s has non-ascending interval: %+v", item.ProviderPlaceID, interval)
			}
		}
	}
}

func newTestFoursquareProvider(t *testing.T, baseURL string) *FoursquarePlaceProvider {
	t.Helper()

	provider, err := NewFoursquarePlaceProvider(config.PlaceProviderConfig{
		FoursquareAPIKey:         "test-key",
		FoursquareBaseURL:        baseURL,
		FoursquareTimeoutSeconds: 1,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("new foursquare provider: %v", err)
	}
	return provider
}

func assertProviderErrorKind(t *testing.T, err error, wantKind string) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected provider error kind %q, got nil", wantKind)
	}

	var providerErr *ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("expected ProviderError, got %T: %v", err, err)
	}
	if providerErr.Kind != wantKind {
		t.Fatalf("expected error kind %q, got %q: %v", wantKind, providerErr.Kind, err)
	}
}
