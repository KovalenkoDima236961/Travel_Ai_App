package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
	tripobs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/observability"
)

const (
	defaultInvitationTTL = 14 * 24 * time.Hour
	maxInvitationTTL     = 90 * 24 * time.Hour
	maxInviteMessage     = 500
	maxSuggestionComment = 2000
	maxTargetIDLength    = 200
)

type tripInvitationRepository interface {
	CreateTripInvitation(ctx context.Context, invitation *entity.TripInvitation) (*entity.TripInvitation, error)
	ListTripInvitationsByTrip(ctx context.Context, tripID uuid.UUID) ([]entity.TripInvitation, error)
	ListPendingTripInvitationsForUser(ctx context.Context, userID uuid.UUID, email string) ([]entity.TripInvitation, error)
	GetTripInvitationByID(ctx context.Context, tripID, invitationID uuid.UUID) (*entity.TripInvitation, error)
	LinkTripInvitationToUser(ctx context.Context, invitationID, userID uuid.UUID, email string) (*entity.TripInvitation, error)
	UpdateTripInvitationStatus(ctx context.Context, tripID, invitationID uuid.UUID, status entity.TripInvitationStatus, actorUserID *uuid.UUID, appliedAt time.Time) (*entity.TripInvitation, error)
	ResendTripInvitation(ctx context.Context, tripID, invitationID, inviterUserID uuid.UUID, expiresAt time.Time, message string) (*entity.TripInvitation, error)
}

type tripOwnershipRepository interface {
	TransferTripOwnership(ctx context.Context, tripID, currentOwnerID, nextOwnerID uuid.UUID) (*entity.Trip, error)
	RemoveTripCollaboratorByUser(ctx context.Context, tripID, userID uuid.UUID) (*entity.TripCollaborator, error)
}

type tripSuggestionRepository interface {
	CreateTripSuggestion(ctx context.Context, suggestion *entity.TripSuggestion) (*entity.TripSuggestion, error)
	ListTripSuggestionsByTrip(ctx context.Context, tripID uuid.UUID, status *entity.TripSuggestionStatus) ([]entity.TripSuggestion, error)
	GetTripSuggestionByID(ctx context.Context, tripID, suggestionID uuid.UUID) (*entity.TripSuggestion, error)
	UpdateTripSuggestionStatus(ctx context.Context, tripID, suggestionID, actorUserID uuid.UUID, status entity.TripSuggestionStatus, appliedRevision *int) (*entity.TripSuggestion, error)
}

type tripVoteRepository interface {
	UpsertTripVote(ctx context.Context, vote *entity.TripVote) (*entity.TripVote, error)
	GetTripVoteByID(ctx context.Context, tripID, voteID uuid.UUID) (*entity.TripVote, error)
	DeleteTripVote(ctx context.Context, tripID, voteID, userID uuid.UUID) error
	ListTripVotesByTrip(ctx context.Context, tripID uuid.UUID) ([]entity.TripVote, error)
	ListTripVotesByTarget(ctx context.Context, tripID uuid.UUID, targetType entity.TripVoteTargetType, targetID string) ([]entity.TripVote, error)
}

type commentThreadRepository interface {
	ResolveItineraryComment(ctx context.Context, tripID, commentID, actorUserID uuid.UUID) (*entity.ItineraryComment, error)
	ReopenItineraryComment(ctx context.Context, tripID, commentID uuid.UUID) (*entity.ItineraryComment, error)
}

func (s *Service) CreateTripInvitation(ctx context.Context, tripID uuid.UUID, in appdto.CreateTripInvitationInput) (appdto.TripInvitationInfo, error) {
	repo, ok := s.repo.(tripInvitationRepository)
	if !ok {
		return appdto.TripInvitationInfo{}, apperrs.NewDependencyError("trip invitation repository is not configured")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}
	trip, _, err := s.requireOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}
	ownerID, err := tripOwnerID(trip)
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}

	role, err := normalizeCollaboratorRole(in.Role)
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}
	email, err := normalizeCollaboratorEmail(in.Email)
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}
	message, err := normalizeInviteMessage(in.Message)
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}
	expiresAt, err := normalizeInvitationExpiration(in.ExpiresAt)
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}

	var invitedUserID *uuid.UUID
	var found *appdto.UserLookupResult
	if s.userLookupProvider != nil {
		found, err = s.userLookupProvider.LookupByEmail(ctx, email)
		if err != nil && !errors.Is(err, domainerrs.ErrNotFound) {
			return appdto.TripInvitationInfo{}, err
		}
		if found != nil && found.UserID != uuid.Nil {
			if found.UserID == ownerID {
				return appdto.TripInvitationInfo{}, apperrs.NewInvalidInput("owner cannot be invited as a collaborator")
			}
			invitedUserID = &found.UserID
		}
	}

	invitation, err := repo.CreateTripInvitation(ctx, &entity.TripInvitation{
		ID:            uuid.New(),
		TripID:        tripID,
		InviterUserID: user.ID,
		InvitedUserID: invitedUserID,
		Email:         email,
		Role:          role,
		Status:        entity.TripInvitationStatusPending,
		Message:       message,
		ExpiresAt:     expiresAt,
	})
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}

	if invitedUserID != nil {
		if _, err := s.repo.UpsertTripCollaborator(ctx, &entity.TripCollaborator{
			ID:              uuid.New(),
			TripID:          tripID,
			UserID:          *invitedUserID,
			Email:           email,
			Role:            role,
			Status:          entity.CollaboratorStatusPending,
			InvitedByUserID: user.ID,
			Message:         message,
			ExpiresAt:       &expiresAt,
			Permissions:     map[string]any{},
		}); err != nil {
			return appdto.TripInvitationInfo{}, err
		}
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventCollaboratorInvited,
		EntityType:  activityEntityType(activity.EntityCollaborator),
		EntityID:    activityEntityID(invitation.ID),
		Metadata: map[string]any{
			"email":        email,
			"role":         string(role),
			"invitationId": invitation.ID.String(),
			"expiresAt":    invitation.ExpiresAt.Format(time.RFC3339),
		},
	})

	if invitedUserID != nil {
		destination := tripDestination(trip)
		s.notifyDirect(ctx, *invitedUserID, tripID, user.ID,
			notifications.TypeCollaborationInvited,
			"You were invited to collaborate on a trip",
			fmt.Sprintf("You were invited to collaborate on %s as %s.", destination, role),
			notifications.EntityCollaborator,
			activityEntityID(invitation.ID),
			map[string]any{
				"tripId":       tripID.String(),
				"destination":  destination,
				"role":         string(role),
				"invitationId": invitation.ID.String(),
			})
	}

	info := appdto.TripInvitationInfo{Invitation: *invitation}
	if invitation.Email != "" {
		info.Email = &invitation.Email
	}
	tripobs.RecordCollaborationEvent("invite_created", "success")
	return info, nil
}

