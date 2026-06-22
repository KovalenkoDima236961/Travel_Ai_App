package trip

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Status represents the lifecycle state of a trip planning request.
type Status string

const (
	StatusDraft      Status = "DRAFT"
	StatusProcessing Status = "PROCESSING"
	StatusCompleted  Status = "COMPLETED"
	StatusFailed     Status = "FAILED"
)

// Trip is the domain model, using plain Go types. The repository maps between
// this and the database representation.
type Trip struct {
	ID             uuid.UUID
	UserID         *uuid.UUID
	Destination    string
	StartDate      *time.Time
	Days           int32
	BudgetAmount   *float64
	BudgetCurrency string
	Travelers      int32
	Interests      []string
	Pace           string
	Status         Status
	Itinerary      json.RawMessage
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// Itinerary is the mock plan generated locally until the AI Planning Service
// is integrated over async messaging.
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
	Day        int      `json:"day"`
	Title      string   `json:"title"`
	Activities []string `json:"activities"`
}
