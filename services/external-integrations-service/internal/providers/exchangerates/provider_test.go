package exchangerates

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/config"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/cache"
)

func TestMockExchangeRateProviderConvertsBothDirections(t *testing.T) {
	provider := NewMockExchangeRateProvider()

	jpy, err := provider.Convert(context.Background(), 10, "EUR", "JPY")
	if err != nil {
		t.Fatalf("convert EUR->JPY: %v", err)
	}
	if jpy.ConvertedAmount != 1705 {
		t.Fatalf("expected 1705 JPY, got %v", jpy.ConvertedAmount)
	}

	eur, err := provider.Convert(context.Background(), 2500, "JPY", "EUR")
	if err != nil {
		t.Fatalf("convert JPY->EUR: %v", err)
	}
	if eur.ConvertedAmount != 14.66 {
		t.Fatalf("expected 14.66 EUR, got %v", eur.ConvertedAmount)
	}
}

func TestMockExchangeRateProviderIdentityAndUnsupportedCurrency(t *testing.T) {
	provider := NewMockExchangeRateProvider()

	identity, err := provider.Convert(context.Background(), 25, "EUR", "EUR")
	if err != nil {
		t.Fatalf("identity conversion: %v", err)
	}
	if identity.Provider != "identity" || identity.ConvertedAmount != 25 || identity.Rate != 1 {
		t.Fatalf("unexpected identity result: %+v", identity)
	}

	if _, err := provider.Latest(context.Background(), "XXX"); !errors.Is(err, ErrUnsupportedCurrency) {
		t.Fatalf("expected unsupported currency, got %v", err)
	}
}

func TestExchangeRateFallbackToMock(t *testing.T) {
	cfg := &config.Config{
		ExchangeRateProvider: config.ExchangeRateProviderConfig{
			Provider:       config.ExchangeRateProviderHost,
			FallbackToMock: true,
		},
	}
	provider, err := New(cfg, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	result, err := provider.Convert(context.Background(), 2500, "JPY", "EUR")
	if err != nil {
		t.Fatalf("convert with fallback: %v", err)
	}
	if result.Provider != "mock" || !result.FallbackUsed {
		t.Fatalf("expected mock fallback result, got %+v", result)
	}
}

func TestExchangeRateFallbackDisabledReturnsProviderUnavailable(t *testing.T) {
	cfg := &config.Config{
		ExchangeRateProvider: config.ExchangeRateProviderConfig{
			Provider:       config.ExchangeRateProviderHost,
			FallbackToMock: false,
		},
	}
	provider, err := New(cfg, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}

	_, err = provider.Latest(context.Background(), "EUR")
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) || providerErr.Kind != providerErrorUnavailable {
		t.Fatalf("expected unavailable provider error, got %v", err)
	}
}

func TestCachingExchangeRateProviderHitAvoidsSecondProviderCall(t *testing.T) {
	counting := &countingProvider{next: NewMockExchangeRateProvider()}
	cached := newCachingExchangeRateProvider("mock", counting, cache.New(10), time.Hour, zap.NewNop())

	for i := 0; i < 2; i++ {
		if _, err := cached.Convert(context.Background(), 2500, "JPY", "EUR"); err != nil {
			t.Fatalf("convert %d: %v", i, err)
		}
	}
	if counting.latestCalls != 1 {
		t.Fatalf("expected one latest call after cache hit, got %d", counting.latestCalls)
	}
}

func TestCachingExchangeRateProviderExpires(t *testing.T) {
	counting := &countingProvider{next: NewMockExchangeRateProvider()}
	cached := newCachingExchangeRateProvider("mock", counting, cache.New(10), time.Nanosecond, zap.NewNop())

	if _, err := cached.Latest(context.Background(), "EUR"); err != nil {
		t.Fatalf("first latest: %v", err)
	}
	time.Sleep(time.Millisecond)
	if _, err := cached.Latest(context.Background(), "EUR"); err != nil {
		t.Fatalf("second latest: %v", err)
	}
	if counting.latestCalls != 2 {
		t.Fatalf("expected cache expiry to call provider twice, got %d", counting.latestCalls)
	}
}

func TestDisabledCacheCallsProviderEachTime(t *testing.T) {
	counting := &countingProvider{next: NewMockExchangeRateProvider()}
	var provider service.ExchangeRateProvider = counting
	for i := 0; i < 2; i++ {
		if _, err := provider.Convert(context.Background(), 2500, "JPY", "EUR"); err != nil {
			t.Fatalf("convert %d: %v", i, err)
		}
	}
	if counting.convertCalls != 2 {
		t.Fatalf("expected disabled cache to call provider twice, got %d", counting.convertCalls)
	}
}

type countingProvider struct {
	next         service.ExchangeRateProvider
	latestCalls  int
	convertCalls int
}

func (p *countingProvider) Latest(ctx context.Context, base string) (*entity.ExchangeRateTable, error) {
	p.latestCalls++
	return p.next.Latest(ctx, base)
}

func (p *countingProvider) Convert(ctx context.Context, amount float64, from string, to string) (*entity.CurrencyConversionResult, error) {
	p.convertCalls++
	return p.next.Convert(ctx, amount, from, to)
}
