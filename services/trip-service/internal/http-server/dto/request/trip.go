package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

// CreateTrip is the JSON body accepted by POST /trips. Validation tags are
// enforced by the project's validation package in the handler.
type CreateTrip struct {
	Destination    string   `json:"destination" validate:"required"`
	WorkspaceID    *string  `json:"workspaceId" validate:"omitempty,uuid"`
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
	var workspaceID *uuid.UUID
	if r.WorkspaceID != nil {
		parsed := uuid.MustParse(*r.WorkspaceID)
		workspaceID = &parsed
	}
	return appdto.CreateTripInput{
		Destination:    r.Destination,
		WorkspaceID:    workspaceID,
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

type CreateTripTraveler struct {
	Name         string                  `json:"name"`
	Email        *string                 `json:"email"`
	LinkedUserID *string                 `json:"linkedUserId"`
	Role         entity.TripTravelerRole `json:"role"`
}

func (r CreateTripTraveler) ToInput() (appdto.CreateTripTravelerInput, error) {
	var linkedUserID *uuid.UUID
	if r.LinkedUserID != nil && strings.TrimSpace(*r.LinkedUserID) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(*r.LinkedUserID))
		if err != nil {
			return appdto.CreateTripTravelerInput{}, fmt.Errorf("invalid linkedUserId")
		}
		linkedUserID = &parsed
	}
	return appdto.CreateTripTravelerInput{
		Name:         r.Name,
		Email:        r.Email,
		LinkedUserID: linkedUserID,
		Role:         r.Role,
	}, nil
}

type UpdateTripTraveler struct {
	Name  *string                  `json:"name"`
	Email *string                  `json:"email"`
	Role  *entity.TripTravelerRole `json:"role"`
}

func (r UpdateTripTraveler) ToInput() appdto.UpdateTripTravelerInput {
	return appdto.UpdateTripTravelerInput{
		Name:  r.Name,
		Email: r.Email,
		Role:  r.Role,
	}
}

type UpdateItemCostSplit struct {
	ExpectedItineraryRevision *int                     `json:"expectedItineraryRevision"`
	Split                     *aggregate.CostSplitRule `json:"split"`
}

func (r UpdateItemCostSplit) ToInput() appdto.UpdateItemCostSplitInput {
	return appdto.UpdateItemCostSplitInput{
		ExpectedItineraryRevision: r.ExpectedItineraryRevision,
		Split:                     r.Split,
	}
}

type UpdateAccommodationCostSplit struct {
	Split *aggregate.CostSplitRule `json:"split"`
}

func (r UpdateAccommodationCostSplit) ToInput() appdto.UpdateAccommodationCostSplitInput {
	return appdto.UpdateAccommodationCostSplitInput{Split: r.Split}
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

type CreateWorkspaceBudget struct {
	Name        string   `json:"name"`
	Description *string  `json:"description"`
	Amount      *float64 `json:"amount"`
	Currency    string   `json:"currency"`
	PeriodStart *string  `json:"periodStart"`
	PeriodEnd   *string  `json:"periodEnd"`
	IsPrimary   *bool    `json:"isPrimary"`
}

func (r CreateWorkspaceBudget) ToInput() (appdto.CreateWorkspaceBudgetInput, error) {
	if r.Amount == nil {
		return appdto.CreateWorkspaceBudgetInput{}, fmt.Errorf("amount is required")
	}
	periodStart, err := parseOptionalDate(r.PeriodStart, "periodStart")
	if err != nil {
		return appdto.CreateWorkspaceBudgetInput{}, err
	}
	periodEnd, err := parseOptionalDate(r.PeriodEnd, "periodEnd")
	if err != nil {
		return appdto.CreateWorkspaceBudgetInput{}, err
	}
	return appdto.CreateWorkspaceBudgetInput{
		Name:        r.Name,
		Description: r.Description,
		Amount:      *r.Amount,
		Currency:    r.Currency,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		IsPrimary:   r.IsPrimary,
	}, nil
}

type ArchiveWorkspaceBudget struct {
	Reason string `json:"reason"`
}

func DecodeUpdateWorkspaceBudget(body io.Reader) (appdto.UpdateWorkspaceBudgetInput, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return appdto.UpdateWorkspaceBudgetInput{}, err
	}
	var out appdto.UpdateWorkspaceBudgetInput
	for key, value := range raw {
		switch key {
		case "name":
			var name string
			if err := json.Unmarshal(value, &name); err != nil {
				return appdto.UpdateWorkspaceBudgetInput{}, fmt.Errorf("invalid name")
			}
			out.Name = &name
		case "description":
			out.DescriptionSet = true
			if string(bytes.TrimSpace(value)) == "null" {
				out.Description = nil
				continue
			}
			var description string
			if err := json.Unmarshal(value, &description); err != nil {
				return appdto.UpdateWorkspaceBudgetInput{}, fmt.Errorf("invalid description")
			}
			out.Description = &description
		case "amount":
			var amount float64
			if err := json.Unmarshal(value, &amount); err != nil {
				return appdto.UpdateWorkspaceBudgetInput{}, fmt.Errorf("invalid amount")
			}
			out.Amount = &amount
		case "currency":
			var currency string
			if err := json.Unmarshal(value, &currency); err != nil {
				return appdto.UpdateWorkspaceBudgetInput{}, fmt.Errorf("invalid currency")
			}
			out.Currency = &currency
		case "periodStart":
			out.PeriodStartSet = true
			periodStart, err := parseNullableDate(value, "periodStart")
			if err != nil {
				return appdto.UpdateWorkspaceBudgetInput{}, err
			}
			out.PeriodStart = periodStart
		case "periodEnd":
			out.PeriodEndSet = true
			periodEnd, err := parseNullableDate(value, "periodEnd")
			if err != nil {
				return appdto.UpdateWorkspaceBudgetInput{}, err
			}
			out.PeriodEnd = periodEnd
		case "isPrimary":
			var isPrimary bool
			if err := json.Unmarshal(value, &isPrimary); err != nil {
				return appdto.UpdateWorkspaceBudgetInput{}, fmt.Errorf("invalid isPrimary")
			}
			out.IsPrimary = &isPrimary
		}
	}
	return out, nil
}

func parseOptionalDate(value *string, field string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*value))
	if err != nil {
		return nil, fmt.Errorf("%s must be in YYYY-MM-DD format", field)
	}
	return &parsed, nil
}

func parseNullableDate(raw json.RawMessage, field string) (*time.Time, error) {
	if string(bytes.TrimSpace(raw)) == "null" {
		return nil, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, fmt.Errorf("%s must be in YYYY-MM-DD format", field)
	}
	return parseOptionalDate(&value, field)
}
