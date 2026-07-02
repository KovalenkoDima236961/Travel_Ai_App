package observability

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestProviderMetricsNormalizeAndRecordBoundedLabels(t *testing.T) {
	provider := " TestProvider "
	operation := " TestOperation "
	result := " Success "
	errorCode := " TestError "
	fallbackProvider := " MockFallback "

	requestsBefore := testutil.ToFloat64(externalProviderRequests.WithLabelValues("testprovider", "testoperation", "success"))
	failuresBefore := testutil.ToFloat64(externalProviderFailures.WithLabelValues("testprovider", "testoperation", "testerror"))
	fallbackBefore := testutil.ToFloat64(externalProviderFallback.WithLabelValues("testprovider", "testoperation", "mockfallback"))
	hitsBefore := testutil.ToFloat64(externalProviderCacheHits.WithLabelValues("testprovider", "testoperation"))
	missesBefore := testutil.ToFloat64(externalProviderCacheMisses.WithLabelValues("testprovider", "testoperation"))

	RecordProviderRequest(provider, operation, result, time.Millisecond)
	RecordProviderFailure(provider, operation, errorCode)
	RecordProviderFallback(provider, operation, fallbackProvider)
	RecordProviderCacheHit(provider, operation)
	RecordProviderCacheMiss(provider, operation)

	if got := testutil.ToFloat64(externalProviderRequests.WithLabelValues("testprovider", "testoperation", "success")); got != requestsBefore+1 {
		t.Fatalf("provider requests = %v, want %v", got, requestsBefore+1)
	}
	if got := testutil.ToFloat64(externalProviderFailures.WithLabelValues("testprovider", "testoperation", "testerror")); got != failuresBefore+1 {
		t.Fatalf("provider failures = %v, want %v", got, failuresBefore+1)
	}
	if got := testutil.ToFloat64(externalProviderFallback.WithLabelValues("testprovider", "testoperation", "mockfallback")); got != fallbackBefore+1 {
		t.Fatalf("provider fallbacks = %v, want %v", got, fallbackBefore+1)
	}
	if got := testutil.ToFloat64(externalProviderCacheHits.WithLabelValues("testprovider", "testoperation")); got != hitsBefore+1 {
		t.Fatalf("provider cache hits = %v, want %v", got, hitsBefore+1)
	}
	if got := testutil.ToFloat64(externalProviderCacheMisses.WithLabelValues("testprovider", "testoperation")); got != missesBefore+1 {
		t.Fatalf("provider cache misses = %v, want %v", got, missesBefore+1)
	}
}
