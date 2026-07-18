package featureflags

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

var ErrUnknownFlag = errors.New("unknown feature flag")

type Config struct {
	Enabled         bool `yaml:"enabled" env:"FEATURE_FLAGS_ENABLED" env-default:"true"`
	CacheTTLSeconds int  `yaml:"cache_ttl_seconds" env:"FEATURE_FLAGS_CACHE_TTL_SECONDS" env-default:"30" validate:"min=1,max=300"`
	FailClosed      bool `yaml:"fail_closed" env:"FEATURE_FLAGS_FAIL_CLOSED" env-default:"false"`
}

type EvaluationContext struct {
	UserID      *uuid.UUID
	WorkspaceID *uuid.UUID
	Environment string
	ServiceName string
	RequestID   string
}

type Metadata struct {
	Source      string     `json:"source"`
	Scope       string     `json:"scope"`
	Reason      string     `json:"reason,omitempty"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"`
	DefaultUsed bool       `json:"defaultUsed"`
}

type ResolvedFlag struct {
	Key                        string    `json:"key"`
	Value                      bool      `json:"value"`
	Type                       ValueType `json:"type"`
	Category                   string    `json:"category"`
	Description                string    `json:"description"`
	SafeForFrontend            bool      `json:"safeForFrontend"`
	RequiresBackendEnforcement bool      `json:"requiresBackendEnforcement"`
	Metadata                   Metadata  `json:"metadata"`
}

