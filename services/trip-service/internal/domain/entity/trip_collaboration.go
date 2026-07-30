package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TripInvitationStatus string

const (
	TripInvitationStatusPending  TripInvitationStatus = "pending"
	TripInvitationStatusAccepted TripInvitationStatus = "accepted"
	TripInvitationStatusDeclined TripInvitationStatus = "declined"
	TripInvitationStatusExpired  TripInvitationStatus = "expired"
	TripInvitationStatusRevoked  TripInvitationStatus = "revoked"
)

type TripInvitation struct {
	ID            uuid.UUID
	TripID        uuid.UUID
	InviterUserID uuid.UUID
	InvitedUserID *uuid.UUID
	Email         string
	Role          CollaboratorRole
	Status        TripInvitationStatus
	Message       string
	ExpiresAt     time.Time
	AcceptedAt    *time.Time
	DeclinedAt    *time.Time
	RevokedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type TripMemberStatus string

const (
	TripMemberStatusActive  TripMemberStatus = "active"
	TripMemberStatusInvited TripMemberStatus = "invited"
	TripMemberStatusRemoved TripMemberStatus = "removed"
)

type TripMember struct {
	UserID      uuid.UUID
	Email       string
	DisplayName string
	Role        string
	Status      TripMemberStatus
	JoinedAt    *time.Time
	InvitedBy   *uuid.UUID
	LastSeenAt  *time.Time
	Permissions map[string]bool
}

type TripSuggestionType string

const (
	TripSuggestionActivityReplacement TripSuggestionType = "activity_replacement"
	TripSuggestionTimeChange          TripSuggestionType = "time_change"
	TripSuggestionBudgetAdjustment    TripSuggestionType = "budget_adjustment"
	TripSuggestionRouteChange         TripSuggestionType = "route_change"
	TripSuggestionNote                TripSuggestionType = "note"
)

func (t TripSuggestionType) Valid() bool {
	switch t {
	case TripSuggestionActivityReplacement,
		TripSuggestionTimeChange,
		TripSuggestionBudgetAdjustment,
		TripSuggestionRouteChange,
		TripSuggestionNote:
		return true
	default:
		return false
	}
}

type TripSuggestionTargetType string

const (
	TripSuggestionTargetTrip          TripSuggestionTargetType = "trip"
	TripSuggestionTargetDay           TripSuggestionTargetType = "day"
	TripSuggestionTargetItineraryItem TripSuggestionTargetType = "itinerary_item"
	TripSuggestionTargetBudgetItem    TripSuggestionTargetType = "budget_item"
	TripSuggestionTargetRoute         TripSuggestionTargetType = "route"
	TripSuggestionTargetAttachment    TripSuggestionTargetType = "attachment"
)

func (t TripSuggestionTargetType) Valid() bool {
	switch t {
	case TripSuggestionTargetTrip,
		TripSuggestionTargetDay,
		TripSuggestionTargetItineraryItem,
		TripSuggestionTargetBudgetItem,
		TripSuggestionTargetRoute,
		TripSuggestionTargetAttachment:
		return true
	default:
		return false
	}
}

type TripSuggestionStatus string

const (
	TripSuggestionStatusOpen     TripSuggestionStatus = "open"
	TripSuggestionStatusAccepted TripSuggestionStatus = "accepted"
	TripSuggestionStatusRejected TripSuggestionStatus = "rejected"
	TripSuggestionStatusResolved TripSuggestionStatus = "resolved"
)

type TripSuggestion struct {
	ID                       uuid.UUID
	TripID                   uuid.UUID
	AuthorUserID             uuid.UUID
	SuggestionType           TripSuggestionType
	TargetType               TripSuggestionTargetType
	TargetID                 string
	Status                   TripSuggestionStatus
	Before                   json.RawMessage
	After                    json.RawMessage
	Comment                  string
	Metadata                 map[string]any
	AppliedItineraryRevision *int
	CreatedAt                time.Time
	UpdatedAt                time.Time
	ResolvedAt               *time.Time
	ResolvedByUserID         *uuid.UUID
}

type TripVoteTargetType string

const (
	TripVoteTargetActivity    TripVoteTargetType = "activity"
	TripVoteTargetRestaurant  TripVoteTargetType = "restaurant"
	TripVoteTargetHotel       TripVoteTargetType = "hotel"
	TripVoteTargetDestination TripVoteTargetType = "destination"
	TripVoteTargetSuggestion  TripVoteTargetType = "suggestion"
)

func (t TripVoteTargetType) Valid() bool {
	switch t {
	case TripVoteTargetActivity,
		TripVoteTargetRestaurant,
		TripVoteTargetHotel,
		TripVoteTargetDestination,
		TripVoteTargetSuggestion:
		return true
	default:
		return false
	}
}

type TripVoteType string

const (
	TripVoteThumbsUp   TripVoteType = "thumbs_up"
	TripVoteThumbsDown TripVoteType = "thumbs_down"
	TripVoteHeart      TripVoteType = "heart"
	TripVoteStar       TripVoteType = "star"
)

func (t TripVoteType) Valid() bool {
	switch t {
	case TripVoteThumbsUp, TripVoteThumbsDown, TripVoteHeart, TripVoteStar:
		return true
	default:
		return false
	}
}

type TripVote struct {
	ID         uuid.UUID
	TripID     uuid.UUID
	TargetType TripVoteTargetType
	TargetID   string
	UserID     uuid.UUID
	VoteType   TripVoteType
	Metadata   map[string]any
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type TripVoteSummary struct {
	TargetType  TripVoteTargetType
	TargetID    string
	Counts      map[TripVoteType]int
	CurrentVote *TripVoteType
}
