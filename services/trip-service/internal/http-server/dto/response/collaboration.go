package response

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type TripInvitation struct {
	ID            uuid.UUID                   `json:"id"`
	TripID        uuid.UUID                   `json:"tripId"`
	InviterUserID uuid.UUID                   `json:"inviterUserId"`
	InvitedUserID *uuid.UUID                  `json:"invitedUserId,omitempty"`
	Email         string                      `json:"email"`
	Role          entity.CollaboratorRole     `json:"role"`
	Status        entity.TripInvitationStatus `json:"status"`
	Message       string                      `json:"message,omitempty"`
	ExpiresAt     time.Time                   `json:"expiresAt"`
	AcceptedAt    *time.Time                  `json:"acceptedAt,omitempty"`
	DeclinedAt    *time.Time                  `json:"declinedAt,omitempty"`
	RevokedAt     *time.Time                  `json:"revokedAt,omitempty"`
	CreatedAt     time.Time                   `json:"createdAt"`
	UpdatedAt     time.Time                   `json:"updatedAt"`
}

type TripMember struct {
	UserID      uuid.UUID               `json:"userId"`
	Email       string                  `json:"email,omitempty"`
	DisplayName string                  `json:"displayName,omitempty"`
	Role        string                  `json:"role"`
	Status      entity.TripMemberStatus `json:"status"`
	JoinedAt    *time.Time              `json:"joinedAt,omitempty"`
	InvitedBy   *uuid.UUID              `json:"invitedBy,omitempty"`
	LastSeenAt  *time.Time              `json:"lastSeenAt,omitempty"`
	Permissions map[string]bool         `json:"permissions"`
	IsSelf      bool                    `json:"isSelf"`
}

type TripSuggestion struct {
	ID                       uuid.UUID                       `json:"id"`
	TripID                   uuid.UUID                       `json:"tripId"`
	AuthorUserID             uuid.UUID                       `json:"authorUserId"`
	SuggestionType           entity.TripSuggestionType       `json:"suggestionType"`
	TargetType               entity.TripSuggestionTargetType `json:"targetType"`
	TargetID                 string                          `json:"targetId,omitempty"`
	Status                   entity.TripSuggestionStatus     `json:"status"`
	Before                   json.RawMessage                 `json:"before,omitempty"`
	After                    json.RawMessage                 `json:"after,omitempty"`
	Comment                  string                          `json:"comment,omitempty"`
	Metadata                 map[string]any                  `json:"metadata,omitempty"`
	AppliedItineraryRevision *int                            `json:"appliedItineraryRevision,omitempty"`
	ResolvedAt               *time.Time                      `json:"resolvedAt,omitempty"`
	ResolvedByUserID         *uuid.UUID                      `json:"resolvedByUserId,omitempty"`
	CreatedAt                time.Time                       `json:"createdAt"`
	UpdatedAt                time.Time                       `json:"updatedAt"`
	IsAuthor                 bool                            `json:"isAuthor"`
	CanResolve               bool                            `json:"canResolve"`
}

type TripVoteSummary struct {
	TargetType  entity.TripVoteTargetType `json:"targetType"`
	TargetID    string                    `json:"targetId"`
	Counts      map[string]int            `json:"counts"`
	CurrentVote *string                   `json:"currentVote,omitempty"`
}

type ListTripInvitations struct {
	Items []TripInvitation `json:"items"`
}

type ListTripMembers struct {
	Items []TripMember `json:"items"`
}

type ListTripSuggestions struct {
	Items []TripSuggestion `json:"items"`
}

type ListTripVoteSummaries struct {
	Items []TripVoteSummary `json:"items"`
}

