package httpserver

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

// twoRomeStops is a valid two-stop walking request between two mock Rome places.
const twoRomeStops = `{
  "mode": "walking",
  "stops": [
    {"name": "Colosseum", "latitude": 41.8902, "longitude": 12.4922},
    {"name": "Trevi Fountain", "latitude": 41.9009, "longitude": 12.4833}
  ]
}`

func TestRouteEstimateValidWalkingReturnsOK(t *testing.T) {
	resp := performRequest(newTestRouter(), http.MethodPost, "/routes/estimate", twoRomeStops)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", resp.Code, resp.Body.String())
	}

	estimate := decodeEstimate(t, resp)
	if estimate.Provider != "mock" {
		t.Fatalf("expected provider mock, got %q", estimate.Provider)
	}
	if estimate.Mode != "walking" {
		t.Fatalf("expected mode walking, got %q", estimate.Mode)
	}
	if estimate.DistanceKm <= 0 {
		t.Fatalf("expected positive distance, got %v", estimate.DistanceKm)
	}
	if estimate.DurationMinutes <= 0 {
		t.Fatalf("expected positive duration, got %v", estimate.DurationMinutes)
	}
	if len(estimate.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(estimate.Segments))
	}
}

func TestRouteEstimateMissingModeReturnsBadRequest(t *testing.T) {
	body := `{"stops":[{"name":"A","latitude":41.0,"longitude":12.0},{"name":"B","latitude":41.1,"longitude":12.1}]}`
	resp := performRequest(newTestRouter(), http.MethodPost, "/routes/estimate", body)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRouteEstimateUnsupportedModeReturnsBadRequest(t *testing.T) {
	body := `{"mode":"driving","stops":[{"name":"A","latitude":41.0,"longitude":12.0},{"name":"B","latitude":41.1,"longitude":12.1}]}`
	resp := performRequest(newTestRouter(), http.MethodPost, "/routes/estimate", body)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRouteEstimateTooFewStopsReturnsBadRequest(t *testing.T) {
	body := `{"mode":"walking","stops":[{"name":"A","latitude":41.0,"longitude":12.0}]}`
	resp := performRequest(newTestRouter(), http.MethodPost, "/routes/estimate", body)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRouteEstimateTooManyStopsReturnsBadRequest(t *testing.T) {
	stops := make([]entity.RouteStop, 26)
	for i := range stops {
		stops[i] = entity.RouteStop{Name: fmt.Sprintf("Stop %d", i), Latitude: 41.0, Longitude: 12.0}
	}
	body := marshalRequest(t, entity.RouteEstimateRequest{Mode: "walking", Stops: stops})
	resp := performRequest(newTestRouter(), http.MethodPost, "/routes/estimate", body)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRouteEstimateInvalidLatitudeReturnsBadRequest(t *testing.T) {
	body := `{"mode":"walking","stops":[{"name":"A","latitude":120.0,"longitude":12.0},{"name":"B","latitude":41.1,"longitude":12.1}]}`
	resp := performRequest(newTestRouter(), http.MethodPost, "/routes/estimate", body)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRouteEstimateInvalidLongitudeReturnsBadRequest(t *testing.T) {
	body := `{"mode":"walking","stops":[{"name":"A","latitude":41.0,"longitude":12.0},{"name":"B","latitude":41.1,"longitude":200.0}]}`
	resp := performRequest(newTestRouter(), http.MethodPost, "/routes/estimate", body)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRouteEstimateMissingStopNameReturnsBadRequest(t *testing.T) {
	body := `{"mode":"walking","stops":[{"name":"","latitude":41.0,"longitude":12.0},{"name":"B","latitude":41.1,"longitude":12.1}]}`
	resp := performRequest(newTestRouter(), http.MethodPost, "/routes/estimate", body)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", resp.Code, resp.Body.String())
	}
}

func TestRouteEstimateIsDeterministic(t *testing.T) {
	router := newTestRouter()
	first := decodeEstimate(t, performRequest(router, http.MethodPost, "/routes/estimate", twoRomeStops))
	second := decodeEstimate(t, performRequest(router, http.MethodPost, "/routes/estimate", twoRomeStops))

	if first.DistanceKm != second.DistanceKm || first.DurationMinutes != second.DurationMinutes {
		t.Fatalf("expected deterministic estimate, got %+v vs %+v", first, second)
	}
}

func TestRouteEstimateSegmentCountEqualsStopsMinusOne(t *testing.T) {
	body := `{"mode":"walking","stops":[
		{"name":"Colosseum","latitude":41.8902,"longitude":12.4922},
		{"name":"Roman Forum","latitude":41.8925,"longitude":12.4853},
		{"name":"Trevi Fountain","latitude":41.9009,"longitude":12.4833}
	]}`
	estimate := decodeEstimate(t, performRequest(newTestRouter(), http.MethodPost, "/routes/estimate", body))
	if len(estimate.Segments) != 2 {
		t.Fatalf("expected 2 segments for 3 stops, got %d", len(estimate.Segments))
	}
}

func TestRouteEstimateTotalEqualsSumOfSegments(t *testing.T) {
	body := `{"mode":"walking","stops":[
		{"name":"Colosseum","latitude":41.8902,"longitude":12.4922},
		{"name":"Roman Forum","latitude":41.8925,"longitude":12.4853},
		{"name":"Trevi Fountain","latitude":41.9009,"longitude":12.4833}
	]}`
	estimate := decodeEstimate(t, performRequest(newTestRouter(), http.MethodPost, "/routes/estimate", body))

	var sumDistance float64
	var sumDuration int
	for _, segment := range estimate.Segments {
		sumDistance += segment.DistanceKm
		sumDuration += segment.DurationMinutes
	}

	if math.Abs(estimate.DistanceKm-sumDistance) >= 0.01 {
		t.Fatalf("total distance %v does not match sum of segments %v", estimate.DistanceKm, sumDistance)
	}
	if estimate.DurationMinutes != sumDuration {
		t.Fatalf("total duration %d does not match sum of segments %d", estimate.DurationMinutes, sumDuration)
	}
}

func TestRouteEstimateCORSPreflightAllowsPOST(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/routes/estimate", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	resp := httptest.NewRecorder()

	newTestRouter().ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.Code)
	}
	if got := resp.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, "POST") {
		t.Fatalf("expected POST in allowed methods, got %q", got)
	}
}

func decodeEstimate(t *testing.T, resp *httptest.ResponseRecorder) entity.RouteEstimate {
	t.Helper()
	var estimate entity.RouteEstimate
	if err := json.NewDecoder(resp.Body).Decode(&estimate); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return estimate
}

func marshalRequest(t *testing.T, req entity.RouteEstimateRequest) string {
	t.Helper()
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	return string(data)
}
