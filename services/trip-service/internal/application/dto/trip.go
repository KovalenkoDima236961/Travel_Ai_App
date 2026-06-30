package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

// CreateTripInput is the validated, application-level representation of a create
// request.
type CreateTripInput struct {
	Destination    string
	StartDate      string
	Days           int32
	BudgetAmount   *float64
	BudgetCurrency string
	Travelers      int32
	Interests      []string
	Pace           string
}

// UpdateItineraryInput is the application-level payload for replacing a trip's
// itinerary JSON.
type UpdateItineraryInput struct {
	Itinerary                 json.RawMessage
	ExpectedItineraryRevision *int
}

// UpdateTripBudgetInput is the application-level payload for a trip budget
// update. Clear is true when the request explicitly clears the budget
// (budget: null or a budget object without an amount); otherwise Amount and
// Currency carry the new budget.
type UpdateTripBudgetInput struct {
	Amount   *float64
	Currency string
	Clear    bool
}

// UpdateTripAccommodationInput is the application-level payload for creating or
// replacing a trip's structured accommodation.
type UpdateTripAccommodationInput struct {
	Accommodation *aggregate.Accommodation
}

// GenerateItineraryInput is the application-level payload for full itinerary
// generation.
type GenerateItineraryInput struct {
	ExpectedItineraryRevision *int
}

// RegenerateItineraryPartInput is the application-level payload for partial AI
// regeneration. Instruction is optional and normalized by the service.
type RegenerateItineraryPartInput struct {
	Instruction               string
	ExpectedItineraryRevision *int
}

// RestoreItineraryVersionInput is the application-level payload for restoring a
// saved itinerary version.
type RestoreItineraryVersionInput struct {
	ExpectedItineraryRevision *int
}

// CreateTripShareInput holds optional initial controls for a new or re-enabled
// public share link.
type CreateTripShareInput struct {
	ExpiresAt *time.Time
	Password  string
}

// UpdateTripShareInput holds owner-controlled settings for an existing share.
type UpdateTripShareInput struct {
	ExpiresAt       *time.Time
	ClearExpiration bool
	Password        string
	ClearPassword   bool
}

// TripShareInfo is the application-level share-link status returned to the
// owning user. Disabled shares intentionally omit the token/URL.
type TripShareInfo struct {
	ShareToken       string
	ShareURL         string
	Enabled          bool
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
	DisabledAt       *time.Time
	ExpiresAt        *time.Time
	Expired          bool
	PasswordRequired bool
}

type PublicShareStatus struct {
	Available        bool
	PasswordRequired bool
	Expired          bool
}

type PublicShareUnlockResponse struct {
	AccessToken string
	ExpiresAt   time.Time
}

type InviteTripCollaboratorInput struct {
	Email string
	Role  entity.CollaboratorRole
}

type UpdateTripCollaboratorInput struct {
	Role entity.CollaboratorRole
}

type UserLookupResult struct {
	UserID      uuid.UUID
	Email       string
	DisplayName string
}

type TripCollaboratorInfo struct {
	Collaborator entity.TripCollaborator
	Email        *string
	DisplayName  *string
}

type CollaborationInvitation struct {
	CollaboratorID  uuid.UUID
	TripID          uuid.UUID
	Destination     string
	Role            entity.CollaboratorRole
	InvitedByUserID uuid.UUID
	InvitedAt       time.Time
}
