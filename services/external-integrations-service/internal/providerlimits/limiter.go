package providerlimits

import (
	"math"
	"sync"
	"time"
)

// Limiter is an in-memory, per-key token-bucket rate limiter. It is process
// local and resets on restart (documented v1 limitation). A key is typically a
// provider category, since exactly one provider is active per category.
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	now     func() time.Time
}

type tokenBucket struct {
	tokens       float64
	maxTokens    float64
	refillPerSec float64
	last         time.Time
}

// RateSpec configures a bucket. A non-positive PerMinute means unlimited.
type RateSpec struct {
	PerMinute int
	Burst     int
}

// Unlimited reports whether the spec disables rate limiting.
func (s RateSpec) Unlimited() bool { return s.PerMinute <= 0 }

// NewLimiter builds an empty limiter.
func NewLimiter() *Limiter {
	return &Limiter{buckets: map[string]*tokenBucket{}, now: time.Now}
}

// Allow attempts to consume cost tokens for the key under the given spec. It
// returns whether the call is allowed and, when denied, the duration until
// enough tokens are available again. Unlimited specs always allow.
func (l *Limiter) Allow(key string, spec RateSpec, cost int64) (bool, time.Duration) {
	if spec.Unlimited() {
		return true, 0
	}
	if cost < 1 {
		cost = 1
	}
	refillPerSec := float64(spec.PerMinute) / 60.0
	maxTokens := float64(spec.Burst)
	if maxTokens < 1 {
		maxTokens = 1
	}
	// The burst can never be smaller than a single call's cost, otherwise the
	// call could never be admitted.
	if maxTokens < float64(cost) {
		maxTokens = float64(cost)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	bucket, ok := l.buckets[key]
	if !ok {
		bucket = &tokenBucket{tokens: maxTokens, maxTokens: maxTokens, refillPerSec: refillPerSec, last: now}
		l.buckets[key] = bucket
	}
	// Reconfigure on the fly when limits change (e.g. via reload/tests).
	bucket.maxTokens = maxTokens
	bucket.refillPerSec = refillPerSec

	elapsed := now.Sub(bucket.last).Seconds()
	if elapsed > 0 {
		bucket.tokens = math.Min(bucket.maxTokens, bucket.tokens+elapsed*refillPerSec)
		bucket.last = now
	}

	if bucket.tokens >= float64(cost) {
		bucket.tokens -= float64(cost)
		return true, 0
	}

	needed := float64(cost) - bucket.tokens
	wait := time.Duration(needed / refillPerSec * float64(time.Second))
	if wait <= 0 {
		wait = time.Second
	}
	return false, wait
}

// Tokens returns the currently available tokens for a key, for the optional
// tokens-available gauge. It never mutates bucket state.
func (l *Limiter) Tokens(key string) (float64, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	bucket, ok := l.buckets[key]
	if !ok {
		return 0, false
	}
	now := l.now()
	elapsed := now.Sub(bucket.last).Seconds()
	tokens := bucket.tokens
	if elapsed > 0 {
		tokens = math.Min(bucket.maxTokens, tokens+elapsed*bucket.refillPerSec)
	}
	return tokens, true
}
