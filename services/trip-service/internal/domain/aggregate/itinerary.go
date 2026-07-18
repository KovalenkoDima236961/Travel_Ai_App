package aggregate

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TravelStatus records an explicit, lightweight execution update made while a
// traveler is following the itinerary. It deliberately lives with the item so
// older itinerary JSON remains compatible and revision conflict protection
// remains in effect.
type TravelStatus struct {
	Status          string    `json:"status"`
	UpdatedAt       time.Time `json:"updatedAt"`
	UpdatedByUserID uuid.UUID `json:"updatedByUserId"`
	Note            string    `json:"note,omitempty"`
}

// Itinerary is the generated plan associated with a trip. It is stored on the
// trip as JSONB. The typed shape is shared by local and remote itinerary
// generators.
type Itinerary struct {
	Destination string         `json:"destination"`
	Summary     string         `json:"summary"`
	Travelers   int32          `json:"travelers"`
	Pace        string         `json:"pace"`
	Currency    string         `json:"currency"`
	TotalBudget *float64       `json:"totalBudget,omitempty"`
	Days        []ItineraryDay `json:"days"`
	GeneratedAt time.Time      `json:"generatedAt"`
	Source      string         `json:"source"`
}

// ItineraryDay is a single day within a generated itinerary.
type ItineraryDay struct {
	Day           int             `json:"day"`
	Date          string          `json:"date,omitempty"`
	Title         string          `json:"title"`
	PrimaryStopID string          `json:"primaryStopId,omitempty"`
	LocationName  string          `json:"locationName,omitempty"`
	TransferDay   bool            `json:"transferDay,omitempty"`
	Items         []ItineraryItem `json:"items"`
}

// ItineraryItem is a single planned activity within a day.
type ItineraryItem struct {
	Time              string                 `json:"time"`
	EndTime           string                 `json:"endTime,omitempty"`
	Type              string                 `json:"type"`
	Category          string                 `json:"category,omitempty"`
	TransportMode     string                 `json:"transportMode,omitempty"`
	DurationMinutes   *int                   `json:"durationMinutes,omitempty"`
	WalkingDistanceKm *float64               `json:"walkingDistanceKm,omitempty"`
	Name              string                 `json:"name"`
	Note              string                 `json:"note,omitempty"`
	EstimatedCost     *EstimatedCost         `json:"estimatedCost,omitempty"`
	Transfer          *TransferDetails       `json:"transfer,omitempty"`
	Place             *PlaceRef              `json:"place,omitempty"`
	PlaceEnrichment   *PlaceEnrichmentMeta   `json:"placeEnrichment,omitempty"`
	PriceEnrichment   *PriceEnrichmentMeta   `json:"priceEnrichment,omitempty"`
	AvailabilityCheck *AvailabilityCheckMeta `json:"availabilityCheck,omitempty"`
	TravelStatus      *TravelStatus          `json:"travelStatus,omitempty"`
}

// EstimatedCost is the structured, item-level cost estimate stored on an
// itinerary item as part of the itinerary JSONB. All fields are optional; an
// item without a meaningful estimate omits the whole object. Currency, when
// present, is an uppercase ISO-like 3-letter code; an empty currency is treated
// as the trip/itinerary currency by budget computations.
type EstimatedCost struct {
	Amount     *float64       `json:"amount,omitempty"`
	Currency   string         `json:"currency,omitempty"`
	Category   string         `json:"category,omitempty"`
	Confidence string         `json:"confidence,omitempty"`
	Source     string         `json:"source,omitempty"`
	Note       string         `json:"note,omitempty"`
	Split      *CostSplitRule `json:"split,omitempty"`
}

// CostSplitRule is planning-only metadata that describes how one estimated
// cost should be allocated across trip travelers.
type CostSplitRule struct {
	Type        string             `json:"type"`
	TravelerIDs []string           `json:"travelerIds,omitempty"`
	Percentages map[string]float64 `json:"percentages,omitempty"`
}

