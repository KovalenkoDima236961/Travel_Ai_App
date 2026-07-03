package routes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

const orsTwoStopResponse = `{
  "routes": [
    {
      "summary": {"distance": 1234.5, "duration": 600.0},
      "segments": [
        {"distance": 1234.5, "duration": 600.0}
      ],
      "geometry": "abc123_polyline"
    }
  ]
}`

func orsTwoStopRequest() entity.RouteEstimateRequest {
	return entity.RouteEstimateRequest{
		Mode: "walking",
		Stops: []entity.RouteStop{
			{Name: "Colosseum", Latitude: 41.8902, Longitude: 12.4922},
			{Name: "Trevi Fountain", Latitude: 41.9009, Longitude: 12.4833},
		},
	}
}

func newTestORSProvider(t *testing.T, baseURL string) *OpenRouteServiceProvider {
	t.Helper()
	provider, err := NewOpenRouteServiceProvider(config.RouteProviderConfig{
		ORSAPIKey:      "test-key",
		ORSBaseURL:     baseURL,
		TimeoutSeconds: 5,
	}, zap.NewNop())
	if err != nil {
		t.Fatalf("construct ORS provider: %v", err)
	}
	return provider
}

func TestORSProviderSendsAPIKeyHeaderAndCoordinateOrder(t *testing.T) {
	var gotAuth string
	var gotBody orsDirectionsRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(orsTwoStopResponse))
	}))
	defer server.Close()

	provider := newTestORSProvider(t, server.URL)
	if _, err := provider.EstimateRoute(context.Background(), orsTwoStopRequest()); err != nil {
		t.Fatalf("estimate: %v", err)
	}

	if gotAuth != "test-key" {
		t.Fatalf("expected raw API key in Authorization header, got %q", gotAuth)
	}
	if len(gotBody.Coordinates) != 2 {
		t.Fatalf("expected 2 coordinates, got %d", len(gotBody.Coordinates))
	}
	// ORS requires [longitude, latitude] order.
	if gotBody.Coordinates[0][0] != 12.4922 || gotBody.Coordinates[0][1] != 41.8902 {
		t.Fatalf("expected first coordinate [lng,lat]=[12.4922,41.8902], got %v", gotBody.Coordinates[0])
	}
}

func TestORSProviderMapsProfiles(t *testing.T) {
	cases := map[string]string{
		"walking": "/v2/directions/foot-walking",
		"driving": "/v2/directions/driving-car",
		"cycling": "/v2/directions/cycling-regular",
	}
	for mode, wantPath := range cases {
		t.Run(mode, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				_, _ = w.Write([]byte(orsTwoStopResponse))
			}))
			defer server.Close()

			provider := newTestORSProvider(t, server.URL)
			req := orsTwoStopRequest()
			req.Mode = mode
			if _, err := provider.EstimateRoute(context.Background(), req); err != nil {
				t.Fatalf("estimate: %v", err)
			}
			if gotPath != wantPath {
				t.Fatalf("expected path %q, got %q", wantPath, gotPath)
			}
		})
	}
}

func TestORSProviderParsesDistanceAndDuration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(orsTwoStopResponse))
	}))
	defer server.Close()

	provider := newTestORSProvider(t, server.URL)
	estimate, err := provider.EstimateRoute(context.Background(), orsTwoStopRequest())
	if err != nil {
		t.Fatalf("estimate: %v", err)
	}

	if estimate.Provider != "ors" {
		t.Fatalf("expected provider ors, got %q", estimate.Provider)
	}
	if estimate.DistanceKm != 1.23 {
		t.Fatalf("expected 1.23 km (1234.5m), got %v", estimate.DistanceKm)
	}
	if estimate.DurationMinutes != 10 {
		t.Fatalf("expected 10 minutes (600s), got %d", estimate.DurationMinutes)
	}
	if len(estimate.Segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(estimate.Segments))
	}
	if estimate.RouteGeometry != "abc123_polyline" {
		t.Fatalf("expected geometry passthrough, got %v", estimate.RouteGeometry)
	}
}

func TestORSProviderClassifiesStatusCodes(t *testing.T) {
	cases := map[int]string{
		http.StatusUnauthorized:        providerErrorAuthConfig,
		http.StatusForbidden:           providerErrorAuthConfig,
		http.StatusTooManyRequests:     providerErrorRateLimit,
		http.StatusInternalServerError: providerErrorUnavailable,
		http.StatusBadGateway:          providerErrorUnavailable,
	}
	for status, wantKind := range cases {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
			}))
			defer server.Close()

			provider := newTestORSProvider(t, server.URL)
			_, err := provider.EstimateRoute(context.Background(), orsTwoStopRequest())
			var providerErr *ProviderError
			if !errors.As(err, &providerErr) {
				t.Fatalf("expected ProviderError, got %v", err)
			}
			if providerErr.Kind != wantKind {
				t.Fatalf("status %d: expected kind %q, got %q", status, wantKind, providerErr.Kind)
			}
		})
	}
}

func TestORSProviderHandlesMalformedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("{not-json"))
	}))
	defer server.Close()

	provider := newTestORSProvider(t, server.URL)
	_, err := provider.EstimateRoute(context.Background(), orsTwoStopRequest())
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Kind != providerErrorResponse {
		t.Fatalf("expected bad_response provider error, got %v", err)
	}
}

func TestORSProviderUnsupportedModeReturnsError(t *testing.T) {
	provider := newTestORSProvider(t, "https://example.invalid")
	req := orsTwoStopRequest()
	req.Mode = "swimming"
	if _, err := provider.EstimateRoute(context.Background(), req); err == nil {
		t.Fatal("expected error for unsupported mode")
	}
}

func TestNewMissingORSKeyWithFallbackUsesMock(t *testing.T) {
	provider, err := New(&config.Config{RouteProvider: config.RouteProviderConfig{
		Provider:       config.RouteProviderORS,
		FallbackToMock: true,
	}}, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("expected fallback to mock, got error: %v", err)
	}

	estimate, err := provider.EstimateRoute(context.Background(), orsTwoStopRequest())
	if err != nil {
		t.Fatalf("estimate: %v", err)
	}
	if estimate.Provider != "mock" {
		t.Fatalf("expected mock provider when ORS key missing, got %q", estimate.Provider)
	}
}

func TestNewMissingORSKeyWithoutFallbackFailsStartup(t *testing.T) {
	_, err := New(&config.Config{RouteProvider: config.RouteProviderConfig{
		Provider:       config.RouteProviderORS,
		FallbackToMock: false,
	}}, nil, zap.NewNop())
	if err == nil {
		t.Fatal("expected startup error when ORS key missing and fallback disabled")
	}
}

func TestFallbackRouteProviderUsesMockWhenPrimaryFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	primary := newTestORSProvider(t, server.URL)
	provider := newFallbackRouteProvider(config.RouteProviderORS, primary, NewMockRouteProvider(), zap.NewNop())

	estimate, err := provider.EstimateRoute(context.Background(), orsTwoStopRequest())
	if err != nil {
		t.Fatalf("expected fallback to succeed, got %v", err)
	}
	if estimate.Provider != "mock" {
		t.Fatalf("expected provider mock after fallback, got %q", estimate.Provider)
	}
	if !estimate.FallbackUsed {
		t.Fatal("expected fallbackUsed=true after fallback")
	}
}
