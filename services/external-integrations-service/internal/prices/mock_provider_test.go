package prices

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/infrastructure/cache"
)

func TestMockProviderReturnsDeterministicMuseumPrice(t *testing.T) {
	provider := NewMockPriceProvider()
	input := priceInput("Rome", "EUR", "Museum of Rome", "museum")

	first, err := provider.EstimatePrice(context.Background(), input)
	if err != nil {
		t.Fatalf("first estimate: %v", err)
	}
	second, err := provider.EstimatePrice(context.Background(), input)
	if err != nil {
		t.Fatalf("second estimate: %v", err)
	}
	if !first.Matched || first.EstimatedCost == nil {
		t.Fatalf("expected matched cost, got %+v", first)
	}
	if *first.EstimatedCost.Amount != *second.EstimatedCost.Amount {
		t.Fatalf("expected deterministic amount, got %v and %v", *first.EstimatedCost.Amount, *second.EstimatedCost.Amount)
	}
	if first.EstimatedCost.Source != "provider" || first.EstimatedCost.Category != "ticket" {
		t.Fatalf("unexpected cost shape: %+v", first.EstimatedCost)
	}
}

func TestMockProviderReturnsLandmarkPrice(t *testing.T) {
	result, err := NewMockPriceProvider().EstimatePrice(
		context.Background(),
		priceInput("Rome", "USD", "Colosseum", "landmark"),
	)
	if err != nil {
		t.Fatalf("estimate landmark: %v", err)
	}
	if !result.Matched || result.EstimatedCost == nil {
		t.Fatalf("expected matched landmark price, got %+v", result)
	}
	if result.EstimatedCost.Currency != "USD" {
		t.Fatalf("expected USD, got %q", result.EstimatedCost.Currency)
	}
}

func TestMockProviderReturnsNoMatchForPark(t *testing.T) {
	result, err := NewMockPriceProvider().EstimatePrice(
		context.Background(),
		priceInput("Paris", "EUR", "Luxembourg Gardens", "park"),
	)
	if err != nil {
		t.Fatalf("estimate park: %v", err)
	}
	if result.Matched || result.EstimatedCost != nil {
		t.Fatalf("expected no_match for park, got %+v", result)
	}
}

func TestMockProviderRejectsUnsupportedCurrency(t *testing.T) {
	_, err := NewMockPriceProvider().EstimatePrice(
		context.Background(),
		priceInput("Rome", "XXX", "Colosseum", "landmark"),
	)
	if err == nil {
		t.Fatal("expected unsupported currency error")
	}
}

func TestPriceCacheHitAvoidsSecondProviderCall(t *testing.T) {
	fake := &countingProvider{result: &PriceEstimateResult{
		EstimatedCost:   &EstimatedCost{Amount: floatPtr(18), Currency: "EUR", Category: "ticket", Confidence: "medium", Source: "provider"},
		Provider:        "fake",
		PriceType:       stringPtr("ticket"),
		Matched:         true,
		MatchConfidence: 0.8,
	}}
	provider := newCachingProvider("mock", fake, cache.New(10), time.Minute, zap.NewNop())
	input := priceInput("Rome", "EUR", "Colosseum", "landmark")

	if _, err := provider.EstimatePrice(context.Background(), input); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := provider.EstimatePrice(context.Background(), input); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if fake.calls != 1 {
		t.Fatalf("expected one provider call, got %d", fake.calls)
	}
}

func TestPriceCacheExpires(t *testing.T) {
	fake := &countingProvider{result: noMatch("none", 0.2)}
	provider := newCachingProvider("mock", fake, cache.New(10), time.Millisecond, zap.NewNop())
	input := priceInput("Rome", "EUR", "Unknown", "attraction")

	if _, err := provider.EstimatePrice(context.Background(), input); err != nil {
		t.Fatalf("first call: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	if _, err := provider.EstimatePrice(context.Background(), input); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if fake.calls != 2 {
		t.Fatalf("expected cache expiry to call provider twice, got %d", fake.calls)
	}
}

func TestPriceCacheDoesNotStoreFallbackResults(t *testing.T) {
	fake := &countingProvider{result: &PriceEstimateResult{
		EstimatedCost:   &EstimatedCost{Amount: floatPtr(18), Currency: "EUR", Category: "ticket", Confidence: "medium", Source: "provider"},
		Provider:        "mock",
		FallbackUsed:    true,
		PriceType:       stringPtr("ticket"),
		Matched:         true,
		MatchConfidence: 0.8,
	}}
	provider := newCachingProvider("api", fake, cache.New(10), time.Minute, zap.NewNop())
	input := priceInput("Rome", "EUR", "Colosseum", "landmark")

	if _, err := provider.EstimatePrice(context.Background(), input); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := provider.EstimatePrice(context.Background(), input); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if fake.calls != 2 {
		t.Fatalf("expected fallback result not to be cached, got %d provider call(s)", fake.calls)
	}
}

type countingProvider struct {
	result *PriceEstimateResult
	err    error
	calls  int
}

func (p *countingProvider) EstimatePrice(context.Context, PriceEstimateInput) (*PriceEstimateResult, error) {
	p.calls++
	if p.err != nil {
		return nil, p.err
	}
	out := copyResult(*p.result)
	return &out, nil
}

func priceInput(destination, currency, name, category string) PriceEstimateInput {
	return PriceEstimateInput{
		Destination: destination,
		Currency:    currency,
		Place: &PricePlace{
			Name:     name,
			Category: category,
		},
		ItemContext: &PriceItemContext{Name: name, Type: category},
	}
}

func floatPtr(value float64) *float64 {
	return &value
}
