package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	appservice "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/httpserver/handler"
	routeprovider "github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providers/routes"
)

// failingRouteProvider always fails, standing in for a real provider outage when
// fallback is disabled.
type failingRouteProvider struct{}

func (failingRouteProvider) EstimateRoute(context.Context, entity.RouteEstimateRequest) (*entity.RouteEstimate, error) {
	return nil, errors.New("provider down")
}

// failingWeatherProvider always fails.
type failingWeatherProvider struct{}

func (failingWeatherProvider) GetForecast(context.Context, entity.WeatherForecastRequest) (*entity.WeatherForecast, error) {
	return nil, errors.New("provider down")
}

func twoStopBody(mode string) string {
	return fmt.Sprintf(
		`{"mode":%q,"stops":[{"name":"A","latitude":41.0,"longitude":12.0},{"name":"B","latitude":41.1,"longitude":12.1}]}`,
		mode,
	)
}

func TestRouteEstimateORSProviderAllowsDrivingAndCycling(t *testing.T) {
	r := chi.NewRouter()
	svc := appservice.NewRoutesService(routeprovider.NewMockRouteProvider(), zap.NewNop())
	handler.NewRoutesHandler(svc, zap.NewNop(), "ors").RegisterRoutes(r)

	for _, mode := range []string{"walking", "driving", "cycling"} {
		resp := performRequest(r, http.MethodPost, "/routes/estimate", twoStopBody(mode))
		if resp.Code != http.StatusOK {
			t.Fatalf("mode %q: expected 200, got %d body=%s", mode, resp.Code, resp.Body.String())
		}
	}
}

func TestRouteEstimateORSProviderRejectsUnknownMode(t *testing.T) {
	r := chi.NewRouter()
	svc := appservice.NewRoutesService(routeprovider.NewMockRouteProvider(), zap.NewNop())
	handler.NewRoutesHandler(svc, zap.NewNop(), "ors").RegisterRoutes(r)

	resp := performRequest(r, http.MethodPost, "/routes/estimate", twoStopBody("swimming"))
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown mode, got %d", resp.Code)
	}
}

func TestRouteEstimateProviderUnavailableReturnsSafeError(t *testing.T) {
	r := chi.NewRouter()
	svc := appservice.NewRoutesService(failingRouteProvider{}, zap.NewNop())
	handler.NewRoutesHandler(svc, zap.NewNop(), "ors").RegisterRoutes(r)

	resp := performRequest(r, http.MethodPost, "/routes/estimate", twoStopBody("walking"))
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body["error"] != "route_provider_unavailable" {
		t.Fatalf("expected safe error code, got %+v", body)
	}
}

func TestWeatherForecastProviderUnavailableReturnsSafeError(t *testing.T) {
	r := chi.NewRouter()
	svc := appservice.NewWeatherService(failingWeatherProvider{}, zap.NewNop())
	handler.NewWeatherHandler(svc, zap.NewNop()).RegisterRoutes(r)

	resp := performRequest(r, http.MethodGet, "/weather/forecast?destination=Rome&startDate=2026-08-10&days=3", "")
	if resp.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d body=%s", resp.Code, resp.Body.String())
	}

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body["error"] != "weather_provider_unavailable" {
		t.Fatalf("expected safe error code, got %+v", body)
	}
}
