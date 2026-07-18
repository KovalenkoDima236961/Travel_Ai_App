// Package verification evaluates how much of a trip is backed by current,
// real-world data. It deliberately consumes persisted trip metadata only; it
// never implies a booking, purchase, or provider guarantee.
package verification

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type Status string

const (
	StatusVerified      Status = "verified"
	StatusNeedsReview   Status = "needs_review"
	StatusEstimated     Status = "estimated"
	StatusStale         Status = "stale"
	StatusMissing       Status = "missing"
	StatusUnavailable   Status = "unavailable"
	StatusFailed        Status = "failed"
	StatusNotApplicable Status = "not_applicable"
)

type Source string

const (
	SourceProvider     Source = "provider"
	SourceManual       Source = "manual"
	SourceReceipt      Source = "receipt"
	SourceCalendarSync Source = "calendar_sync"
	SourceAI           Source = "ai"
	SourceMock         Source = "mock"
	SourceFallback     Source = "fallback"
	SourceHeuristic    Source = "heuristic"
	SourceImported     Source = "imported"
	SourceUnknown      Source = "unknown"
)

type Scope string

const (
	ScopeTransport     Scope = "transport"
	ScopePlace         Scope = "place"
	ScopeOpeningHours  Scope = "opening_hours"
	ScopePrice         Scope = "price"
	ScopeAvailability  Scope = "availability"
	ScopeWeather       Scope = "weather"
	ScopeRouteEstimate Scope = "route_estimate"
	ScopeCalendarSync  Scope = "calendar_sync"
	ScopeAccommodation Scope = "accommodation"
	ScopeItineraryItem Scope = "itinerary_item"
	ScopeBudget        Scope = "budget"
	ScopePublicShare   Scope = "public_share"
	ScopeOther         Scope = "other"
)

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type Action struct {
	Type  string `json:"type"`
	Label string `json:"label"`
	Href  string `json:"href"`
}

type Detail struct {
	Scope      Scope          `json:"scope"`
	EntityType string         `json:"entityType"`
	EntityID   string         `json:"entityId"`
	Status     Status         `json:"status"`
	Source     Source         `json:"source"`
	Provider   string         `json:"provider,omitempty"`
	CheckedAt  *time.Time     `json:"checkedAt,omitempty"`
	ExpiresAt  *time.Time     `json:"expiresAt,omitempty"`
	Confidence *float64       `json:"confidence,omitempty"`
	Title      string         `json:"title"`
	Message    string         `json:"message"`
	Severity   Severity       `json:"severity"`
	Action     *Action        `json:"action,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type Section struct {
	Scope   Scope    `json:"scope"`
	Score   int      `json:"score"`
	Status  Status   `json:"status"`
	Details []Detail `json:"details"`
}

type Summary struct {
	VerifiedCount    int `json:"verifiedCount"`
	NeedsReviewCount int `json:"needsReviewCount"`
	EstimatedCount   int `json:"estimatedCount"`
	StaleCount       int `json:"staleCount"`
	MissingCount     int `json:"missingCount"`
	UnavailableCount int `json:"unavailableCount"`
	FailedCount      int `json:"failedCount"`
}

type Level string

const (
	LevelReady       Level = "ready"
	LevelMostlyReady Level = "mostly_ready"
	LevelNeedsReview Level = "needs_review"
	LevelNotReady    Level = "not_ready"
)

type Response struct {
	TripID             uuid.UUID `json:"tripId"`
	Score              int       `json:"score"`
	Level              Level     `json:"level"`
	Summary            Summary   `json:"summary"`
	Sections           []Section `json:"sections"`
	TopIssues          []Detail  `json:"topIssues"`
	RecommendedActions []Action  `json:"recommendedActions"`
	ComputedAt         time.Time `json:"computedAt"`
}

// CalendarState is intentionally a compact, private projection. It excludes
// calendar event titles, attendee details, links, and provider credentials.
type CalendarState struct {
	Connected                bool
	Synced                   bool
	LastSyncedAt             *time.Time
	SyncedItineraryRevision  int
	CurrentItineraryRevision int
	OutOfDate                bool
	Provider                 string
}

type Input struct {
	Trip      *entity.Trip
	Itinerary aggregate.Itinerary
	Calendar  *CalendarState
	Now       time.Time
	Config    Config
}

type Config struct {
	Enabled                   bool
	CacheEnabled              bool
	CacheTTLSeconds           int
	WeatherStaleHoursNearTrip int
	WeatherStaleHoursFarTrip  int
	TransportStaleDays        int
	AvailabilityStaleHours    int
	PriceStaleDays            int
	PlaceStaleDays            int
	RouteEstimateStaleDays    int
	CalendarSyncStaleDays     int
	NearTripDays              int
	MaxDetails                int
	PlaceMinConfidence        float64
}

func DefaultConfig() Config {
	return Config{
		Enabled:                   true,
		CacheEnabled:              true,
		CacheTTLSeconds:           60,
		WeatherStaleHoursNearTrip: 12,
		WeatherStaleHoursFarTrip:  24,
		TransportStaleDays:        7,
		AvailabilityStaleHours:    48,
		PriceStaleDays:            7,
		PlaceStaleDays:            30,
		RouteEstimateStaleDays:    14,
		CalendarSyncStaleDays:     7,
		NearTripDays:              7,
		MaxDetails:                100,
		PlaceMinConfidence:        0.75,
	}
}

type ActionRequest struct {
	ActionType string `json:"actionType"`
	Scope      Scope  `json:"scope"`
	EntityType string `json:"entityType,omitempty"`
	EntityID   string `json:"entityId,omitempty"`
}

type ActionResult struct {
	Status              string   `json:"status"`
	Message             string   `json:"message"`
	UpdatedVerification Response `json:"updatedVerification"`
}