func NewTripInvitation(info appdto.TripInvitationInfo) TripInvitation {
	invitation := info.Invitation
	return TripInvitation{
		ID:            invitation.ID,
		TripID:        invitation.TripID,
		InviterUserID: invitation.InviterUserID,
		InvitedUserID: invitation.InvitedUserID,
		Email:         invitation.Email,
		Role:          invitation.Role,
		Status:        invitation.Status,
		Message:       invitation.Message,
		ExpiresAt:     invitation.ExpiresAt,
		AcceptedAt:    invitation.AcceptedAt,
		DeclinedAt:    invitation.DeclinedAt,
		RevokedAt:     invitation.RevokedAt,
		CreatedAt:     invitation.CreatedAt,
		UpdatedAt:     invitation.UpdatedAt,
	}
}

func NewListTripInvitations(infos []appdto.TripInvitationInfo) ListTripInvitations {
	items := make([]TripInvitation, 0, len(infos))
	for _, info := range infos {
		items = append(items, NewTripInvitation(info))
	}
	return ListTripInvitations{Items: items}
}

func NewTripMember(info appdto.TripMemberInfo) TripMember {
	member := info.Member
	permissions := map[string]bool{}
	for key, allowed := range member.Permissions {
		permissions[key] = allowed
	}
	return TripMember{
		UserID:      member.UserID,
		Email:       member.Email,
		DisplayName: member.DisplayName,
		Role:        member.Role,
		Status:      member.Status,
		JoinedAt:    member.JoinedAt,
		InvitedBy:   member.InvitedBy,
		LastSeenAt:  member.LastSeenAt,
		Permissions: permissions,
		IsSelf:      info.IsSelf,
	}
}

func NewListTripMembers(infos []appdto.TripMemberInfo) ListTripMembers {
	items := make([]TripMember, 0, len(infos))
	for _, info := range infos {
		items = append(items, NewTripMember(info))
	}
	return ListTripMembers{Items: items}
}

func NewTripSuggestion(info appdto.TripSuggestionInfo) TripSuggestion {
	suggestion := info.Suggestion
	return TripSuggestion{
		ID:                       suggestion.ID,
		TripID:                   suggestion.TripID,
		AuthorUserID:             suggestion.AuthorUserID,
		SuggestionType:           suggestion.SuggestionType,
		TargetType:               suggestion.TargetType,
		TargetID:                 suggestion.TargetID,
		Status:                   suggestion.Status,
		Before:                   cloneRawJSON(suggestion.Before),
		After:                    cloneRawJSON(suggestion.After),
		Comment:                  suggestion.Comment,
		Metadata:                 suggestion.Metadata,
		AppliedItineraryRevision: suggestion.AppliedItineraryRevision,
		ResolvedAt:               suggestion.ResolvedAt,
		ResolvedByUserID:         suggestion.ResolvedByUserID,
		CreatedAt:                suggestion.CreatedAt,
		UpdatedAt:                suggestion.UpdatedAt,
		IsAuthor:                 info.IsAuthor,
		CanResolve:               info.CanResolve,
	}
}

func NewListTripSuggestions(infos []appdto.TripSuggestionInfo) ListTripSuggestions {
	items := make([]TripSuggestion, 0, len(infos))
	for _, info := range infos {
		items = append(items, NewTripSuggestion(info))
	}
	return ListTripSuggestions{Items: items}
}

func NewTripVoteSummary(summary entity.TripVoteSummary) TripVoteSummary {
	counts := make(map[string]int, len(summary.Counts))
	for voteType, count := range summary.Counts {
		counts[string(voteType)] = count
	}
	var current *string
	if summary.CurrentVote != nil {
		value := string(*summary.CurrentVote)
		current = &value
	}
	return TripVoteSummary{
		TargetType:  summary.TargetType,
		TargetID:    summary.TargetID,
		Counts:      counts,
		CurrentVote: current,
	}
}

func NewListTripVoteSummaries(summaries []entity.TripVoteSummary) ListTripVoteSummaries {
	items := make([]TripVoteSummary, 0, len(summaries))
	for _, summary := range summaries {
		items = append(items, NewTripVoteSummary(summary))
	}
	return ListTripVoteSummaries{Items: items}
}

func cloneRawJSON(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	out := make([]byte, len(raw))
	copy(out, raw)
	return out
}
