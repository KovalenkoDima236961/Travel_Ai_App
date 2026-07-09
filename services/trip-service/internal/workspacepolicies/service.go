package workspacepolicies

import (
	"context"
	"errors"

	"github.com/google/uuid"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
)

type WorkspaceAccess interface {
	AccessCheck(context.Context, uuid.UUID, uuid.UUID) (*workspaces.Access, error)
}

type Service struct {
	repository Repository
	workspaces WorkspaceAccess
}

func New(repository Repository, workspaces WorkspaceAccess) *Service {
	return &Service{repository: repository, workspaces: workspaces}
}

func (s *Service) Get(ctx context.Context, workspaceID uuid.UUID) (GetResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return GetResponse{}, err
	}
	if _, err := s.requireRole(ctx, user.ID, workspaceID, false); err != nil {
		return GetResponse{}, err
	}
	policy, err := s.repository.GetActive(ctx, workspaceID)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			defaults := DefaultRules()
			return GetResponse{Policy: nil, Defaults: &defaults}, nil
		}
		return GetResponse{}, err
	}
	return GetResponse{Policy: policy}, nil
}

func (s *Service) Upsert(
	ctx context.Context,
	workspaceID uuid.UUID,
	input UpsertInput,
) (*Policy, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireRole(ctx, user.ID, workspaceID, true); err != nil {
		return nil, err
	}
	if err := ValidateInput(&input); err != nil {
		return nil, apperrs.NewInvalidInput("%s", err)
	}
	return s.repository.UpsertActive(ctx, workspaceID, user.ID, input)
}

func (s *Service) Archive(ctx context.Context, workspaceID uuid.UUID) (*Policy, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireRole(ctx, user.ID, workspaceID, true); err != nil {
		return nil, err
	}
	return s.repository.ArchiveActive(ctx, workspaceID, user.ID)
}

func (s *Service) GetActive(
	ctx context.Context,
	workspaceID uuid.UUID,
) (*Policy, error) {
	return s.repository.GetActive(ctx, workspaceID)
}

func (s *Service) requireRole(
	ctx context.Context,
	userID, workspaceID uuid.UUID,
	manage bool,
) (*workspaces.Access, error) {
	if s.workspaces == nil {
		return nil, apperrs.ErrForbidden
	}
	access, err := s.workspaces.AccessCheck(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	if access == nil || !access.HasAccess || access.WorkspaceArchived {
		return nil, apperrs.ErrForbidden
	}
	if manage && access.Role != workspaces.RoleOwner && access.Role != workspaces.RoleAdmin {
		return nil, apperrs.ErrForbidden
	}
	return access, nil
}
