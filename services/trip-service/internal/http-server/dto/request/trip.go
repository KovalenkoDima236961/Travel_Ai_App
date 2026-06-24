package request

import (
	"encoding/json"
	"time"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
)

// CreateTrip is the JSON body accepted by POST /trips. Validation tags are
// enforced by the project's validation package in the handler.
type CreateTrip struct {
	Destination    string   `json:"destination" validate:"required"`
	StartDate      string   `json:"startDate" validate:"omitempty,datetime=2006-01-02"`
	Days           int32    `json:"days" validate:"required,gte=1,lte=30"`
	BudgetAmount   *float64 `json:"budgetAmount" validate:"omitempty,gte=0"`
	BudgetCurrency string   `json:"budgetCurrency" validate:"omitempty,len=3"`
	Travelers      int32    `json:"travelers" validate:"required,gte=1"`
	Interests      []string `json:"interests" validate:"omitempty,dive,required"`
	Pace           string   `json:"pace" validate:"omitempty,oneof=relaxed balanced packed"`
}

// ToInput maps the transport request to the application-level input.
func (r CreateTrip) ToInput() appdto.CreateTripInput {
	return appdto.CreateTripInput{
		Destination:    r.Destination,
		StartDate:      r.StartDate,
		Days:           r.Days,
		BudgetAmount:   r.BudgetAmount,
		BudgetCurrency: r.BudgetCurrency,
		Travelers:      r.Travelers,
		Interests:      r.Interests,
		Pace:           r.Pace,
	}
}

// UpdateTripItinerary is the JSON body accepted by PUT /trips/{id}/itinerary.
type UpdateTripItinerary struct {
	Itinerary json.RawMessage `json:"itinerary"`
}

// ToInput maps the transport request to the application-level input.
func (r UpdateTripItinerary) ToInput() appdto.UpdateItineraryInput {
	return appdto.UpdateItineraryInput{Itinerary: r.Itinerary}
}

// RegenerateItineraryPart is the JSON body accepted by partial itinerary
// regeneration endpoints.
type RegenerateItineraryPart struct {
	Instruction string `json:"instruction"`
}

// ToInput maps the transport request to the application-level input.
func (r RegenerateItineraryPart) ToInput() appdto.RegenerateItineraryPartInput {
	return appdto.RegenerateItineraryPartInput{Instruction: r.Instruction}
}

// CreateTripShare is the optional JSON body accepted by POST /trips/{id}/share.
type CreateTripShare struct {
	ExpiresAt *time.Time `json:"expiresAt"`
	Password  string     `json:"password"`
}

func (r CreateTripShare) ToInput() appdto.CreateTripShareInput {
	return appdto.CreateTripShareInput{
		ExpiresAt: r.ExpiresAt,
		Password:  r.Password,
	}
}

// UpdateTripShareSettings is the JSON body accepted by PATCH /trips/{id}/share.
type UpdateTripShareSettings struct {
	ExpiresAt       *time.Time `json:"expiresAt"`
	ClearExpiration bool       `json:"clearExpiration"`
	Password        string     `json:"password"`
	ClearPassword   bool       `json:"clearPassword"`
}

func (r UpdateTripShareSettings) ToInput() appdto.UpdateTripShareInput {
	return appdto.UpdateTripShareInput{
		ExpiresAt:       r.ExpiresAt,
		ClearExpiration: r.ClearExpiration,
		Password:        r.Password,
		ClearPassword:   r.ClearPassword,
	}
}

// PublicShareUnlock is the JSON body accepted by
// POST /public/trips/{shareToken}/unlock.
type PublicShareUnlock struct {
	Password string `json:"password"`
}
