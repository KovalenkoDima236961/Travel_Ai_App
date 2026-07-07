package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

func TestWeatherForecastValidRequestReturnsOK(t *testing.T) {
	resp := performRequest(newTestRouter(), http.MethodGet, "/weather/forecast?destination=Rome&startDate=2026-08-10&days=3", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	forecast := decodeForecast(t, resp)
	if forecast.Provider != "mock" {
		t.Fatalf("expected provider mock, got %q", forecast.Provider)
	}
	if forecast.Destination != "Rome" {
		t.Fatalf("expected destination Rome, got %q", forecast.Destination)
	}
	if len(forecast.Days) != 3 {
		t.Fatalf("expected exactly 3 forecast days, got %d", len(forecast.Days))
	}
	if forecast.Days[0].Date != "2026-08-10" {
		t.Fatalf("expected first day 2026-08-10, got %q", forecast.Days[0].Date)
	}
}

func TestWeatherForecastMissingDestinationReturnsBadRequest(t *testing.T) {
	resp := performRequest(newTestRouter(), http.MethodGet, "/weather/forecast?startDate=2026-08-10&days=3", "")
	assertWeatherBadRequest(t, resp)
}

func TestWeatherForecastInvalidStartDateReturnsBadRequest(t *testing.T) {
	resp := performRequest(newTestRouter(), http.MethodGet, "/weather/forecast?destination=Rome&startDate=08-10-2026&days=3", "")
	assertWeatherBadRequest(t, resp)
}

func TestWeatherForecastDaysTooLowReturnsBadRequest(t *testing.T) {
	resp := performRequest(newTestRouter(), http.MethodGet, "/weather/forecast?destination=Rome&startDate=2026-08-10&days=0", "")
	assertWeatherBadRequest(t, resp)
}

func TestWeatherForecastDaysTooHighReturnsBadRequest(t *testing.T) {
	resp := performRequest(newTestRouter(), http.MethodGet, "/weather/forecast?destination=Rome&startDate=2026-08-10&days=31", "")
	assertWeatherBadRequest(t, resp)
}

func TestWeatherForecastIsDeterministic(t *testing.T) {
	path := "/weather/forecast?destination=Paris&startDate=2026-04-10&days=4"
	first := decodeForecast(t, performRequest(newTestRouter(), http.MethodGet, path, ""))
	second := decodeForecast(t, performRequest(newTestRouter(), http.MethodGet, path, ""))

	if len(first.Days) != len(second.Days) {
		t.Fatalf("expected same day count, got %d vs %d", len(first.Days), len(second.Days))
	}
	for i := range first.Days {
		if !reflect.DeepEqual(first.Days[i], second.Days[i]) {
			t.Fatalf("forecast day %d differs: %+v vs %+v", i, first.Days[i], second.Days[i])
		}
	}
}

func TestWeatherForecastRomeSummerIncludesHotSunnyStyleData(t *testing.T) {
	forecast := decodeForecast(t, performRequest(newTestRouter(), http.MethodGet, "/weather/forecast?destination=Rome&startDate=2026-08-10&days=3", ""))

	foundHotSunny := false
	for _, day := range forecast.Days {
		if day.TemperatureMaxC >= 32 && (day.Condition == "hot" || day.Condition == "sunny") {
			foundHotSunny = true
		}
	}
	if !foundHotSunny {
		t.Fatalf("expected Rome summer to include hot/sunny-style data, got %+v", forecast.Days)
	}
}

func TestWeatherForecastWarningsAreGenerated(t *testing.T) {
	rome := decodeForecast(t, performRequest(newTestRouter(), http.MethodGet, "/weather/forecast?destination=Rome&startDate=2026-08-10&days=1", ""))
	if !forecastContainsWarning(rome, "High heat") {
		t.Fatalf("expected high heat warning, got %+v", rome.Days)
	}

	paris := decodeForecast(t, performRequest(newTestRouter(), http.MethodGet, "/weather/forecast?destination=Paris&startDate=2026-04-10&days=10", ""))
	if !forecastContainsWarning(paris, "Rain likely") {
		t.Fatalf("expected rain warning, got %+v", paris.Days)
	}
}

func TestWeatherForecastCORSPreflightAllowsGET(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/weather/forecast", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	resp := httptest.NewRecorder()

	newTestRouter().ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.Code)
	}
	if got := resp.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "GET") {
		t.Fatalf("expected GET in allowed methods, got %q", got)
	}
}

func decodeForecast(t *testing.T, resp *httptest.ResponseRecorder) entity.WeatherForecast {
	t.Helper()
	var forecast entity.WeatherForecast
	if err := json.NewDecoder(resp.Body).Decode(&forecast); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return forecast
}

func assertWeatherBadRequest(t *testing.T, resp *httptest.ResponseRecorder) {
	t.Helper()
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if strings.TrimSpace(body["error"]) == "" {
		t.Fatalf("expected JSON error body, got %+v", body)
	}
}

func forecastContainsWarning(forecast entity.WeatherForecast, prefix string) bool {
	for _, day := range forecast.Days {
		for _, warning := range day.Warnings {
			if strings.HasPrefix(warning, prefix) {
				return true
			}
		}
	}
	return false
}
