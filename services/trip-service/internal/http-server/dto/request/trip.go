// Package request holds the inbound HTTP payloads for the trip endpoints and
// their mapping to application-level inputs.
package request

import (
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
