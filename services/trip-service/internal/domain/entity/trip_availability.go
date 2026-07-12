package entity

import (
	"time"

	"github.com/google/uuid"
)

type AvailabilityDateRange struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

type TripAvailabilityResponse struct {
	ID                uuid.UUID
	TripID            uuid.UUID
	UserID            uuid.UUID
	AvailableRanges   []AvailabilityDateRange
	UnavailableRanges []AvailabilityDateRange
	PreferredRanges   []AvailabilityDateRange
	MinTripDays       *int
	MaxTripDays       *int
	Timezone          string
	Notes             string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
