package request

import (
	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type SaveTripAsTemplate struct {
	Title           string   `json:"title"`
	Description     *string  `json:"description"`
	Visibility      string   `json:"visibility"`
	WorkspaceID     *string  `json:"workspaceId"`
	DestinationHint *string  `json:"destinationHint"`
	DefaultCurrency *string  `json:"defaultCurrency"`
	Tags            []string `json:"tags"`
}

func (r SaveTripAsTemplate) ToInput() (appdto.SaveTripAsTemplateInput, error) {
	workspaceID, err := parseOptionalUUID(r.WorkspaceID)
	if err != nil {
		return appdto.SaveTripAsTemplateInput{}, err
	}
	return appdto.SaveTripAsTemplateInput{
		Title:           r.Title,
		Description:     r.Description,
		Visibility:      entity.TripTemplateVisibility(r.Visibility),
		WorkspaceID:     workspaceID,
		DestinationHint: r.DestinationHint,
		DefaultCurrency: r.DefaultCurrency,
		Tags:            r.Tags,
	}, nil
}

type UpdateTripTemplate struct {
	Title           *string  `json:"title"`
	Description     *string  `json:"description"`
	DestinationHint *string  `json:"destinationHint"`
	DefaultCurrency *string  `json:"defaultCurrency"`
	Tags            []string `json:"tags"`
}

func (r UpdateTripTemplate) ToInput() appdto.UpdateTripTemplateInput {
	return appdto.UpdateTripTemplateInput{
		Title:           r.Title,
		Description:     r.Description,
		DestinationHint: r.DestinationHint,
		DefaultCurrency: r.DefaultCurrency,
		Tags:            r.Tags,
		ReplaceTags:     r.Tags != nil,
	}
}

type ArchiveTripTemplate struct {
	Reason string `json:"reason"`
}

type DuplicateTripTemplate struct {
	Title       string  `json:"title"`
	Visibility  string  `json:"visibility"`
	WorkspaceID *string `json:"workspaceId"`
}

func (r DuplicateTripTemplate) ToInput() (appdto.DuplicateTripTemplateInput, error) {
	workspaceID, err := parseOptionalUUID(r.WorkspaceID)
	if err != nil {
		return appdto.DuplicateTripTemplateInput{}, err
	}
	return appdto.DuplicateTripTemplateInput{
		Title:       r.Title,
		Visibility:  entity.TripTemplateVisibility(r.Visibility),
		WorkspaceID: workspaceID,
	}, nil
}

type CreateTripFromTemplate struct {
	Title          string   `json:"title"`
	Destination    string   `json:"destination"`
	StartDate      string   `json:"startDate"`
	WorkspaceID    *string  `json:"workspaceId"`
	Budget         *Budget  `json:"budget"`
	Travelers      *int32   `json:"travelers"`
	Pace           string   `json:"pace"`
	BudgetAmount   *float64 `json:"budgetAmount"`
	BudgetCurrency string   `json:"budgetCurrency"`
}

type Budget struct {
	Amount   *float64 `json:"amount"`
	Currency string   `json:"currency"`
}

func (r CreateTripFromTemplate) ToInput() (appdto.CreateTripFromTemplateInput, error) {
	workspaceID, err := parseOptionalUUID(r.WorkspaceID)
	if err != nil {
		return appdto.CreateTripFromTemplateInput{}, err
	}
	budgetAmount := r.BudgetAmount
	budgetCurrency := r.BudgetCurrency
	if r.Budget != nil {
		budgetAmount = r.Budget.Amount
		budgetCurrency = r.Budget.Currency
	}
	return appdto.CreateTripFromTemplateInput{
		Title:          r.Title,
		Destination:    r.Destination,
		StartDate:      r.StartDate,
		WorkspaceID:    workspaceID,
		BudgetAmount:   budgetAmount,
		BudgetCurrency: budgetCurrency,
		Travelers:      r.Travelers,
		Pace:           r.Pace,
	}, nil
}

func parseOptionalUUID(raw *string) (*uuid.UUID, error) {
	if raw == nil || *raw == "" {
		return nil, nil
	}
	parsed, err := uuid.Parse(*raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
