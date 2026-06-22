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
	Time          string   `json:"time"`
	Type          string   `json:"type"`
	Name          string   `json:"name"`
	Note          string   `json:"note,omitempty"`
	EstimatedCost *float64 `json:"estimatedCost,omitempty"`
}
