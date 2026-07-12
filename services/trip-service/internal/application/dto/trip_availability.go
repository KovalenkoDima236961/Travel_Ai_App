package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type AvailabilityDateRange = entity.AvailabilityDateRange

type UpsertTripAvailabilityInput struct {
	AvailableRanges   []AvailabilityDateRange
	UnavailableRanges []AvailabilityDateRange
	PreferredRanges   []AvailabilityDateRange
	MinTripDays       *int
	MaxTripDays       *int
	Timezone          string
	Notes             string
}

type TripAvailabilityUserSummary struct {
	UserID      uuid.UUID `json:"userId"`
	DisplayName string    `json:"displayName"`
}

type TripAvailabilityResponseInfo struct {
	UserID            uuid.UUID               `json:"userId"`
	DisplayName       string                  `json:"displayName"`
	AvailableRanges   []AvailabilityDateRange `json:"availableRanges"`
	UnavailableRanges []AvailabilityDateRange `json:"unavailableRanges"`
	PreferredRanges   []AvailabilityDateRange `json:"preferredRanges"`
	MinTripDays       *int                    `json:"minTripDays,omitempty"`
	MaxTripDays       *int                    `json:"maxTripDays,omitempty"`
	Timezone          string                  `json:"timezone,omitempty"`
	Notes             string                  `json:"notes,omitempty"`
	Submitted         bool                    `json:"submitted"`
	UpdatedAt         *time.Time              `json:"updatedAt,omitempty"`
}

type TripAvailabilitySummary struct {
	TotalCollaborators int                           `json:"totalCollaborators"`
	SubmittedCount     int                           `json:"submittedCount"`
	MissingCount       int                           `json:"missingCount"`
	MissingUsers       []TripAvailabilityUserSummary `json:"missingUsers"`
}

type TripAvailabilityList struct {
	TripID    uuid.UUID                      `json:"tripId"`
	Responses []TripAvailabilityResponseInfo `json:"responses"`
	Summary   TripAvailabilitySummary        `json:"summary"`
}

type DateOptionUserSummary struct {
	UserID      uuid.UUID `json:"userId"`
	DisplayName string    `json:"displayName"`
}

type DateOptionConflict struct {
	UserID      uuid.UUID `json:"userId"`
	DisplayName string    `json:"displayName"`
	Reason      string    `json:"reason"`
}

type DateOption struct {
	ID                       string                  `json:"id"`
	StartDate                string                  `json:"startDate"`
	EndDate                  string                  `json:"endDate"`
	DurationDays             int                     `json:"durationDays"`
	Score                    int                     `json:"score"`
	AvailableUserCount       int                     `json:"availableUserCount"`
	TotalUserCount           int                     `json:"totalUserCount"`
	PreferredUserCount       int                     `json:"preferredUserCount"`
	ConflictUserCount        int                     `json:"conflictUserCount"`
	MissingResponseUserCount int                     `json:"missingResponseUserCount"`
	AvailableUsers           []DateOptionUserSummary `json:"availableUsers"`
	Conflicts                []DateOptionConflict    `json:"conflicts"`
	MissingResponses         []DateOptionUserSummary `json:"missingResponses"`
	Pros                     []string                `json:"pros"`
	Cons                     []string                `json:"cons"`
	Warnings                 []string                `json:"warnings"`
}

type DateOptionsInput struct {
	MinDays         *int
	MaxDays         *int
	SearchStartDate string
	SearchEndDate   string
	PreferWeekends  *bool
	Limit           int
}

type DateOptionsSummary struct {
	ResponseCount        int    `json:"responseCount"`
	TotalCollaborators   int    `json:"totalCollaborators"`
	RecommendedOptionID  string `json:"recommendedOptionId,omitempty"`
	MissingResponseCount int    `json:"missingResponseCount"`
}

type DateOptionsResult struct {
	Options []DateOption       `json:"options"`
	Summary DateOptionsSummary `json:"summary"`
}

type ApplyDateOptionInput struct {
	ExpectedItineraryRevision *int
	RegenerateItinerary       bool
}

type ApplyDateOptionResult struct {
	Trip                      *entity.Trip `json:"-"`
	AppliedOption             DateOption   `json:"appliedOption"`
	ItineraryStale            bool         `json:"itineraryStale"`
	RouteShifted              bool         `json:"routeShifted"`
	RegenerateItinerary       bool         `json:"regenerateItinerary"`
	Warnings                  []string     `json:"warnings"`
	ExpectedItineraryRevision int          `json:"expectedItineraryRevision"`
}

type CreateDateOptionsPollInput struct {
	Title     string
	OptionIDs []string
}

type RequestAvailabilityInput struct {
	Message string
}
