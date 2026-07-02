package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetoptimization"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
	tripobs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/observability"
)

func (s *Service) ListBudgetOptimizationProposals(
	ctx context.Context,
	tripID uuid.UUID,
	status string,
	limit int,
) ([]entity.BudgetOptimizationProposal, int, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	if _, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, 0, err
	}
	if limit == 0 {
		limit = defaultLimit
	}
	if limit < 1 || limit > maxLimit {
		return nil, 0, apperrs.NewInvalidInput("limit must be between 1 and %d", maxLimit)
	}

	var filter *entity.BudgetOptimizationProposalStatus
	if trimmed := strings.TrimSpace(status); trimmed != "" {
		normalized := entity.BudgetOptimizationProposalStatus(strings.ToLower(trimmed))
		if !validBudgetOptimizationStatus(normalized) {
			return nil, 0, apperrs.NewInvalidInput("status is invalid")
		}
		filter = &normalized
	}

	proposals, err := s.repo.ListBudgetOptimizationProposalsByTrip(ctx, tripID, filter, limit)
	return proposals, limit, err
}

func (s *Service) GetBudgetOptimizationProposal(
	ctx context.Context,
	tripID, proposalID uuid.UUID,
) (*entity.BudgetOptimizationProposal, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}
	return s.repo.GetBudgetOptimizationProposalByIDAndTrip(ctx, proposalID, tripID)
}

func (s *Service) OptimizeBudgetDayForActor(
	ctx context.Context,
	tripID, actorUserID uuid.UUID,
	jobID *uuid.UUID,
	dayNumber int,
	instruction string,
	expectedRevision int,
	payload budgetoptimization.JobPayload,
) (*entity.Trip, error) {
	ctx = contextWithActor(ctx, actorUserID)
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}

	current, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	if err := checkCurrentItineraryRevision(expectedRevision, current.ItineraryRevision); err != nil {
		return nil, err
	}
	currentItinerary, _, err := currentItineraryAndDayIndex(current, dayNumber)
	if err != nil {
		return nil, err
	}

	userContext, err := s.loadUserContext(ctx, user, tripID)
	if err != nil {
		return nil, err
	}
	weatherForecast, err := s.loadWeatherContext(ctx, *current, tripID)
	if err != nil {
		return nil, err
	}
	summary, err := s.budgetSummaryForTrip(ctx, current)
	if err != nil {
		if s.budgetConversionFailOpen {
			summary = budget.CalculateBudgetSummary(budget.TripBudget{
				Amount:        current.BudgetAmount,
				Currency:      current.BudgetCurrency,
				Days:          int(current.Days),
				Accommodation: current.Accommodation,
			}, currentItinerary)
		} else {
			return nil, apperrs.ErrBudgetConversionFailed
		}
	}

	input, err := budgetoptimization.BuildOptimizeDayInput(
		*current,
		currentItinerary,
		dayNumber,
		summary,
		payload,
		instruction,
		userContext.Profile,
		userContext.Preferences,
		weatherForecast,
	)
	if err != nil {
		return nil, err
	}

	content, err := s.generator.OptimizeBudgetDay(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := budgetoptimization.NormalizeProposalContent(content, dayNumber, input.BudgetContext.Currency); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}

	proposal, err := s.repo.CreateBudgetOptimizationProposal(ctx, &entity.BudgetOptimizationProposal{
		ID:                        uuid.New(),
		TripID:                    tripID,
		JobID:                     jobID,
		CreatedByUserID:           user.ID,
		Scope:                     entity.BudgetOptimizationScopeDay,
		DayNumber:                 &dayNumber,
		ExpectedItineraryRevision: expectedRevision,
		BaseItineraryRevision:     current.ItineraryRevision,
		Status:                    entity.BudgetOptimizationProposalStatusPending,
		Currency:                  content.Currency,
		TargetReductionAmount:     payload.TargetReductionAmount,
		EstimatedSavingsAmount:    &content.EstimatedSavingsAmount,
		ProposalJSON:              raw,
	})
	if err != nil {
		return nil, err
	}
	tripobs.RecordBudgetOptimizationProposalCreated(string(proposal.Status))

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventBudgetOptimizationProposed,
		EntityType:  activityEntityType(activity.EntityItineraryDay),
		EntityID:    activityEntityID(proposal.ID),
		Metadata: map[string]any{
			"proposalId":             proposal.ID.String(),
			"jobId":                  idString(jobID),
			"dayNumber":              dayNumber,
			"estimatedSavingsAmount": content.EstimatedSavingsAmount,
			"currency":               content.Currency,
		},
	})

	trip := tripID
	proposalID := proposal.ID
	s.sendNotifications(ctx, []notifications.NotificationCreateInput{{
		UserID:     user.ID,
		TripID:     &trip,
		Type:       notifications.TypeBudgetOptimizationReady,
		Title:      "Budget proposal ready",
		Message:    fmt.Sprintf("A budget optimization proposal for Day %d is ready to review.", dayNumber),
		EntityType: activityEntityType(notifications.EntityItineraryDay),
		EntityID:   &proposalID,
		Metadata: map[string]any{
			"tripId":     tripID.String(),
			"proposalId": proposal.ID.String(),
			"dayNumber":  dayNumber,
		},
	}})

	return current, nil
}

