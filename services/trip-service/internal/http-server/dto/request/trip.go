package request

import (
	"bytes"
	"encoding/json"
	"time"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
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
	Itinerary                 json.RawMessage `json:"itinerary"`
	ExpectedItineraryRevision *int            `json:"expectedItineraryRevision"`
}

// ToInput maps the transport request to the application-level input.
func (r UpdateTripItinerary) ToInput() appdto.UpdateItineraryInput {
	return appdto.UpdateItineraryInput{
		Itinerary:                 r.Itinerary,
		ExpectedItineraryRevision: r.ExpectedItineraryRevision,
	}
}

// UpdateTripBudget is the JSON body accepted by PUT /trips/{id}/budget. The
// budget field is an object {amount, currency} to set a budget, or null to clear
// it. A budget object without an amount is treated as a clear.
type UpdateTripBudget struct {
	Budget json.RawMessage `json:"budget"`
}

// UpdateTripAccommodation is the JSON body accepted by
// PUT /trips/{id}/accommodation.
type UpdateTripAccommodation struct {
	Accommodation *aggregate.Accommodation `json:"accommodation"`
}

func (r UpdateTripAccommodation) ToInput() appdto.UpdateTripAccommodationInput {
	return appdto.UpdateTripAccommodationInput{Accommodation: r.Accommodation}
}

type budgetBody struct {
	Amount   *float64 `json:"amount"`
	Currency string   `json:"currency"`
}

// ToInput maps the transport request to the application-level input. It returns
// an error only when a present budget object is malformed JSON.
func (r UpdateTripBudget) ToInput() (appdto.UpdateTripBudgetInput, error) {
	raw := bytes.TrimSpace(r.Budget)
	if len(raw) == 0 || string(raw) == "null" {
		return appdto.UpdateTripBudgetInput{Clear: true}, nil
	}

	var body budgetBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return appdto.UpdateTripBudgetInput{}, err
	}
	if body.Amount == nil {
		return appdto.UpdateTripBudgetInput{Clear: true}, nil
	}
	return appdto.UpdateTripBudgetInput{Amount: body.Amount, Currency: body.Currency}, nil
}

// GenerateTripItinerary is the JSON body accepted by POST /trips/{id}/generate.
type GenerateTripItinerary struct {
	ExpectedItineraryRevision *int `json:"expectedItineraryRevision"`
}

func (r GenerateTripItinerary) ToInput() appdto.GenerateItineraryInput {
	return appdto.GenerateItineraryInput{
		ExpectedItineraryRevision: r.ExpectedItineraryRevision,
	}
}

// RegenerateItineraryPart is the JSON body accepted by partial itinerary
// regeneration endpoints.
type RegenerateItineraryPart struct {
	Instruction               string `json:"instruction"`
	ExpectedItineraryRevision *int   `json:"expectedItineraryRevision"`
}

// ToInput maps the transport request to the application-level input.
func (r RegenerateItineraryPart) ToInput() appdto.RegenerateItineraryPartInput {
	return appdto.RegenerateItineraryPartInput{
		Instruction:               r.Instruction,
		ExpectedItineraryRevision: r.ExpectedItineraryRevision,
	}
}

type RestoreItineraryVersion struct {
	ExpectedItineraryRevision *int `json:"expectedItineraryRevision"`
}

func (r RestoreItineraryVersion) ToInput() appdto.RestoreItineraryVersionInput {
	return appdto.RestoreItineraryVersionInput{
		ExpectedItineraryRevision: r.ExpectedItineraryRevision,
	}
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

type InviteTripCollaborator struct {
	Email string                  `json:"email"`
	Role  entity.CollaboratorRole `json:"role"`
}

func (r InviteTripCollaborator) ToInput() appdto.InviteTripCollaboratorInput {
	return appdto.InviteTripCollaboratorInput{
		Email: r.Email,
		Role:  r.Role,
	}
}

type UpdateTripCollaborator struct {
	Role entity.CollaboratorRole `json:"role"`
}

func (r UpdateTripCollaborator) ToInput() appdto.UpdateTripCollaboratorInput {
	return appdto.UpdateTripCollaboratorInput{Role: r.Role}
}

// PublicShareUnlock is the JSON body accepted by
// POST /public/trips/{shareToken}/unlock.
type PublicShareUnlock struct {
	Password string `json:"password"`
}
