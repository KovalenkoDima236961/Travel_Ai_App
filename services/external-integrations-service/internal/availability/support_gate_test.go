package availability

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/cache"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/providerlimits"
)

// supportSpyProvider records SearchAvailability calls and reports item support so
// tests can prove the cache/quota chain is never entered for unsupported items.
type supportSpyProvider struct {
	supports    bool
	searchCalls int
}

func (s *supportSpyProvider) Name() string { return "spy" }

func (s *supportSpyProvider) SupportsItem(AvailabilityItem) bool { return s.supports }

func (s *supportSpyProvider) SearchAvailability(context.Context, AvailabilitySearchRequest) (*AvailabilitySearchResult, error) {
	s.searchCalls++
	return &AvailabilitySearchResult{
		Status:              StatusAvailable,
		Result:              ProviderResultSuccess,
		Provider:            "spy",
		ProviderDisplayName: "Spy",
		Match:               AvailabilityMatch{Matched: true, Confidence: 0.9},
		Options: []AvailabilityOption{{
			ID:           "spy-1",
			Title:        "Spy option",
			Availability: StatusAvailable,
			PriceType:    PriceTypePerPerson,
			ProviderName: "Spy",
			BookingURL:   "https://example.com/book/spy",
		}},
	}, nil
}

func TestServiceGatesUnsupportedItemBeforeProviderChain(t *testing.T) {
	spy := &supportSpyProvider{supports: false}
	svc := NewService(spy, zap.NewNop(), true)

	result, err := svc.SearchAvailability(context.Background(), availabilityInput("Rome", "Lunch break", "rest"))
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	// The whole decorator chain (cache + quota guard + provider) is skipped, so no
	// quota can be consumed for an unsupported item type.
	if spy.searchCalls != 0 {
		t.Fatalf("expected provider chain to be skipped for unsupported item, got %d calls", spy.searchCalls)
	}
	if result.Status != StatusUnknown {
		t.Fatalf("expected unknown status, got %q", result.Status)
	}
	if !containsWarning(result.Warnings, "does not support this item type") {
		t.Fatalf("expected unsupported warning, got %v", result.Warnings)
	}
}

func TestServiceRunsProviderForSupportedItem(t *testing.T) {
	spy := &supportSpyProvider{supports: true}
	svc := NewService(spy, zap.NewNop(), true)

	if _, err := svc.SearchAvailability(context.Background(), availabilityInput("Vienna", "Live concert", "concert")); err != nil {
		t.Fatalf("search: %v", err)
	}
	if spy.searchCalls != 1 {
		t.Fatalf("expected provider to run once for supported item, got %d calls", spy.searchCalls)
	}
}

func TestSupportsItemForwardsThroughDecorators(t *testing.T) {
	spy := &supportSpyProvider{supports: false}
	guard := providerlimits.NewGuard(providerlimits.GuardParams{Enabled: false, Logger: zap.NewNop()})

	var provider AvailabilityProvider = newFallbackProvider("spy", spy, NewMockAvailabilityProvider(), zap.NewNop())
	provider = newGuardedProvider(guard, "spy", provider, NewMockAvailabilityProvider(), true, zap.NewNop())
	provider = newCachingProvider("spy", provider, cache.New(10), time.Minute, zap.NewNop())

	restItem := AvailabilityItem{Name: "Lunch break", Type: "rest"}
	if providerSupportsItem(provider, restItem) {
		t.Fatal("expected composed chain to report the item as unsupported")
	}

	spy.supports = true
	concertItem := AvailabilityItem{Name: "Live concert", Type: "concert"}
	if !providerSupportsItem(provider, concertItem) {
		t.Fatal("expected composed chain to report the item as supported")
	}
}
