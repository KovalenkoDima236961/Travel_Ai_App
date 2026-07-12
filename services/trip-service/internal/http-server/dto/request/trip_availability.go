package request

import (
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type UpsertTripAvailability struct {
	AvailableRanges   []entity.AvailabilityDateRange `json:"availableRanges"`
	UnavailableRanges []entity.AvailabilityDateRange `json:"unavailableRanges"`
	PreferredRanges   []entity.AvailabilityDateRange `json:"preferredRanges"`
	MinTripDays       *int                           `json:"minTripDays"`
	MaxTripDays       *int                           `json:"maxTripDays"`
	Timezone          string                         `json:"timezone"`
	Notes             string                         `json:"notes"`
}

func (r UpsertTripAvailability) ToInput() appdto.UpsertTripAvailabilityInput {
	return appdto.UpsertTripAvailabilityInput{
		AvailableRanges:   r.AvailableRanges,
		UnavailableRanges: r.UnavailableRanges,
		PreferredRanges:   r.PreferredRanges,
		MinTripDays:       r.MinTripDays,
		MaxTripDays:       r.MaxTripDays,
		Timezone:          r.Timezone,
		Notes:             r.Notes,
	}
}

type GenerateDateOptions struct {
	MinDays         *int   `json:"minDays"`
	MaxDays         *int   `json:"maxDays"`
	SearchStartDate string `json:"searchStartDate"`
	SearchEndDate   string `json:"searchEndDate"`
	PreferWeekends  *bool  `json:"preferWeekends"`
	Limit           int    `json:"limit"`
}

func (r GenerateDateOptions) ToInput() appdto.DateOptionsInput {
	return appdto.DateOptionsInput{
		MinDays:         r.MinDays,
		MaxDays:         r.MaxDays,
		SearchStartDate: r.SearchStartDate,
		SearchEndDate:   r.SearchEndDate,
		PreferWeekends:  r.PreferWeekends,
		Limit:           r.Limit,
	}
}

type ApplyDateOption struct {
	ExpectedItineraryRevision *int `json:"expectedItineraryRevision"`
	RegenerateItinerary       bool `json:"regenerateItinerary"`
}

func (r ApplyDateOption) ToInput() appdto.ApplyDateOptionInput {
	return appdto.ApplyDateOptionInput{
		ExpectedItineraryRevision: r.ExpectedItineraryRevision,
		RegenerateItinerary:       r.RegenerateItinerary,
	}
}

type CreateDateOptionsPoll struct {
	Title     string   `json:"title"`
	OptionIDs []string `json:"optionIds"`
}

func (r CreateDateOptionsPoll) ToInput() appdto.CreateDateOptionsPollInput {
	return appdto.CreateDateOptionsPollInput{
		Title:     r.Title,
		OptionIDs: r.OptionIDs,
	}
}

type RequestAvailability struct {
	Message string `json:"message"`
}

func (r RequestAvailability) ToInput() appdto.RequestAvailabilityInput {
	return appdto.RequestAvailabilityInput{Message: r.Message}
}
