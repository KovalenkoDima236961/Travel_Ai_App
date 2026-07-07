package observability

import (
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type HTTPMetrics struct {
	service  string
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight *prometheus.GaugeVec
}

var (
	defaultHTTPMetricsMu sync.Mutex
	defaultHTTPMetrics   = map[string]*HTTPMetrics{}
	uuidSegmentPattern   = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	numberSegmentPattern = regexp.MustCompile(`^\d+$`)
)

func DefaultHTTPMetrics(service string) *HTTPMetrics {
	defaultHTTPMetricsMu.Lock()
	defer defaultHTTPMetricsMu.Unlock()
	if metrics := defaultHTTPMetrics[service]; metrics != nil {
		return metrics
	}
	metrics := NewHTTPMetrics(service, prometheus.DefaultRegisterer)
	defaultHTTPMetrics[service] = metrics
	return metrics
}

func NewHTTPMetrics(service string, registerer prometheus.Registerer) *HTTPMetrics {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}
	return &HTTPMetrics{
		service: service,
		requests: registerCounterVec(registerer, prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total HTTP requests handled by service.",
			},
			[]string{"service", "method", "route", "status"},
		)),
		duration: registerHistogramVec(registerer, prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration by service, method, route, and status.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"service", "method", "route", "status"},
		)),
		inFlight: registerGaugeVec(registerer, prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "http_requests_in_flight",
				Help: "In-flight HTTP requests by service, method, and route.",
			},
			[]string{"service", "method", "route"},
		)),
	}
}

func HTTPMetricsMiddleware(metrics *HTTPMetrics) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if metrics == nil {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()
			route := routeLabel(r)
			metrics.inFlight.WithLabelValues(metrics.service, r.Method, route).Inc()
			defer metrics.inFlight.WithLabelValues(metrics.service, r.Method, route).Dec()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			route = routeLabel(r)
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			statusLabel := strconv.Itoa(status)
			metrics.requests.WithLabelValues(metrics.service, r.Method, route, statusLabel).Inc()
			metrics.duration.WithLabelValues(metrics.service, r.Method, route, statusLabel).Observe(time.Since(start).Seconds())
		})
	}
}

func MetricsHandler(gatherer prometheus.Gatherer) http.Handler {
	if gatherer == nil {
		gatherer = prometheus.DefaultGatherer
	}
	return promhttp.HandlerFor(gatherer, promhttp.HandlerOpts{})
}

func RoutePattern(r *http.Request) string {
	return routeLabel(r)
}

func routeLabel(r *http.Request) string {
	if r == nil {
		return "unknown"
	}
	if rctx := chi.RouteContext(r.Context()); rctx != nil {
		if pattern := rctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return sanitizePath(r.URL.Path)
}

func sanitizePath(path string) string {
	if path == "" {
		return "/"
	}
	parts := stringsSplit(path, '/')
	for i, part := range parts {
		if uuidSegmentPattern.MatchString(part) {
			parts[i] = "{uuid}"
			continue
		}
		if numberSegmentPattern.MatchString(part) {
			parts[i] = "{number}"
		}
	}
	if path[0] == '/' {
		return "/" + stringsJoin(parts, "/")
	}
	return stringsJoin(parts, "/")
}

func stringsSplit(s string, sep byte) []string {
	parts := make([]string, 0, 8)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			if i > start {
				parts = append(parts, s[start:i])
			}
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func stringsJoin(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for _, part := range parts[1:] {
		out += sep + part
	}
	return out
}

func registerCounterVec(registerer prometheus.Registerer, collector *prometheus.CounterVec) *prometheus.CounterVec {
	if err := registerer.Register(collector); err != nil {
		if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := already.ExistingCollector.(*prometheus.CounterVec); ok {
				return existing
			}
		}
	}
	return collector
}

func registerHistogramVec(registerer prometheus.Registerer, collector *prometheus.HistogramVec) *prometheus.HistogramVec {
	if err := registerer.Register(collector); err != nil {
		if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := already.ExistingCollector.(*prometheus.HistogramVec); ok {
				return existing
			}
		}
	}
	return collector
}

func registerGaugeVec(registerer prometheus.Registerer, collector *prometheus.GaugeVec) *prometheus.GaugeVec {
	if err := registerer.Register(collector); err != nil {
		if already, ok := err.(prometheus.AlreadyRegisteredError); ok {
			if existing, ok := already.ExistingCollector.(*prometheus.GaugeVec); ok {
				return existing
			}
		}
	}
	return collector
}