// UnmarshalJSON accepts both the structured object form and the legacy bare
// number form (where estimatedCost was a single float). This keeps itineraries
// stored or generated before Budget Tracking v1 readable. A null payload leaves
// the value zeroed (callers hold it behind a pointer, so null stays nil).
func (c *EstimatedCost) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil
	}
	if trimmed[0] == '{' {
		// Alias avoids recursing back into this method.
		type alias EstimatedCost
		var a alias
		if err := json.Unmarshal(trimmed, &a); err != nil {
			return err
		}
		*c = EstimatedCost(a)
		return nil
	}

	// Legacy form: a bare number meaning the amount only.
	var amount float64
	if err := json.Unmarshal(trimmed, &amount); err != nil {
		return err
	}
	c.Amount = &amount
	return nil
}

// OpeningHoursInterval is one local-time opening interval for an attached
// place. DayOfWeek uses the app-wide convention: 1 = Monday, 7 = Sunday.
type OpeningHoursInterval struct {
	DayOfWeek int    `json:"dayOfWeek"`
	Open      string `json:"open"`
	Close     string `json:"close"`
}

// PlaceRef is optional real-place metadata attached to an itinerary item.
type PlaceRef struct {
	Provider        string                 `json:"provider"`
	ProviderPlaceID string                 `json:"providerPlaceId"`
	Name            string                 `json:"name"`
	Address         string                 `json:"address"`
	Latitude        *float64               `json:"latitude,omitempty"`
	Longitude       *float64               `json:"longitude,omitempty"`
	Rating          *float64               `json:"rating,omitempty"`
	RatingCount     *int                   `json:"ratingCount,omitempty"`
	MapURL          string                 `json:"mapUrl,omitempty"`
	Category        string                 `json:"category,omitempty"`
	Website         string                 `json:"website,omitempty"`
	OpeningHours    []OpeningHoursInterval `json:"openingHours,omitempty"`
}

// PlaceEnrichmentMeta describes automatic Trip Service place matching for an
// itinerary item. It is optional and preserved with itinerary JSON snapshots.
type PlaceEnrichmentMeta struct {
	Status       string  `json:"status"`
	ReviewStatus string  `json:"reviewStatus,omitempty"`
	Confidence   float64 `json:"confidence,omitempty"`
	Query        string  `json:"query,omitempty"`
	Provider     string  `json:"provider,omitempty"`
	MatchedAt    string  `json:"matchedAt,omitempty"`
	Reason       string  `json:"reason,omitempty"`
}

// PriceEnrichmentMeta describes automatic attraction/ticket price estimation
// for an itinerary item. It is optional and preserved with itinerary snapshots.
type PriceEnrichmentMeta struct {
	Status          string  `json:"status"`
	Provider        string  `json:"provider,omitempty"`
	MatchConfidence float64 `json:"matchConfidence,omitempty"`
	PriceType       string  `json:"priceType,omitempty"`
	ReviewStatus    string  `json:"reviewStatus,omitempty"`
	UpdatedAt       string  `json:"updatedAt,omitempty"`
	Reason          string  `json:"reason,omitempty"`
}

// AvailabilityCheckMeta is a lightweight snapshot of the last external
// availability check a user applied to an item. It is optional, stored inside
// the itinerary JSONB, and used by the approval checklist to surface richer
// availability signals (low-confidence match, fallback data, price change). It
// deliberately stores only summary fields — never the raw provider response,
// full option lists, secrets, or booking-session data.
type AvailabilityCheckMeta struct {
	Provider         string  `json:"provider,omitempty"`
	Status           string  `json:"status,omitempty"`
	CheckedAt        string  `json:"checkedAt,omitempty"`
	MatchConfidence  float64 `json:"matchConfidence,omitempty"`
	SelectedOptionID string  `json:"selectedOptionId,omitempty"`
	FallbackUsed     bool    `json:"fallbackUsed,omitempty"`
	PriceChanged     bool    `json:"priceChanged,omitempty"`
}
