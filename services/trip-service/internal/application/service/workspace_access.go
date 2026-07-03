package service

import (
	"context"

	"github.com/google/uuid"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
)

func (s *Service) requireWorkspaceTripCreateAccess(ctx context.Context, userID, workspaceID uuid.UUID) error {
	if !s.workspacesEnabled || s.workspaceProvider == nil {
		return apperrs.ErrForbidden
	}
	access, err := s.workspaceProvider.AccessCheck(ctx, userID, workspaceID)
	if err != nil {
		return err
	}
	if access == nil || !access.HasAccess {
		return apperrs.ErrForbidden
	}
	switch access.Role {
	case workspaces.RoleOwner, workspaces.RoleAdmin, workspaces.RoleMember:
		return nil
	default:
		return apperrs.ErrForbidden
	}
}

func (s *Service) accessibleWorkspaceIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if !s.workspacesEnabled || s.workspaceProvider == nil {
		return []uuid.UUID{}, nil
	}
	rows, err := s.workspaceProvider.ListForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids, nil
}

func normalizeTripListScope(scope appdto.TripListScope) appdto.TripListScope {
	switch scope {
	case appdto.TripListScopePersonal, appdto.TripListScopeWorkspace:
		return scope
	default:
		return appdto.TripListScopeAll
	}
}
