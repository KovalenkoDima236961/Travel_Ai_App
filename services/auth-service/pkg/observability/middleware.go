package observability

import (
	"net/http"
	"strings"
)

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get(HeaderRequestID))
		correlationID := strings.TrimSpace(r.Header.Get(HeaderCorrelationID))
		ctx := ContextWithRequestIDs(r.Context(), requestID, correlationID)

		w.Header().Set(HeaderRequestID, RequestIDFromContext(ctx))
		w.Header().Set(HeaderCorrelationID, CorrelationIDFromContext(ctx))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
