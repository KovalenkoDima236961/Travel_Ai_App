package cache

import (
	"sync"
	"time"
)

// defaultMaxEntries bounds memory use when a caller does not specify a size.
const defaultMaxEntries = 4096

type entry struct {
	value     any
	expiresAt time.Time
}

// TTLCache is a concurrency-safe map of string keys to arbitrary values, each
// with its own expiry. Expired entries are removed lazily on read and during
// opportunistic cleanup on write. An optional maximum size bounds memory; when
// the cache is full a best-effort eviction frees space for new entries.
type TTLCache struct {
	mu      sync.RWMutex
	entries map[string]entry
	maxSize int
	now     func() time.Time
}

// New constructs a cache holding at most maxSize entries. A non-positive
// maxSize falls back to a sane default so the cache can never grow unbounded.
func New(maxSize int) *TTLCache {
	if maxSize <= 0 {
		maxSize = defaultMaxEntries
	}
	return &TTLCache{
		entries: make(map[string]entry),
		maxSize: maxSize,
		now:     time.Now,
	}
}

// Get returns the cached value for key when present and unexpired. Expired
// entries are evicted and reported as a miss.
func (c *TTLCache) Get(key string) (any, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}

	if c.now().After(e.expiresAt) {
		c.mu.Lock()
		if current, ok := c.entries[key]; ok && c.now().After(current.expiresAt) {
			delete(c.entries, key)
		}
		c.mu.Unlock()
		return nil, false
	}

	return e.value, true
}

// Set stores value under key with the given TTL. A non-positive TTL is a no-op
// so callers can disable caching by passing a zero duration.
func (c *TTLCache) Set(key string, value any, ttl time.Duration) {
	if ttl <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[key]; !exists && len(c.entries) >= c.maxSize {
		c.evictLocked()
	}
	c.entries[key] = entry{value: value, expiresAt: c.now().Add(ttl)}
}

// Len reports the number of stored entries, including any not yet lazily
// expired. It is primarily useful for tests.
func (c *TTLCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// evictLocked frees space for a new entry. It first drops expired entries and,
// if the cache is still full, removes one arbitrary entry. Callers must hold
// the write lock.
func (c *TTLCache) evictLocked() {
	now := c.now()
	for key, e := range c.entries {
		if now.After(e.expiresAt) {
			delete(c.entries, key)
		}
	}
	if len(c.entries) < c.maxSize {
		return
	}
	for key := range c.entries {
		delete(c.entries, key)
		return
	}
}
