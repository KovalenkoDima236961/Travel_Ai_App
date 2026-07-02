package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestHTTPMetricsMiddlewareRecordsRouteTemplate(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewHTTPMetrics("test-service", registry)

	r := chi.NewRouter()
	r.Use(HTTPMetricsMiddleware(metrics))
	r.Get("/trips/{tripID}/generation-jobs/{jobID}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	req := httptest.NewRequest(
		http.MethodGet,
		"/trips/0f6186e2-52b3-4e41-b077-2e72f2fdf8f1/generation-jobs/71f1d0ac-4650-450f-9f20-5a34af5e5851",
		nil,
	)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	expected := `
# HELP http_requests_total Total HTTP requests handled by service.
# TYPE http_requests_total counter
http_requests_total{method="GET",route="/trips/{tripID}/generation-jobs/{jobID}",service="test-service",status="202"} 1
`
	if err := testutil.GatherAndCompare(registry, strings.NewReader(expected), "http_requests_total"); err != nil {
		t.Fatal(err)
	}
}

func TestRoutePatternSanitizesRawUUIDAndNumberSegments(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/trips/0f6186e2-52b3-4e41-b077-2e72f2fdf8f1/days/12/items",
		nil,
	)

	if got := RoutePattern(req); got != "/trips/{uuid}/days/{number}/items" {
		t.Fatalf("route pattern = %q, want sanitized route", got)
	}
}
