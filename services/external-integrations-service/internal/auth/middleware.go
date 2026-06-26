package auth

import (
	"encoding/json"
	"net/http"
	"strings"
)

type MiddlewareConfig struct {
	JWTAccessSecret string
	HeaderName      string
}

func Middleware(cfg MiddlewareConfig) func(http.Handler) http.Handler {
	headerName := strings.TrimSpace(cfg.HeaderName)
	if headerName == "" {
		headerName = "Authorization"
	}
	validator := NewTokenValidator(cfg.JWTAccessSecret)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r.Header.Get(headerName))
			if !ok {
				writeUnauthorized(w)
				return
			}
			user, err := validator.ValidateAccessToken(token)
			if err != nil {
				writeUnauthorized(w)
				return
			}
			user.AccessToken = token
			next.ServeHTTP(w, r.WithContext(WithUser(r.Context(), user)))
		})
	}
}

func bearerToken(header string) (string, bool) {
	const prefix = "bearer "
	value := strings.TrimSpace(header)
	if len(value) <= len(prefix) || strings.ToLower(value[:len(prefix)]) != prefix {
		return "", false
	}
	token := strings.TrimSpace(value[len(prefix):])
	return token, token != ""
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
}