func (s *Service) ListTripInvitations(ctx context.Context, tripID uuid.UUID) ([]appdto.TripInvitationInfo, error) {
	repo, ok := s.repo.(tripInvitationRepository)
	if !ok {
		return nil, apperrs.NewDependencyError("trip invitation repository is not configured")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}
	invitations, err := repo.ListTripInvitationsByTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	return invitationInfos(invitations), nil
}

func (s *Service) ResendTripInvitation(ctx context.Context, tripID, invitationID uuid.UUID, in appdto.CreateTripInvitationInput) (appdto.TripInvitationInfo, error) {
	repo, ok := s.repo.(tripInvitationRepository)
	if !ok {
		return appdto.TripInvitationInfo{}, apperrs.NewDependencyError("trip invitation repository is not configured")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}
	trip, _, err := s.requireOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}
	existing, err := repo.GetTripInvitationByID(ctx, tripID, invitationID)
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}
	if existing.Status == entity.TripInvitationStatusAccepted {
		return appdto.TripInvitationInfo{}, apperrs.NewConflict("accepted invitations cannot be resent")
	}
	message, err := normalizeInviteMessage(firstNonEmpty(in.Message, existing.Message))
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}
	expiresAt, err := normalizeInvitationExpiration(in.ExpiresAt)
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}
	updated, err := repo.ResendTripInvitation(ctx, tripID, invitationID, user.ID, expiresAt, message)
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}

	if updated.InvitedUserID != nil {
		s.notifyDirect(ctx, *updated.InvitedUserID, tripID, user.ID,
			notifications.TypeCollaborationInvited,
			"Trip invitation resent",
			fmt.Sprintf("Your invitation to collaborate on %s was resent.", tripDestination(trip)),
			notifications.EntityCollaborator,
			activityEntityID(updated.ID),
			map[string]any{"tripId": tripID.String(), "invitationId": updated.ID.String(), "role": string(updated.Role)})
	}

	info := appdto.TripInvitationInfo{Invitation: *updated}
	if updated.Email != "" {
		info.Email = &updated.Email
	}
	tripobs.RecordCollaborationEvent("invite_resent", "success")
	return info, nil
}

func (s *Service) RevokeTripInvitation(ctx context.Context, tripID, invitationID uuid.UUID) error {
	repo, ok := s.repo.(tripInvitationRepository)
	if !ok {
		return apperrs.NewDependencyError("trip invitation repository is not configured")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	if _, _, err := s.requireOwner(ctx, tripID, user.ID); err != nil {
		return err
	}
	invitation, err := repo.GetTripInvitationByID(ctx, tripID, invitationID)
	if err != nil {
		return err
	}
	if invitation.Status == entity.TripInvitationStatusAccepted {
		return apperrs.NewConflict("accepted invitations cannot be revoked")
	}
	if invitation.Status == entity.TripInvitationStatusRevoked {
		return nil
	}
	updated, err := repo.UpdateTripInvitationStatus(ctx, tripID, invitationID, entity.TripInvitationStatusRevoked, &user.ID, time.Now().UTC())
	if err != nil {
		return err
	}
	if updated.InvitedUserID != nil {
		if collaborator, collabErr := s.repo.GetTripCollaboratorByTripAndUser(ctx, tripID, *updated.InvitedUserID); collabErr == nil && collaborator.Status == entity.CollaboratorStatusPending {
			_, _ = s.repo.RemoveTripCollaborator(ctx, tripID, collaborator.ID)
		}
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventCollaboratorRevoked,
		EntityType:  activityEntityType(activity.EntityCollaborator),
		EntityID:    activityEntityID(invitationID),
		Metadata:    map[string]any{"invitationId": invitationID.String(), "email": updated.Email},
	})
	tripobs.RecordCollaborationEvent("invite_revoked", "success")
	return nil
}

