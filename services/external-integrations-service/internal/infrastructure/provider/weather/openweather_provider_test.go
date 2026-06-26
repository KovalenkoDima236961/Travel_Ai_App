package weather

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

func owmEntry(ts time.Time, tempMin, tempMax, pop, windMS float64, main string) owmForecastEntry {
	return owmForecastEntry{
		Dt:      ts.Unix(),
		Main:    owmMain{Temp: (tempMin + tempMax) / 2, TempMin: tempMin, TempMax: tempMax},
		Weather: []owmWeather{{Main: main, Description: strings.ToLower(main)}},
		Wind:    owmWind{Speed: windMS},
		Pop:     pop,
		DtTxt:   ts.Format("2006-01-02 15:04:05"),
	}
}

// newOWMServer routes geocoding and forecast paths. The forecast handler may
// override the default behaviour to simulate provider failures.
func newOWMServer(geo []owmGeoResult, forecast *owmForecastResponse, forecastHandler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/geo/") {
			_ = json.NewEncoder(w).Encode(geo)
			return
		}
		if forecastHandler != nil {
			forecastHandler(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(forecast)
	}))
}

func newTestOpenWeatherProvider(t *testing.T, baseURL string) *OpenWeatherProvider {
	t.Helper()
	provider, err := NewOpenWeatherProvider(config.WeatherProviderConfig{
		OpenWeatherAPIKey:  "test-key",
		OpenWeatherBaseURL: baseURL,
		OpenWeatherUnits:   "metric",
		TimeoutSeconds:     5,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("construct OpenWeather provider: %v", err)
	}
	return provider
}

func TestOpenWeatherGeocodingAndForecastRequestsFormedCorrectly(t *testing.T) {
	var geoQuery, forecastQuery url.Values

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/geo/") {
			geoQuery = r.URL.Query()
			_ = json.NewEncoder(w).Encode([]owmGeoResult{{Name: "Rome", Lat: 41.89, Lon: 12.49, Country: "IT"}})
			return
		}
		forecastQuery = r.URL.Query()
		ts := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
		_ = json.NewEncoder(w).Encode(owmForecastResponse{
			List: []owmForecastEntry{owmEntry(ts, 20, 28, 0.2, 4, "Clear")},
			City: owmCity{Timezone: 0},
		})
	}))
	defer server.Close()

	provider := newTestOpenWeatherProvider(t, server.URL)
	req := entity.WeatherForecastRequest{Destination: "Rome", StartDate: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), Days: 1}
	if _, err := provider.GetForecast(context.Background(), req); err != nil {
		t.Fatalf("forecast: %v", err)
	}

	if geoQuery.Get("q") != "Rome" || geoQuery.Get("limit") != "1" || geoQuery.Get("appid") != "test-key" {
		t.Fatalf("unexpected geocoding query: %v", geoQuery)
	}
	if forecastQuery.Get("lat") == "" || forecastQuery.Get("lon") == "" {
		t.Fatalf("expected lat/lon in forecast query: %v", forecastQuery)
	}
	if forecastQuery.Get("units") != "metric" || forecastQuery.Get("appid") != "test-key" {
		t.Fatalf("unexpected forecast query: %v", forecastQuery)
	}
}

func TestOpenWeatherGroupsThreeHourEntriesByDay(t *testing.T) {
	date := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	forecast := &owmForecastResponse{
		List: []owmForecastEntry{
			owmEntry(date.Add(6*time.Hour), 18, 22, 0.1, 2, "Clear"),
			owmEntry(date.Add(12*time.Hour), 24, 30, 0.4, 5, "Clouds"),
			owmEntry(date.Add(18*time.Hour), 20, 26, 0.2, 3, "Clouds"),
		},
		City: owmCity{Timezone: 0},
	}

	server := newOWMServer([]owmGeoResult{{Name: "Rome", Lat: 41.89, Lon: 12.49}}, forecast, nil)
	defer server.Close()

	provider := newTestOpenWeatherProvider(t, server.URL)
	result, err := provider.GetForecast(context.Background(), entity.WeatherForecastRequest{
		Destination: "Rome", StartDate: date, Days: 1,
	})
	if err != nil {
		t.Fatalf("forecast: %v", err)
	}

	if len(result.Days) != 1 {
		t.Fatalf("expected 1 day, got %d", len(result.Days))
	}
	day := result.Days[0]
	if day.Date != "2026-08-10" {
		t.Fatalf("expected date 2026-08-10, got %q", day.Date)
	}
	if day.TemperatureMinC != 18 || day.TemperatureMaxC != 30 {
		t.Fatalf("expected min 18 / max 30, got %v / %v", day.TemperatureMinC, day.TemperatureMaxC)
	}
	if day.PrecipitationChance != 40 {
		t.Fatalf("expected precipitation 40, got %d", day.PrecipitationChance)
	}
	if day.WindSpeedKph != 18 { // max 5 m/s * 3.6
		t.Fatalf("expected wind 18 kph, got %v", day.WindSpeedKph)
	}
	if day.Condition != "partly_cloudy" { // Clouds is dominant (2 vs 1)
		t.Fatalf("expected partly_cloudy condition, got %q", day.Condition)
	}
	if day.Summary != "Partly cloudy" {
		t.Fatalf("expected summary selected from dominant condition, got %q", day.Summary)
	}
	if result.Provider != "openweathermap" {
		t.Fatalf("expected provider openweathermap, got %q", result.Provider)
	}
}

