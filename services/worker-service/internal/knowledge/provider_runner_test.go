package knowledge

import (
	"context"
	"testing"

	tripknowledge "github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge"
	tripprovider "github.com/KovalenkoDima236961/Travel_Ai_App/internal/knowledge/provider"
)

// CI must never reach a real provider. These tests pin the selection rules that
// guarantee that.
func TestSelectProviderDefaultsToMock(t *testing.T) {
	selected, err := SelectProvider(ProviderConfig{})
	if err != nil {
		t.Fatalf("SelectProvider() error = %v", err)
	}
	if selected.ProviderName() != tripprovider.ProviderMock {
		t.Fatalf("an unset provider must default to mock, got %q", selected.ProviderName())
	}
}

func TestSelectProviderFallsBackToMockWhenConfigured(t *testing.T) {
	selected, err := SelectProvider(ProviderConfig{Provider: "foursquare", FallbackToMock: true})
	if err != nil {
		t.Fatalf("SelectProvider() error = %v", err)
	}
	if selected.ProviderName() != tripprovider.ProviderMock {
		t.Fatalf("fallback must yield the mock provider, got %q", selected.ProviderName())
	}
}

// Without fallback, an unconfigured real provider must fail loudly rather than
// silently doing nothing or attempting a network call.
func TestSelectProviderFailsLoudlyWithoutFallback(t *testing.T) {
	if _, err := SelectProvider(ProviderConfig{Provider: "foursquare", FallbackToMock: false}); err == nil {
		t.Fatal("an unconfigured real provider without fallback must error")
	}
	if _, err := SelectProvider(ProviderConfig{Provider: "definitely-not-a-provider"}); err == nil {
		t.Fatal("an unsupported provider name must error")
	}
}

func TestProviderConfigFromEnvUsesSafeDefaults(t *testing.T) {
	cfg := ProviderConfigFromEnv()
	if cfg.Provider != tripprovider.ProviderMock {
		t.Fatalf("default provider must be mock, got %q", cfg.Provider)
	}
	if cfg.AllowRawPayload {
		t.Fatal("raw payload retention must be off by default")
	}
	if !cfg.FallbackToMock {
		t.Fatal("fallback to mock must be on by default")
	}
}

func TestProviderConfigFromEnvReadsOverrides(t *testing.T) {
	t.Setenv(EnvProvider, "wikidata")
	t.Setenv(EnvMaxResults, "25")
	t.Setenv(EnvAllowRawPayload, "true")
	t.Setenv(EnvStrongMinQuality, "0.8")

	cfg := ProviderConfigFromEnv()
	if cfg.Provider != "wikidata" || cfg.MaxResults != 25 || !cfg.AllowRawPayload || cfg.StrongMinQuality != 0.8 {
		t.Fatalf("environment overrides were not applied: %+v", cfg)
	}
}

// A malformed or out-of-range threshold must not disable the quality gate.
func TestProviderConfigFromEnvIgnoresInvalidValues(t *testing.T) {
	t.Setenv(EnvStrongMinQuality, "not-a-number")
	t.Setenv(EnvRejectBelow, "9000")
	t.Setenv(EnvMaxResults, "-5")

	defaults := DefaultProviderConfig()
	cfg := ProviderConfigFromEnv()
	if cfg.StrongMinQuality != defaults.StrongMinQuality {
		t.Fatalf("unparseable threshold must fall back to the default, got %v", cfg.StrongMinQuality)
	}
	if cfg.RejectBelowQualty != defaults.RejectBelowQualty {
		t.Fatalf("out-of-range threshold must fall back to the default, got %v", cfg.RejectBelowQualty)
	}
	if cfg.MaxResults != defaults.MaxResults {
		t.Fatalf("non-positive limit must fall back to the default, got %d", cfg.MaxResults)
	}
}

func TestProviderConfigThresholdsRespectOverrides(t *testing.T) {
	cfg := DefaultProviderConfig()
	cfg.StrongMinQuality = 0.9
	cfg.StaleAfterDays = 7
	thresholds := cfg.thresholds()
	if thresholds.StrongMinQuality != 0.9 || thresholds.StaleAfterDays != 7 {
		t.Fatalf("thresholds did not pick up overrides: %+v", thresholds)
	}
	// Unset values keep the documented defaults.
	if thresholds.WeakMinQuality != tripknowledge.DefaultThresholds().WeakMinQuality {
		t.Fatalf("unset threshold must keep its default, got %v", thresholds.WeakMinQuality)
	}
}

func TestProviderConfigSourcePolicyDefaultsToWithholdingRawPayload(t *testing.T) {
	if DefaultProviderConfig().sourcePolicy().AllowRawPayload {
		t.Fatal("the default source policy must withhold raw payloads")
	}
	if !DefaultProviderConfig().sourcePolicy().RequireLicense {
		t.Fatal("the default source policy must require a license")
	}
}

func TestNewProviderRunnerRequiresStore(t *testing.T) {
	if _, err := NewProviderRunner(nil, DefaultProviderConfig()); err == nil {
		t.Fatal("a runner without a store must error")
	}
}

// An unsupported job type must be rejected rather than silently treated as an
// ingestion run.
func TestRunRejectsUnknownJobType(t *testing.T) {
	runner := &ProviderRunner{cfg: DefaultProviderConfig()}
	if _, err := runner.Run(context.Background(), ProviderRequest{JobType: "not_a_job"}); err == nil {
		t.Fatal("an unknown job type must error")
	}
}

func TestRunRequiresDestinationForDuplicateDetection(t *testing.T) {
	store := &tripknowledge.Store{}
	runner, err := NewProviderRunner(store, DefaultProviderConfig())
	if err != nil {
		t.Fatalf("NewProviderRunner() error = %v", err)
	}
	if _, err := runner.Run(context.Background(), ProviderRequest{
		JobType: tripknowledge.JobDuplicateDetection,
	}); err == nil {
		t.Fatal("duplicate detection without a destination must error")
	}
}

// Disabling refresh must be honoured without touching the store.
func TestRefreshDisabledReturnsWarningInsteadOfRunning(t *testing.T) {
	cfg := DefaultProviderConfig()
	cfg.RefreshEnabled = false
	runner, err := NewProviderRunner(&tripknowledge.Store{}, cfg)
	if err != nil {
		t.Fatalf("NewProviderRunner() error = %v", err)
	}
	result, err := runner.Run(context.Background(), ProviderRequest{
		JobType: tripknowledge.JobRefreshStalePlaces,
	})
	if err != nil {
		t.Fatalf("a disabled refresh must not error: %v", err)
	}
	if len(result.Warnings) == 0 {
		t.Fatal("a disabled refresh must report why it did nothing")
	}
}
