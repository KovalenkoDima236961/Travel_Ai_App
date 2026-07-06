package dto

import (
	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type TemplateVisibilityFilter string

const (
	TemplateVisibilityFilterAll       TemplateVisibilityFilter = "all"
	TemplateVisibilityFilterPrivate   TemplateVisibilityFilter = "private"
	TemplateVisibilityFilterWorkspace TemplateVisibilityFilter = "workspace"
)

type ListTripTemplatesInput struct {
	Limit       int
	Offset      int
	Visibility  entity.TripTemplateVisibility
	Status      entity.TripTemplateStatus
	WorkspaceID *uuid.UUID
	Tag         string
	Query       string
}

type SaveTripAsTemplateInput struct {
	Title           string
	Description     *string
	Visibility      entity.TripTemplateVisibility
	WorkspaceID     *uuid.UUID
	DestinationHint *string
	DefaultCurrency *string
	Tags            []string
}

type UpdateTripTemplateInput struct {
	Title           *string
	Description     *string
	DestinationHint *string
	DefaultCurrency *string
	Tags            []string
	ReplaceTags     bool
}

type DuplicateTripTemplateInput struct {
	Title       string
	Visibility  entity.TripTemplateVisibility
	WorkspaceID *uuid.UUID
}

type CreateTripFromTemplateInput struct {
	Title          string
	Destination    string
	StartDate      string
	WorkspaceID    *uuid.UUID
	BudgetAmount   *float64
	BudgetCurrency string
	Travelers      *int32
	Pace           string
}

// CreateTemplateAdaptationInput is the validated request to start an AI template
// adaptation job. It creates a draft trip re-targeted to a new destination.
type CreateTemplateAdaptationInput struct {
	Title                   string
	Destination             string
	StartDate               string
	DurationDays            int
	WorkspaceID             *uuid.UUID
	BudgetAmount            *float64
	BudgetCurrency          string
	Travelers               *int32
	Pace                    string
	Interests               []string
	Avoid                   []string
	SpecialInstructions     string
	FallbackToDeterministic bool
}

type TripTemplateAccess struct {
	Role         string
	Source       string
	CanUse       bool
	CanEdit      bool
	CanArchive   bool
	CanDuplicate bool
}

type TripTemplateWithAccess struct {
	Template entity.TripTemplate
	Access   TripTemplateAccess
}
