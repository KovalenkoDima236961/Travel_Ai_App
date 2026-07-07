package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

const InternalServiceTokenHeader = "X-Internal-Service-Token"

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
