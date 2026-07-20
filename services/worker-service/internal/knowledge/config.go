package knowledge

import (
	"os"
	"strconv"
	"strings"
)

// Knowledge provider configuration is read from the environment rather than the
// worker config file so it can be toggled per deployment without a release.
// Every default is the safe one: mock provider, no raw payload retention,
// fallback enabled, so a misconfigured environment degrades to deterministic
// synthetic data instead of unexpected provider traffic.
const (
	EnvProvider         = "KNOWLEDGE_PROVIDER"
	EnvFallbackToMock   = "KNOWLEDGE_PROVIDER_FALLBACK_TO_MOCK"
	EnvTimeoutSeconds   = "KNOWLEDGE_PROVIDER_TIMEOUT_SECONDS"
	EnvMaxResults       = "KNOWLEDGE_PROVIDER_MAX_RESULTS_PER_DESTINATION"
	EnvRefreshEnabled   = "KNOWLEDGE_PROVIDER_REFRESH_ENABLED"
	EnvStaleAfterDays   = "KNOWLEDGE_REFRESH_STALE_AFTER_DAYS"
	EnvRefreshBatchSize = "KNOWLEDGE_REFRESH_BATCH_SIZE"
	EnvAllowRawPayload  = "KNOWLEDGE_PROVIDER_STORE_RAW_PAYLOAD"
	EnvStrongMinQuality = "KNOWLEDGE_AI_STRONG_MIN_QUALITY"
	EnvWeakMinQuality   = "KNOWLEDGE_AI_WEAK_MIN_QUALITY"
	EnvNeedsReviewBelow = "KNOWLEDGE_NEEDS_REVIEW_BELOW_QUALITY"
	EnvRejectBelow      = "KNOWLEDGE_REJECT_BELOW_QUALITY"
	// EnvTimeoutSeconds is consumed by real adapters in External Integrations
	// Service; it is declared here so the contract lives in one place.
)

// ProviderConfigFromEnv reads the KNOWLEDGE_* contract, falling back to the
// documented defaults for anything unset or unparseable.
func ProviderConfigFromEnv() ProviderConfig {
	cfg := DefaultProviderConfig()
	if value := strings.TrimSpace(os.Getenv(EnvProvider)); value != "" {
		cfg.Provider = value
	}
	cfg.FallbackToMock = envBool(EnvFallbackToMock, cfg.FallbackToMock)
	cfg.RefreshEnabled = envBool(EnvRefreshEnabled, cfg.RefreshEnabled)
	cfg.AllowRawPayload = envBool(EnvAllowRawPayload, cfg.AllowRawPayload)
	cfg.MaxResults = envInt(EnvMaxResults, cfg.MaxResults)
	cfg.StaleAfterDays = envInt(EnvStaleAfterDays, cfg.StaleAfterDays)
	cfg.RefreshBatchSize = envInt(EnvRefreshBatchSize, cfg.RefreshBatchSize)
	cfg.StrongMinQuality = envFloat(EnvStrongMinQuality, cfg.StrongMinQuality)
	cfg.WeakMinQuality = envFloat(EnvWeakMinQuality, cfg.WeakMinQuality)
	cfg.NeedsReviewBelow = envFloat(EnvNeedsReviewBelow, cfg.NeedsReviewBelow)
	cfg.RejectBelowQualty = envFloat(EnvRejectBelow, cfg.RejectBelowQualty)
	return cfg
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

// envFloat ignores out-of-range values: a quality threshold outside 0..1 is a
// configuration mistake and must not silently disable the quality gate.
func envFloat(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < 0 || parsed > 1 {
		return fallback
	}
	return parsed
}
