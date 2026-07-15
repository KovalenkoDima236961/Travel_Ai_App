package transport

import (
	"context"

	"go.uber.org/zap"
)

type fallbackProvider struct {
	providerName string
	primary      TransportProvider
	fallback     TransportProvider
	log          *zap.Logger
}

func newFallbackProvider(providerName string, primary TransportProvider, fallback TransportProvider, log *zap.Logger) TransportProvider {
	if log == nil {
		log = zap.NewNop()
	}
	return &fallbackProvider{providerName: providerName, primary: primary, fallback: fallback, log: log}
}

func (p *fallbackProvider) SearchTransportOptions(ctx context.Context, req TransportSearchRequest) (TransportSearchResponse, error) {
	result, err := p.primary.SearchTransportOptions(ctx, req)
	if err == nil {
		return result, nil
	}
	p.log.Warn("transport provider fallback used",
		zap.String("operation", "transport_search"),
		zap.String("provider", p.providerName),
		zap.String("fallbackProvider", ProviderMock),
		zap.Bool("fallbackUsed", true),
		zap.String("errorType", providerErrorKind(err)),
		zap.Error(err),
	)
	fallbackResult, fallbackErr := p.fallback.SearchTransportOptions(ctx, req)
	if fallbackErr != nil {
		return TransportSearchResponse{}, err
	}
	fallbackResult.Summary.Provider = ProviderMock
	fallbackResult.Summary.FallbackUsed = true
	return fallbackResult, nil
}
