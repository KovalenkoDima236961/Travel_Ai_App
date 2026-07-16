package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	tripsecurity "github.com/KovalenkoDima236961/Travel_Ai_App/internal/security"
)

type AccessLevel string

const (
	AccessLevelOwner  AccessLevel = "owner"
	AccessLevelEditor AccessLevel = "editor"
	AccessLevelViewer AccessLevel = "viewer"
	AccessLevelNone   AccessLevel = "none"
)

type TripAccess struct {
	Level         AccessLevel
	Source        string
	WorkspaceRole string
}

func (a TripAccess) CanView() bool {
	return a.Allows(tripsecurity.PermissionTripView)
}

func (a TripAccess) CanEdit() bool {
	return a.Allows(tripsecurity.PermissionTripEdit)
}

func (a TripAccess) CanManageCollaborators() bool {
	return a.Allows(tripsecurity.PermissionCollaboratorsManage)
}

func (a TripAccess) CanManageShare() bool {
	return a.Allows(tripsecurity.PermissionShareManage)
}

func (a TripAccess) CanRestoreVersion() bool {
	return a.CanEdit()
}

func (a TripAccess) CanDelete() bool {
	return a.Allows(tripsecurity.PermissionTripDelete)
}

func (a TripAccess) Allows(permission tripsecurity.TripPermission) bool {
	return tripsecurity.Authorize(tripsecurity.TripAccessContext{
		Principal:     tripsecurity.Principal{Type: tripsecurity.PrincipalAuthenticatedUser},
		Role:          string(a.Level),
		WorkspaceRole: a.WorkspaceRole,
		Accepted:      a.Level != AccessLevelNone,
	}, permission).Allowed
}

func (a TripAccess) Role() string {
	if a.Source == "workspace" && a.WorkspaceRole != "" {
		return a.WorkspaceRole
	}
	switch a.Level {
	case AccessLevelOwner:
		return "owner"
	case AccessLevelEditor:
		return "editor"
	case AccessLevelViewer:
		return "viewer"
	default:
		return "none"
	}
}

func (a TripAccess) SourceName() string {
	if a.Source != "" {
		return a.Source
	}
	return "collaborator"
}

// GetTripAccess resolves the current authenticated user's private access level
// for a trip. Public share viewers are intentionally not represented here.
func (s *Service) GetTripAccess(ctx context.Context, tripID uuid.UUID) (TripAccess, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return TripAccess{Level: AccessLevelNone}, err
	}
	_, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	return access, err
}

func (s *Service) GetTripForActor(ctx context.Context, tripID, actorUserID uuid.UUID) (*entity.Trip, TripAccess, error) {
	return s.requireViewerEditorOrOwner(ctx, tripID, actorUserID)
}

func (s *Service) tripForAccess(ctx context.Context, tripID, actorUserID uuid.UUID) (*entity.Trip, TripAccess, error) {
	trip, err := s.repo.GetByID(ctx, tripID)
	if err != nil {
		return nil, TripAccess{Level: AccessLevelNone}, err
	}
	access := TripAccess{Level: AccessLevelNone}
	if trip.WorkspaceID == nil && trip.UserID != nil && *trip.UserID == actorUserID {
		access = strongestAccess(access, TripAccess{Level: AccessLevelOwner, Source: "owner"})
	}
	if trip.WorkspaceID != nil {
		workspaceAccess, err := s.workspaceTripAccess(ctx, actorUserID, *trip.WorkspaceID)
		if err != nil {
			return nil, TripAccess{Level: AccessLevelNone}, err
		}
		access = strongestAccess(access, workspaceAccess)
	}

	collaborator, err := s.repo.GetTripCollaboratorByTripAndUser(ctx, tripID, actorUserID)
	if err != nil {
		if !errors.Is(err, domainerrs.ErrNotFound) {
			return nil, TripAccess{Level: AccessLevelNone}, err
		}
	} else if collaborator.Status == entity.CollaboratorStatusAccepted {
		switch collaborator.Role {
		case entity.CollaboratorRoleEditor:
			access = strongestAccess(access, TripAccess{Level: AccessLevelEditor, Source: "collaborator"})
		case entity.CollaboratorRoleViewer:
			access = strongestAccess(access, TripAccess{Level: AccessLevelViewer, Source: "collaborator"})
		}
	}
	if !access.CanView() {
		return nil, TripAccess{Level: AccessLevelNone}, domainerrs.ErrNotFound
	}
	return trip, access, nil
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

func (s *Service) workspaceTripAccess(ctx context.Context, actorUserID, workspaceID uuid.UUID) (TripAccess, error) {
	if !s.workspacesEnabled || s.workspaceProvider == nil {
		return TripAccess{Level: AccessLevelNone}, nil
	}
	access, err := s.workspaceProvider.AccessCheck(ctx, actorUserID, workspaceID)
	if err != nil {
		return TripAccess{Level: AccessLevelNone}, err
	}
	if access == nil || !access.HasAccess {
		return TripAccess{Level: AccessLevelNone}, nil
	}
	switch access.Role {
	case "owner", "admin":
		return TripAccess{Level: AccessLevelOwner, Source: "workspace", WorkspaceRole: string(access.Role)}, nil
	case "member":
		return TripAccess{Level: AccessLevelEditor, Source: "workspace", WorkspaceRole: string(access.Role)}, nil
	case "viewer":
		return TripAccess{Level: AccessLevelViewer, Source: "workspace", WorkspaceRole: string(access.Role)}, nil
	default:
		return TripAccess{Level: AccessLevelNone}, nil
	}
}

func strongestAccess(current, candidate TripAccess) TripAccess {
	if accessRank(candidate.Level) > accessRank(current.Level) {
		return candidate
	}
	return current
}

func accessRank(level AccessLevel) int {
	switch level {
	case AccessLevelOwner:
		return 3
	case AccessLevelEditor:
		return 2
	case AccessLevelViewer:
		return 1
	default:
		return 0
	}
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
