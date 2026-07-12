package response

import (
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/generationjobs"
)

type TripAvailability = dto.TripAvailabilityList
type TripAvailabilityResponse = dto.TripAvailabilityResponseInfo
type DateOptionsResponse = dto.DateOptionsResult
type DateOption = dto.DateOption
type TripAvailabilitySummary = dto.TripAvailabilitySummary

type ApplyDateOptionResponse struct {
	Trip           Trip                        `json:"trip"`
	AppliedOption  dto.DateOption              `json:"appliedOption"`
	ItineraryStale bool                        `json:"itineraryStale"`
	RouteShifted   bool                        `json:"routeShifted"`
	Warnings       []string                    `json:"warnings"`
	GenerationJob  *generationjobs.JobResponse `json:"generationJob,omitempty"`
}

func NewApplyDateOptionResponse(result dto.ApplyDateOptionResult, job *generationjobs.JobResponse) ApplyDateOptionResponse {
	warnings := result.Warnings
	if warnings == nil {
		warnings = []string{}
	}
	return ApplyDateOptionResponse{
		Trip:           NewTrip(result.Trip),
		AppliedOption:  result.AppliedOption,
		ItineraryStale: result.ItineraryStale,
		RouteShifted:   result.RouteShifted,
		Warnings:       warnings,
		GenerationJob:  job,
	}
}