func (s *Service) AcceptTripInvitation(ctx context.Context, tripID, invitationID uuid.UUID) (appdto.TripInvitationInfo, error) {
	repo, ok := s.repo.(tripInvitationRepository)
	if !ok {
		return appdto.TripInvitationInfo{}, apperrs.NewDependencyError("trip invitation repository is not configured")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}
	invitation, err := repo.GetTripInvitationByID(ctx, tripID, invitationID)
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}
	if err := ensureInvitationAcceptable(ctx, repo, invitation, user); err != nil {
		return appdto.TripInvitationInfo{}, err
	}
	if invitation.InvitedUserID == nil && user.Email != "" {
		if linked, linkErr := repo.LinkTripInvitationToUser(ctx, invitationID, user.ID, user.Email); linkErr == nil {
			invitation = linked
		}
	}
	expiresAt := invitation.ExpiresAt
	collaborator, err := s.repo.UpsertTripCollaborator(ctx, &entity.TripCollaborator{
		ID:              uuid.New(),
		TripID:          tripID,
		UserID:          user.ID,
		Email:           invitation.Email,
		Role:            invitation.Role,
		Status:          entity.CollaboratorStatusPending,
		InvitedByUserID: invitation.InviterUserID,
		Message:         invitation.Message,
		ExpiresAt:       &expiresAt,
		Permissions:     map[string]any{},
	})
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}
	if collaborator.Status != entity.CollaboratorStatusAccepted {
		collaborator, err = s.repo.AcceptTripCollaborator(ctx, tripID, collaborator.ID, user.ID)
		if err != nil {
			return appdto.TripInvitationInfo{}, err
		}
	}
	updated, err := repo.UpdateTripInvitationStatus(ctx, tripID, invitationID, entity.TripInvitationStatusAccepted, &user.ID, time.Now().UTC())
	if err != nil {
		return appdto.TripInvitationInfo{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventCollaboratorAccepted,
		EntityType:  activityEntityType(activity.EntityCollaborator),
		EntityID:    activityEntityID(collaborator.ID),
		Metadata: map[string]any{
			"invitationId":       invitationID.String(),
			"collaboratorUserId": user.ID.String(),
			"role":               string(collaborator.Role),
		},
	})
	if trip, loadErr := s.repo.GetByID(ctx, tripID); loadErr == nil && trip.UserID != nil {
		s.notifyDirect(ctx, *trip.UserID, tripID, user.ID,
			notifications.TypeCollaborationAccepted,
			"Collaboration invitation accepted",
			fmt.Sprintf("A collaborator accepted your invitation for %s.", tripDestination(trip)),
			notifications.EntityCollaborator,
			activityEntityID(collaborator.ID),
			map[string]any{"tripId": tripID.String(), "invitationId": invitationID.String(), "role": string(collaborator.Role)})
	}
	info := appdto.TripInvitationInfo{Invitation: *updated}
	if updated.Email != "" {
		info.Email = &updated.Email
	}
	tripobs.RecordCollaborationEvent("invite_accepted", "success")
	return info, nil
}

func (s *Service) DeclineTripInvitation(ctx context.Context, tripID, invitationID uuid.UUID) error {
	repo, ok := s.repo.(tripInvitationRepository)
	if !ok {
		return apperrs.NewDependencyError("trip invitation repository is not configured")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	invitation, err := repo.GetTripInvitationByID(ctx, tripID, invitationID)
	if err != nil {
		return err
	}
	if !invitationMatchesUser(invitation, user) {
		return apperrs.ErrForbidden
	}
	if invitation.Status != entity.TripInvitationStatusPending {
		return apperrs.NewConflict("invitation is not pending")
	}
	updated, err := repo.UpdateTripInvitationStatus(ctx, tripID, invitationID, entity.TripInvitationStatusDeclined, &user.ID, time.Now().UTC())
	if err != nil {
		return err
	}
	if collaborator, collabErr := s.repo.GetTripCollaboratorByTripAndUser(ctx, tripID, user.ID); collabErr == nil && collaborator.Status == entity.CollaboratorStatusPending {
		_, _ = s.repo.DeclineTripCollaborator(ctx, tripID, collaborator.ID, user.ID)
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventCollaboratorDeclined,
		EntityType:  activityEntityType(activity.EntityCollaborator),
		EntityID:    activityEntityID(updated.ID),
		Metadata:    map[string]any{"invitationId": invitationID.String(), "role": string(updated.Role)},
	})
	tripobs.RecordCollaborationEvent("invite_declined", "success")
	return nil
}

