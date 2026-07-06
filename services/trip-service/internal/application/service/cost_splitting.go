package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/aggregate"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
)

const (
	maxTravelerNameLength = 80

	splitTypeAllEqual          = "all_equal"
	splitTypeSelectedEqual     = "selected_equal"
	splitTypeCustomPercentages = "custom_percentages"

	ruleSourceExplicit = "explicit"
	ruleSourceDefault  = "default"
)

var costSplitCategoryOrder = []string{
	budget.CategoryFood,
	budget.CategoryTransport,
	budget.CategoryTicket,
	budget.CategoryActivity,
	budget.CategoryAccommodation,
	budget.CategoryShopping,
	budget.CategoryOther,
}

func (s *Service) ListTripTravelers(ctx context.Context, tripID uuid.UUID) ([]entity.TripTraveler, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}
	return s.repo.ListTripTravelersByTrip(ctx, tripID)
}

func (s *Service) CreateTripTraveler(
	ctx context.Context,
	tripID uuid.UUID,
	in appdto.CreateTripTravelerInput,
) (*entity.TripTraveler, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}

	name, err := normalizeTravelerName(in.Name)
	if err != nil {
		return nil, err
	}
	email, err := normalizeTravelerEmail(in.Email)
	if err != nil {
		return nil, err
	}
	role, err := normalizeTravelerRole(in.Role)
	if err != nil {
		return nil, err
	}
	if err := s.ensureTravelerUnique(ctx, tripID, uuid.Nil, email, in.LinkedUserID); err != nil {
		return nil, err
	}

	traveler := &entity.TripTraveler{
		ID:              uuid.New(),
		TripID:          tripID,
		Name:            name,
		Email:           email,
		LinkedUserID:    in.LinkedUserID,
		Role:            role,
		Status:          entity.TripTravelerStatusActive,
		CreatedByUserID: user.ID,
	}
	created, err := s.repo.CreateTripTraveler(ctx, traveler)
	if err != nil {
		if errors.Is(err, domainerrs.ErrConflict) {
			return nil, apperrs.NewInvalidInput("an active traveler with that email or linked user already exists")
		}
		return nil, err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTripTravelerAdded,
		EntityType:  activityEntityType(activity.EntityTripTraveler),
		EntityID:    activityEntityID(created.ID),
		Metadata: map[string]any{
			"travelerId":   created.ID.String(),
			"travelerName": created.Name,
		},
	})

	s.ResetApprovalIfApproved(ctx, tripID, user.ID, "Travelers changed")

	return created, nil
}

func (s *Service) UpdateTripTraveler(
	ctx context.Context,
	tripID, travelerID uuid.UUID,
	in appdto.UpdateTripTravelerInput,
) (*entity.TripTraveler, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetTripTravelerByID(ctx, tripID, travelerID)
	if err != nil {
		return nil, err
	}
	if existing.Status == entity.TripTravelerStatusRemoved {
		return nil, domainerrs.ErrNotFound
	}

	next := *existing
	if in.Name != nil {
		name, err := normalizeTravelerName(*in.Name)
		if err != nil {
			return nil, err
		}
		next.Name = name
	}
	if in.Email != nil {
		email, err := normalizeTravelerEmail(in.Email)
		if err != nil {
			return nil, err
		}
		next.Email = email
	}
	if in.Role != nil {
		role, err := normalizeTravelerRole(*in.Role)
		if err != nil {
			return nil, err
		}
		next.Role = role
	}
	if err := s.ensureTravelerUnique(ctx, tripID, travelerID, next.Email, next.LinkedUserID); err != nil {
		return nil, err
	}

	updated, err := s.repo.UpdateTripTraveler(ctx, &next)
	if err != nil {
		if errors.Is(err, domainerrs.ErrConflict) {
			return nil, apperrs.NewInvalidInput("an active traveler with that email or linked user already exists")
		}
		return nil, err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTripTravelerUpdated,
		EntityType:  activityEntityType(activity.EntityTripTraveler),
		EntityID:    activityEntityID(updated.ID),
		Metadata: map[string]any{
			"travelerId":   updated.ID.String(),
			"travelerName": updated.Name,
		},
	})

	s.ResetApprovalIfApproved(ctx, tripID, user.ID, "Travelers changed")

	return updated, nil
}

