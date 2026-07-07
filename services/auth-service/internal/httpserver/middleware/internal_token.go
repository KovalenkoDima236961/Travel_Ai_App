package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// InternalServiceTokenHeader is the header internal callers must present.
const InternalServiceTokenHeader = "X-Internal-Service-Token"

// InternalServiceToken guards internal service-to-service endpoints. A request
// must present the shared token in the X-Internal-Service-Token header; any
// missing or mismatched token is rejected with 401. The comparison is
// constant-time so the token cannot be discovered by timing.
//
// This is the v1 scheme. It deliberately does NOT validate a user JWT: internal
// callers (e.g. Notification Service) are trusted to supply recipient user ids.
// It can be replaced later by mTLS or signed service tokens without changing
// callers. It mirrors the identical middleware in Notification Service.
func InternalServiceToken(expectedToken string) func(http.Handler) http.Handler {
	expected := strings.TrimSpace(expectedToken)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			provided := strings.TrimSpace(r.Header.Get(InternalServiceTokenHeader))
			if expected == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
				writeUnauthorized(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
