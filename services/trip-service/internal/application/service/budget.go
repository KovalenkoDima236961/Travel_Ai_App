package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

// GetBudgetSummary computes the on-demand budget summary for a trip from its
// budget and itinerary. Any accepted collaborator (owner/editor/viewer) may
// read it; non-collaborators get a not-found error from the access check.
func (s *Service) GetBudgetSummary(ctx context.Context, tripID uuid.UUID) (budget.Summary, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return budget.Summary{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return budget.Summary{}, err
	}
	return budgetSummaryForTrip(trip), nil
}

// UpdateTripBudget validates and persists the trip-level budget. Only owner and
// editor may update it. The update touches only the budget columns and does not
// mutate itinerary_revision, since the itinerary JSON is unchanged.
func (s *Service) UpdateTripBudget(ctx context.Context, tripID uuid.UUID, in appdto.UpdateTripBudgetInput) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	current, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	ownerID, err := tripOwnerID(current)
	if err != nil {
		return nil, err
	}

	var (
		amount   *float64
		currency string
	)
	if !in.Clear {
		amount, currency, err = budget.NormalizeBudgetInput(in.Amount, in.Currency, current.BudgetCurrency)
		if err != nil {
			return nil, apperrs.NewInvalidInput("%s", err.Error())
		}
	}

	updated, err := s.repo.UpdateTripBudget(ctx, tripID, ownerID, amount, currency)
	if err != nil {
		return nil, err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTripBudgetUpdated,
		EntityType:  activityEntityType(activity.EntityTrip),
		EntityID:    activityEntityID(tripID),
		Metadata:    budgetActivityMetadata(updated),
	})

	return updated, nil
}

func budgetSummaryForTrip(trip *entity.Trip) budget.Summary {
	return budget.CalculateBudgetSummary(budget.TripBudget{
		Amount:   trip.BudgetAmount,
		Currency: trip.BudgetCurrency,
		Days:     int(trip.Days),
	}, parseItineraryLenient(trip.Itinerary))
}

func budgetActivityMetadata(trip *entity.Trip) map[string]any {
	if trip.BudgetAmount == nil {
		return map[string]any{"cleared": true}
	}
	return map[string]any{
		"amount":   *trip.BudgetAmount,
		"currency": trip.BudgetCurrency,
	}
}

// parseItineraryLenient decodes the stored itinerary JSON without enforcing the
// strict validation used by mutation paths, so a budget summary is always
// computable even when the itinerary is slightly off.
func parseItineraryLenient(raw json.RawMessage) aggregate.Itinerary {
	if len(raw) == 0 || strings.EqualFold(strings.TrimSpace(string(raw)), "null") {
		return aggregate.Itinerary{}
	}
	var itinerary aggregate.Itinerary
	if err := json.Unmarshal(raw, &itinerary); err != nil {
		return aggregate.Itinerary{}
	}
	return itinerary
}
