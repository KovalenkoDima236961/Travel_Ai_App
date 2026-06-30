package exchangerates

import (
	"context"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/application/service"
	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

type fallbackExchangeRateProvider struct {
	providerName         string
	fallbackProviderName string
	primary              service.ExchangeRateProvider
	fallback             service.ExchangeRateProvider
	log                  *zap.Logger
}

func newFallbackExchangeRateProvider(
	providerName string,
	primary service.ExchangeRateProvider,
	fallback service.ExchangeRateProvider,
	log *zap.Logger,
) service.ExchangeRateProvider {
	if log == nil {
		log = zap.NewNop()
	}
	return &fallbackExchangeRateProvider{
		providerName:         providerName,
		fallbackProviderName: mockProviderName,
		primary:              primary,
		fallback:             fallback,
		log:                  log,
	}
}

func (p *fallbackExchangeRateProvider) Latest(ctx context.Context, base string) (*entity.ExchangeRateTable, error) {
	table, err := p.primary.Latest(ctx, base)
	if err == nil {
		return table, nil
	}
	p.logFallback("exchange_rate_latest", err)

	fallbackTable, fallbackErr := p.fallback.Latest(ctx, base)
	if fallbackErr != nil {
		p.log.Warn("exchange rate provider fallback failed",
			zap.String("action", "exchange_rate_latest"),
			zap.String("provider", p.providerName),
			zap.String("fallbackProvider", p.fallbackProviderName),
			zap.String("errorType", providerErrorKind(fallbackErr)),
			zap.Error(fallbackErr),
		)
		return nil, err
	}
	fallbackTable.Provider = p.fallbackProviderName
	fallbackTable.FallbackUsed = true
	return fallbackTable, nil
}

func (p *fallbackExchangeRateProvider) Convert(ctx context.Context, amount float64, from string, to string) (*entity.CurrencyConversionResult, error) {
	result, err := p.primary.Convert(ctx, amount, from, to)
	if err == nil {
		return result, nil
	}
	p.logFallback("exchange_rate_convert", err)

	fallbackResult, fallbackErr := p.fallback.Convert(ctx, amount, from, to)
	if fallbackErr != nil {
		p.log.Warn("exchange rate provider fallback failed",
			zap.String("action", "exchange_rate_convert"),
			zap.String("provider", p.providerName),
			zap.String("fallbackProvider", p.fallbackProviderName),
			zap.String("errorType", providerErrorKind(fallbackErr)),
			zap.Error(fallbackErr),
		)
		return nil, err
	}
	fallbackResult.Provider = p.fallbackProviderName
	fallbackResult.FallbackUsed = true
	return fallbackResult, nil
}

func (p *fallbackExchangeRateProvider) logFallback(action string, err error) {
	p.log.Warn("exchange rate provider fallback used",
		zap.String("action", action),
		zap.String("provider", p.providerName),
		zap.String("fallbackProvider", p.fallbackProviderName),
		zap.Bool("fallbackUsed", true),
		zap.String("errorType", providerErrorKind(err)),
		zap.Error(err),
	)
}
