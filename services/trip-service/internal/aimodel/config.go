package aimodel

import (
	"fmt"
	"strings"
	"time"
)

type Config struct {
	ModelServingEnabled            bool
	DefaultDeploymentKey           string
	ShadowEnabled                  bool
	ShadowSamplePercent            float64
	ShadowMaxConcurrent            int
	ShadowQueueName                string
	ShadowTimeout                  time.Duration
	ShadowMaxQueueAge              time.Duration
	ShadowSkipWhenQueueDepthAbove  int
	ShadowFailOpen                 bool
	CandidateRolloutPercent        float64
	CandidateInternalOnly          bool
	CandidateFallbackToBaseline    bool
	DeploymentAssignmentSalt       string
	ComparisonRetentionDays        int
	AssignmentRetentionDays        int
	ShadowInputRetention           time.Duration
	InternalRolloutEnabled         bool
	AllowlistRolloutEnabled        bool
	PercentageRolloutEnabled       bool
	UserOptInEnabled               bool
	AutomaticGuardrailPauseEnabled bool
	Guardrails                     GuardrailConfig
}

func DefaultConfig() Config {
	return Config{
		ModelServingEnabled:            true,
		DefaultDeploymentKey:           "grounded-baseline",
		ShadowEnabled:                  false,
		ShadowSamplePercent:            0,
		ShadowMaxConcurrent:            1,
		ShadowQueueName:                "ai.shadow.evaluations",
		ShadowTimeout:                  180 * time.Second,
		ShadowMaxQueueAge:              900 * time.Second,
		ShadowSkipWhenQueueDepthAbove:  100,
		ShadowFailOpen:                 true,
		CandidateRolloutPercent:        0,
		CandidateInternalOnly:          true,
		CandidateFallbackToBaseline:    true,
		ComparisonRetentionDays:        90,
		AssignmentRetentionDays:        90,
		ShadowInputRetention:           48 * time.Hour,
		InternalRolloutEnabled:         false,
		AllowlistRolloutEnabled:        false,
		PercentageRolloutEnabled:       false,
		UserOptInEnabled:               false,
		AutomaticGuardrailPauseEnabled: false,
		Guardrails:                     DefaultGuardrailConfig(),
	}
}

func NormalizeConfig(cfg Config) Config {
	defaults := DefaultConfig()
	if strings.TrimSpace(cfg.DefaultDeploymentKey) == "" {
		cfg.DefaultDeploymentKey = defaults.DefaultDeploymentKey
	}
	if cfg.ShadowMaxConcurrent <= 0 {
		cfg.ShadowMaxConcurrent = defaults.ShadowMaxConcurrent
	}
	if strings.TrimSpace(cfg.ShadowQueueName) == "" {
		cfg.ShadowQueueName = defaults.ShadowQueueName
	}
	if cfg.ShadowTimeout <= 0 {
		cfg.ShadowTimeout = defaults.ShadowTimeout
	}
	if cfg.ShadowMaxQueueAge <= 0 {
		cfg.ShadowMaxQueueAge = defaults.ShadowMaxQueueAge
	}
	if cfg.ShadowSkipWhenQueueDepthAbove <= 0 {
		cfg.ShadowSkipWhenQueueDepthAbove = defaults.ShadowSkipWhenQueueDepthAbove
	}
	if cfg.ComparisonRetentionDays <= 0 {
		cfg.ComparisonRetentionDays = defaults.ComparisonRetentionDays
	}
	if cfg.AssignmentRetentionDays <= 0 {
		cfg.AssignmentRetentionDays = defaults.AssignmentRetentionDays
	}
	if cfg.ShadowInputRetention <= 0 {
		cfg.ShadowInputRetention = defaults.ShadowInputRetention
	}
	cfg.ShadowSamplePercent = clampPercent(cfg.ShadowSamplePercent)
	cfg.CandidateRolloutPercent = clampPercent(cfg.CandidateRolloutPercent)
	cfg.Guardrails = NormalizeGuardrailConfig(cfg.Guardrails)
	return cfg
}

func ValidateConfig(cfg Config, strict bool) error {
	cfg = NormalizeConfig(cfg)
	if !cfg.ModelServingEnabled {
		return nil
	}
	if strings.TrimSpace(cfg.DefaultDeploymentKey) == "" {
		return fmt.Errorf("AI_DEFAULT_DEPLOYMENT_KEY is required")
	}
	if cfg.ShadowSamplePercent < 0 || cfg.ShadowSamplePercent > 100 {
		return fmt.Errorf("AI_SHADOW_SAMPLE_PERCENT must be between 0 and 100")
	}
	if cfg.CandidateRolloutPercent < 0 || cfg.CandidateRolloutPercent > 100 {
		return fmt.Errorf("AI_CANDIDATE_ROLLOUT_PERCENT must be between 0 and 100")
	}
	if strict && (cfg.ShadowEnabled || cfg.InternalRolloutEnabled || cfg.AllowlistRolloutEnabled || cfg.PercentageRolloutEnabled) {
		if strings.TrimSpace(cfg.DeploymentAssignmentSalt) == "" {
			return fmt.Errorf("AI_DEPLOYMENT_ASSIGNMENT_SALT is required when candidate rollout is enabled")
		}
	}
	return ValidateGuardrailConfig(cfg.Guardrails)
}

func clampPercent(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}
