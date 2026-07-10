package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"

	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/planningconstraints"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/usercontext"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

func (s *Service) PreviewPlanningConstraints(
	ctx context.Context,
	req planningconstraints.PreviewRequest,
) (*planningconstraints.PreviewResponse, error) {
	if err := planningconstraints.ValidatePreviewRequest(req); err != nil {
		return nil, apperrs.NewInvalidInput("%s", err.Error())
	}
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	var trip *entity.Trip
	if req.TripID != nil && planningconstraints.IncludeTripState(req.IncludeTripState) {
		trip, _, err = s.requireViewerEditorOrOwner(ctx, *req.TripID, user.ID)
		if err != nil {
			return nil, err
		}
		if req.WorkspaceID != nil && (trip.WorkspaceID == nil || *trip.WorkspaceID != *req.WorkspaceID) {
			return nil, apperrs.NewInvalidInput("workspaceId does not match trip workspace")
		}
	}
	workspaceID := req.WorkspaceID
	if workspaceID == nil && trip != nil {
		workspaceID = trip.WorkspaceID
	}
	if workspaceID != nil && trip == nil {
		if err := s.requireWorkspaceTripCreateAccess(ctx, user.ID, *workspaceID); err != nil {
			return nil, err
		}
	}

	userCtx, err := s.loadUserContext(ctx, user, tripIDForContext(trip))
	if err != nil {
		return nil, err
	}
	policy, err := s.activeWorkspacePolicy(ctx, workspaceID, planningconstraints.IncludeWorkspacePolicy(req.IncludeWorkspacePolicy))
	if err != nil {
		return nil, err
	}
	previousTrips, err := s.previousTripsForPlanningConstraints(
		ctx,
		user.ID,
		planningconstraints.IncludePreviousSignals(req.Source, req.IncludePreviousTripSignals),
	)
	if err != nil {
		return nil, err
	}

	constraints := planningconstraints.Build(planningconstraints.BuildInput{
		UserID:                     user.ID,
		Trip:                       trip,
		WorkspaceID:                workspaceID,
		Source:                     req.Source,
		Request:                    req.Request,
		UserContext:                userCtx,
		WorkspacePolicy:            policy,
		PreviousTrips:              previousTrips,
		IncludePreviousTripSignals: planningconstraints.IncludePreviousSignals(req.Source, req.IncludePreviousTripSignals),
		IncludeRoute:               planningconstraints.IncludeRoute(req.IncludeRoute),
	})
	return &planningconstraints.PreviewResponse{
		Constraints: constraints,
		Summary:     planningconstraints.SummaryFor(constraints),
		Warnings:    constraints.Warnings,
		Blockers:    constraints.Blockers,
	}, nil
}

func (s *Service) buildPlanningConstraints(
	ctx context.Context,
	user auth.AuthenticatedUser,
	source planningconstraints.Source,
	trip *entity.Trip,
	request planningconstraints.RequestOverride,
	userCtx usercontext.UserContext,
	includePrevious bool,
) (*planningconstraints.PlanningConstraints, error) {
	var workspaceID *uuid.UUID
	if trip != nil {
		workspaceID = trip.WorkspaceID
	}
	policy, err := s.activeWorkspacePolicy(ctx, workspaceID, true)
	if err != nil {
		return nil, err
	}
	previousTrips, err := s.previousTripsForPlanningConstraints(ctx, user.ID, includePrevious)
	if err != nil {
		return nil, err
	}
	constraints := planningconstraints.Build(planningconstraints.BuildInput{
		UserID:                     user.ID,
		Trip:                       trip,
		WorkspaceID:                workspaceID,
		Source:                     source,
		Request:                    request,
		UserContext:                userCtx,
		WorkspacePolicy:            policy,
		PreviousTrips:              previousTrips,
		IncludePreviousTripSignals: includePrevious,
		IncludeRoute:               true,
	})
	s.logPlanningConstraintsSummary(trip, constraints)
	return &constraints, nil
}

func (s *Service) requireNoPlanningBlockers(
	constraints *planningconstraints.PlanningConstraints,
	source planningconstraints.Source,
) error {
	if constraints == nil || len(constraints.Blockers) == 0 || planningconstraints.AllowsBlockingOverride(source) {
		return nil
	}
	return planningconstraints.NewBlockingError(*constraints)
}

func (s *Service) activeWorkspacePolicy(
	ctx context.Context,
	workspaceID *uuid.UUID,
	include bool,
) (*workspacepolicies.Policy, error) {
	if !include || workspaceID == nil || s.workspacePolicyProvider == nil {
		return nil, nil
	}
	policy, err := s.workspacePolicyProvider.GetActive(ctx, *workspaceID)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return policy, nil
}

func (s *Service) previousTripsForPlanningConstraints(
	ctx context.Context,
	userID uuid.UUID,
	include bool,
) ([]entity.Trip, error) {
	if !include {
		return nil, nil
	}
	trips, err := s.repo.ListByUser(ctx, userID, 15, 0)
	if err != nil {
		s.log.Warn("planning constraints: previous trips unavailable", zap.Error(err))
		return nil, nil
	}
	return trips, nil
}

func tripIDForContext(trip *entity.Trip) uuid.UUID {
	if trip == nil {
		return uuid.Nil
	}
	return trip.ID
}

func (s *Service) logPlanningConstraintsSummary(trip *entity.Trip, constraints planningconstraints.PlanningConstraints) {
	fields := []zap.Field{
		zap.String("source", string(constraints.Source)),
		zap.String("language", constraints.Language),
		zap.Int("warning_count", len(constraints.Warnings)),
		zap.Int("blocker_count", len(constraints.Blockers)),
	}
	if trip != nil {
		fields = append(fields, zap.String("trip_id", trip.ID.String()))
	}
	if constraints.WorkspaceID != nil {
		fields = append(fields, zap.String("workspace_id", constraints.WorkspaceID.String()))
	}
	s.log.Info("planning constraints built", fields...)
}
