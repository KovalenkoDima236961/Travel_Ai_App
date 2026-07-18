package featureflags

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type fakeRepository struct {
	override *Override
	err      error
	gets     int
	audits   []AuditEvent
}

func (f *fakeRepository) GetGlobalOverride(context.Context, string, string) (*Override, error) {
	f.gets++
	return f.override, f.err
}
func (f *fakeRepository) SaveGlobalOverride(_ context.Context, override Override, audit AuditEvent) (*Override, error) {
	f.override = &override
	f.audits = append(f.audits, audit)
	return &override, nil
}
func (f *fakeRepository) DeleteGlobalOverride(_ context.Context, _, _ string, audit AuditEvent) error {
	f.override = nil
	f.audits = append(f.audits, audit)
	return nil
}
func (f *fakeRepository) ListAudit(context.Context, string, int) ([]AuditEvent, error) {
	return f.audits, nil
}

func TestGetFlagUsesEnvironmentThenDatabaseOverrideAndCache(t *testing.T) {
	t.Setenv("FEATURE_COPILOT_ENABLED", "false")
	value := true
	updated := time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC)
	repo := &fakeRepository{override: &Override{Key: CopilotEnabled, ValueType: ValueTypeBoolean, BoolValue: &value, Enabled: true, ScopeType: "global", UpdatedAt: updated}}
	service := New(repo, Config{Enabled: true, CacheTTLSeconds: 60}, "local", nil)

	resolved, err := service.GetFlag(context.Background(), CopilotEnabled, EvaluationContext{})
	if err != nil {
		t.Fatalf("GetFlag() error = %v", err)
	}
	if !resolved.Value || resolved.Metadata.Source != "db" {
		t.Fatalf("resolved = %#v, want db true", resolved)
	}
	if repo.gets != 1 {
		t.Fatalf("lookups = %d, want 1", repo.gets)
	}
	if _, err := service.GetFlag(context.Background(), CopilotEnabled, EvaluationContext{}); err != nil {
		t.Fatal(err)
	}
	if repo.gets != 1 {
		t.Fatalf("cache did not prevent second lookup; got %d", repo.gets)
	}
}

func TestGetFlagFailsClosedForEnforcedProductionFlags(t *testing.T) {
	service := New(&fakeRepository{err: errors.New("database unavailable")}, Config{Enabled: true, CacheTTLSeconds: 30}, "production", nil)
	resolved, err := service.GetFlag(context.Background(), AIGenerationEnabled, EvaluationContext{})
	if err == nil {
		t.Fatal("GetFlag() error = nil, want database error")
	}
	if resolved.Value || resolved.Metadata.Source != "fail_closed" {
		t.Fatalf("resolved = %#v, want fail-closed false", resolved)
	}
}

func TestUpdateAndResetCreateAuditableOverride(t *testing.T) {
	repo := &fakeRepository{}
	service := New(repo, Config{Enabled: true, CacheTTLSeconds: 30}, "staging", nil)
	actor := uuid.New()
	if _, err := service.UpdateGlobal(context.Background(), PublicSharingEnabled, "", true, "staging verification", "request-1", &actor); err != nil {
		t.Fatalf("UpdateGlobal() error = %v", err)
	}
	if repo.override == nil || repo.override.BoolValue == nil || !*repo.override.BoolValue {
		t.Fatalf("override = %#v, want true", repo.override)
	}
	if len(repo.audits) != 1 || repo.audits[0].Action != "created" || repo.audits[0].Reason != "staging verification" {
		t.Fatalf("audit = %#v", repo.audits)
	}
	if err := service.ResetGlobal(context.Background(), PublicSharingEnabled, "", "rollback", "request-2", &actor); err != nil {
		t.Fatalf("ResetGlobal() error = %v", err)
	}
	if repo.override != nil {
		t.Fatalf("override = %#v, want nil", repo.override)
	}
	if got := repo.audits[len(repo.audits)-1].Action; got != "reset_to_default" {
		t.Fatalf("reset audit action = %q", got)
	}
}

func TestUpdateRequiresReasonOutsideLocalDevelopment(t *testing.T) {
	service := New(&fakeRepository{}, Config{Enabled: true, CacheTTLSeconds: 30}, "production", nil)
	if _, err := service.UpdateGlobal(context.Background(), CopilotEnabled, "", false, "", "", nil); err == nil {
		t.Fatal("UpdateGlobal() error = nil, want required reason")
	}
}

func TestUpdateDoesNotOverwriteGlobalDefaultOverrideWhenCreatingEnvironmentOverride(t *testing.T) {
	value := false
	globalOverrideID := uuid.New()
	repo := &fakeRepository{override: &Override{
		ID:          globalOverrideID,
		Key:         CopilotEnabled,
		ValueType:   ValueTypeBoolean,
		BoolValue:   &value,
		Environment: nil,
		Enabled:     true,
		ScopeType:   "global",
		CreatedAt:   time.Now().UTC(),
	}}
	service := New(repo, Config{Enabled: true, CacheTTLSeconds: 30}, "production", nil)
	updated, err := service.UpdateGlobal(context.Background(), CopilotEnabled, "production", true, "staging promotion", "request-3", nil)
	if err != nil {
		t.Fatalf("UpdateGlobal() error = %v", err)
	}
	if updated.Metadata.Source != "db" || repo.override.Environment == nil || *repo.override.Environment != "production" {
		t.Fatalf("override = %#v, want a production db override", repo.override)
	}
	if repo.override.ID == uuid.Nil || repo.override.ID == globalOverrideID {
		t.Fatalf("environment override reused global override id: %#v", repo.override)
	}
}

func TestResetDeletesTheOverrideScopeThatWasResolved(t *testing.T) {
	value := true
	repo := &fakeRepository{override: &Override{
		ID:        uuid.New(),
		Key:       CopilotEnabled,
		ValueType: ValueTypeBoolean,
		BoolValue: &value,
		Enabled:   true,
		ScopeType: "global",
	}}
	service := New(repo, Config{Enabled: true, CacheTTLSeconds: 30}, "production", nil)
	if err := service.ResetGlobal(context.Background(), CopilotEnabled, "production", "remove shared override", "request-4", nil); err != nil {
		t.Fatalf("ResetGlobal() error = %v", err)
	}
	if repo.override != nil {
		t.Fatalf("override = %#v, want reset generic override", repo.override)
	}
	if event := repo.audits[len(repo.audits)-1]; event.Environment != nil {
		t.Fatalf("audit environment = %q, want generic environment", *event.Environment)
	}
}