func (s *Service) ListTripMembers(ctx context.Context, tripID uuid.UUID) ([]appdto.TripMemberInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	collaborators, err := s.repo.ListTripCollaborators(ctx, tripID)
	if err != nil {
		return nil, err
	}
	out := make([]appdto.TripMemberInfo, 0, len(collaborators)+1)
	if trip.UserID != nil {
		out = append(out, appdto.TripMemberInfo{
			Member: entity.TripMember{
				UserID:      *trip.UserID,
				Role:        "owner",
				Status:      entity.TripMemberStatusActive,
				JoinedAt:    &trip.CreatedAt,
				Permissions: permissionsForRole("owner"),
			},
			IsSelf: *trip.UserID == user.ID,
		})
	}
	for _, collaborator := range collaborators {
		if collaborator.Status == entity.CollaboratorStatusRemoved ||
			collaborator.Status == entity.CollaboratorStatusDeclined ||
			collaborator.Status == entity.CollaboratorStatusRevoked ||
			collaborator.Status == entity.CollaboratorStatusExpired {
			continue
		}
		status := entity.TripMemberStatusInvited
		joinedAt := collaborator.AcceptedAt
		if collaborator.Status == entity.CollaboratorStatusAccepted {
			status = entity.TripMemberStatusActive
		}
		out = append(out, appdto.TripMemberInfo{
			Member: entity.TripMember{
				UserID:      collaborator.UserID,
				Email:       collaborator.Email,
				Role:        string(collaborator.Role),
				Status:      status,
				JoinedAt:    joinedAt,
				InvitedBy:   &collaborator.InvitedByUserID,
				LastSeenAt:  collaborator.LastSeenAt,
				Permissions: permissionsForRole(string(collaborator.Role)),
			},
			IsSelf: collaborator.UserID == user.ID,
		})
	}
	if !access.CanManageCollaborators() {
		for i := range out {
			out[i].Member.Email = ""
		}
	}
	return out, nil
}

func (s *Service) TransferTripOwnership(ctx context.Context, tripID uuid.UUID, in appdto.TransferTripOwnershipInput) (*entity.Trip, error) {
	repo, ok := s.repo.(tripOwnershipRepository)
	if !ok {
		return nil, apperrs.NewDependencyError("trip ownership repository is not configured")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	trip, _, err := s.requireOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	if trip.UserID == nil {
		return nil, apperrs.NewConflict("workspace-owned trips cannot transfer direct ownership")
	}
	if in.NewOwnerUserID == uuid.Nil {
		return nil, apperrs.NewInvalidInput("newOwnerUserId is required")
	}
	if in.NewOwnerUserID == user.ID {
		return nil, apperrs.NewInvalidInput("new owner must be a different user")
	}
	nextCollaborator, err := s.repo.GetTripCollaboratorByTripAndUser(ctx, tripID, in.NewOwnerUserID)
	if err != nil {
		return nil, apperrs.NewInvalidInput("new owner must be an accepted trip collaborator")
	}
	if nextCollaborator.Status != entity.CollaboratorStatusAccepted {
		return nil, apperrs.NewInvalidInput("new owner must be an accepted trip collaborator")
	}

	updated, err := repo.TransferTripOwnership(ctx, tripID, user.ID, in.NewOwnerUserID)
	if err != nil {
		return nil, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventOwnershipTransferred,
		EntityType:  activityEntityType(activity.EntityTrip),
		EntityID:    activityEntityID(tripID),
		Metadata: map[string]any{
			"oldOwnerUserId": user.ID.String(),
			"newOwnerUserId": in.NewOwnerUserID.String(),
		},
	})
	s.notifyDirect(ctx, in.NewOwnerUserID, tripID, user.ID,
		notifications.TypeOwnershipTransferred,
		"Trip ownership transferred",
		fmt.Sprintf("You are now the owner of %s.", tripDestination(trip)),
		notifications.EntityTrip,
		activityEntityID(tripID),
		map[string]any{"tripId": tripID.String(), "oldOwnerUserId": user.ID.String()})
	tripobs.RecordCollaborationEvent("ownership_transferred", "success")
	return updated, nil
}

func (s *Service) LeaveTrip(ctx context.Context, tripID uuid.UUID) error {
	repo, ok := s.repo.(tripOwnershipRepository)
	if !ok {
		return apperrs.NewDependencyError("trip ownership repository is not configured")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return err
	}
	if access.Level == AccessLevelOwner && trip.UserID != nil && *trip.UserID == user.ID {
		return apperrs.NewConflict("owner must transfer ownership before leaving the trip")
	}
	removed, err := repo.RemoveTripCollaboratorByUser(ctx, tripID, user.ID)
	if err != nil {
		return err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventCollaboratorRemoved,
		EntityType:  activityEntityType(activity.EntityCollaborator),
		EntityID:    activityEntityID(removed.ID),
		Metadata:    map[string]any{"collaboratorUserId": user.ID.String(), "role": string(removed.Role), "selfRemoved": true},
	})
	tripobs.RecordCollaborationEvent("member_left", "success")
	return nil
}

