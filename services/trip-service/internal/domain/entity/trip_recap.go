package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// TripRecap is a private, user-editable record of a completed trip. The JSON
// payload is versioned so a later recap format can be introduced safely.
type TripRecap struct {
	ID              uuid.UUID
	TripID          uuid.UUID
	CreatedByUserID uuid.UUID
	UpdatedByUserID *uuid.UUID
	Status          TripRecapStatus
	RecapJSON       json.RawMessage
	SourceSummary   json.RawMessage
	AIMetadata      json.RawMessage
	FinalizedAt     *time.Time
	ArchivedAt      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type TripRecapStatus string

const (
	TripRecapStatusDraft     TripRecapStatus = "draft"
	TripRecapStatusGenerated TripRecapStatus = "generated"
	TripRecapStatusEdited    TripRecapStatus = "edited"
	TripRecapStatusFinalized TripRecapStatus = "finalized"
	TripRecapStatusArchived  TripRecapStatus = "archived"
)

type TripRecapFeedback struct {
	ID                         uuid.UUID
	TripID                     uuid.UUID
	RecapID                    uuid.UUID
	UserID                     uuid.UUID
	FeedbackType               string
	EntityType                 *string
	EntityID                   *string
	Label                      string
	Value                      *string
	ApprovedForPersonalization bool
	Metadata                   map[string]any
	CreatedAt                  time.Time
	UpdatedAt                  time.Time
}
