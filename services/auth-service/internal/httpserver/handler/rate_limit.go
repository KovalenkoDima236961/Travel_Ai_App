package handler

import (
	"context"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

type authRateBucket struct {
	start time.Time
	count int
}

// originalRemoteAddrKey preserves the transport peer address before chi's
// RealIP middleware may replace RemoteAddr with a caller-controlled forwarding
// header. Authentication rate limiting must not be bypassable by cycling
// X-Forwarded-For values on a directly exposed service.
type originalRemoteAddrKey struct{}

// OriginalRemoteAddrMiddleware captures the direct peer address for sensitive
// rate-limit keys. It must run before middleware.RealIP.
func OriginalRemoteAddrMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), originalRemoteAddrKey{}, r.RemoteAddr)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type authRateLimiter struct {
	mu      sync.Mutex
	limit   int
	buckets map[string]authRateBucket
}

var authRateLimited = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "auth_rate_limited_total", Help: "Sensitive Auth Service requests denied by rate limiting.",
}, []string{"endpoint"})

func init() { prometheus.MustRegister(authRateLimited) }

func newAuthRateLimiter(limit int) *authRateLimiter {
	if limit <= 0 {
		limit = 1
	}
	return &authRateLimiter{limit: limit, buckets: make(map[string]authRateBucket)}
}

func (l *authRateLimiter) allow(key string) bool {
	now := time.Now().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()
	bucket := l.buckets[key]
	if bucket.start.IsZero() || now.Sub(bucket.start) >= time.Minute {
		l.buckets[key] = authRateBucket{start: now, count: 1}
		return true
	}
	if bucket.count >= l.limit {
		return false
	}
	bucket.count++
	l.buckets[key] = bucket
	return true
}

func clientRateKey(r *http.Request) string {
	remoteAddr := r.RemoteAddr
	if original, ok := r.Context().Value(originalRemoteAddrKey{}).(string); ok && original != "" {
		remoteAddr = original
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err == nil && host != "" {
		return host
	}
	if value := strings.TrimSpace(remoteAddr); value != "" {
		return value
	}
	return "unknown"
}

func (h *Handler) allowSensitive(w http.ResponseWriter, r *http.Request, endpoint string, limiter *authRateLimiter) bool {
	if limiter.allow(clientRateKey(r)) {
		return true
	}
	authRateLimited.WithLabelValues(endpoint).Inc()
	h.log.Warn("security_audit", zap.String("action", "auth_rate_limit"), zap.String("resource_type", endpoint), zap.String("outcome", "rate_limited"))
	w.Header().Set("Retry-After", "60")
	writeJSON(w, http.StatusTooManyRequests, map[string]any{
		"error": map[string]string{
			"code":    "rate_limited",
			"message": "Too many attempts. Please try again later.",
		},
	})
	return false
}
