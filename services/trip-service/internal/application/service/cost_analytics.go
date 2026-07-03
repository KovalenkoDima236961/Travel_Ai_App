package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/analytics"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
)

const maxWorkspaceAnalyticsTrips = 500

func (s *Service) GetTripCostAnalytics(ctx context.Context, tripID uuid.UUID, currency string) (analytics.TripCostAnalytics, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return analytics.TripCostAnalytics{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return analytics.TripCostAnalytics{}, err
	}
	return s.tripCostAnalyticsForTrip(ctx, trip, currency, time.Now().UTC())
}

func (s *Service) GetWorkspaceCostAnalytics(
	ctx context.Context,
	workspaceID uuid.UUID,
	in appdto.WorkspaceCostAnalyticsInput,
) (analytics.WorkspaceCostAnalytics, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return analytics.WorkspaceCostAnalytics{}, err
	}
	if err := s.requireWorkspaceAnalyticsAccess(ctx, user.ID, workspaceID); err != nil {
		return analytics.WorkspaceCostAnalytics{}, err
	}
	if in.From != nil && in.To != nil && in.From.After(*in.To) {
		return analytics.WorkspaceCostAnalytics{}, apperrs.NewInvalidInput("from must be before or equal to to")
	}
	targetCurrency := in.Currency
	if targetCurrency == "" {
		targetCurrency = budget.DefaultCurrency
	}

	trips, err := s.repo.ListAccessible(
		ctx,
		user.ID,
		[]uuid.UUID{workspaceID},
		appdto.TripListScopeWorkspace,
		&workspaceID,
		maxWorkspaceAnalyticsTrips,
		0,
	)
	if err != nil {
		return analytics.WorkspaceCostAnalytics{}, err
	}

	generatedAt := time.Now().UTC()
	items := make([]analytics.WorkspaceTripInput, 0, len(trips))
	for _, trip := range trips {
		if !tripInAnalyticsDateRange(trip, in.From, in.To) {
			continue
		}
		tripAnalytics, err := s.tripCostAnalyticsForTrip(ctx, &trip, targetCurrency, generatedAt)
		if err != nil {
			return analytics.WorkspaceCostAnalytics{}, err
		}
		items = append(items, analytics.WorkspaceTripInput{
			Trip:      trip,
			Analytics: tripAnalytics,
		})
	}

	result := analytics.CalculateWorkspaceCost(analytics.WorkspaceInput{
		WorkspaceID: workspaceID,
		Currency:    targetCurrency,
		GeneratedAt: generatedAt,
		From:        in.From,
		To:          in.To,
		Trips:       items,
	})
	if len(trips) >= maxWorkspaceAnalyticsTrips {
		result.Warnings = append(result.Warnings, "Workspace analytics include the first 500 accessible trips.")
	}
	return result, nil
}

func (s *Service) tripCostAnalyticsForTrip(
	ctx context.Context,
	trip *entity.Trip,
	requestedCurrency string,
	generatedAt time.Time,
) (analytics.TripCostAnalytics, error) {
	itinerary := parseItineraryLenient(trip.Itinerary)
	targetCurrency := analytics.ResolveTripCurrency(requestedCurrency, trip, itinerary)

	summary, err := budget.CalculateBudgetSummaryWithConversion(ctx, budget.TripBudget{
		Amount:        nil,
		Currency:      targetCurrency,
		Days:          int(trip.Days),
		Accommodation: trip.Accommodation,
	}, itinerary, s.budgetConversionProvider, budget.ConversionOptions{
		Enabled:  s.budgetConversionEnabled,
		FailOpen: s.budgetConversionFailOpen,
	})
	if err != nil {
		if s.budgetConversionFailOpen {
			summary = budget.CalculateBudgetSummary(budget.TripBudget{
				Amount:        nil,
				Currency:      targetCurrency,
				Days:          int(trip.Days),
				Accommodation: trip.Accommodation,
			}, itinerary)
		} else {
			return analytics.TripCostAnalytics{}, apperrs.ErrBudgetConversionFailed
		}
	}

	result, err := analytics.CalculateTripCost(ctx, analytics.TripInput{
		Trip:               trip,
		Itinerary:          itinerary,
		BudgetSummary:      summary,
		Currency:           targetCurrency,
		GeneratedAt:        generatedAt,
		Converter:          s.budgetConversionProvider,
		ConversionEnabled:  s.budgetConversionEnabled,
		ConversionFailOpen: s.budgetConversionFailOpen,
	})
	if err != nil {
		if s.budgetConversionFailOpen {
			return analytics.CalculateTripCost(ctx, analytics.TripInput{
				Trip:      trip,
				Itinerary: itinerary,
				BudgetSummary: budget.CalculateBudgetSummary(budget.TripBudget{
					Amount:        nil,
					Currency:      targetCurrency,
					Days:          int(trip.Days),
					Accommodation: trip.Accommodation,
				}, itinerary),
				Currency:           targetCurrency,
				GeneratedAt:        generatedAt,
				ConversionEnabled:  false,
				ConversionFailOpen: true,
			})
		}
		return analytics.TripCostAnalytics{}, apperrs.ErrBudgetConversionFailed
	}
	return result, nil
}

func (s *Service) requireWorkspaceAnalyticsAccess(ctx context.Context, userID, workspaceID uuid.UUID) error {
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
	case workspaces.RoleOwner, workspaces.RoleAdmin, workspaces.RoleMember, workspaces.RoleViewer:
		return nil
	default:
		return apperrs.ErrForbidden
	}
}

func tripInAnalyticsDateRange(trip entity.Trip, from, to *time.Time) bool {
	if from == nil && to == nil {
		return true
	}
	if trip.StartDate == nil {
		return false
	}
	start := truncateDate(*trip.StartDate)
	if from != nil && start.Before(truncateDate(*from)) {
		return false
	}
	if to != nil && start.After(truncateDate(*to)) {
		return false
	}
	return true
}

func truncateDate(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}
