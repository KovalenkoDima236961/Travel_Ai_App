package transport

import "context"

type TransportProvider interface {
	SearchTransportOptions(ctx context.Context, req TransportSearchRequest) (TransportSearchResponse, error)
}