func (s *Service) RemoveTripTraveler(ctx context.Context, tripID, travelerID uuid.UUID) (*entity.TripTraveler, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if _, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID); err != nil {
		return nil, err
	}
	existing, err := s.repo.GetTripTravelerByID(ctx, tripID, travelerID)
	if err != nil {
		return nil, err
	}
	if existing.Status == entity.TripTravelerStatusRemoved {
		return nil, domainerrs.ErrNotFound
	}

	removed, err := s.repo.RemoveTripTraveler(ctx, tripID, travelerID)
	if err != nil {
		return nil, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventTripTravelerRemoved,
		EntityType:  activityEntityType(activity.EntityTripTraveler),
		EntityID:    activityEntityID(removed.ID),
		Metadata: map[string]any{
			"travelerId":   removed.ID.String(),
			"travelerName": removed.Name,
		},
	})

	s.ResetApprovalIfApproved(ctx, tripID, user.ID, "Travelers changed")

	return removed, nil
}

func (s *Service) UpdateItemCostSplit(
	ctx context.Context,
	tripID uuid.UUID,
	dayNumber int,
	itemIndex int,
	in appdto.UpdateItemCostSplitInput,
) (*entity.Trip, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	current, _, err := s.requireEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return nil, err
	}
	expectedRevision, err := requireExpectedItineraryRevision(in.ExpectedItineraryRevision)
	if err != nil {
		return nil, err
	}
	if err := checkCurrentItineraryRevision(expectedRevision, current.ItineraryRevision); err != nil {
		return nil, err
	}
	ownerID, err := tripOwnerID(current)
	if err != nil {
		return nil, err
	}
	itinerary, dayIndex, err := currentItineraryAndDayIndex(current, dayNumber)
	if err != nil {
		return nil, err
	}
	if itemIndex < 0 || itemIndex >= len(itinerary.Days[dayIndex].Items) {
		return nil, currentItineraryInvalidError()
	}
	item := &itinerary.Days[dayIndex].Items[itemIndex]
	if !costHasUsableAmount(item.EstimatedCost) {
		return nil, apperrs.NewInvalidInput("cost_missing")
	}
	activeTravelers, err := s.repo.ListActiveTripTravelersByTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	split, err := validateCostSplitForSave(in.Split, activeTravelers)
	if err != nil {
		return nil, err
	}
	item.EstimatedCost.Split = split

	updated, err := s.saveRegeneratedItinerary(
		ctx,
		tripID,
		ownerID,
		user.ID,
		itinerary,
		expectedRevision,
		entity.ItineraryVersionSourceCostSplitUpdated,
		map[string]any{
			"source":    "cost_split_updated",
			"dayNumber": dayNumber,
			"itemIndex": itemIndex,
			"splitType": split.Type,
		},
	)
	if err != nil {
		return nil, err
	}

	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventCostSplitUpdated,
		EntityType:  activityEntityType(activity.EntityItineraryItem),
		EntityID:    nil,
		Metadata: map[string]any{
			"dayNumber": dayNumber,
			"itemIndex": itemIndex,
			"splitType": split.Type,
		},
	})

	s.ResetApprovalIfApproved(ctx, tripID, user.ID, "Cost split changed")

	return updated, nil
}

func (s *Service) UpdateAccommodationCostSplit(
	ctx context.Context,
	tripID uuid.UUID,
	in appdto.UpdateAccommodationCostSplitInput,
) (*entity.Trip, error) {
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
	if current.Accommodation == nil || !costHasUsableAmount(current.Accommodation.EstimatedCost) {
		return nil, apperrs.NewInvalidInput("cost_missing")
	}
	activeTravelers, err := s.repo.ListActiveTripTravelersByTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	split, err := validateCostSplitForSave(in.Split, activeTravelers)
	if err != nil {
		return nil, err
	}
	accommodation := *current.Accommodation
	estimatedCost := *accommodation.EstimatedCost
	estimatedCost.Split = split
	accommodation.EstimatedCost = &estimatedCost

	updated, err := s.repo.UpdateTripAccommodation(ctx, tripID, ownerID, &accommodation)
	if err != nil {
		return nil, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventAccommodationSplitUpdated,
		EntityType:  activityEntityType(activity.EntityAccommodation),
		EntityID:    activityEntityID(tripID),
		Metadata: map[string]any{
			"splitType": split.Type,
		},
	})

	s.ResetApprovalIfApproved(ctx, tripID, user.ID, "Cost split changed")

	return updated, nil
}