func (s *Service) ApplyBudgetOptimizationProposal(
	ctx context.Context,
	tripID, proposalID uuid.UUID,
	expectedItineraryRevision *int,
) (*entity.Trip, *entity.BudgetOptimizationProposal, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, nil, err
	}
	current, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, nil, err
	}
	expectedRevision, err := requireExpectedItineraryRevision(expectedItineraryRevision)
	if err != nil {
		return nil, nil, err
	}
	if err := checkCurrentItineraryRevision(expectedRevision, current.ItineraryRevision); err != nil {
		return nil, nil, err
	}
	ownerID, err := tripOwnerID(current)
	if err != nil {
		return nil, nil, err
	}

	proposal, err := s.repo.GetBudgetOptimizationProposalByIDAndTrip(ctx, proposalID, tripID)
	if err != nil {
		return nil, nil, err
	}
	if proposal.Status != entity.BudgetOptimizationProposalStatusPending {
		return nil, nil, apperrs.NewInvalidInput("proposal is not pending")
	}
	if proposal.BaseItineraryRevision != current.ItineraryRevision {
		_, _ = s.repo.MarkBudgetOptimizationProposalExpired(ctx, proposal.ID)
		return nil, nil, apperrs.NewItineraryConflict(current.ItineraryRevision)
	}
	if proposal.DayNumber == nil {
		return nil, nil, apperrs.NewInvalidInput("proposal is invalid")
	}

	var content budgetoptimization.ProposalContent
	if err := json.Unmarshal(proposal.ProposalJSON, &content); err != nil {
		return nil, nil, apperrs.NewInvalidInput("proposal is invalid")
	}
	if err := budgetoptimization.NormalizeProposalContent(&content, *proposal.DayNumber, proposal.Currency); err != nil {
		return nil, nil, err
	}

	currentItinerary, dayIndex, err := currentItineraryAndDayIndex(current, *proposal.DayNumber)
	if err != nil {
		return nil, nil, err
	}
	currentItinerary.Days[dayIndex] = content.ProposedDay

	updated, err := s.saveRegeneratedItinerary(
		ctx,
		tripID,
		ownerID,
		user.ID,
		currentItinerary,
		expectedRevision,
		entity.ItineraryVersionSourceBudgetOptimizationApplied,
		map[string]any{
			"source":     "budget_optimization_applied",
			"proposalId": proposal.ID.String(),
			"dayNumber":  *proposal.DayNumber,
		},
	)
	if err != nil {
		return nil, nil, err
	}

	applied, err := s.repo.MarkBudgetOptimizationProposalApplied(ctx, proposal.ID, updated.ItineraryRevision)
	if err != nil {
		return nil, nil, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventBudgetOptimizationApplied,
		EntityType:  activityEntityType(activity.EntityItineraryDay),
		EntityID:    activityEntityID(proposal.ID),
		Metadata: map[string]any{
			"proposalId":             proposal.ID.String(),
			"dayNumber":              *proposal.DayNumber,
			"estimatedSavingsAmount": applied.EstimatedSavingsAmount,
			"currency":               applied.Currency,
		},
	})
	return updated, applied, nil
}

func (s *Service) DiscardBudgetOptimizationProposal(
	ctx context.Context,
	tripID, proposalID uuid.UUID,
) (*entity.BudgetOptimizationProposal, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}
	proposal, err := s.repo.GetBudgetOptimizationProposalByIDAndTrip(ctx, proposalID, tripID)
	if err != nil {
		return nil, err
	}
	if proposal.Status != entity.BudgetOptimizationProposalStatusPending {
		return proposal, nil
	}
	discarded, err := s.repo.MarkBudgetOptimizationProposalDiscarded(ctx, proposalID)
	if err != nil {
		if errorsIsNotFound(err) {
			return proposal, nil
		}
		return nil, err
	}
	dayNumber := 0
	if discarded.DayNumber != nil {
		dayNumber = *discarded.DayNumber
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventBudgetOptimizationDiscarded,
		EntityType:  activityEntityType(activity.EntityItineraryDay),
		EntityID:    activityEntityID(discarded.ID),
		Metadata: map[string]any{
			"proposalId": discarded.ID.String(),
			"dayNumber":  dayNumber,
		},
	})
	return discarded, nil
}

func validBudgetOptimizationStatus(status entity.BudgetOptimizationProposalStatus) bool {
	switch status {
	case entity.BudgetOptimizationProposalStatusPending,
		entity.BudgetOptimizationProposalStatusApplied,
		entity.BudgetOptimizationProposalStatusDiscarded,
		entity.BudgetOptimizationProposalStatusExpired,
		entity.BudgetOptimizationProposalStatusFailed:
		return true
	default:
		return false
	}
}

func idString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func errorsIsNotFound(err error) bool {
	return errors.Is(err, domainerrs.ErrNotFound)
}
