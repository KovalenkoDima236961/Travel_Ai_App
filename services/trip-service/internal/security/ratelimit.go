package security

import (
	"sync"
	"time"
)

type rateBucket struct {
	windowStart time.Time
	count       int
}

// RateLimiter is a concurrent in-memory fixed-window limiter. It is suitable
// for the v1 single-instance boundary; deployments with multiple replicas
// should replace its storage with a shared backend.
type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]rateBucket
	now     func() time.Time
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	return &RateLimiter{
		limit: limit, window: window, buckets: make(map[string]rateBucket), now: time.Now,
	}
}

func (l *RateLimiter) Allow(key string) bool {
	if l == nil || key == "" {
		return false
	}
	now := l.now().UTC()
	l.mu.Lock()
	defer l.mu.Unlock()
	bucket := l.buckets[key]
	if bucket.windowStart.IsZero() || now.Sub(bucket.windowStart) >= l.window {
		l.buckets[key] = rateBucket{windowStart: now, count: 1}
		return true
	}
	if bucket.count >= l.limit {
		return false
	}
	bucket.count++
	l.buckets[key] = bucket
	return true
}
