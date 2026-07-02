package observability

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const (
	HeaderRequestID     = "X-Request-ID"
	HeaderCorrelationID = "X-Correlation-ID"
)

type requestIDContextKey struct{}
type correlationIDContextKey struct{}

func ContextWithRequestIDs(ctx context.Context, requestID, correlationID string) context.Context {
	requestID = strings.TrimSpace(requestID)
	correlationID = strings.TrimSpace(correlationID)
	if requestID == "" {
		requestID = uuid.NewString()
	}
	if correlationID == "" {
		correlationID = requestID
	}
	ctx = context.WithValue(ctx, requestIDContextKey{}, requestID)
	return context.WithValue(ctx, correlationIDContextKey{}, correlationID)
}

func ContextWithGeneratedRequestIDs(ctx context.Context) context.Context {
	return ContextWithRequestIDs(ctx, "", "")
}

func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if value, ok := ctx.Value(requestIDContextKey{}).(string); ok {
		return value
	}
	return ""
}

func CorrelationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if value, ok := ctx.Value(correlationIDContextKey{}).(string); ok {
		return value
	}
	return ""
}

func EnsureRequestIDs(ctx context.Context) (context.Context, string, string) {
	requestID := RequestIDFromContext(ctx)
	correlationID := CorrelationIDFromContext(ctx)
	if requestID == "" || correlationID == "" {
		ctx = ContextWithRequestIDs(ctx, requestID, correlationID)
		requestID = RequestIDFromContext(ctx)
		correlationID = CorrelationIDFromContext(ctx)
	}
	return ctx, requestID, correlationID
}

func PropagateRequestIDs(req *http.Request) {
	if req == nil {
		return
	}
	ctx, requestID, correlationID := EnsureRequestIDs(req.Context())
	*req = *req.WithContext(ctx)
	req.Header.Set(HeaderRequestID, requestID)
	req.Header.Set(HeaderCorrelationID, correlationID)
}
