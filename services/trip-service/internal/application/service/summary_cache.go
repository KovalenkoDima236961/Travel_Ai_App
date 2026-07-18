package service

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	tripobs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/observability"
)

type summaryCacheEntry struct {
	value     any
	expiresAt time.Time
	createdAt time.Time
	summary   string
}

func summaryCacheKey(summary string, trip *entity.Trip, viewerID uuid.UUID, variants ...any) string {
	parts := []string{summary, viewerID.String()}
	if trip != nil {
		parts = append(parts,
			trip.ID.String(),
			fmt.Sprint(trip.ItineraryRevision),
			fmt.Sprint(trip.UpdatedAt.UTC().UnixNano()),
		)
	}
	for _, variant := range variants {
		parts = append(parts, fmt.Sprint(variant))
	}
	return strings.Join(parts, ":")
}

type summaryCache struct {
	mu       sync.Mutex
	enabled  bool
	ttl      time.Duration
	maxItems int
	items    map[string]summaryCacheEntry
	now      func() time.Time
}

func newSummaryCache(enabled bool, ttl time.Duration, maxItems int) *summaryCache {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	if maxItems < 1 {
		maxItems = 1000
	}
	return &summaryCache{
		enabled:  enabled,
		ttl:      ttl,
		maxItems: maxItems,
		items:    make(map[string]summaryCacheEntry),
		now:      time.Now,
	}
}

func (c *summaryCache) get(summary, key string) (any, bool) {
	if c == nil || !c.enabled {
		tripobs.RecordSummaryCacheMiss(summary)
		return nil, false
	}
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.items[key]
	if !ok {
		tripobs.RecordSummaryCacheMiss(summary)
		return nil, false
	}
	if !now.Before(entry.expiresAt) {
		delete(c.items, key)
		tripobs.RecordSummaryCacheMiss(summary)
		tripobs.RecordSummaryCacheEviction(entry.summary)
		return nil, false
	}
	tripobs.RecordSummaryCacheHit(summary)
	return entry.value, true
}

func (c *summaryCache) set(summary, key string, value any) {
	c.setWithTTL(summary, key, value, 0)
}

func (c *summaryCache) setWithTTL(summary, key string, value any, ttl time.Duration) {
	if c == nil || !c.enabled {
		return
	}
	if ttl <= 0 {
		ttl = c.ttl
	}
	now := c.now()
	c.mu.Lock()
	defer c.mu.Unlock()

	for itemKey, entry := range c.items {
		if !now.Before(entry.expiresAt) {
			delete(c.items, itemKey)
			tripobs.RecordSummaryCacheEviction(entry.summary)
		}
	}
	if _, exists := c.items[key]; !exists && len(c.items) >= c.maxItems {
		oldestKey := ""
		var oldest summaryCacheEntry
		for itemKey, entry := range c.items {
			if oldestKey == "" || entry.createdAt.Before(oldest.createdAt) {
				oldestKey = itemKey
				oldest = entry
			}
		}
		if oldestKey != "" {
			delete(c.items, oldestKey)
			tripobs.RecordSummaryCacheEviction(oldest.summary)
		}
	}
	c.items[key] = summaryCacheEntry{
		value:     value,
		expiresAt: now.Add(ttl),
		createdAt: now,
		summary:   summary,
	}
}

func (c *summaryCache) clear(summary string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for key, entry := range c.items {
		if entry.summary == summary {
			delete(c.items, key)
			tripobs.RecordSummaryCacheEviction(entry.summary)
		}
	}
}
