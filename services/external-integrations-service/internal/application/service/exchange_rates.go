package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/internal/domain/entity"
)

// ExchangeRateProvider is implemented by exchange-rate providers.
type ExchangeRateProvider interface {
	Latest(ctx context.Context, base string) (*entity.ExchangeRateTable, error)
	Convert(ctx context.Context, amount float64, from string, to string) (*entity.CurrencyConversionResult, error)
}

// ExchangeRateService contains exchange-rate use cases over the configured
// provider. The handler owns transport validation.
type ExchangeRateService struct {
	provider ExchangeRateProvider
	log      *zap.Logger
}

func NewExchangeRateService(provider ExchangeRateProvider, log *zap.Logger) *ExchangeRateService {
	if log == nil {
		log = zap.NewNop()
	}
	return &ExchangeRateService{provider: provider, log: log}
}

func (s *ExchangeRateService) Latest(ctx context.Context, base string) (*entity.ExchangeRateTable, error) {
	start := time.Now()
	table, err := s.provider.Latest(ctx, base)
	if err != nil {
		s.log.Warn("exchange_rate_latest",
			zap.String("action", "exchange_rate_latest"),
			zap.String("base", base),
			zap.Int64("durationMs", time.Since(start).Milliseconds()),
			zap.Bool("success", false),
			zap.Error(err),
		)
		return nil, err
	}

	s.log.Info("exchange_rate_latest",
		zap.String("action", "exchange_rate_latest"),
		zap.String("provider", table.Provider),
		zap.String("base", table.Base),
		zap.Int("rateCount", len(table.Rates)),
		zap.Int64("durationMs", time.Since(start).Milliseconds()),
		zap.Bool("fallbackUsed", table.FallbackUsed),
		zap.Bool("success", true),
	)
	return table, nil
}

func (s *ExchangeRateService) Convert(ctx context.Context, amount float64, from string, to string) (*entity.CurrencyConversionResult, error) {
	start := time.Now()
	result, err := s.provider.Convert(ctx, amount, from, to)
	if err != nil {
		s.log.Warn("exchange_rate_convert",
			zap.String("action", "exchange_rate_convert"),
			zap.String("from", from),
			zap.String("to", to),
			zap.Int64("durationMs", time.Since(start).Milliseconds()),
			zap.Bool("success", false),
			zap.Error(err),
		)
		return nil, err
	}

	s.log.Info("exchange_rate_convert",
		zap.String("action", "exchange_rate_convert"),
		zap.String("provider", result.Provider),
		zap.String("from", result.From),
		zap.String("to", result.To),
		zap.Int64("durationMs", time.Since(start).Milliseconds()),
		zap.Bool("fallbackUsed", result.FallbackUsed),
		zap.Bool("success", true),
	)
	return result, nil
}