func (s *Service) GetCostSplittingSummary(
	ctx context.Context,
	tripID uuid.UUID,
	currency string,
) (appdto.CostSplittingSummary, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.CostSplittingSummary{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.CostSplittingSummary{}, err
	}
	travelers, err := s.repo.ListActiveTripTravelersByTrip(ctx, tripID)
	if err != nil {
		return appdto.CostSplittingSummary{}, err
	}
	itinerary := parseItineraryLenient(trip.Itinerary)
	targetCurrency, err := resolveCostSplitCurrency(currency, trip, itinerary)
	if err != nil {
		return appdto.CostSplittingSummary{}, err
	}
	return s.calculateCostSplittingSummary(ctx, trip, itinerary, travelers, targetCurrency, time.Now().UTC())
}

func (s *Service) calculateCostSplittingSummary(
	ctx context.Context,
	trip *entity.Trip,
	itinerary aggregate.Itinerary,
	travelers []entity.TripTraveler,
	currency string,
	generatedAt time.Time,
) (appdto.CostSplittingSummary, error) {
	accumulators := make(map[uuid.UUID]*travelerAllocationAccumulator, len(travelers))
	activeByID := make(map[string]entity.TripTraveler, len(travelers))
	for i := range travelers {
		traveler := travelers[i]
		activeByID[traveler.ID.String()] = traveler
		accumulators[traveler.ID] = &travelerAllocationAccumulator{
			traveler:       traveler,
			categoryTotals: make(map[string]float64),
			dayTotals:      make(map[int]float64),
			items:          make([]appdto.TravelerAllocatedItem, 0),
		}
	}

	summary := appdto.CostSplittingSummary{
		TripID:          trip.ID,
		Currency:        currency,
		GeneratedAt:     generatedAt,
		Travelers:       make([]appdto.TravelerCostAllocation, 0, len(travelers)),
		UnassignedCosts: make([]appdto.UnassignedCost, 0),
		ByCategory:      make([]appdto.CostSplitCategoryTotal, 0),
		ByDay:           make([]appdto.CostSplitDayTotal, 0),
		Warnings:        make([]string, 0),
	}
	summary.Summary.TravelerCount = len(travelers)
	globalCategoryTotals := make(map[string]float64)
	globalDayTotals := make(map[int]float64)

	for _, entry := range costSplitEntries(trip, itinerary, currency) {
		if entry.Cost == nil || entry.Cost.Amount == nil {
			if entry.NeedsEstimate {
				summary.Summary.MissingEstimateCount++
			}
			continue
		}
		if *entry.Cost.Amount < 0 {
			summary.Summary.InvalidSplitCount++
			summary.UnassignedCosts = append(summary.UnassignedCosts, entry.unassigned(*entry.Cost.Amount, entry.originalCurrency(currency), "invalid_cost"))
			continue
		}

		convertedAmount, conversion, ok, reason, err := s.convertCostSplitAmount(
			ctx,
			*entry.Cost.Amount,
			entry.originalCurrency(currency),
			currency,
		)
		if err != nil {
			return appdto.CostSplittingSummary{}, err
		}
		if !ok {
			summary.Summary.UnconvertedItemCount++
			summary.UnassignedCosts = append(summary.UnassignedCosts, entry.unassigned(*entry.Cost.Amount, entry.originalCurrency(currency), reason))
			continue
		}
		if conversion != nil {
			summary.Summary.ConvertedItemCount++
			mergeCostSplitExchangeRateInfo(&summary, conversion)
		}

		summary.Summary.EstimatedTotal += convertedAmount
		allocation, invalidRule := resolveCostSplitAllocation(entry.Cost.Split, travelers, activeByID)
		if allocation.RuleSource == ruleSourceDefault {
			summary.Summary.DefaultSplitCount++
		}
		if invalidRule {
			summary.Summary.InvalidSplitCount++
		}
		if len(allocation.Shares) == 0 {
			reason := allocation.UnassignedReason
			if reason == "" {
				reason = "invalid_split_rule"
			}
			summary.Summary.UnassignedTotal += convertedAmount
			summary.UnassignedCosts = append(summary.UnassignedCosts, entry.unassigned(round2(convertedAmount), currency, reason))
			continue
		}

		category := normalizeCostSplitCategory(entry.Category)
		for travelerID, share := range allocation.Shares {
			allocatedAmount := convertedAmount * share
			accumulator := accumulators[travelerID]
			if accumulator == nil {
				continue
			}
			accumulator.allocatedTotal += allocatedAmount
			accumulator.categoryTotals[category] += allocatedAmount
			if entry.DayNumber != nil {
				accumulator.dayTotals[*entry.DayNumber] += allocatedAmount
			}
			accumulator.items = append(accumulator.items, entry.allocatedItem(
				round2(allocatedAmount),
				category,
				allocation.SplitType,
				allocation.RuleSource,
			))
			summary.Summary.AllocatedTotal += allocatedAmount
			globalCategoryTotals[category] += allocatedAmount
			if entry.DayNumber != nil {
				globalDayTotals[*entry.DayNumber] += allocatedAmount
			}
		}
	}

	summary.Summary.EstimatedTotal = round2(summary.Summary.EstimatedTotal)
	summary.Summary.AllocatedTotal = round2(summary.Summary.AllocatedTotal)
	summary.Summary.UnassignedTotal = round2(summary.Summary.UnassignedTotal)
	summary.ByCategory = buildCostSplitCategoryTotals(globalCategoryTotals)
	summary.ByDay = buildCostSplitDayTotals(globalDayTotals)

	for i := range travelers {
		accumulator := accumulators[travelers[i].ID]
		if accumulator == nil {
			continue
		}
		summary.Travelers = append(summary.Travelers, accumulator.toDTO(summary.Summary.AllocatedTotal))
	}
	summary.Warnings = buildCostSplitWarnings(summary.Summary, currency)
	return summary, nil
}

