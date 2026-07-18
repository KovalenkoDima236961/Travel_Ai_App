package entity

import (
	"encoding/json"
	"strings"
	"time"
)

// TripLifecycle is a derived organizational state. It intentionally does not
// replace Status, which continues to describe itinerary-generation work.
type TripLifecycle string

const (
	TripLifecycleDraft     TripLifecycle = "draft"
	TripLifecyclePlanning  TripLifecycle = "planning"
	TripLifecycleReady     TripLifecycle = "ready"
	TripLifecycleActive    TripLifecycle = "active"
	TripLifecycleCompleted TripLifecycle = "completed"
	TripLifecycleArchived  TripLifecycle = "archived"
)

type LifecycleOptions struct {
	Now                        time.Time
	ReadyHealthScoreThreshold  int
	ReadyVerificationThreshold int
}

// DeriveLifecycle keeps lifecycle deterministic and resilient to older rows.
// Optional readiness snapshots may be stored in creation_metadata by existing
// integrations; unavailable scores deliberately fall back to planning.
func DeriveLifecycle(trip *Trip, options LifecycleOptions) TripLifecycle {
	if trip == nil {
		return TripLifecycleDraft
	}
	if trip.ArchivedAt != nil {
		return TripLifecycleArchived
	}
	now := options.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	today := dateOnly(now)
	if trip.StartDate != nil {
		start := dateOnly(*trip.StartDate)
		end := start.AddDate(0, 0, max(1, int(trip.Days))-1)
		if !today.Before(start) && !today.After(end) {
			return TripLifecycleActive
		}
		if today.After(end) {
			return TripLifecycleCompleted
		}
	}
	if hasTripItinerary(trip.Itinerary) && readinessPasses(trip.CreationMetadata, options) {
		return TripLifecycleReady
	}
	if hasTripItinerary(trip.Itinerary) || trip.Route != nil || hasSignificantPlanningData(trip) {
		return TripLifecyclePlanning
	}
	return TripLifecycleDraft
}

func dateOnly(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func hasTripItinerary(raw json.RawMessage) bool {
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}
	var payload struct {
		Days []json.RawMessage `json:"days"`
	}
	return json.Unmarshal(raw, &payload) == nil && len(payload.Days) > 0
}

func hasSignificantPlanningData(trip *Trip) bool {
	return trip.BudgetAmount != nil || len(trip.Interests) > 0 || strings.TrimSpace(trip.Pace) != "" && trip.Pace != "balanced"
}

func readinessPasses(metadata map[string]any, options LifecycleOptions) bool {
	if metadata == nil {
		return false
	}
	if ready, ok := metadata["ready"].(bool); ok && ready {
		return true
	}
	health, healthOK := numericMetadata(metadata, "tripHealthScore")
	verification, verificationOK := numericMetadata(metadata, "verificationScore")
	if !healthOK || !verificationOK {
		return false
	}
	healthThreshold := options.ReadyHealthScoreThreshold
	if healthThreshold <= 0 {
		healthThreshold = 80
	}
	verificationThreshold := options.ReadyVerificationThreshold
	if verificationThreshold <= 0 {
		verificationThreshold = 75
	}
	return health >= float64(healthThreshold) && verification >= float64(verificationThreshold)
}

func numericMetadata(metadata map[string]any, key string) (float64, bool) {
	value, ok := metadata[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	default:
		return 0, false
	}
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
