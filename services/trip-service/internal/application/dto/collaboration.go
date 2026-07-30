package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type CreateTripInvitationInput struct {
	Email     string
	Role      entity.CollaboratorRole
	Message   string
	ExpiresAt *time.Time
}

type TripInvitationInfo struct {
	Invitation entity.TripInvitation
	Email      *string
}

type TripMemberInfo struct {
	Member entity.TripMember
	IsSelf bool
}

type TransferTripOwnershipInput struct {
	NewOwnerUserID uuid.UUID
}

type CreateTripSuggestionInput struct {
	SuggestionType entity.TripSuggestionType
	TargetType     entity.TripSuggestionTargetType
	TargetID       string
	Before         json.RawMessage
	After          json.RawMessage
	Comment        string
	Metadata       map[string]any
}

type UpdateTripSuggestionStatusInput struct {
	ExpectedItineraryRevision *int
}

type TripSuggestionInfo struct {
	Suggestion entity.TripSuggestion
	IsAuthor   bool
	CanResolve bool
}

type SetTripVoteInput struct {
	TargetType entity.TripVoteTargetType
	TargetID   string
	VoteType   entity.TripVoteType
	Metadata   map[string]any
}

type ListTripVotesInput struct {
	TargetType entity.TripVoteTargetType
	TargetID   string
}