func (s *Service) CreateTripSuggestion(ctx context.Context, tripID uuid.UUID, in appdto.CreateTripSuggestionInput) (appdto.TripSuggestionInfo, error) {
	repo, ok := s.repo.(tripSuggestionRepository)
	if !ok {
		return appdto.TripSuggestionInfo{}, apperrs.NewDependencyError("trip suggestion repository is not configured")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripSuggestionInfo{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripSuggestionInfo{}, err
	}
	normalized, err := normalizeSuggestionInput(in)
	if err != nil {
		return appdto.TripSuggestionInfo{}, err
	}
	created, err := repo.CreateTripSuggestion(ctx, &entity.TripSuggestion{
		ID:             uuid.New(),
		TripID:         tripID,
		AuthorUserID:   user.ID,
		SuggestionType: normalized.SuggestionType,
		TargetType:     normalized.TargetType,
		TargetID:       normalized.TargetID,
		Status:         entity.TripSuggestionStatusOpen,
		Before:         normalized.Before,
		After:          normalized.After,
		Comment:        normalized.Comment,
		Metadata:       normalized.Metadata,
	})
	if err != nil {
		return appdto.TripSuggestionInfo{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventSuggestionCreated,
		EntityType:  activityEntityType(activity.EntityTripSuggestion),
		EntityID:    activityEntityID(created.ID),
		Metadata: map[string]any{
			"suggestionType": string(created.SuggestionType),
			"targetType":     string(created.TargetType),
			"targetId":       created.TargetID,
		},
	})
	s.notifyTripBroadcast(ctx, trip, user.ID,
		notifications.TypeTripSuggestionCreated,
		"New trip suggestion",
		"A collaborator created a suggestion.",
		notifications.EntityTripSuggestion,
		activityEntityID(created.ID),
		map[string]any{"tripId": tripID.String(), "suggestionId": created.ID.String(), "suggestionType": string(created.SuggestionType)})
	tripobs.RecordCollaborationEvent("suggestion_created", "success")
	return suggestionInfo(*created, user.ID, access), nil
}

func (s *Service) ListTripSuggestions(ctx context.Context, tripID uuid.UUID, status *entity.TripSuggestionStatus) ([]appdto.TripSuggestionInfo, error) {
	repo, ok := s.repo.(tripSuggestionRepository)
	if !ok {
		return nil, apperrs.NewDependencyError("trip suggestion repository is not configured")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	_, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	suggestions, err := repo.ListTripSuggestionsByTrip(ctx, tripID, status)
	if err != nil {
		return nil, err
	}
	out := make([]appdto.TripSuggestionInfo, 0, len(suggestions))
	for _, suggestion := range suggestions {
		out = append(out, suggestionInfo(suggestion, user.ID, access))
	}
	return out, nil
}

func (s *Service) AcceptTripSuggestion(ctx context.Context, tripID, suggestionID uuid.UUID, in appdto.UpdateTripSuggestionStatusInput) (appdto.TripSuggestionInfo, error) {
	return s.resolveTripSuggestion(ctx, tripID, suggestionID, entity.TripSuggestionStatusAccepted, in)
}

func (s *Service) RejectTripSuggestion(ctx context.Context, tripID, suggestionID uuid.UUID) (appdto.TripSuggestionInfo, error) {
	return s.resolveTripSuggestion(ctx, tripID, suggestionID, entity.TripSuggestionStatusRejected, appdto.UpdateTripSuggestionStatusInput{})
}

func (s *Service) ResolveTripSuggestion(ctx context.Context, tripID, suggestionID uuid.UUID) (appdto.TripSuggestionInfo, error) {
	return s.resolveTripSuggestion(ctx, tripID, suggestionID, entity.TripSuggestionStatusResolved, appdto.UpdateTripSuggestionStatusInput{})
}

func (s *Service) resolveTripSuggestion(
	ctx context.Context,
	tripID, suggestionID uuid.UUID,
	status entity.TripSuggestionStatus,
	in appdto.UpdateTripSuggestionStatusInput,
) (appdto.TripSuggestionInfo, error) {
	repo, ok := s.repo.(tripSuggestionRepository)
	if !ok {
		return appdto.TripSuggestionInfo{}, apperrs.NewDependencyError("trip suggestion repository is not configured")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripSuggestionInfo{}, err
	}
	trip, access, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripSuggestionInfo{}, err
	}
	suggestion, err := repo.GetTripSuggestionByID(ctx, tripID, suggestionID)
	if err != nil {
		return appdto.TripSuggestionInfo{}, err
	}
	if suggestion.Status != entity.TripSuggestionStatusOpen {
		return appdto.TripSuggestionInfo{}, apperrs.NewConflict("suggestion is not open")
	}
	var appliedRevision *int
	if status == entity.TripSuggestionStatusAccepted {
		appliedRevision, err = s.applyAcceptedSuggestion(ctx, trip, user.ID, *suggestion, in.ExpectedItineraryRevision)
		if err != nil {
			tripobs.RecordCollaborationEvent("suggestion_accepted", "failure")
			return appdto.TripSuggestionInfo{}, err
		}
	}
	updated, err := repo.UpdateTripSuggestionStatus(ctx, tripID, suggestionID, user.ID, status, appliedRevision)
	if err != nil {
		return appdto.TripSuggestionInfo{}, err
	}
	eventType := activity.EventSuggestionResolved
	notificationType := notifications.TypeTripSuggestionAccepted
	if status == entity.TripSuggestionStatusAccepted {
		eventType = activity.EventSuggestionAccepted
	} else if status == entity.TripSuggestionStatusRejected {
		eventType = activity.EventSuggestionRejected
		notificationType = notifications.TypeTripSuggestionCreated
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   eventType,
		EntityType:  activityEntityType(activity.EntityTripSuggestion),
		EntityID:    activityEntityID(updated.ID),
		Metadata: map[string]any{
			"suggestionType": string(updated.SuggestionType),
			"targetType":     string(updated.TargetType),
			"targetId":       updated.TargetID,
			"status":         string(updated.Status),
		},
	})
	if updated.AuthorUserID != user.ID {
		s.notifyDirect(ctx, updated.AuthorUserID, tripID, user.ID,
			notificationType,
			"Trip suggestion updated",
			fmt.Sprintf("Your suggestion for %s was %s.", tripDestination(trip), updated.Status),
			notifications.EntityTripSuggestion,
			activityEntityID(updated.ID),
			map[string]any{"tripId": tripID.String(), "suggestionId": updated.ID.String(), "status": string(updated.Status)})
	}
	tripobs.RecordCollaborationEvent("suggestion_"+string(status), "success")
	return suggestionInfo(*updated, user.ID, access), nil
}

