package request

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type CreateTripInvitation struct {
	Email     string                  `json:"email"`
	Role      entity.CollaboratorRole `json:"role"`
	Message   string                  `json:"message"`
	ExpiresAt *time.Time              `json:"expiresAt"`
}

func (r CreateTripInvitation) ToInput() appdto.CreateTripInvitationInput {
	return appdto.CreateTripInvitationInput{
		Email:     r.Email,
		Role:      r.Role,
		Message:   r.Message,
		ExpiresAt: r.ExpiresAt,
	}
}

type ResendTripInvitation struct {
	Message   string     `json:"message"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

func (r ResendTripInvitation) ToInput() appdto.CreateTripInvitationInput {
	return appdto.CreateTripInvitationInput{
		Message:   r.Message,
		ExpiresAt: r.ExpiresAt,
	}
}

type TransferTripOwnership struct {
	NewOwnerUserID uuid.UUID `json:"newOwnerUserId"`
}

func (r TransferTripOwnership) ToInput() appdto.TransferTripOwnershipInput {
	return appdto.TransferTripOwnershipInput{NewOwnerUserID: r.NewOwnerUserID}
}

type CreateTripSuggestion struct {
	SuggestionType entity.TripSuggestionType       `json:"suggestionType"`
	TargetType     entity.TripSuggestionTargetType `json:"targetType"`
	TargetID       string                          `json:"targetId"`
	Before         json.RawMessage                 `json:"before,omitempty"`
	After          json.RawMessage                 `json:"after,omitempty"`
	Comment        string                          `json:"comment"`
	Metadata       map[string]any                  `json:"metadata,omitempty"`
}

func (r CreateTripSuggestion) ToInput() appdto.CreateTripSuggestionInput {
	return appdto.CreateTripSuggestionInput{
		SuggestionType: r.SuggestionType,
		TargetType:     r.TargetType,
		TargetID:       r.TargetID,
		Before:         r.Before,
		After:          r.After,
		Comment:        r.Comment,
		Metadata:       r.Metadata,
	}
}

type UpdateTripSuggestionStatus struct {
	ExpectedItineraryRevision *int `json:"expectedItineraryRevision"`
}

func (r UpdateTripSuggestionStatus) ToInput() appdto.UpdateTripSuggestionStatusInput {
	return appdto.UpdateTripSuggestionStatusInput{ExpectedItineraryRevision: r.ExpectedItineraryRevision}
}

type SetTripVote struct {
	TargetType entity.TripVoteTargetType `json:"targetType"`
	TargetID   string                    `json:"targetId"`
	VoteType   entity.TripVoteType       `json:"voteType"`
	Metadata   map[string]any            `json:"metadata,omitempty"`
}

func (r SetTripVote) ToInput() appdto.SetTripVoteInput {
	return appdto.SetTripVoteInput{
		TargetType: r.TargetType,
		TargetID:   r.TargetID,
		VoteType:   r.VoteType,
		Metadata:   r.Metadata,
	}
}
