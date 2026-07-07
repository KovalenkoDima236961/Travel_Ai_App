package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCacheReturnsValueBeforeTTL(t *testing.T) {
	c := New(0)
	c.Set("k", "v", time.Minute)

	got, ok := c.Get("k")
	if !ok {
		t.Fatal("expected cache hit before TTL")
	}
	if got != "v" {
		t.Fatalf("expected value v, got %v", got)
	}
}

func TestCacheMissForUnknownKey(t *testing.T) {
	c := New(0)
	if _, ok := c.Get("missing"); ok {
		t.Fatal("expected miss for unknown key")
	}
}

func TestCacheExpiredEntryNotReturned(t *testing.T) {
	c := New(0)
	current := time.Unix(1_700_000_000, 0)
	c.now = func() time.Time { return current }

	c.Set("k", "v", 30*time.Second)
	if _, ok := c.Get("k"); !ok {
		t.Fatal("expected hit immediately after set")
	}

	// Advance past the TTL.
	current = current.Add(31 * time.Second)
	if _, ok := c.Get("k"); ok {
		t.Fatal("expected expired entry to be a miss")
	}
	if c.Len() != 0 {
		t.Fatalf("expected expired entry to be evicted, len=%d", c.Len())
	}
}

func TestCacheZeroTTLIsNoOp(t *testing.T) {
	c := New(0)
	c.Set("k", "v", 0)
	if _, ok := c.Get("k"); ok {
		t.Fatal("expected zero-TTL set to store nothing")
	}
}

func TestCacheRespectsMaxSize(t *testing.T) {
	c := New(2)
	c.Set("a", 1, time.Minute)
	c.Set("b", 2, time.Minute)
	c.Set("c", 3, time.Minute)
	if c.Len() > 2 {
		t.Fatalf("expected cache to stay within max size, len=%d", c.Len())
	}
}

func TestCacheConcurrentAccessIsSafe(t *testing.T) {
	c := New(0)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			key := fmt.Sprintf("key-%d", n%10)
			c.Set(key, n, time.Minute)
			_, _ = c.Get(key)
		}(i)
	}
	wg.Wait()
}