func (s *Service) SetTripVote(ctx context.Context, tripID uuid.UUID, in appdto.SetTripVoteInput) (entity.TripVoteSummary, error) {
	repo, ok := s.repo.(tripVoteRepository)
	if !ok {
		return entity.TripVoteSummary{}, apperrs.NewDependencyError("trip vote repository is not configured")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return entity.TripVoteSummary{}, err
	}
	if _, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return entity.TripVoteSummary{}, err
	}
	normalized, err := normalizeVoteInput(in)
	if err != nil {
		return entity.TripVoteSummary{}, err
	}
	vote, err := repo.UpsertTripVote(ctx, &entity.TripVote{
		ID:         uuid.New(),
		TripID:     tripID,
		TargetType: normalized.TargetType,
		TargetID:   normalized.TargetID,
		UserID:     user.ID,
		VoteType:   normalized.VoteType,
		Metadata:   normalized.Metadata,
	})
	if err != nil {
		return entity.TripVoteSummary{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTripVoteAdded,
		EntityType:  activityEntityType(activity.EntityTripVote),
		EntityID:    activityEntityID(vote.ID),
		Metadata: map[string]any{
			"targetType": string(vote.TargetType),
			"targetId":   vote.TargetID,
			"voteType":   string(vote.VoteType),
		},
	})
	tripobs.RecordCollaborationEvent("vote_set", "success")
	return s.TripVoteSummary(ctx, tripID, vote.TargetType, vote.TargetID)
}

func (s *Service) ListTripVotes(ctx context.Context, tripID uuid.UUID, in appdto.ListTripVotesInput) ([]entity.TripVoteSummary, error) {
	repo, ok := s.repo.(tripVoteRepository)
	if !ok {
		return nil, apperrs.NewDependencyError("trip vote repository is not configured")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}
	var votes []entity.TripVote
	if in.TargetType != "" || strings.TrimSpace(in.TargetID) != "" {
		if !in.TargetType.Valid() {
			return nil, apperrs.NewInvalidInput("targetType is invalid")
		}
		targetID := strings.TrimSpace(in.TargetID)
		if targetID == "" {
			return nil, apperrs.NewInvalidInput("targetId is required")
		}
		var err error
		votes, err = repo.ListTripVotesByTarget(ctx, tripID, in.TargetType, targetID)
		if err != nil {
			return nil, err
		}
		return summarizeTripVotes(votes, user.ID), nil
	}
	votes, err = repo.ListTripVotesByTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	return summarizeTripVotes(votes, user.ID), nil
}

func (s *Service) TripVoteSummary(ctx context.Context, tripID uuid.UUID, targetType entity.TripVoteTargetType, targetID string) (entity.TripVoteSummary, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return entity.TripVoteSummary{}, err
	}
	summaries, err := s.ListTripVotes(ctx, tripID, appdto.ListTripVotesInput{TargetType: targetType, TargetID: targetID})
	if err != nil {
		return entity.TripVoteSummary{}, err
	}
	if len(summaries) == 0 {
		return entity.TripVoteSummary{TargetType: targetType, TargetID: targetID, Counts: map[entity.TripVoteType]int{}}, nil
	}
	_ = user
	return summaries[0], nil
}

func (s *Service) DeleteTripVote(ctx context.Context, tripID, voteID uuid.UUID) error {
	repo, ok := s.repo.(tripVoteRepository)
	if !ok {
		return apperrs.NewDependencyError("trip vote repository is not configured")
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	_, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return err
	}
	vote, err := repo.GetTripVoteByID(ctx, tripID, voteID)
	if err != nil {
		return err
	}
	if vote.UserID != user.ID && access.Level != AccessLevelOwner {
		return apperrs.ErrForbidden
	}
	deleteUserID := vote.UserID
	if err := repo.DeleteTripVote(ctx, tripID, voteID, deleteUserID); err != nil {
		return err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTripVoteRemoved,
		EntityType:  activityEntityType(activity.EntityTripVote),
		EntityID:    activityEntityID(vote.ID),
		Metadata:    map[string]any{"targetType": string(vote.TargetType), "targetId": vote.TargetID},
	})
	tripobs.RecordCollaborationEvent("vote_deleted", "success")
	return nil
}

func invitationInfos(invitations []entity.TripInvitation) []appdto.TripInvitationInfo {
	out := make([]appdto.TripInvitationInfo, 0, len(invitations))
	for i := range invitations {
		info := appdto.TripInvitationInfo{Invitation: invitations[i]}
		if invitations[i].Email != "" {
			info.Email = &invitations[i].Email
		}
		out = append(out, info)
	}
	return out
}

