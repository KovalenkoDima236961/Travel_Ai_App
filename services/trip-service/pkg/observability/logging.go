package observability

import (
	"context"

	"go.uber.org/zap"
)

func RequestIDFields(ctx context.Context) []zap.Field {
	fields := make([]zap.Field, 0, 2)
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		fields = append(fields, zap.String("requestId", requestID))
	}
	if correlationID := CorrelationIDFromContext(ctx); correlationID != "" {
		fields = append(fields, zap.String("correlationId", correlationID))
	}
	return fields
}

func LoggerWithRequestIDs(log *zap.Logger, ctx context.Context) *zap.Logger {
	if log == nil {
		log = zap.NewNop()
	}
	if fields := RequestIDFields(ctx); len(fields) > 0 {
		return log.With(fields...)
	}
	return log
}