func TestOpenWeatherCoverageGapReturnsError(t *testing.T) {
	date := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	forecast := &owmForecastResponse{
		List: []owmForecastEntry{owmEntry(date.Add(12*time.Hour), 20, 28, 0.2, 4, "Clear")},
		City: owmCity{Timezone: 0},
	}
	server := newOWMServer([]owmGeoResult{{Name: "Rome", Lat: 41.89, Lon: 12.49}}, forecast, nil)
	defer server.Close()

	provider := newTestOpenWeatherProvider(t, server.URL)
	// Request two days but only the first is covered.
	_, err := provider.GetForecast(context.Background(), entity.WeatherForecastRequest{
		Destination: "Rome", StartDate: date, Days: 2,
	})
	if err == nil {
		t.Fatal("expected coverage error when a requested day has no data")
	}
}

func TestOpenWeatherUnknownDestinationReturnsError(t *testing.T) {
	server := newOWMServer([]owmGeoResult{}, nil, nil)
	defer server.Close()

	provider := newTestOpenWeatherProvider(t, server.URL)
	_, err := provider.GetForecast(context.Background(), entity.WeatherForecastRequest{
		Destination: "Nowhereville", StartDate: time.Now(), Days: 1,
	})
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Kind != providerErrorNotFound {
		t.Fatalf("expected not_found provider error, got %v", err)
	}
}

func TestOpenWeatherClassifiesForecastStatusCodes(t *testing.T) {
	cases := map[int]string{
		http.StatusUnauthorized:        providerErrorAuthConfig,
		http.StatusForbidden:           providerErrorAuthConfig,
		http.StatusTooManyRequests:     providerErrorRateLimit,
		http.StatusInternalServerError: providerErrorUnavailable,
	}
	for status, wantKind := range cases {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := newOWMServer(
				[]owmGeoResult{{Name: "Rome", Lat: 41.89, Lon: 12.49}},
				nil,
				func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) },
			)
			defer server.Close()

			provider := newTestOpenWeatherProvider(t, server.URL)
			_, err := provider.GetForecast(context.Background(), entity.WeatherForecastRequest{
				Destination: "Rome", StartDate: time.Now(), Days: 1,
			})
			var providerErr *ProviderError
			if !errors.As(err, &providerErr) || providerErr.Kind != wantKind {
				t.Fatalf("status %d: expected kind %q, got %v", status, wantKind, err)
			}
		})
	}
}

func TestOpenWeatherMalformedForecastJSONReturnsError(t *testing.T) {
	server := newOWMServer(
		[]owmGeoResult{{Name: "Rome", Lat: 41.89, Lon: 12.49}},
		nil,
		func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("{not-json")) },
	)
	defer server.Close()

	provider := newTestOpenWeatherProvider(t, server.URL)
	_, err := provider.GetForecast(context.Background(), entity.WeatherForecastRequest{
		Destination: "Rome", StartDate: time.Now(), Days: 1,
	})
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Kind != providerErrorResponse {
		t.Fatalf("expected bad_response provider error, got %v", err)
	}
}

func TestNewMissingOpenWeatherKeyWithFallbackUsesMock(t *testing.T) {
	provider, err := New(&config.Config{WeatherProvider: config.WeatherProviderConfig{
		Provider:       config.WeatherProviderOpenWeather,
		FallbackToMock: true,
	}}, zap.NewNop())
	if err != nil {
		t.Fatalf("expected fallback to mock, got error: %v", err)
	}

	forecast, err := provider.GetForecast(context.Background(), entity.WeatherForecastRequest{
		Destination: "Rome", StartDate: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), Days: 3,
	})
	if err != nil {
		t.Fatalf("forecast: %v", err)
	}
	if forecast.Provider != "mock" {
		t.Fatalf("expected mock provider when key missing, got %q", forecast.Provider)
	}
}

func TestNewMissingOpenWeatherKeyWithoutFallbackFailsStartup(t *testing.T) {
	_, err := New(&config.Config{WeatherProvider: config.WeatherProviderConfig{
		Provider:       config.WeatherProviderOpenWeather,
		FallbackToMock: false,
	}}, zap.NewNop())
	if err == nil {
		t.Fatal("expected startup error when key missing and fallback disabled")
	}
}

func TestFallbackWeatherProviderUsesMockWhenPrimaryFails(t *testing.T) {
	server := newOWMServer(
		[]owmGeoResult{{Name: "Rome", Lat: 41.89, Lon: 12.49}},
		nil,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	)
	defer server.Close()

	primary := newTestOpenWeatherProvider(t, server.URL)
	provider := newFallbackWeatherProvider(config.WeatherProviderOpenWeather, primary, NewMockWeatherProvider(), zap.NewNop())

	forecast, err := provider.GetForecast(context.Background(), entity.WeatherForecastRequest{
		Destination: "Rome", StartDate: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), Days: 3,
	})
	if err != nil {
		t.Fatalf("expected fallback to succeed, got %v", err)
	}
	if forecast.Provider != "mock" || !forecast.FallbackUsed {
		t.Fatalf("expected mock fallback with fallbackUsed=true, got provider=%q fallbackUsed=%v", forecast.Provider, forecast.FallbackUsed)
	}
	if len(forecast.Days) != 3 {
		t.Fatalf("expected 3 days from mock fallback, got %d", len(forecast.Days))
	}
}
