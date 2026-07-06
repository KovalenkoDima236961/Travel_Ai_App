package availability

import (
	"testing"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
)

func TestSelectProviderMockByDefault(t *testing.T) {
	provider, err := selectProvider(config.AvailabilityProviderMock, config.AvailabilityConfig{Provider: config.AvailabilityProviderMock}, zap.NewNop())
	if err != nil {
		t.Fatalf("selectProvider mock: %v", err)
	}
	if provider.Name() != mockProviderName {
		t.Fatalf("expected mock provider, got %q", provider.Name())
	}
}

func TestSelectProviderTicketmasterWhenConfigured(t *testing.T) {
	cfg := config.AvailabilityConfig{
		Provider:            config.AvailabilityProviderTicketmaster,
		FallbackToMock:      true,
		TicketmasterAPIKey:  "test-key",
		TicketmasterBaseURL: "https://app.ticketmaster.com/discovery/v2",
		MinMatchConfidence:  0.55,
		MaxOptions:          10,
	}
	provider, err := selectProvider(config.AvailabilityProviderTicketmaster, cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("selectProvider ticketmaster: %v", err)
	}
	if provider.Name() != ticketmasterProviderName {
		t.Fatalf("expected ticketmaster provider, got %q", provider.Name())
	}
}

func TestSelectProviderTicketmasterMissingKeyFallsBackToMock(t *testing.T) {
	cfg := config.AvailabilityConfig{
		Provider:       config.AvailabilityProviderTicketmaster,
		FallbackToMock: true,
	}
	provider, err := selectProvider(config.AvailabilityProviderTicketmaster, cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("expected fallback, got error: %v", err)
	}
	if provider.Name() != mockProviderName {
		t.Fatalf("expected mock fallback, got %q", provider.Name())
	}
}

func TestSelectProviderTicketmasterMissingKeyNoFallbackErrors(t *testing.T) {
	cfg := config.AvailabilityConfig{
		Provider:       config.AvailabilityProviderTicketmaster,
		FallbackToMock: false,
	}
	if _, err := selectProvider(config.AvailabilityProviderTicketmaster, cfg, zap.NewNop()); err == nil {
		t.Fatal("expected error when ticketmaster key missing and fallback disabled")
	}
}

func TestSelectProviderUnknownErrors(t *testing.T) {
	if _, err := selectProvider("does-not-exist", config.AvailabilityConfig{Provider: "does-not-exist"}, zap.NewNop()); err == nil {
		t.Fatal("expected error for unknown provider")
	}
}
