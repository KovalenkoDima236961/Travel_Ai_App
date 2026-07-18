package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

func TestSummaryCacheExpiresAndSeparatesViewers(t *testing.T) {
	now := time.Date(2026, time.July, 17, 10, 0, 0, 0, time.UTC)
	cache := newSummaryCache(true, 30*time.Second, 10)
	cache.now = func() time.Time { return now }
	trip := &entity.Trip{ID: uuid.New(), ItineraryRevision: 4, UpdatedAt: now}
	viewerA, viewerB := uuid.New(), uuid.New()
	keyA := summaryCacheKey("command_center", trip, viewerA, "viewer")
	keyB := summaryCacheKey("command_center", trip, viewerB, "viewer")
	if keyA == keyB {
		t.Fatal("viewer-scoped summary keys must differ")
	}
	trip.ItineraryRevision++
	if revised := summaryCacheKey("command_center", trip, viewerA, "viewer"); revised == keyA {
		t.Fatal("itinerary revision must change the summary cache key")
	}
	trip.ItineraryRevision--
	cache.set("command_center", keyA, "value")
	if value, ok := cache.get("command_center", keyA); !ok || value != "value" {
		t.Fatalf("expected cache hit, got value=%v hit=%v", value, ok)
	}
	if _, ok := cache.get("command_center", keyB); ok {
		t.Fatal("a second viewer must not receive the first viewer's summary")
	}
	now = now.Add(31 * time.Second)
	if _, ok := cache.get("command_center", keyA); ok {
		t.Fatal("expired summary should miss")
	}
}

func TestNormalizeListLimit(t *testing.T) {
	if got := normalizeListLimit(0, 50, 100); got != 50 {
		t.Fatalf("default limit = %d, want 50", got)
	}
	if got := normalizeListLimit(500, 50, 100); got != 100 {
		t.Fatalf("max limit = %d, want 100", got)
	}
}

func TestSummaryCacheEvictsOldestAtCapacity(t *testing.T) {
	now := time.Date(2026, time.July, 17, 10, 0, 0, 0, time.UTC)
	cache := newSummaryCache(true, time.Minute, 1)
	cache.now = func() time.Time { return now }
	cache.set("health", "first", 1)
	now = now.Add(time.Second)
	cache.set("health", "second", 2)
	if _, ok := cache.get("health", "first"); ok {
		t.Fatal("oldest entry should be evicted")
	}
	if value, ok := cache.get("health", "second"); !ok || value != 2 {
		t.Fatalf("newest entry should remain, got value=%v hit=%v", value, ok)
	}
}

func TestSummaryCacheCustomTTLAndClear(t *testing.T) {
	now := time.Date(2026, time.July, 17, 10, 0, 0, 0, time.UTC)
	cache := newSummaryCache(true, 30*time.Second, 10)
	cache.now = func() time.Time { return now }
	cache.setWithTTL("library_insights", "insights", "value", 5*time.Minute)
	now = now.Add(31 * time.Second)
	if value, ok := cache.get("library_insights", "insights"); !ok || value != "value" {
		t.Fatalf("custom cache TTL should keep entry alive, got value=%v hit=%v", value, ok)
	}
	cache.clear("library_insights")
	if _, ok := cache.get("library_insights", "insights"); ok {
		t.Fatal("cleared summary should not remain cached")
	}
}
