package entity

import (
	"time"

	"github.com/google/uuid"
)

type TripCalendarSync struct {
	ID                 uuid.UUID
	TripID             uuid.UUID
	UserID             uuid.UUID
	Provider           string
	ExternalCalendarID string
	ExternalEventID    string
	ExternalEventLink  *string
	DayNumber          int
	ItemIndex          int
	ItineraryRevision  int
	SyncKey            string
	Status             string
	LastSyncedAt       time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          *time.Time
}
