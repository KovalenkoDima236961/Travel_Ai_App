package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/analytics"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

func (s *Service) EvaluateTripPolicy(
	ctx context.Context,
	tripID uuid.UUID,
) (workspacepolicies.Evaluation, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return workspacepolicies.Evaluation{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return workspacepolicies.Evaluation{}, err
	}
	return s.evaluateTripPolicyForTrip(ctx, trip)
}

func (s *Service) evaluateTripPolicyForTrip(
	ctx context.Context,
	trip *entity.Trip,
) (workspacepolicies.Evaluation, error) {
	if trip.WorkspaceID == nil {
		return workspacepolicies.NotApplicableEvaluation(trip.ID, nil, "personal_trip"), nil
	}
	if s.workspacePolicyProvider == nil {
		return workspacepolicies.NotApplicableEvaluation(
			trip.ID, trip.WorkspaceID, "no_active_policy",
		), nil
	}
	policy, err := s.workspacePolicyProvider.GetActive(ctx, *trip.WorkspaceID)
	if err != nil {
		if errors.Is(err, domainerrs.ErrNotFound) {
			return workspacepolicies.NotApplicableEvaluation(
				trip.ID, trip.WorkspaceID, "no_active_policy",
			), nil
		}
		return workspacepolicies.Evaluation{}, err
	}
	itinerary := parseItineraryLenient(trip.Itinerary)
	analyticsByCurrency := make(map[string]analytics.TripCostAnalytics)
	for _, currency := range policyCurrencies(policy) {
		value, err := s.tripCostAnalyticsForTrip(ctx, trip, currency, time.Now().UTC())
		if err != nil {
			s.log.Warn("workspace policy: cost analytics unavailable",
				zap.String("trip_id", trip.ID.String()),
				zap.String("currency", currency),
				zap.Error(err))
			continue
		}
		analyticsByCurrency[currency] = value
	}

	var splitting *workspacepolicies.CostSplittingSnapshot
	if policy.Rules.Rules.RequireCostSplitting.Enabled {
		travelers, err := s.repo.ListActiveTripTravelersByTrip(ctx, trip.ID)
		if err != nil {
			return workspacepolicies.Evaluation{}, err
		}
		currency, err := resolveCostSplitCurrency("", trip, itinerary)
		if err == nil {
			summary, err := s.calculateCostSplittingSummary(
				ctx, trip, itinerary, travelers, currency, time.Now().UTC(),
			)
			if err != nil {
				return workspacepolicies.Evaluation{}, err
			}
			splitting = &workspacepolicies.CostSplittingSnapshot{
				Currency:          summary.Currency,
				TravelerCount:     summary.Summary.TravelerCount,
				UnassignedTotal:   summary.Summary.UnassignedTotal,
				DefaultSplitCount: summary.Summary.DefaultSplitCount,
				InvalidSplitCount: summary.Summary.InvalidSplitCount,
			}
		}
	}
	return workspacepolicies.Evaluate(ctx, workspacepolicies.EvaluationInput{
		Trip:                trip,
		Policy:              policy,
		Itinerary:           itinerary,
		AnalyticsByCurrency: analyticsByCurrency,
		CostSplitting:       splitting,
		Converter:           s.budgetConversionProvider,
		ConversionEnabled:   s.budgetConversionEnabled,
	}), nil
}

func (s *Service) workspacePolicyAIConstraints(
	ctx context.Context,
	trip *entity.Trip,
) *workspacepolicies.AIConstraints {
	if trip == nil || trip.WorkspaceID == nil || s.workspacePolicyProvider == nil {
		return nil
	}
	policy, err := s.workspacePolicyProvider.GetActive(ctx, *trip.WorkspaceID)
	if err != nil {
		if !errors.Is(err, domainerrs.ErrNotFound) {
			s.log.Warn("workspace policy: failed to load AI constraints",
				zap.String("trip_id", trip.ID.String()), zap.Error(err))
		}
		return nil
	}
	return workspacepolicies.BuildAIConstraints(policy)
}

func policyCurrencies(policy *workspacepolicies.Policy) []string {
	if policy == nil {
		return nil
	}
	rules := policy.Rules.Rules
	values := []struct {
		enabled  bool
		currency string
	}{
		{rules.MaxTripBudget.Enabled, rules.MaxTripBudget.Currency},
		{rules.MaxDailyBudget.Enabled, rules.MaxDailyBudget.Currency},
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !value.enabled {
			continue
		}
		currency := strings.ToUpper(strings.TrimSpace(value.currency))
		if currency == "" {
			continue
		}
		if _, ok := seen[currency]; ok {
			continue
		}
		seen[currency] = struct{}{}
		result = append(result, currency)
	}
	return result
}
