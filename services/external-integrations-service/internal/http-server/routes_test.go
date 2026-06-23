package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"

	appservice "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/http-server/handler"
	placeprovider "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/provider/places"
	routeprovider "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/provider/routes"
	weatherprovider "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/provider/weather"
)

func TestHealthReturnsOK(t *testing.T) {
	resp := performRequest(newTestRouter(), http.MethodGet, "/health", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestReadyReturnsOK(t *testing.T) {
	resp := performRequest(newTestRouter(), http.MethodGet, "/ready", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestSearchMissingQueryReturnsBadRequest(t *testing.T) {
	resp := performRequest(newTestRouter(), http.MethodGet, "/places/search?destination=Rome", "")
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestSearchColosseumRomeReturnsColosseum(t *testing.T) {
	resp := performRequest(newTestRouter(), http.MethodGet, "/places/search?query=Colosseum&destination=Rome", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body handler.SearchPlacesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) == 0 || body.Items[0].Name != "Colosseum" {
		t.Fatalf("expected Colosseum result, got %+v", body.Items)
	}
	if len(body.Items[0].OpeningHours) == 0 {
		t.Fatalf("expected Colosseum search result to include opening hours, got %+v", body.Items[0])
	}
	if body.Items[0].OpeningHours[0].DayOfWeek != 1 ||
		body.Items[0].OpeningHours[0].Open != "08:30" ||
		body.Items[0].OpeningHours[0].Close != "19:15" {
		t.Fatalf("unexpected Colosseum opening hours: %+v", body.Items[0].OpeningHours)
	}
}

func TestSearchIsCaseInsensitive(t *testing.T) {
	resp := performRequest(newTestRouter(), http.MethodGet, "/places/search?query=coLoSSeUm&destination=rOmE", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body handler.SearchPlacesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) == 0 || body.Items[0].ProviderPlaceID != "mock-colosseum-rome" {
		t.Fatalf("expected case-insensitive Colosseum result, got %+v", body.Items)
	}
}

func TestSearchDestinationFiltersResults(t *testing.T) {
	resp := performRequest(newTestRouter(), http.MethodGet, "/places/search?query=museum&destination=Paris", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body handler.SearchPlacesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) == 0 {
		t.Fatal("expected Paris museum results")
	}
	for _, item := range body.Items {
		if !strings.Contains(item.ProviderPlaceID, "paris") {
			t.Fatalf("expected destination-filtered Paris result, got %+v", item)
		}
	}
}

func TestSearchUnknownQueryReturnsCityFallback(t *testing.T) {
	resp := performRequest(newTestRouter(), http.MethodGet, "/places/search?query=no-match-here&destination=Rome", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body handler.SearchPlacesResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Items) == 0 || len(body.Items) > 3 {
		t.Fatalf("expected up to 3 Rome fallback results, got %+v", body.Items)
	}
}

func TestGetDetailsReturnsPlace(t *testing.T) {
	resp := performRequest(newTestRouter(), http.MethodGet, "/places/mock-colosseum-rome", "")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body entity.Place
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != "Colosseum" || body.Provider != "mock" {
		t.Fatalf("unexpected place response: %+v", body)
	}
	if len(body.OpeningHours) == 0 {
		t.Fatalf("expected place details to include opening hours, got %+v", body)
	}
}

func TestPlaceWithoutOpeningHoursOmitsField(t *testing.T) {
	raw, err := json.Marshal(entity.Place{
		Provider:        "mock",
		ProviderPlaceID: "mock-no-hours",
		Name:            "No Hours Place",
		Address:         "Unknown",
	})
	if err != nil {
		t.Fatalf("marshal place: %v", err)
	}
	if strings.Contains(string(raw), "openingHours") {
		t.Fatalf("expected openingHours to be omitted when empty, got %s", raw)
	}
}

func TestGetDetailsUnknownIDReturnsNotFound(t *testing.T) {
	resp := performRequest(newTestRouter(), http.MethodGet, "/places/unknown-place", "")
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestCORSPreflightWorks(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/places/search", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	resp := httptest.NewRecorder()

	newTestRouter().ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.Code)
	}
	if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected allowed origin header, got %q", got)
	}
}

func newTestRouter() http.Handler {
	cfg := testConfig()
	svc := appservice.New(placeprovider.NewMockPlaceProvider(), zap.NewNop())
	placesHandler := handler.NewPlacesHandler(svc, zap.NewNop(), cfg.PlaceProvider.Provider)
	routesSvc := appservice.NewRoutesService(routeprovider.NewMockRouteProvider(), zap.NewNop())
	routesHandler := handler.NewRoutesHandler(routesSvc, zap.NewNop())
	weatherSvc := appservice.NewWeatherService(weatherprovider.NewMockWeatherProvider(), zap.NewNop())
	weatherHandler := handler.NewWeatherHandler(weatherSvc, zap.NewNop())
	return NewRouter(zap.NewNop(), placesHandler, routesHandler, weatherHandler, NewReadinessHandler(zap.NewNop()), cfg.CORS)
}

func testConfig() *config.Config {
	return &config.Config{
		Env: "test",
		HTTPServer: config.HTTPServer{
			Address: ":0",
		},
		PlaceProvider:   config.PlaceProviderConfig{Provider: "mock"},
		RouteProvider:   config.RouteProviderConfig{Provider: "mock"},
		WeatherProvider: config.WeatherProviderConfig{Provider: "mock"},
		CORS: config.CORSConfig{
			AllowedOrigins: "http://localhost:3000",
			AllowedMethods: "GET,POST,OPTIONS",
			AllowedHeaders: "Content-Type,Authorization",
		},
	}
}

func performRequest(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	return resp
}