func normalizeTravelerName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", apperrs.NewInvalidInput("name is required")
	}
	if len([]rune(name)) > maxTravelerNameLength {
		return "", apperrs.NewInvalidInput("name must be at most %d characters", maxTravelerNameLength)
	}
	return name, nil
}

func normalizeTravelerEmail(raw *string) (*string, error) {
	if raw == nil {
		return nil, nil
	}
	email := strings.ToLower(strings.TrimSpace(*raw))
	if email == "" {
		return nil, nil
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil || !strings.EqualFold(parsed.Address, email) {
		return nil, apperrs.NewInvalidInput("email must be valid")
	}
	return &email, nil
}

func normalizeTravelerRole(raw entity.TripTravelerRole) (entity.TripTravelerRole, error) {
	role := entity.TripTravelerRole(strings.ToLower(strings.TrimSpace(string(raw))))
	if role == "" {
		return entity.TripTravelerRoleTraveler, nil
	}
	switch role {
	case entity.TripTravelerRoleOrganizer, entity.TripTravelerRoleTraveler:
		return role, nil
	default:
		return "", apperrs.NewInvalidInput("role must be organizer or traveler")
	}
}

func (s *Service) ensureTravelerUnique(
	ctx context.Context,
	tripID uuid.UUID,
	currentTravelerID uuid.UUID,
	email *string,
	linkedUserID *uuid.UUID,
) error {
	if email != nil {
		travelers, err := s.repo.ListActiveTripTravelersByTrip(ctx, tripID)
		if err != nil {
			return err
		}
		for i := range travelers {
			if travelers[i].ID == currentTravelerID || travelers[i].Email == nil {
				continue
			}
			if strings.EqualFold(*travelers[i].Email, *email) {
				return apperrs.NewInvalidInput("an active traveler with that email already exists")
			}
		}
	}
	if linkedUserID != nil {
		existing, err := s.repo.GetTripTravelerByLinkedUser(ctx, tripID, *linkedUserID)
		if err != nil {
			if errors.Is(err, domainerrs.ErrNotFound) {
				return nil
			}
			return err
		}
		if existing.ID != currentTravelerID {
			return apperrs.NewInvalidInput("an active traveler with that linked user already exists")
		}
	}
	return nil
}

func validateCostSplitForSave(
	raw *aggregate.CostSplitRule,
	activeTravelers []entity.TripTraveler,
) (*aggregate.CostSplitRule, error) {
	if raw == nil {
		return nil, apperrs.NewInvalidInput("split is required")
	}
	split := normalizeSplitRule(raw)
	active := make(map[string]struct{}, len(activeTravelers))
	for i := range activeTravelers {
		active[activeTravelers[i].ID.String()] = struct{}{}
	}
	switch split.Type {
	case splitTypeAllEqual:
		if len(split.TravelerIDs) > 0 || len(split.Percentages) > 0 {
			return nil, apperrs.NewInvalidInput("all_equal split must not include travelerIds or percentages")
		}
	case splitTypeSelectedEqual:
		if len(split.TravelerIDs) == 0 {
			return nil, apperrs.NewInvalidInput("selected_equal split requires travelerIds")
		}
		for _, travelerID := range split.TravelerIDs {
			if _, ok := active[travelerID]; !ok {
				return nil, apperrs.NewInvalidInput("selected_equal split references an inactive or unknown traveler")
			}
		}
		split.Percentages = nil
	case splitTypeCustomPercentages:
		if len(split.Percentages) == 0 {
			return nil, apperrs.NewInvalidInput("custom_percentages split requires percentages")
		}
		sum := 0.0
		for travelerID, percent := range split.Percentages {
			if _, ok := active[travelerID]; !ok {
				return nil, apperrs.NewInvalidInput("custom_percentages split references an inactive or unknown traveler")
			}
			if percent <= 0 {
				return nil, apperrs.NewInvalidInput("custom_percentages values must be greater than 0")
			}
			sum += percent
		}
		if math.Abs(sum-100) > 0.01 {
			return nil, apperrs.NewInvalidInput("custom_percentages must sum to 100")
		}
		if len(split.TravelerIDs) > 0 && !travelerIDSetMatches(split.TravelerIDs, split.Percentages) {
			return nil, apperrs.NewInvalidInput("travelerIds must match percentage keys")
		}
		split.TravelerIDs = percentageKeys(split.Percentages)
	default:
		return nil, apperrs.NewInvalidInput("split.type must be all_equal, selected_equal, or custom_percentages")
	}
	return split, nil
}

func normalizeSplitRule(raw *aggregate.CostSplitRule) *aggregate.CostSplitRule {
	out := &aggregate.CostSplitRule{
		Type: strings.ToLower(strings.TrimSpace(raw.Type)),
	}
	if len(raw.TravelerIDs) > 0 {
		seen := make(map[string]struct{}, len(raw.TravelerIDs))
		for _, id := range raw.TravelerIDs {
			trimmed := strings.TrimSpace(id)
			if trimmed == "" {
				continue
			}
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			out.TravelerIDs = append(out.TravelerIDs, trimmed)
		}
	}
	if len(raw.Percentages) > 0 {
		out.Percentages = make(map[string]float64, len(raw.Percentages))
		for id, percent := range raw.Percentages {
			trimmed := strings.TrimSpace(id)
			if trimmed == "" {
				continue
			}
			out.Percentages[trimmed] = percent
		}
	}
	return out
}

func travelerIDSetMatches(ids []string, percentages map[string]float64) bool {
	if len(ids) != len(percentages) {
		return false
	}
	for _, id := range ids {
		if _, ok := percentages[id]; !ok {
			return false
		}
	}
	return true
}

func percentageKeys(percentages map[string]float64) []string {
	keys := make([]string, 0, len(percentages))
	for id := range percentages {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	return keys
}

type costSplitEntry struct {
	Type          string
	DayNumber     *int
	ItemIndex     *int
	Name          string
	Category      string
	ItemType      string
	Cost          *aggregate.EstimatedCost
	NeedsEstimate bool
}

func costSplitEntries(trip *entity.Trip, itinerary aggregate.Itinerary, currency string) []costSplitEntry {
	entries := make([]costSplitEntry, 0)
	for _, day := range itinerary.Days {
		dayNumber := day.Day
		for itemIndex := range day.Items {
			item := day.Items[itemIndex]
			index := itemIndex
			entry := costSplitEntry{
				Type:          "itinerary_item",
				DayNumber:     &dayNumber,
				ItemIndex:     &index,
				Name:          item.Name,
				Category:      costCategoryForSplit(item.EstimatedCost, item.Type),
				ItemType:      item.Type,
				Cost:          item.EstimatedCost,
				NeedsEstimate: budget.ItemNeedsCost(item.Type),
			}
			entries = append(entries, entry)
		}
	}
	if trip.Accommodation != nil {
		entries = append(entries, costSplitEntry{
			Type:     "accommodation",
			Name:     trip.Accommodation.Name,
			Category: budget.CategoryAccommodation,
			Cost:     trip.Accommodation.EstimatedCost,
		})
	}
	return entries
}

func (e costSplitEntry) originalCurrency(targetCurrency string) string {
	if e.Cost == nil {
		return targetCurrency
	}
	return costCurrencyForSplit(e.Cost.Currency, targetCurrency)
}

func (e costSplitEntry) unassigned(amount float64, currency, reason string) appdto.UnassignedCost {
	return appdto.UnassignedCost{
		Type:      e.Type,
		DayNumber: e.DayNumber,
		ItemIndex: e.ItemIndex,
		Name:      e.Name,
		Amount:    round2(amount),
		Currency:  currency,
		Reason:    reason,
	}
}

func (e costSplitEntry) allocatedItem(
	allocatedAmount float64,
	category string,
	splitType string,
	ruleSource string,
) appdto.TravelerAllocatedItem {
	originalAmount := 0.0
	originalCurrency := ""
	if e.Cost != nil && e.Cost.Amount != nil {
		originalAmount = *e.Cost.Amount
		originalCurrency = strings.ToUpper(strings.TrimSpace(e.Cost.Currency))
	}
	return appdto.TravelerAllocatedItem{
		Type:                 e.Type,
		DayNumber:            e.DayNumber,
		ItemIndex:            e.ItemIndex,
		Name:                 e.Name,
		Category:             category,
		AllocatedAmount:      allocatedAmount,
		OriginalCostAmount:   round2(originalAmount),
		OriginalCostCurrency: originalCurrency,
		SplitType:            splitType,
		RuleSource:           ruleSource,
	}
}

type costSplitAllocation struct {
	Shares           map[uuid.UUID]float64
	SplitType        string
	RuleSource       string
	UnassignedReason string
}

func resolveCostSplitAllocation(
	raw *aggregate.CostSplitRule,
	activeTravelers []entity.TripTraveler,
	activeByID map[string]entity.TripTraveler,
) (costSplitAllocation, bool) {
	if len(activeTravelers) == 0 {
		return costSplitAllocation{UnassignedReason: "no_travelers"}, raw != nil
	}
	if raw == nil || strings.TrimSpace(raw.Type) == "" {
		return equalAllocation(activeTravelers, splitTypeAllEqual, ruleSourceDefault), false
	}
	split := normalizeSplitRule(raw)
	switch split.Type {
	case splitTypeAllEqual:
		return equalAllocation(activeTravelers, splitTypeAllEqual, ruleSourceExplicit), false
	case splitTypeSelectedEqual:
		selected := make([]entity.TripTraveler, 0, len(split.TravelerIDs))
		invalid := false
		seen := make(map[string]struct{}, len(split.TravelerIDs))
		for _, id := range split.TravelerIDs {
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			traveler, ok := activeByID[id]
			if !ok {
				invalid = true
				continue
			}
			selected = append(selected, traveler)
		}
		if len(selected) == 0 {
			return costSplitAllocation{
				SplitType:        splitTypeSelectedEqual,
				RuleSource:       ruleSourceExplicit,
				UnassignedReason: "invalid_split_rule",
			}, true
		}
		return equalAllocation(selected, splitTypeSelectedEqual, ruleSourceExplicit), invalid
	case splitTypeCustomPercentages:
		allocation := costSplitAllocation{
			Shares:     make(map[uuid.UUID]float64, len(split.Percentages)),
			SplitType:  splitTypeCustomPercentages,
			RuleSource: ruleSourceExplicit,
		}
		sum := 0.0
		for id, percent := range split.Percentages {
			traveler, ok := activeByID[id]
			if !ok || percent <= 0 {
				return costSplitAllocation{
					SplitType:        splitTypeCustomPercentages,
					RuleSource:       ruleSourceExplicit,
					UnassignedReason: "invalid_split_rule",
				}, true
			}
			sum += percent
			allocation.Shares[traveler.ID] = percent / 100
		}
		if math.Abs(sum-100) > 0.01 {
			return costSplitAllocation{
				SplitType:        splitTypeCustomPercentages,
				RuleSource:       ruleSourceExplicit,
				UnassignedReason: "invalid_split_rule",
			}, true
		}
		if len(split.TravelerIDs) > 0 && !travelerIDSetMatches(split.TravelerIDs, split.Percentages) {
			return costSplitAllocation{
				SplitType:        splitTypeCustomPercentages,
				RuleSource:       ruleSourceExplicit,
				UnassignedReason: "invalid_split_rule",
			}, true
		}
		return allocation, false
	default:
		return costSplitAllocation{
			SplitType:        split.Type,
			RuleSource:       ruleSourceExplicit,
			UnassignedReason: "invalid_split_rule",
		}, true
	}
}

func equalAllocation(
	travelers []entity.TripTraveler,
	splitType string,
	ruleSource string,
) costSplitAllocation {
	if len(travelers) == 0 {
		return costSplitAllocation{UnassignedReason: "no_travelers"}
	}
	share := 1 / float64(len(travelers))
	out := costSplitAllocation{
		Shares:     make(map[uuid.UUID]float64, len(travelers)),
		SplitType:  splitType,
		RuleSource: ruleSource,
	}
	for i := range travelers {
		out.Shares[travelers[i].ID] = share
	}
	return out
}

type travelerAllocationAccumulator struct {
	traveler       entity.TripTraveler
	allocatedTotal float64
	categoryTotals map[string]float64
	dayTotals      map[int]float64
	items          []appdto.TravelerAllocatedItem
}

func (a *travelerAllocationAccumulator) toDTO(grandAllocatedTotal float64) appdto.TravelerCostAllocation {
	percentage := 0.0
	if grandAllocatedTotal > 0 {
		percentage = round2((a.allocatedTotal / grandAllocatedTotal) * 100)
	}
	return appdto.TravelerCostAllocation{
		TravelerID:        a.traveler.ID,
		Name:              a.traveler.Name,
		Email:             a.traveler.Email,
		LinkedUserID:      a.traveler.LinkedUserID,
		Role:              a.traveler.Role,
		AllocatedTotal:    round2(a.allocatedTotal),
		PercentageOfTotal: percentage,
		ByCategory:        buildCostSplitCategoryTotals(a.categoryTotals),
		ByDay:             buildCostSplitDayTotals(a.dayTotals),
		Items:             a.items,
	}
}

func (s *Service) convertCostSplitAmount(
	ctx context.Context,
	amount float64,
	from string,
	to string,
) (float64, *budget.CurrencyConversionResult, bool, string, error) {
	from = strings.ToUpper(strings.TrimSpace(from))
	to = strings.ToUpper(strings.TrimSpace(to))
	if from == "" || from == to {
		return amount, nil, true, "", nil
	}
	if !s.budgetConversionEnabled || s.budgetConversionProvider == nil {
		return 0, nil, false, "conversion_disabled", nil
	}
	result, err := s.budgetConversionProvider.Convert(ctx, amount, from, to)
	if err != nil {
		if s.budgetConversionFailOpen {
			return 0, nil, false, "conversion_unavailable", nil
		}
		return 0, nil, false, "conversion_unavailable", apperrs.ErrBudgetConversionFailed
	}
	return result.ConvertedAmount, result, true, "", nil
}

func mergeCostSplitExchangeRateInfo(summary *appdto.CostSplittingSummary, conversion *budget.CurrencyConversionResult) {
	if conversion == nil {
		return
	}
	if summary.ExchangeRateInfo == nil {
		summary.ExchangeRateInfo = &budget.ExchangeRateInfo{
			Provider:     conversion.Provider,
			AsOf:         conversion.AsOf,
			FallbackUsed: conversion.FallbackUsed,
		}
		return
	}
	if summary.ExchangeRateInfo.Provider == "" {
		summary.ExchangeRateInfo.Provider = conversion.Provider
	}
	if summary.ExchangeRateInfo.AsOf.IsZero() || conversion.AsOf.After(summary.ExchangeRateInfo.AsOf) {
		summary.ExchangeRateInfo.AsOf = conversion.AsOf
	}
	summary.ExchangeRateInfo.FallbackUsed = summary.ExchangeRateInfo.FallbackUsed || conversion.FallbackUsed
}

func resolveCostSplitCurrency(currency string, trip *entity.Trip, itinerary aggregate.Itinerary) (string, error) {
	if c := strings.ToUpper(strings.TrimSpace(currency)); c != "" {
		if len(c) != 3 {
			return "", apperrs.NewInvalidInput("currency must be a 3-letter code")
		}
		return c, nil
	}
	if c := strings.ToUpper(strings.TrimSpace(trip.BudgetCurrency)); c != "" {
		return c, nil
	}
	if c := strings.ToUpper(strings.TrimSpace(itinerary.Currency)); c != "" {
		return c, nil
	}
	return budget.DefaultCurrency, nil
}

func costHasUsableAmount(cost *aggregate.EstimatedCost) bool {
	return cost != nil && cost.Amount != nil && *cost.Amount >= 0
}

func costCurrencyForSplit(itemCurrency, targetCurrency string) string {
	normalized := strings.ToUpper(strings.TrimSpace(itemCurrency))
	if normalized == "" {
		return targetCurrency
	}
	return normalized
}

func costCategoryForSplit(cost *aggregate.EstimatedCost, itemType string) string {
	if cost != nil && strings.TrimSpace(cost.Category) != "" {
		return normalizeCostSplitCategory(cost.Category)
	}
	return normalizeCostSplitCategory(itemType)
}

func normalizeCostSplitCategory(raw string) string {
	category := strings.ToLower(strings.TrimSpace(raw))
	switch category {
	case budget.CategoryFood,
		budget.CategoryTransport,
		budget.CategoryTicket,
		budget.CategoryActivity,
		budget.CategoryAccommodation,
		budget.CategoryShopping:
		return category
	default:
		return budget.CategoryOther
	}
}

func buildCostSplitCategoryTotals(totals map[string]float64) []appdto.CostSplitCategoryTotal {
	out := make([]appdto.CostSplitCategoryTotal, 0, len(totals))
	seen := make(map[string]struct{}, len(totals))
	for _, category := range costSplitCategoryOrder {
		amount := round2(totals[category])
		if amount == 0 {
			continue
		}
		out = append(out, appdto.CostSplitCategoryTotal{Category: category, Amount: amount})
		seen[category] = struct{}{}
	}
	extra := make([]string, 0)
	for category := range totals {
		if _, ok := seen[category]; !ok && round2(totals[category]) != 0 {
			extra = append(extra, category)
		}
	}
	sort.Strings(extra)
	for _, category := range extra {
		out = append(out, appdto.CostSplitCategoryTotal{Category: category, Amount: round2(totals[category])})
	}
	return out
}

func buildCostSplitDayTotals(totals map[int]float64) []appdto.CostSplitDayTotal {
	days := make([]int, 0, len(totals))
	for dayNumber, amount := range totals {
		if round2(amount) != 0 {
			days = append(days, dayNumber)
		}
	}
	sort.Ints(days)
	out := make([]appdto.CostSplitDayTotal, 0, len(days))
	for _, dayNumber := range days {
		out = append(out, appdto.CostSplitDayTotal{DayNumber: dayNumber, Amount: round2(totals[dayNumber])})
	}
	return out
}

func buildCostSplitWarnings(totals appdto.CostSplittingTotals, currency string) []string {
	warnings := make([]string, 0)
	if totals.TravelerCount == 0 {
		warnings = append(warnings, "No active travelers are configured, so estimated costs are unassigned.")
	}
	if totals.DefaultSplitCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d costs use the default equal split.", totals.DefaultSplitCount))
	}
	if totals.InvalidSplitCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d costs have invalid split rules.", totals.InvalidSplitCount))
	}
	if totals.UnconvertedItemCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d costs could not be converted to %s.", totals.UnconvertedItemCount, currency))
	}
	if totals.MissingEstimateCount > 0 {
		warnings = append(warnings, fmt.Sprintf("%d items likely need a cost estimate.", totals.MissingEstimateCount))
	}
	return warnings
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
