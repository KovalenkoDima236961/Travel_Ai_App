package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TripTemplateVisibility string

const (
	TripTemplateVisibilityPrivate   TripTemplateVisibility = "private"
	TripTemplateVisibilityWorkspace TripTemplateVisibility = "workspace"
)

type TripTemplateStatus string

const (
	TripTemplateStatusActive   TripTemplateStatus = "active"
	TripTemplateStatusArchived TripTemplateStatus = "archived"
)

// TripTemplate stores sanitized reusable itinerary structure. It intentionally
// has no comments, collaborators, share links, jobs, calendar sync, or live
// availability state.
type TripTemplate struct {
	ID                     uuid.UUID
	WorkspaceID            *uuid.UUID
	CreatedByUserID        uuid.UUID
	SourceTripID           *uuid.UUID
	Title                  string
	Description            *string
	DestinationHint        *string
	DurationDays           int32
	DefaultCurrency        *string
	Visibility             TripTemplateVisibility
	TemplateJSON           json.RawMessage
	Tags                   []string
	EstimatedTotalAmount   *float64
	EstimatedTotalCurrency *string
	Status                 TripTemplateStatus
	CreatedAt              time.Time
	UpdatedAt              time.Time
	ArchivedAt             *time.Time
	ArchivedByUserID       *uuid.UUID
}
