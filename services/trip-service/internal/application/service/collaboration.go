package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
)

var ErrRegisteredUserNotFound = errors.New("registered user not found")

func (s *Service) InviteTripCollaborator(ctx context.Context, tripID uuid.UUID, in appdto.InviteTripCollaboratorInput) (appdto.TripCollaboratorInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripCollaboratorInfo{}, err
	}
	trip, _, err := s.requireOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripCollaboratorInfo{}, err
	}
	ownerID, err := tripOwnerID(trip)
	if err != nil {
		return appdto.TripCollaboratorInfo{}, err
	}

	role, err := normalizeCollaboratorRole(in.Role)
	if err != nil {
		return appdto.TripCollaboratorInfo{}, err
	}
	email, err := normalizeCollaboratorEmail(in.Email)
	if err != nil {
		return appdto.TripCollaboratorInfo{}, err
	}
	if s.userLookupProvider == nil {
		return appdto.TripCollaboratorInfo{}, apperrs.NewDependencyError("user lookup is not configured")
	}

	found, err := s.userLookupProvider.LookupByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			return appdto.TripCollaboratorInfo{}, ErrRegisteredUserNotFound
		}
		return appdto.TripCollaboratorInfo{}, err
	}
	if found == nil || found.UserID == uuid.Nil {
		return appdto.TripCollaboratorInfo{}, ErrRegisteredUserNotFound
	}
	if found.UserID == ownerID {
		return appdto.TripCollaboratorInfo{}, apperrs.NewInvalidInput("owner cannot be invited as a collaborator")
	}

	collaborator, err := s.repo.UpsertTripCollaborator(ctx, &entity.TripCollaborator{
		ID:              uuid.New(),
		TripID:          tripID,
		UserID:          found.UserID,
		Role:            role,
		Status:          entity.CollaboratorStatusPending,
		InvitedByUserID: user.ID,
	})
	if err != nil {
		return appdto.TripCollaboratorInfo{}, err
	}

	info := appdto.TripCollaboratorInfo{Collaborator: *collaborator}
	if found.Email != "" {
		info.Email = &found.Email
	}
	if found.DisplayName != "" {
		info.DisplayName = &found.DisplayName
	}

	inviteMetadata := map[string]any{
		"collaboratorUserId": collaborator.UserID.String(),
		"role":               string(collaborator.Role),
	}
	if found.Email != "" {
		inviteMetadata["collaboratorEmail"] = found.Email
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventCollaboratorInvited,
		EntityType:  activityEntityType(activity.EntityCollaborator),
		EntityID:    activityEntityID(collaborator.ID),
		Metadata:    inviteMetadata,
	})

	return info, nil
}

func (s *Service) ListTripCollaborators(ctx context.Context, tripID uuid.UUID) ([]appdto.TripCollaboratorInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}

	collaborators, err := s.repo.ListTripCollaborators(ctx, tripID)
	if err != nil {
		return nil, err
	}
	out := make([]appdto.TripCollaboratorInfo, 0, len(collaborators))
	for _, collaborator := range collaborators {
		out = append(out, appdto.TripCollaboratorInfo{Collaborator: collaborator})
	}
	return out, nil
}

func (s *Service) UpdateTripCollaborator(ctx context.Context, tripID, collaboratorID uuid.UUID, in appdto.UpdateTripCollaboratorInput) (appdto.TripCollaboratorInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripCollaboratorInfo{}, err
	}
	if _, _, err := s.requireOwner(ctx, tripID, user.ID); err != nil {
		return appdto.TripCollaboratorInfo{}, err
	}
	role, err := normalizeCollaboratorRole(in.Role)
	if err != nil {
		return appdto.TripCollaboratorInfo{}, err
	}

	existing, err := s.repo.GetTripCollaboratorByID(ctx, tripID, collaboratorID)
	if err != nil {
		return appdto.TripCollaboratorInfo{}, err
	}

	updated, err := s.repo.UpdateTripCollaboratorRole(ctx, tripID, collaboratorID, role)
	if err != nil {
		return appdto.TripCollaboratorInfo{}, err
	}

	if existing.Role != updated.Role {
		s.recordActivity(ctx, activity.RecordActivityInput{
			TripID:      tripID,
			ActorUserID: &user.ID,
			EventType:   activity.EventCollaboratorRoleChanged,
			EntityType:  activityEntityType(activity.EntityCollaborator),
			EntityID:    activityEntityID(updated.ID),
			Metadata: map[string]any{
				"collaboratorUserId": updated.UserID.String(),
				"oldRole":            string(existing.Role),
				"newRole":            string(updated.Role),
			},
		})
	}

	return appdto.TripCollaboratorInfo{Collaborator: *updated}, nil
}