type Override struct {
	ID          uuid.UUID
	Key         string
	ValueType   ValueType
	BoolValue   *bool
	Environment *string
	ScopeType   string
	ScopeID     *string
	Description *string
	Enabled     bool
	Source      string
	CreatedBy   *uuid.UUID
	UpdatedBy   *uuid.UUID
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type AuditEvent struct {
	ID          uuid.UUID      `json:"id"`
	FlagKey     string         `json:"flagKey"`
	Environment *string        `json:"environment,omitempty"`
	ScopeType   string         `json:"scopeType"`
	ScopeID     *string        `json:"scopeId,omitempty"`
	ActorUserID *uuid.UUID     `json:"actorUserId,omitempty"`
	Action      string         `json:"action"`
	OldValue    map[string]any `json:"oldValue,omitempty"`
	NewValue    map[string]any `json:"newValue,omitempty"`
	Reason      string         `json:"reason,omitempty"`
	RequestID   string         `json:"requestId,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
}

type Repository interface {
	GetGlobalOverride(ctx context.Context, key, environment string) (*Override, error)
	SaveGlobalOverride(ctx context.Context, override Override, audit AuditEvent) (*Override, error)
	DeleteGlobalOverride(ctx context.Context, key, environment string, audit AuditEvent) error
	ListAudit(ctx context.Context, key string, limit int) ([]AuditEvent, error)
}

type cacheEntry struct {
	flag      ResolvedFlag
	expiresAt time.Time
}

type Service struct {
	repo        Repository
	config      Config
	environment string
	log         *zap.Logger
	cacheMu     sync.RWMutex
	cache       map[string]cacheEntry
	now         func() time.Time
}

func New(repo Repository, cfg Config, environment string, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	if cfg.CacheTTLSeconds <= 0 {
		cfg.CacheTTLSeconds = 30
	}
	return &Service{repo: repo, config: cfg, environment: strings.ToLower(strings.TrimSpace(environment)), log: log, cache: make(map[string]cacheEntry), now: time.Now}
}

func (s *Service) IsEnabled(ctx context.Context, key string, eval EvaluationContext) (bool, Metadata, error) {
	resolved, err := s.GetFlag(ctx, key, eval)
	return resolved.Value, resolved.Metadata, err
}

func (s *Service) GetFlag(ctx context.Context, key string, eval EvaluationContext) (ResolvedFlag, error) {
	definition, ok := DefinitionFor(key)
	if !ok {
		return ResolvedFlag{}, fmt.Errorf("%w: %s", ErrUnknownFlag, key)
	}
	environment := s.resolveEnvironment(eval.Environment)
	cacheKey := definition.Key + ":" + environment
	if cached, ok := s.getCached(cacheKey); ok {
		featureFlagCacheHits.Inc()
		return cached, nil
	}
	featureFlagCacheMisses.Inc()

	value, source, defaultErr := EnvironmentDefault(definition, environment)
	resolved := ResolvedFlag{
		Key: definition.Key, Value: value, Type: definition.Type, Category: definition.Category,
		Description: definition.Description, SafeForFrontend: definition.SafeForFrontend,
		RequiresBackendEnforcement: definition.RequiresBackendEnforcement,
		Metadata:                   Metadata{Source: source, Scope: "global", DefaultUsed: source == "default"},
	}
	if defaultErr != nil {
		s.log.Warn("invalid feature flag environment default", zap.String("flag", definition.Key), zap.Error(defaultErr))
	}

	if !s.config.Enabled || s.repo == nil {
		s.storeCached(cacheKey, resolved)
		featureFlagEvaluations.WithLabelValues(definition.Key, resolved.Metadata.Source, boolLabel(resolved.Value)).Inc()
		return resolved, nil
	}

	override, err := s.repo.GetGlobalOverride(ctx, definition.Key, environment)
	if err != nil {
		s.log.Error("feature flag lookup failed", zap.String("flag", definition.Key), zap.String("environment", environment), zap.String("requestId", eval.RequestID), zap.Error(err))
		if s.failClosed() && definition.RequiresBackendEnforcement {
			resolved.Value = false
			resolved.Metadata = Metadata{Source: "fail_closed", Scope: "global", Reason: "runtime lookup failed"}
		}
		featureFlagEvaluations.WithLabelValues(definition.Key, resolved.Metadata.Source, boolLabel(resolved.Value)).Inc()
		return resolved, err
	}
	if override != nil && override.Enabled && override.BoolValue != nil {
		resolved.Value = *override.BoolValue
		resolved.Metadata = Metadata{Source: "db", Scope: override.ScopeType, UpdatedAt: &override.UpdatedAt}
	}
	s.storeCached(cacheKey, resolved)
	featureFlagEvaluations.WithLabelValues(definition.Key, resolved.Metadata.Source, boolLabel(resolved.Value)).Inc()
	return resolved, nil
}

func (s *Service) ListFlags(ctx context.Context, environment string, frontendOnly bool) ([]ResolvedFlag, error) {
	flags := make([]ResolvedFlag, 0, len(registry))
	for _, definition := range Definitions() {
		if frontendOnly && !definition.SafeForFrontend {
			continue
		}
		flag, err := s.GetFlag(ctx, definition.Key, EvaluationContext{Environment: environment})
		if err != nil {
			// Return the deterministic resolved fallback. In production it is
			// fail-closed for enforceable flags; local development remains usable
			// when a database has not been started yet.
			flags = append(flags, flag)
			continue
		}
		flags = append(flags, flag)
	}
	return flags, nil
}

func (s *Service) UpdateGlobal(ctx context.Context, key, environment string, value bool, reason, requestID string, actor *uuid.UUID) (ResolvedFlag, error) {
	definition, ok := DefinitionFor(key)
	if !ok {
		return ResolvedFlag{}, fmt.Errorf("%w: %s", ErrUnknownFlag, key)
	}
	if definition.Type != ValueTypeBoolean {
		return ResolvedFlag{}, fmt.Errorf("%s is not a boolean feature flag", key)
	}
	environment = s.resolveEnvironment(environment)
	if s.requiresReason() && strings.TrimSpace(reason) == "" {
		return ResolvedFlag{}, errors.New("reason is required outside local development")
	}
	old, err := s.repo.GetGlobalOverride(ctx, definition.Key, environment)
	if err != nil {
		return ResolvedFlag{}, fmt.Errorf("get current feature flag: %w", err)
	}
	now := s.now().UTC()
	override := Override{Key: definition.Key, ValueType: definition.Type, BoolValue: &value, Enabled: true, Source: "db", ScopeType: "global", UpdatedBy: actor, UpdatedAt: now}
	override.Environment = &environment
	sameEnvironment := old != nil && old.Environment != nil && *old.Environment == environment
	if !sameEnvironment {
		override.ID, override.CreatedBy, override.CreatedAt = uuid.New(), actor, now
	} else {
		override.ID, override.CreatedBy, override.CreatedAt = old.ID, old.CreatedBy, old.CreatedAt
	}
	audit := AuditEvent{ID: uuid.New(), FlagKey: definition.Key, Environment: &environment, ScopeType: "global", ActorUserID: actor, Reason: strings.TrimSpace(reason), RequestID: requestID, CreatedAt: now}
	if !sameEnvironment {
		audit.Action = "created"
	} else if old.BoolValue != nil && *old.BoolValue == value {
		audit.Action = "updated"
	} else if value {
		audit.Action = "enabled"
	} else {
		audit.Action = "disabled"
	}
	audit.OldValue = overrideValue(old)
	audit.NewValue = overrideValue(&override)
	if _, err := s.repo.SaveGlobalOverride(ctx, override, audit); err != nil {
		return ResolvedFlag{}, fmt.Errorf("save feature flag: %w", err)
	}
	s.Invalidate(definition.Key)
	featureFlagUpdates.WithLabelValues(definition.Key, audit.Action).Inc()
	return s.GetFlag(ctx, definition.Key, EvaluationContext{Environment: environment, RequestID: requestID})
}

func (s *Service) ResetGlobal(ctx context.Context, key, environment, reason, requestID string, actor *uuid.UUID) error {
	definition, ok := DefinitionFor(key)
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnknownFlag, key)
	}
	environment = s.resolveEnvironment(environment)
	if s.requiresReason() && strings.TrimSpace(reason) == "" {
		return errors.New("reason is required outside local development")
	}
	old, err := s.repo.GetGlobalOverride(ctx, definition.Key, environment)
	if err != nil {
		return fmt.Errorf("get current feature flag: %w", err)
	}
	if old == nil {
		return nil
	}
	deleteEnvironment := environment
	if old.Environment == nil {
		deleteEnvironment = ""
	}
	auditEnvironment := old.Environment
	audit := AuditEvent{ID: uuid.New(), FlagKey: definition.Key, Environment: auditEnvironment, ScopeType: "global", ActorUserID: actor, Action: "reset_to_default", OldValue: overrideValue(old), Reason: strings.TrimSpace(reason), RequestID: requestID, CreatedAt: s.now().UTC()}
	if err := s.repo.DeleteGlobalOverride(ctx, definition.Key, deleteEnvironment, audit); err != nil {
		return fmt.Errorf("reset feature flag: %w", err)
	}
	s.Invalidate(definition.Key)
	featureFlagUpdates.WithLabelValues(definition.Key, audit.Action).Inc()
	return nil
}

func (s *Service) ListAudit(ctx context.Context, key string, limit int) ([]AuditEvent, error) {
	if _, ok := DefinitionFor(key); !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownFlag, key)
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	return s.repo.ListAudit(ctx, key, limit)
}

func (s *Service) Invalidate(key string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	for cacheKey := range s.cache {
		if strings.HasPrefix(cacheKey, key+":") {
			delete(s.cache, cacheKey)
		}
	}
}

func (s *Service) Environment() string { return s.resolveEnvironment("") }

func (s *Service) getCached(key string) (ResolvedFlag, bool) {
	s.cacheMu.RLock()
	entry, ok := s.cache[key]
	s.cacheMu.RUnlock()
	return entry.flag, ok && s.now().Before(entry.expiresAt)
}

func (s *Service) storeCached(key string, flag ResolvedFlag) {
	s.cacheMu.Lock()
	s.cache[key] = cacheEntry{flag: flag, expiresAt: s.now().Add(time.Duration(s.config.CacheTTLSeconds) * time.Second)}
	s.cacheMu.Unlock()
}

func (s *Service) resolveEnvironment(value string) string {
	if normalized := strings.ToLower(strings.TrimSpace(value)); normalized != "" {
		return normalized
	}
	if s.environment != "" {
		return s.environment
	}
	return "local"
}

func (s *Service) failClosed() bool {
	return s.config.FailClosed || s.environment == "staging" || s.environment == "production"
}

func (s *Service) requiresReason() bool {
	return s.environment == "staging" || s.environment == "production"
}

func overrideValue(override *Override) map[string]any {
	if override == nil {
		return nil
	}
	value := map[string]any{"enabled": override.Enabled}
	if override.BoolValue != nil {
		value["value"] = *override.BoolValue
	}
	return value
}

func boolLabel(value bool) string {
	if value {
		return "enabled"
	}
	return "disabled"
}
