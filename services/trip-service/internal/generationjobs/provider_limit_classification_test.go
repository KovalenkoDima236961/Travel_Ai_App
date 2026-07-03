package generationjobs

import (
	"errors"
	"fmt"
	"testing"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/providerlimit"
)

func TestClassifyProviderRateLimitedIsTransient(t *testing.T) {
	err := &providerlimit.Error{Code: providerlimit.CodeRateLimited, Operation: "route_estimate"}
	code, message := ClassifyJobError(err)
	if code != ErrorProviderRateLimited {
		t.Fatalf("expected %s, got %s", ErrorProviderRateLimited, code)
	}
	if !IsRetryableErrorCode(code) {
		t.Fatal("provider_rate_limited must be retryable (transient)")
	}
	if message == "" {
		t.Fatal("expected a safe message")
	}
}

func TestClassifyProviderQuotaExceededIsTerminal(t *testing.T) {
	err := &providerlimit.Error{Code: providerlimit.CodeQuotaExceeded, Operation: "weather_forecast"}
	code, _ := ClassifyJobError(err)
	if code != ErrorProviderQuotaExceeded {
		t.Fatalf("expected %s, got %s", ErrorProviderQuotaExceeded, code)
	}
	if IsRetryableErrorCode(code) {
		t.Fatal("provider_quota_exceeded must be terminal so the worker does not tight-loop")
	}
}

func TestClassifyProviderLimitsUnavailableIsTransient(t *testing.T) {
	err := &providerlimit.Error{Code: providerlimit.CodeLimitsUnavailable, Operation: "place_search"}
	code, _ := ClassifyJobError(err)
	if code != ErrorProviderLimitsUnavailable {
		t.Fatalf("expected %s, got %s", ErrorProviderLimitsUnavailable, code)
	}
	if !IsRetryableErrorCode(code) {
		t.Fatal("provider_limits_unavailable must be retryable (transient)")
	}
}

func TestClassifyProviderLimitThroughWrappedError(t *testing.T) {
	// The enrichment layers wrap client errors with %w; classification must still
	// see the typed limit error through the wrapper.
	wrapped := fmt.Errorf("place search failed: %w", &providerlimit.Error{Code: providerlimit.CodeRateLimited})
	code, _ := ClassifyJobError(wrapped)
	if code != ErrorProviderRateLimited {
		t.Fatalf("expected wrapped limit error to classify as %s, got %s", ErrorProviderRateLimited, code)
	}
}

func TestClassifyNonLimitErrorUnaffected(t *testing.T) {
	code, _ := ClassifyJobError(errors.New("some other failure"))
	if code != ErrorUnknown {
		t.Fatalf("expected unknown_error for a generic error, got %s", code)
	}
}

func TestProviderLimitParse(t *testing.T) {
	body := []byte(`{"error":"provider_quota_exceeded","message":"quota reached","provider":"ors","operation":"route_estimate","retryAfterSeconds":120}`)
	limitErr := providerlimit.Parse(429, body)
	if limitErr == nil {
		t.Fatal("expected a parsed limit error")
	}
	if limitErr.Code != providerlimit.CodeQuotaExceeded || limitErr.Provider != "ors" || limitErr.RetryAfterSeconds != 120 {
		t.Fatalf("unexpected parse result: %+v", limitErr)
	}
	if providerlimit.Parse(500, []byte(`{"error":"weather_provider_unavailable"}`)) != nil {
		t.Fatal("non-limit error codes must not parse as provider-limit errors")
	}
}
