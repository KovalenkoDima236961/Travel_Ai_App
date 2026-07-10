package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
)

// Status represents the lifecycle state of a trip planning request.
type Status string

const (
	StatusDraft      Status = "DRAFT"
	StatusProcessing Status = "PROCESSING"
	StatusCompleted  Status = "COMPLETED"
	StatusFailed     Status = "FAILED"
)

const (
	TripTypeSingleDestination = "single_destination"
	TripTypeMultiDestination  = "multi_destination"
)

// Trip is the domain entity, using plain Go types. Infrastructure adapters map
// between this and their own representations (DB rows, API payloads).
type Trip struct {
	ID                uuid.UUID
	UserID            *uuid.UUID
	WorkspaceID       *uuid.UUID
	TripType          string
	Destination       string
	StartDate         *time.Time
	Days              int32
	BudgetAmount      *float64
	BudgetCurrency    string
	Travelers         int32
	Interests         []string
	Pace              string
	Status            Status
	Route             *aggregate.TripRoute
	Itinerary         json.RawMessage
	Accommodation     *aggregate.Accommodation
	CreationMetadata  map[string]any
	ItineraryRevision int
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