func (s *Service) RemoveTripCollaborator(ctx context.Context, tripID, collaboratorID uuid.UUID) error {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	if _, _, err := s.requireOwner(ctx, tripID, user.ID); err != nil {
		return err
	}
	removed, err := s.repo.RemoveTripCollaborator(ctx, tripID, collaboratorID)
	if err != nil {
		return err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventCollaboratorRemoved,
		EntityType:  activityEntityType(activity.EntityCollaborator),
		EntityID:    activityEntityID(removed.ID),
		Metadata: map[string]any{
			"collaboratorUserId": removed.UserID.String(),
			"role":               string(removed.Role),
		},
	})

	return nil
}

func (s *Service) AcceptTripCollaborator(ctx context.Context, tripID, collaboratorID uuid.UUID) (appdto.TripCollaboratorInfo, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripCollaboratorInfo{}, err
	}
	collaborator, err := s.repo.AcceptTripCollaborator(ctx, tripID, collaboratorID, user.ID)
	if err != nil {
		return appdto.TripCollaboratorInfo{}, err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventCollaboratorAccepted,
		EntityType:  activityEntityType(activity.EntityCollaborator),
		EntityID:    activityEntityID(collaborator.ID),
		Metadata: map[string]any{
			"collaboratorUserId": collaborator.UserID.String(),
			"role":               string(collaborator.Role),
		},
	})

	return appdto.TripCollaboratorInfo{Collaborator: *collaborator}, nil
}

func (s *Service) DeclineTripCollaborator(ctx context.Context, tripID, collaboratorID uuid.UUID) error {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	declined, err := s.repo.DeclineTripCollaborator(ctx, tripID, collaboratorID, user.ID)
	if err != nil {
		return err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventCollaboratorDeclined,
		EntityType:  activityEntityType(activity.EntityCollaborator),
		EntityID:    activityEntityID(declined.ID),
		Metadata: map[string]any{
			"collaboratorUserId": declined.UserID.String(),
			"role":               string(declined.Role),
		},
	})

	return nil
}

func (s *Service) ListCollaborationInvitations(ctx context.Context) ([]appdto.CollaborationInvitation, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	sharedTrips, err := s.repo.ListPendingCollaborationInvitations(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	out := make([]appdto.CollaborationInvitation, 0, len(sharedTrips))
	for _, shared := range sharedTrips {
		out = append(out, appdto.CollaborationInvitation{
			CollaboratorID:  shared.Collaborator.ID,
			TripID:          shared.Trip.ID,
			Destination:     shared.Trip.Destination,
			Role:            shared.Collaborator.Role,
			InvitedByUserID: shared.Collaborator.InvitedByUserID,
			InvitedAt:       shared.Collaborator.InvitedAt,
		})
	}
	return out, nil
}

func (s *Service) ListSharedTrips(ctx context.Context) ([]entity.SharedTrip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListSharedTripsByUser(ctx, user.ID)
}

func normalizeCollaboratorRole(role entity.CollaboratorRole) (entity.CollaboratorRole, error) {
	switch entity.CollaboratorRole(strings.TrimSpace(string(role))) {
	case entity.CollaboratorRoleViewer:
		return entity.CollaboratorRoleViewer, nil
	case entity.CollaboratorRoleEditor:
		return entity.CollaboratorRoleEditor, nil
	default:
		return "", apperrs.NewInvalidInput("role must be viewer or editor")
	}
}

func normalizeCollaboratorEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", apperrs.NewInvalidInput("email is required")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", apperrs.NewInvalidInput("email must be a valid email address")
	}
	return email, nil
}
