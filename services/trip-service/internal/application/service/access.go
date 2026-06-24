package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
)

type AccessLevel string

const (
	AccessLevelOwner  AccessLevel = "owner"
	AccessLevelEditor AccessLevel = "editor"
	AccessLevelViewer AccessLevel = "viewer"
	AccessLevelNone   AccessLevel = "none"
)

type TripAccess struct {
	Level AccessLevel
}

func (a TripAccess) CanView() bool {
	return a.Level == AccessLevelOwner || a.Level == AccessLevelEditor || a.Level == AccessLevelViewer
}

func (a TripAccess) CanEdit() bool {
	return a.Level == AccessLevelOwner || a.Level == AccessLevelEditor
}

func (a TripAccess) CanManageCollaborators() bool {
	return a.Level == AccessLevelOwner
}

func (a TripAccess) CanManageShare() bool {
	return a.Level == AccessLevelOwner
}

func (a TripAccess) CanRestoreVersion() bool {
	return a.CanEdit()
}

func (a TripAccess) CanDelete() bool {
	return a.Level == AccessLevelOwner
}

func (s *Service) tripForAccess(ctx context.Context, tripID, actorUserID uuid.UUID) (*entity.Trip, TripAccess, error) {
	trip, err := s.repo.GetByID(ctx, tripID)
	if err != nil {
		return nil, TripAccess{Level: AccessLevelNone}, err
	}
	if trip.UserID != nil && *trip.UserID == actorUserID {
		return trip, TripAccess{Level: AccessLevelOwner}, nil
	}

	collaborator, err := s.repo.GetTripCollaboratorByTripAndUser(ctx, tripID, actorUserID)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			return nil, TripAccess{Level: AccessLevelNone}, domainerrs.ErrNotFound
		}
		return nil, TripAccess{Level: AccessLevelNone}, err
	}
	if collaborator.Status != entity.CollaboratorStatusAccepted {
		return nil, TripAccess{Level: AccessLevelNone}, domainerrs.ErrNotFound
	}

	switch collaborator.Role {
	case entity.CollaboratorRoleEditor:
		return trip, TripAccess{Level: AccessLevelEditor}, nil
	case entity.CollaboratorRoleViewer:
		return trip, TripAccess{Level: AccessLevelViewer}, nil
	default:
		return nil, TripAccess{Level: AccessLevelNone}, domainerrs.ErrNotFound
	}
}

func (s *Service) requireOwner(ctx context.Context, tripID, actorUserID uuid.UUID) (*entity.Trip, TripAccess, error) {
	trip, access, err := s.tripForAccess(ctx, tripID, actorUserID)
	if err != nil {
		return nil, access, err
	}
	if !access.CanManageCollaborators() {
		return nil, access, apperrs.ErrForbidden
	}
	return trip, access, nil
}

func (s *Service) requireEditorOrOwner(ctx context.Context, tripID, actorUserID uuid.UUID) (*entity.Trip, TripAccess, error) {
	trip, access, err := s.tripForAccess(ctx, tripID, actorUserID)
	if err != nil {
		return nil, access, err
	}
	if !access.CanEdit() {
		return nil, access, apperrs.ErrForbidden
	}
	return trip, access, nil
}

func (s *Service) requireViewerEditorOrOwner(ctx context.Context, tripID, actorUserID uuid.UUID) (*entity.Trip, TripAccess, error) {
	trip, access, err := s.tripForAccess(ctx, tripID, actorUserID)
	if err != nil {
		return nil, access, err
	}
	if !access.CanView() {
		return nil, access, domainerrs.ErrNotFound
	}
	return trip, access, nil
}
