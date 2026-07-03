package providerlimits

import (
	"testing"
	"time"
)

func TestLimiterAllowsUnderLimit(t *testing.T) {
	l := NewLimiter()
	spec := RateSpec{PerMinute: 60, Burst: 5}
	for i := 0; i < 5; i++ {
		allowed, _ := l.Allow("routes", spec, 1)
		if !allowed {
			t.Fatalf("call %d should be allowed under burst", i)
		}
	}
}

func TestLimiterBlocksOverLimit(t *testing.T) {
	l := NewLimiter()
	spec := RateSpec{PerMinute: 1, Burst: 1}

	allowed, _ := l.Allow("routes", spec, 1)
	if !allowed {
		t.Fatal("first call should be allowed")
	}
	allowed, wait := l.Allow("routes", spec, 1)
	if allowed {
		t.Fatal("second call should be blocked over the limit")
	}
	if wait <= 0 {
		t.Fatalf("expected a positive retry-after wait, got %v", wait)
	}
}

func TestLimiterUnlimitedAllowsAll(t *testing.T) {
	l := NewLimiter()
	spec := RateSpec{PerMinute: 0, Burst: 0}
	for i := 0; i < 100; i++ {
		if allowed, _ := l.Allow("routes", spec, 1); !allowed {
			t.Fatalf("unlimited spec should always allow (call %d)", i)
		}
	}
}

func TestLimiterRefillsOverTime(t *testing.T) {
	l := NewLimiter()
	now := time.Now()
	l.now = func() time.Time { return now }
	spec := RateSpec{PerMinute: 60, Burst: 1} // 1 token/sec

	if allowed, _ := l.Allow("routes", spec, 1); !allowed {
		t.Fatal("first call should be allowed")
	}
	if allowed, _ := l.Allow("routes", spec, 1); allowed {
		t.Fatal("second immediate call should be blocked")
	}
	now = now.Add(2 * time.Second)
	if allowed, _ := l.Allow("routes", spec, 1); !allowed {
		t.Fatal("call after refill window should be allowed")
	}
}
