package aggregate

import "time"

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
	EstimatedCost   *float64             `json:"estimatedCost,omitempty"`
	Place           *PlaceRef            `json:"place,omitempty"`
	PlaceEnrichment *PlaceEnrichmentMeta `json:"placeEnrichment,omitempty"`
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
	Status     string  `json:"status"`
	Confidence float64 `json:"confidence,omitempty"`
	Query      string  `json:"query,omitempty"`
	Provider   string  `json:"provider,omitempty"`
	MatchedAt  string  `json:"matchedAt,omitempty"`
	Reason     string  `json:"reason,omitempty"`
}