func ensureInvitationAcceptable(ctx context.Context, repo tripInvitationRepository, invitation *entity.TripInvitation, user auth.AuthenticatedUser) error {
	if !invitationMatchesUser(invitation, user) {
		return apperrs.ErrForbidden
	}
	if invitation.Status != entity.TripInvitationStatusPending {
		return apperrs.NewConflict("invitation is not pending")
	}
	now := time.Now().UTC()
	if !invitation.ExpiresAt.IsZero() && !invitation.ExpiresAt.After(now) {
		_, _ = repo.UpdateTripInvitationStatus(ctx, invitation.TripID, invitation.ID, entity.TripInvitationStatusExpired, &user.ID, now)
		return apperrs.NewConflict("invitation has expired")
	}
	return nil
}

func invitationMatchesUser(invitation *entity.TripInvitation, user auth.AuthenticatedUser) bool {
	if invitation == nil {
		return false
	}
	if invitation.InvitedUserID != nil && *invitation.InvitedUserID == user.ID {
		return true
	}
	return strings.TrimSpace(user.Email) != "" &&
		strings.EqualFold(strings.TrimSpace(invitation.Email), strings.TrimSpace(user.Email))
}

func normalizeInviteMessage(raw string) (string, error) {
	message := strings.TrimSpace(raw)
	if len(message) > maxInviteMessage {
		return "", apperrs.NewInvalidInput("message must be at most %d characters", maxInviteMessage)
	}
	return message, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func normalizeInvitationExpiration(raw *time.Time) (time.Time, error) {
	now := time.Now().UTC()
	if raw == nil {
		return now.Add(defaultInvitationTTL), nil
	}
	expiresAt := raw.UTC()
	if !expiresAt.After(now) {
		return time.Time{}, apperrs.NewInvalidInput("expiresAt must be in the future")
	}
	if expiresAt.After(now.Add(maxInvitationTTL)) {
		return time.Time{}, apperrs.NewInvalidInput("expiresAt must be within 90 days")
	}
	return expiresAt, nil
}

func permissionsForRole(role string) map[string]bool {
	switch role {
	case "owner":
		return map[string]bool{
			"view": true, "edit": true, "comment": true, "vote": true,
			"manageMembers": true, "deleteTrip": true, "transferOwnership": true,
		}
	case "editor":
		return map[string]bool{"view": true, "edit": true, "comment": true, "vote": true}
	case "viewer":
		return map[string]bool{"view": true, "comment": true, "vote": true}
	case "commenter":
		return map[string]bool{"view": true, "comment": true, "vote": true}
	case "guest":
		return map[string]bool{"view": true}
	default:
		return map[string]bool{}
	}
}

func normalizeSuggestionInput(in appdto.CreateTripSuggestionInput) (appdto.CreateTripSuggestionInput, error) {
	if !in.SuggestionType.Valid() {
		return appdto.CreateTripSuggestionInput{}, apperrs.NewInvalidInput("suggestionType is invalid")
	}
	if !in.TargetType.Valid() {
		return appdto.CreateTripSuggestionInput{}, apperrs.NewInvalidInput("targetType is invalid")
	}
	in.TargetID = strings.TrimSpace(in.TargetID)
	if len(in.TargetID) > maxTargetIDLength {
		return appdto.CreateTripSuggestionInput{}, apperrs.NewInvalidInput("targetId must be at most %d characters", maxTargetIDLength)
	}
	in.Comment = strings.TrimSpace(in.Comment)
	if len(in.Comment) > maxSuggestionComment {
		return appdto.CreateTripSuggestionInput{}, apperrs.NewInvalidInput("comment must be at most %d characters", maxSuggestionComment)
	}
	if in.Metadata == nil {
		in.Metadata = map[string]any{}
	}
	return in, nil
}

func suggestionInfo(suggestion entity.TripSuggestion, userID uuid.UUID, access TripAccess) appdto.TripSuggestionInfo {
	return appdto.TripSuggestionInfo{
		Suggestion: suggestion,
		IsAuthor:   suggestion.AuthorUserID == userID,
		CanResolve: access.CanEdit(),
	}
}

func (s *Service) applyAcceptedSuggestion(
	ctx context.Context,
	trip *entity.Trip,
	actorUserID uuid.UUID,
	suggestion entity.TripSuggestion,
	expectedRevision *int,
) (*int, error) {
	if len(suggestion.After) == 0 {
		return nil, nil
	}
	if suggestion.TargetType != entity.TripSuggestionTargetItineraryItem {
		return nil, nil
	}
	expected, err := requireExpectedItineraryRevision(expectedRevision)
	if err != nil {
		return nil, err
	}
	if err := checkCurrentItineraryRevision(expected, trip.ItineraryRevision); err != nil {
		return nil, err
	}
	ownerID, err := tripOwnerID(trip)
	if err != nil {
		return nil, err
	}

	itinerary := parseItineraryLenient(trip.Itinerary)
	dayNumber, itemIndex, err := parseSuggestionItemTarget(suggestion.TargetID, suggestion.Metadata)
	if err != nil {
		return nil, err
	}
	applied, err := applySuggestionToItinerary(&itinerary, suggestion, dayNumber, itemIndex)
	if err != nil {
		return nil, err
	}
	if !applied {
		return nil, nil
	}
	updated, err := s.saveRegeneratedItinerary(ctx, trip.ID, ownerID, actorUserID, itinerary, expected, entity.ItineraryVersionSourceManualEdit, map[string]any{
		"suggestionId":   suggestion.ID.String(),
		"suggestionType": string(suggestion.SuggestionType),
	})
	if err != nil {
		return nil, err
	}
	revision := updated.ItineraryRevision
	return &revision, nil
}

func parseSuggestionItemTarget(targetID string, metadata map[string]any) (int, int, error) {
	if day, item, ok := parseDayItemTarget(targetID); ok {
		return day, item, nil
	}
	day, dayOK := numberFromMetadata(metadata, "dayNumber")
	item, itemOK := numberFromMetadata(metadata, "itemIndex")
	if dayOK && itemOK {
		return day, item, nil
	}
	return 0, 0, apperrs.NewInvalidInput("itinerary item suggestions require targetId formatted as dayNumber:itemIndex")
}

func parseDayItemTarget(value string) (int, int, bool) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	day, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, false
	}
	item, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, false
	}
	return day, item, day >= 1 && item >= 0
}

