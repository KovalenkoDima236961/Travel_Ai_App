package transport

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/services/external-integrations-service/pkg/observability"
)

const transportOperation = "transport_search"

type Service struct {
	provider TransportProvider
	timeout  time.Duration
	log      *zap.Logger
}

func NewService(provider TransportProvider, timeout time.Duration, log *zap.Logger) *Service {
	if log == nil {
		log = zap.NewNop()
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}
	return &Service{provider: provider, timeout: timeout, log: log}
}

func (s *Service) SearchTransportOptions(ctx context.Context, req TransportSearchRequest) (TransportSearchResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	start := time.Now()
	result, err := s.provider.SearchTransportOptions(ctx, req)
	duration := time.Since(start)
	provider := result.Summary.Provider
	if provider == "" {
		provider = "unknown"
	}
	if err != nil {
		code := providerErrorKind(err)
		if code == "unknown" {
			code = ErrorProviderUnavailable
		}
		observability.RecordProviderRequest(provider, transportOperation, "error", duration)
		observability.RecordProviderFailure(provider, transportOperation, code)
		s.log.Warn("transport_search_failed",
			zap.String("provider", provider),
			zap.String("origin", req.Origin.Name),
			zap.String("destination", req.Destination.Name),
			zap.String("errorCode", code),
			zap.Duration("duration", duration),
			zap.Error(err),
		)
		return TransportSearchResponse{}, err
	}
	if result.Options == nil {
		result.Options = []TransportOption{}
	}
	if result.Summary.Provider == "" {
		result.Summary.Provider = provider
	}
	observability.RecordProviderRequest(result.Summary.Provider, transportOperation, "success", duration)
	if result.Summary.FallbackUsed {
		observability.RecordProviderFallback(provider, transportOperation, ProviderMock)
	}
	s.log.Info("transport_search_completed",
		zap.String("provider", result.Summary.Provider),
		zap.String("origin", req.Origin.Name),
		zap.String("destination", req.Destination.Name),
		zap.Int("optionCount", len(result.Options)),
		zap.Bool("fallbackUsed", result.Summary.FallbackUsed),
		zap.Bool("cached", result.Summary.Cached),
		zap.Duration("duration", duration),
	)
	return result, nil
}
