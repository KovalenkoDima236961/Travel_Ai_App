package observability

import (
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type ProviderHealthSnapshot struct {
	LastSuccessAt      *time.Time
	LastFailureAt      *time.Time
	RecentSuccessCount int
	RecentFailureCount int
	LastErrorCode      string
}

type providerHealthState struct {
	mu                 sync.Mutex
	lastSuccessAt      *time.Time
	lastFailureAt      *time.Time
	recentSuccessCount int
	recentFailureCount int
	lastErrorCode      string
}

var providerHealth sync.Map
var providerCircuits sync.Map

const (
	providerCircuitFailureThreshold = 3
	providerCircuitCooldown         = 30 * time.Second
)

type providerCircuitState struct {
	mu                  sync.Mutex
	consecutiveFailures int
	openUntil           time.Time
}

var (
	externalProviderRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "external_provider_requests_total", Help: "Total external provider requests."},
		[]string{"provider", "operation", "result"},
	)
	externalProviderDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "external_provider_duration_seconds",
			Help:    "External provider request duration.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"provider", "operation", "result"},
	)
	externalProviderFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "external_provider_failures_total", Help: "Total external provider failures."},
		[]string{"provider", "operation", "error_code"},
	)
	externalProviderFallback = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "external_provider_fallback_total", Help: "Total external provider fallback uses."},
		[]string{"provider", "operation", "fallback_provider"},
	)
	externalProviderCacheHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "external_provider_cache_hits_total", Help: "Total external provider cache hits."},
		[]string{"provider", "operation"},
	)
	externalProviderCacheMisses = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "external_provider_cache_misses_total", Help: "Total external provider cache misses."},
		[]string{"provider", "operation"},
	)
	externalProviderCircuitOpen = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "external_provider_circuit_open", Help: "Whether a provider operation circuit is currently open."},
		[]string{"provider", "operation"},
	)
	externalProviderShortCircuits = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "external_provider_short_circuits_total", Help: "Provider calls skipped while a cooldown circuit is open."},
		[]string{"provider", "operation"},
	)
)

func init() {
	prometheus.MustRegister(
		externalProviderRequests,
		externalProviderDuration,
		externalProviderFailures,
		externalProviderFallback,
		externalProviderCacheHits,
		externalProviderCacheMisses,
		externalProviderCircuitOpen,
		externalProviderShortCircuits,
	)
}

// ProviderCircuitAllows returns false after repeated primary-provider failures.
// The circuit automatically permits one probe after a short cooldown.
func ProviderCircuitAllows(provider, operation string) bool {
	provider, operation = normalize(provider), normalize(operation)
	state := circuitState(provider, operation)
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.openUntil.IsZero() {
		return true
	}
	if time.Now().UTC().Before(state.openUntil) {
		externalProviderShortCircuits.WithLabelValues(provider, operation).Inc()
		return false
	}
	state.openUntil = time.Time{}
	state.consecutiveFailures = 0
	externalProviderCircuitOpen.WithLabelValues(provider, operation).Set(0)
	return true
}

func RecordProviderCircuitSuccess(provider, operation string) {
	provider, operation = normalize(provider), normalize(operation)
	state := circuitState(provider, operation)
	state.mu.Lock()
	state.consecutiveFailures = 0
	state.openUntil = time.Time{}
	state.mu.Unlock()
	externalProviderCircuitOpen.WithLabelValues(provider, operation).Set(0)
}

func RecordProviderCircuitFailure(provider, operation string) {
	provider, operation = normalize(provider), normalize(operation)
	state := circuitState(provider, operation)
	state.mu.Lock()
	state.consecutiveFailures++
	if state.consecutiveFailures >= providerCircuitFailureThreshold {
		state.openUntil = time.Now().UTC().Add(providerCircuitCooldown)
	}
	open := !state.openUntil.IsZero()
	state.mu.Unlock()
	if open {
		externalProviderCircuitOpen.WithLabelValues(provider, operation).Set(1)
	}
}

func circuitState(provider, operation string) *providerCircuitState {
	value, _ := providerCircuits.LoadOrStore(provider+":"+operation, &providerCircuitState{})
	return value.(*providerCircuitState)
}

func RecordProviderRequest(provider, operation, result string, duration time.Duration) {
	provider = normalize(provider)
	operation = normalize(operation)
	result = normalize(result)
	externalProviderRequests.WithLabelValues(provider, operation, result).Inc()
	externalProviderDuration.WithLabelValues(provider, operation, result).Observe(duration.Seconds())
	if result == "success" {
		healthState(provider).recordSuccess()
	}
}

func RecordProviderFailure(provider, operation, errorCode string) {
	provider = normalize(provider)
	errorCode = normalize(errorCode)
	externalProviderFailures.WithLabelValues(provider, normalize(operation), errorCode).Inc()
	healthState(provider).recordFailure(errorCode)
}

func RecordProviderFallback(provider, operation, fallbackProvider string) {
	externalProviderFallback.WithLabelValues(normalize(provider), normalize(operation), normalize(fallbackProvider)).Inc()
}

func RecordProviderCacheHit(provider, operation string) {
	externalProviderCacheHits.WithLabelValues(normalize(provider), normalize(operation)).Inc()
}

func RecordProviderCacheMiss(provider, operation string) {
	externalProviderCacheMisses.WithLabelValues(normalize(provider), normalize(operation)).Inc()
}

func normalize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "unknown"
	}
	return value
}

func ProviderHealth(provider string) ProviderHealthSnapshot {
	state := healthState(normalize(provider))
	state.mu.Lock()
	defer state.mu.Unlock()
	return ProviderHealthSnapshot{
		LastSuccessAt:      cloneTimePtr(state.lastSuccessAt),
		LastFailureAt:      cloneTimePtr(state.lastFailureAt),
		RecentSuccessCount: state.recentSuccessCount,
		RecentFailureCount: state.recentFailureCount,
		LastErrorCode:      state.lastErrorCode,
	}
}

func healthState(provider string) *providerHealthState {
	value, _ := providerHealth.LoadOrStore(provider, &providerHealthState{})
	return value.(*providerHealthState)
}

func (s *providerHealthState) recordSuccess() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	s.lastSuccessAt = &now
	s.recentSuccessCount++
}

func (s *providerHealthState) recordFailure(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	s.lastFailureAt = &now
	s.recentFailureCount++
	s.lastErrorCode = code
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
