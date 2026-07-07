package exchangerates

import (
	"context"
	"math"
	"strings"
	"time"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

const mockProviderName = "mock"

var mockAsOf = time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC)

// Rates are quoted from EUR into the key currency. They are deliberately stable
// so local tests and smoke tests stay deterministic.
var mockEURRates = map[string]float64{
	"EUR": 1,
	"USD": 1.08,
	"GBP": 0.86,
	"JPY": 170.5,
	"CZK": 24.7,
	"PLN": 4.3,
	"HUF": 390,
	"CHF": 0.96,
	"CAD": 1.47,
	"AUD": 1.63,
}

type MockExchangeRateProvider struct{}

func NewMockExchangeRateProvider() *MockExchangeRateProvider {
	return &MockExchangeRateProvider{}
}

func (p *MockExchangeRateProvider) Latest(_ context.Context, base string) (*entity.ExchangeRateTable, error) {
	base = normalizeCurrency(base)
	baseRate, ok := mockEURRates[base]
	if !ok {
		return nil, ErrUnsupportedCurrency
	}

	rates := make(map[string]float64, len(mockEURRates))
	for currency, rate := range mockEURRates {
		rates[currency] = round6(rate / baseRate)
	}
	return &entity.ExchangeRateTable{
		Provider: mockProviderName,
		Base:     base,
		Rates:    rates,
		AsOf:     mockAsOf,
	}, nil
}

func (p *MockExchangeRateProvider) Convert(ctx context.Context, amount float64, from string, to string) (*entity.CurrencyConversionResult, error) {
	from = normalizeCurrency(from)
	to = normalizeCurrency(to)
	if from == to {
		return identityConversion(amount, from, to), nil
	}
	table, err := p.Latest(ctx, from)
	if err != nil {
		return nil, err
	}
	rate, ok := table.Rates[to]
	if !ok {
		return nil, ErrUnsupportedCurrency
	}
	return &entity.CurrencyConversionResult{
		Provider:        mockProviderName,
		From:            from,
		To:              to,
		Amount:          amount,
		ConvertedAmount: round2(amount * rate),
		Rate:            rate,
		AsOf:            table.AsOf,
	}, nil
}

func identityConversion(amount float64, from string, to string) *entity.CurrencyConversionResult {
	return &entity.CurrencyConversionResult{
		Provider:        "identity",
		From:            normalizeCurrency(from),
		To:              normalizeCurrency(to),
		Amount:          amount,
		ConvertedAmount: round2(amount),
		Rate:            1,
		AsOf:            time.Now().UTC(),
	}
}

func normalizeCurrency(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}

func round6(value float64) float64 {
	return math.Round(value*1_000_000) / 1_000_000
}
