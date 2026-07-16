package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// InternalServiceTokenHeader is the header internal callers must present.
const InternalServiceTokenHeader = "X-Internal-Service-Token"
const InternalServiceNameHeader = "X-Internal-Service-Name"

var (
	internalAuthFailures  = prometheus.NewCounter(prometheus.CounterOpts{Name: "internal_auth_failures_total", Help: "Rejected internal service authentication attempts."})
	internalAuthSuccesses = prometheus.NewCounter(prometheus.CounterOpts{Name: "internal_auth_success_total", Help: "Successful internal service authentication attempts."})
)

func init() { prometheus.MustRegister(internalAuthFailures, internalAuthSuccesses) }

// InternalServiceToken guards internal service-to-service endpoints. A request
// must present the shared token in the X-Internal-Service-Token header; any
// missing or mismatched token is rejected with 401. The comparison is
// constant-time so the token cannot be discovered by timing.
//
// This is the v1 scheme. It deliberately does NOT validate a user JWT: internal
// callers (e.g. Notification Service) are trusted to supply recipient user ids.
// It can be replaced later by mTLS or signed service tokens without changing
// callers. It mirrors the identical middleware in Notification Service.
func InternalServiceToken(expectedToken string, loggers ...*zap.Logger) func(http.Handler) http.Handler {
	expected := activeTokens(expectedToken)
	log := zap.NewNop()
	if len(loggers) > 0 && loggers[0] != nil {
		log = loggers[0]
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provided := strings.TrimSpace(r.Header.Get(InternalServiceTokenHeader))
			if !matchesAnyToken(provided, expected) {
				internalAuthFailures.Inc()
				log.Warn("security_audit",
					zap.String("action", "internal_auth"),
					zap.String("service_name", strings.TrimSpace(r.Header.Get(InternalServiceNameHeader))),
					zap.String("endpoint", r.URL.Path),
					zap.String("request_id", strings.TrimSpace(r.Header.Get("X-Request-ID"))),
					zap.String("outcome", "denied"),
				)
				writeUnauthorized(w)
				return
			}
			internalAuthSuccesses.Inc()
			next.ServeHTTP(w, r)
		})
	}
}

func activeTokens(value string) []string {
	items := make([]string, 0)
	for _, raw := range strings.Split(value, ",") {
		if token := strings.TrimSpace(raw); token != "" {
			items = append(items, token)
		}
	}
	return items
}

func matchesAnyToken(provided string, expected []string) bool {
	if provided == "" || len(expected) == 0 {
		return false
	}
	matched := 0
	for _, token := range expected {
		matched |= subtle.ConstantTimeCompare([]byte(provided), []byte(token))
	}
	return matched == 1
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
