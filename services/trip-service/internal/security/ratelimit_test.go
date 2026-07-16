package security

import (
	"testing"
	"time"
)

func TestRateLimiterIsKeyedAndResets(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	limiter.now = func() time.Time { return now }
	if !limiter.Allow("a") || !limiter.Allow("a") || limiter.Allow("a") {
		t.Fatal("expected third request for key a to be denied")
	}
	if !limiter.Allow("b") {
		t.Fatal("another key must have an independent bucket")
	}
	now = now.Add(time.Minute)
	if !limiter.Allow("a") {
		t.Fatal("bucket should reset after its window")
	}
}
