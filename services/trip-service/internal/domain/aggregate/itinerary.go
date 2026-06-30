package aggregate

import (
	"bytes"
	"encoding/json"
	"time"
)

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
	Day   int             `json:"day"`
	Title string          `json:"title"`
	Items []ItineraryItem `json:"items"`
}

// ItineraryItem is a single planned activity within a day.
type ItineraryItem struct {
	Time            string               `json:"time"`
	Type            string               `json:"type"`
	Name            string               `json:"name"`
	Note            string               `json:"note,omitempty"`
	EstimatedCost   *EstimatedCost       `json:"estimatedCost,omitempty"`
	Place           *PlaceRef            `json:"place,omitempty"`
	PlaceEnrichment *PlaceEnrichmentMeta `json:"placeEnrichment,omitempty"`
}

// EstimatedCost is the structured, item-level cost estimate stored on an
// itinerary item as part of the itinerary JSONB. All fields are optional; an
// item without a meaningful estimate omits the whole object. Currency, when
// present, is an uppercase ISO-like 3-letter code; an empty currency is treated
// as the trip/itinerary currency by budget computations.
type EstimatedCost struct {
	Amount     *float64 `json:"amount,omitempty"`
	Currency   string   `json:"currency,omitempty"`
	Category   string   `json:"category,omitempty"`
	Confidence string   `json:"confidence,omitempty"`
	Source     string   `json:"source,omitempty"`
	Note       string   `json:"note,omitempty"`
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