func numberFromMetadata(metadata map[string]any, key string) (int, bool) {
	value, ok := metadata[key]
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return v, true
	case int32:
		return int(v), true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case json.Number:
		i, err := v.Int64()
		return int(i), err == nil
	default:
		return 0, false
	}
}

func applySuggestionToItinerary(itinerary *aggregate.Itinerary, suggestion entity.TripSuggestion, dayNumber, itemIndex int) (bool, error) {
	for dayIdx := range itinerary.Days {
		if itinerary.Days[dayIdx].Day != dayNumber {
			continue
		}
		if itemIndex < 0 || itemIndex >= len(itinerary.Days[dayIdx].Items) {
			return false, apperrs.NewInvalidInput("target itinerary item does not exist")
		}
		item := &itinerary.Days[dayIdx].Items[itemIndex]
		switch suggestion.SuggestionType {
		case entity.TripSuggestionActivityReplacement:
			var replacement aggregate.ItineraryItem
			if err := json.Unmarshal(suggestion.After, &replacement); err != nil {
				return false, apperrs.NewInvalidInput("after must be an itinerary item")
			}
			itinerary.Days[dayIdx].Items[itemIndex] = replacement
			return true, nil
		case entity.TripSuggestionTimeChange:
			var timeChange struct {
				Time    string `json:"time"`
				EndTime string `json:"endTime"`
			}
			if err := json.Unmarshal(suggestion.After, &timeChange); err != nil {
				return false, apperrs.NewInvalidInput("after must include time fields")
			}
			if strings.TrimSpace(timeChange.Time) != "" {
				item.Time = strings.TrimSpace(timeChange.Time)
			}
			if strings.TrimSpace(timeChange.EndTime) != "" {
				item.EndTime = strings.TrimSpace(timeChange.EndTime)
			}
			return true, nil
		case entity.TripSuggestionBudgetAdjustment:
			var cost aggregate.EstimatedCost
			if err := json.Unmarshal(suggestion.After, &cost); err != nil {
				return false, apperrs.NewInvalidInput("after must be an estimated cost")
			}
			item.EstimatedCost = &cost
			return true, nil
		default:
			return false, nil
		}
	}
	return false, apperrs.NewInvalidInput("target itinerary day does not exist")
}

func normalizeVoteInput(in appdto.SetTripVoteInput) (appdto.SetTripVoteInput, error) {
	if !in.TargetType.Valid() {
		return appdto.SetTripVoteInput{}, apperrs.NewInvalidInput("targetType is invalid")
	}
	in.TargetID = strings.TrimSpace(in.TargetID)
	if in.TargetID == "" {
		return appdto.SetTripVoteInput{}, apperrs.NewInvalidInput("targetId is required")
	}
	if len(in.TargetID) > maxTargetIDLength {
		return appdto.SetTripVoteInput{}, apperrs.NewInvalidInput("targetId must be at most %d characters", maxTargetIDLength)
	}
	if !in.VoteType.Valid() {
		return appdto.SetTripVoteInput{}, apperrs.NewInvalidInput("voteType is invalid")
	}
	if in.Metadata == nil {
		in.Metadata = map[string]any{}
	}
	return in, nil
}

func summarizeTripVotes(votes []entity.TripVote, currentUserID uuid.UUID) []entity.TripVoteSummary {
	byTarget := map[string]*entity.TripVoteSummary{}
	order := make([]string, 0)
	for _, vote := range votes {
		key := string(vote.TargetType) + ":" + vote.TargetID
		summary, ok := byTarget[key]
		if !ok {
			summary = &entity.TripVoteSummary{
				TargetType: vote.TargetType,
				TargetID:   vote.TargetID,
				Counts:     map[entity.TripVoteType]int{},
			}
			byTarget[key] = summary
			order = append(order, key)
		}
		summary.Counts[vote.VoteType]++
		if vote.UserID == currentUserID {
			current := vote.VoteType
			summary.CurrentVote = &current
		}
	}
	out := make([]entity.TripVoteSummary, 0, len(order))
	for _, key := range order {
		out = append(out, *byTarget[key])
	}
	return out
}
