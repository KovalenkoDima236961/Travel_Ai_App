package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/activity"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
)

const (
	maxExpenseTitleLength = 120
	maxExpenseTextLength  = 1000
	settlementTolerance   = 0.01
)

type expenseUser struct {
	ID          uuid.UUID
	DisplayName string
}

type expenseFinancials struct {
	Currency             string
	Expenses             []entity.TripExpense
	Participants         []entity.TripExpenseParticipant
	Settlements          []entity.TripSettlement
	Users                map[uuid.UUID]expenseUser
	Balances             []appdto.ExpenseBalance
	Warnings             []string
	ActualTotal          float64
	OriginalTotals       map[string]float64
	ByCategory           map[entity.ExpenseCategory]float64
	ByPayer              map[uuid.UUID]float64
	CalculationHash      string
	ConvertedPaidCount   int
	UnconvertedPaidCount int
}

type settlementSide struct {
	UserID uuid.UUID
	Amount float64
}

func (s *Service) CreateTripExpense(ctx context.Context, tripID uuid.UUID, in appdto.CreateExpenseInput) (appdto.TripExpense, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	users, travelers, err := s.expenseUsers(ctx, trip, user)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	normalized, participants, err := s.prepareExpenseForCreate(trip, users, travelers, user.ID, access, in)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	created, createdParticipants, err := s.repo.CreateTripExpenseWithParticipants(ctx, normalized, participants)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventExpenseCreated,
		EntityType:  activityEntityType(activity.EntityTripExpense),
		EntityID:    activityEntityID(created.ID),
		Metadata:    expenseActivityMetadata(created, len(createdParticipants)),
	})
	s.notifyExpenseParticipants(ctx, trip, user.ID, created, createdParticipants, users)
	return expenseDTO(created, createdParticipants, users), nil
}

func (s *Service) ListTripExpenses(ctx context.Context, tripID uuid.UUID, filters appdto.ListExpensesInput) (appdto.TripExpensesResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripExpensesResponse{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripExpensesResponse{}, err
	}
	users, _, err := s.expenseUsers(ctx, trip, user)
	if err != nil {
		return appdto.TripExpensesResponse{}, err
	}
	expenses, err := s.repo.ListTripExpensesByTrip(ctx, tripID, filters)
	if err != nil {
		return appdto.TripExpensesResponse{}, err
	}
	items := make([]appdto.TripExpense, 0, len(expenses))
	for i := range expenses {
		participants, err := s.repo.ListExpenseParticipantsByExpense(ctx, tripID, expenses[i].ID)
		if err != nil {
			return appdto.TripExpensesResponse{}, err
		}
		items = append(items, expenseDTO(&expenses[i], participants, users))
	}
	return appdto.TripExpensesResponse{Items: items}, nil
}

func (s *Service) GetTripExpense(ctx context.Context, tripID, expenseID uuid.UUID) (appdto.TripExpense, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	users, _, err := s.expenseUsers(ctx, trip, user)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	expense, err := s.repo.GetTripExpenseByID(ctx, tripID, expenseID)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	if expense.Status != entity.ExpenseStatusActive {
		return appdto.TripExpense{}, domainerrs.ErrNotFound
	}
	participants, err := s.repo.ListExpenseParticipantsByExpense(ctx, tripID, expenseID)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	return expenseDTO(expense, participants, users), nil
}

func (s *Service) UpdateTripExpense(ctx context.Context, tripID, expenseID uuid.UUID, in appdto.UpdateExpenseInput) (appdto.TripExpense, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	existing, err := s.repo.GetTripExpenseByID(ctx, tripID, expenseID)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	if existing.Status != entity.ExpenseStatusActive {
		return appdto.TripExpense{}, domainerrs.ErrNotFound
	}
	if !access.CanEdit() && existing.CreatedByUserID != user.ID {
		return appdto.TripExpense{}, apperrs.ErrForbidden
	}
	users, travelers, err := s.expenseUsers(ctx, trip, user)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	next, participants, replaceParticipants, err := s.prepareExpenseForUpdate(trip, users, travelers, user.ID, access, existing, in)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	updated, updatedParticipants, err := s.repo.UpdateTripExpenseWithParticipants(ctx, next, participants, replaceParticipants)
	if err != nil {
		return appdto.TripExpense{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventExpenseUpdated,
		EntityType:  activityEntityType(activity.EntityTripExpense),
		EntityID:    activityEntityID(updated.ID),
		Metadata:    expenseActivityMetadata(updated, len(updatedParticipants)),
	})
	return expenseDTO(updated, updatedParticipants, users), nil
}

func (s *Service) DeleteTripExpense(ctx context.Context, tripID, expenseID uuid.UUID) error {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return err
	}
	_, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return err
	}
	existing, err := s.repo.GetTripExpenseByID(ctx, tripID, expenseID)
	if err != nil {
		return err
	}
	if existing.Status != entity.ExpenseStatusActive {
		return domainerrs.ErrNotFound
	}
	if !access.CanEdit() && existing.CreatedByUserID != user.ID {
		return apperrs.ErrForbidden
	}
	deleted, err := s.repo.SoftDeleteTripExpense(ctx, tripID, expenseID, user.ID)
	if err != nil {
		return err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventExpenseDeleted,
		EntityType:  activityEntityType(activity.EntityTripExpense),
		EntityID:    activityEntityID(deleted.ID),
		Metadata:    expenseActivityMetadata(deleted, 0),
	})
	return nil
}

func (s *Service) GetTripExpenseSummary(ctx context.Context, tripID uuid.UUID, currency string) (appdto.ExpenseSummary, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.ExpenseSummary{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.ExpenseSummary{}, err
	}
	fin, err := s.expenseFinancials(ctx, trip, user, currency)
	if err != nil {
		return appdto.ExpenseSummary{}, err
	}
	suggestions := settlementSuggestions(tripID, fin.Currency, fin.Balances, fin.Users, fin.CalculationHash)
	paidCount := 0
	for _, settlement := range fin.Settlements {
		if settlement.Status == entity.SettlementStatusPaid {
			paidCount++
		}
	}
	summary := appdto.ExpenseSummary{
		TripID:                 tripID,
		Currency:               fin.Currency,
		ActualTotal:            money(fin.ActualTotal, fin.Currency),
		OriginalCurrencyTotals: originalTotalsDTO(fin.OriginalTotals),
		ByCategory:             categoryTotalsDTO(fin.ByCategory, fin.Currency),
		ByPayer:                payerTotalsDTO(fin.ByPayer, fin.Users, fin.Currency),
		Balances:               fin.Balances,
		ConversionWarnings:     fin.Warnings,
		SettlementSummary: appdto.SettlementSummary{
			PendingCount: len(suggestions),
			PaidCount:    paidCount,
			TotalPending: money(sumSuggestions(suggestions), fin.Currency),
		},
	}
	planned, plannedWarnings := s.plannedExpenseTotal(ctx, trip, fin.Currency)
	summary.ConversionWarnings = append(summary.ConversionWarnings, plannedWarnings...)
	if planned != nil {
		summary.EstimatedTotal = &appdto.MoneyAmount{Amount: round2(*planned), Currency: fin.Currency}
		if *planned > 0 {
			difference := round2(fin.ActualTotal - *planned)
			summary.PlannedVsActual = &appdto.PlannedVsActual{
				Difference:  money(difference, fin.Currency),
				PercentUsed: round2((fin.ActualTotal / *planned) * 100),
			}
		}
	}
	return summary, nil
}

func (s *Service) GetTripSettlements(ctx context.Context, tripID uuid.UUID, currency string) (appdto.SettlementsResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.SettlementsResponse{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.SettlementsResponse{}, err
	}
	return s.settlementsResponse(ctx, trip, user, currency)
}

func (s *Service) MarkTripSettlementPaid(ctx context.Context, tripID uuid.UUID, settlementID string, in appdto.MarkSettlementPaidInput) (appdto.SettlementsResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.SettlementsResponse{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.SettlementsResponse{}, err
	}
	fin, err := s.expenseFinancials(ctx, trip, user, "")
	if err != nil {
		return appdto.SettlementsResponse{}, err
	}
	var createdOrUpdated *entity.TripSettlement
	if strings.HasPrefix(settlementID, "calculated:") {
		suggestions := settlementSuggestions(tripID, fin.Currency, fin.Balances, fin.Users, fin.CalculationHash)
		var selected *appdto.SettlementSuggestion
		for i := range suggestions {
			if suggestions[i].ID == settlementID {
				selected = &suggestions[i]
				break
			}
		}
		if selected == nil {
			return appdto.SettlementsResponse{}, apperrs.NewConflict("settlement suggestion is no longer current")
		}
		if !canMarkSettlementPaid(access, user.ID, selected.FromUserID, selected.ToUserID) {
			return appdto.SettlementsResponse{}, apperrs.ErrForbidden
		}
		hash := selected.CalculationHash
		createdOrUpdated, err = s.repo.CreateTripSettlement(ctx, &entity.TripSettlement{
			ID:              uuid.New(),
			TripID:          tripID,
			FromUserID:      selected.FromUserID,
			ToUserID:        selected.ToUserID,
			Amount:          selected.Amount.Amount,
			Currency:        selected.Amount.Currency,
			Status:          entity.SettlementStatusPaid,
			Source:          entity.SettlementSourceCalculated,
			CalculationHash: &hash,
			PaidAt:          timePtr(time.Now().UTC()),
			PaidByUserID:    &user.ID,
			Notes:           trimOptionalText(in.Notes),
			Metadata:        map[string]any{"suggestionId": selected.ID},
		})
		if err != nil {
			return appdto.SettlementsResponse{}, err
		}
	} else {
		parsed, err := uuid.Parse(settlementID)
		if err != nil {
			return appdto.SettlementsResponse{}, apperrs.NewInvalidInput("invalid settlement id")
		}
		existing, err := s.repo.GetTripSettlementByID(ctx, tripID, parsed)
		if err != nil {
			return appdto.SettlementsResponse{}, err
		}
		if !canMarkSettlementPaid(access, user.ID, existing.FromUserID, existing.ToUserID) {
			return appdto.SettlementsResponse{}, apperrs.ErrForbidden
		}
		createdOrUpdated, err = s.repo.MarkTripSettlementPaid(ctx, tripID, parsed, user.ID, trimOptionalText(in.Notes))
		if err != nil {
			return appdto.SettlementsResponse{}, err
		}
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventSettlementMarkedPaid,
		EntityType:  activityEntityType(activity.EntityTripSettlement),
		EntityID:    activityEntityID(createdOrUpdated.ID),
		Metadata:    settlementActivityMetadata(createdOrUpdated),
	})
	s.notifySettlementPaid(ctx, trip, user.ID, createdOrUpdated, fin.Users)
	return s.settlementsResponse(ctx, trip, user, fin.Currency)
}

func (s *Service) CancelTripSettlement(ctx context.Context, tripID, settlementID uuid.UUID) (appdto.SettlementsResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.SettlementsResponse{}, err
	}
	trip, access, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return appdto.SettlementsResponse{}, err
	}
	existing, err := s.repo.GetTripSettlementByID(ctx, tripID, settlementID)
	if err != nil {
		return appdto.SettlementsResponse{}, err
	}
	if !access.CanEdit() && (existing.PaidByUserID == nil || *existing.PaidByUserID != user.ID) {
		return appdto.SettlementsResponse{}, apperrs.ErrForbidden
	}
	cancelled, err := s.repo.CancelTripSettlement(ctx, tripID, settlementID, user.ID)
	if err != nil {
		return appdto.SettlementsResponse{}, err
	}
	s.recordActivity(ctx, activity.RecordActivityInput{
		TripID:      tripID,
		ActorUserID: &user.ID,
		EventType:   activity.EventSettlementCancelled,
		EntityType:  activityEntityType(activity.EntityTripSettlement),
		EntityID:    activityEntityID(cancelled.ID),
		Metadata:    settlementActivityMetadata(cancelled),
	})
	return s.settlementsResponse(ctx, trip, user, "")
}

func (s *Service) RecalculateTripSettlements(ctx context.Context, tripID uuid.UUID, currency string) (appdto.SettlementsResponse, error) {
	return s.GetTripSettlements(ctx, tripID, currency)
}

func (s *Service) settlementsResponse(ctx context.Context, trip *entity.Trip, user auth.AuthenticatedUser, currency string) (appdto.SettlementsResponse, error) {
	fin, err := s.expenseFinancials(ctx, trip, user, currency)
	if err != nil {
		return appdto.SettlementsResponse{}, err
	}
	paid := make([]appdto.TripSettlement, 0)
	for i := range fin.Settlements {
		if fin.Settlements[i].Status == entity.SettlementStatusPaid {
			paid = append(paid, settlementDTO(&fin.Settlements[i], fin.Users, fin.Currency))
		}
	}
	return appdto.SettlementsResponse{
		Currency:        fin.Currency,
		Suggestions:     settlementSuggestions(trip.ID, fin.Currency, fin.Balances, fin.Users, fin.CalculationHash),
		PaidSettlements: paid,
		Warnings:        fin.Warnings,
	}, nil
}

func (s *Service) prepareExpenseForCreate(
	trip *entity.Trip,
	users map[uuid.UUID]expenseUser,
	travelers []entity.TripTraveler,
	actorID uuid.UUID,
	access TripAccess,
	in appdto.CreateExpenseInput,
) (*entity.TripExpense, []entity.TripExpenseParticipant, error) {
	if !access.CanEdit() && in.PaidByUserID != actorID {
		return nil, nil, apperrs.ErrForbidden
	}
	expense, err := normalizeExpenseInput(trip.ID, actorID, in)
	if err != nil {
		return nil, nil, err
	}
	if _, ok := users[expense.PaidByUserID]; !ok {
		return nil, nil, apperrs.NewInvalidInput("paidByUserId must have trip access")
	}
	if err := validateExpenseLinks(trip, expense); err != nil {
		return nil, nil, err
	}
	participants, err := calculateExpenseParticipants(expense, in.ParticipantUserIDs, in.CustomShares, in.CustomPercentages, users, travelers)
	if err != nil {
		return nil, nil, err
	}
	return expense, participants, nil
}

func (s *Service) prepareExpenseForUpdate(
	trip *entity.Trip,
	users map[uuid.UUID]expenseUser,
	travelers []entity.TripTraveler,
	actorID uuid.UUID,
	access TripAccess,
	existing *entity.TripExpense,
	in appdto.UpdateExpenseInput,
) (*entity.TripExpense, []entity.TripExpenseParticipant, bool, error) {
	next := *existing
	if in.Title != nil {
		next.Title = strings.TrimSpace(*in.Title)
	}
	if in.ClearDescription {
		next.Description = nil
	} else if in.Description != nil {
		next.Description = trimOptionalText(in.Description)
	}
	if in.Amount != nil {
		amount, currency, err := normalizeMoney(in.Amount.Amount, in.Amount.Currency)
		if err != nil {
			return nil, nil, false, err
		}
		next.Amount = amount
		next.Currency = currency
	}
	if in.Category != nil {
		category, err := normalizeExpenseCategory(*in.Category)
		if err != nil {
			return nil, nil, false, err
		}
		next.Category = category
	}
	if in.ExpenseDate != nil {
		next.ExpenseDate = *in.ExpenseDate
	}
	if in.PaidByUserID != nil {
		next.PaidByUserID = *in.PaidByUserID
	}
	if !access.CanEdit() && next.PaidByUserID != actorID {
		return nil, nil, false, apperrs.ErrForbidden
	}
	if _, ok := users[next.PaidByUserID]; !ok {
		return nil, nil, false, apperrs.NewInvalidInput("paidByUserId must have trip access")
	}
	if in.SplitType != nil {
		splitType, err := normalizeExpenseSplitType(*in.SplitType)
		if err != nil {
			return nil, nil, false, err
		}
		next.SplitType = splitType
	}
	if in.LinkedItinerarySet {
		if in.LinkedItinerary == nil {
			next.LinkedDayNumber = nil
			next.LinkedItemIndex = nil
			next.LinkedItemID = nil
		} else {
			next.LinkedDayNumber = &in.LinkedItinerary.DayNumber
			next.LinkedItemIndex = &in.LinkedItinerary.ItemIndex
			next.LinkedItemID = trimOptionalText(in.LinkedItinerary.ItemID)
		}
	}
	if in.LinkedRouteLegIDSet {
		next.LinkedRouteLegID = trimOptionalText(in.LinkedRouteLegID)
	}
	if in.LinkedAccommodation != nil {
		next.LinkedAccommodation = *in.LinkedAccommodation
	}
	if in.ClearNotes {
		next.Notes = nil
	} else if in.Notes != nil {
		next.Notes = trimOptionalText(in.Notes)
	}
	if in.Metadata != nil {
		next.Metadata = in.Metadata
	}
	if err := validateExpenseBasics(next.Title, next.Description, next.Notes); err != nil {
		return nil, nil, false, err
	}
	if err := validateExpenseLinks(trip, &next); err != nil {
		return nil, nil, false, err
	}
	replaceParticipants := in.Amount != nil || in.PaidByUserID != nil || in.SplitType != nil ||
		in.ParticipantUserIDsSet || in.CustomSharesSet || in.CustomPercentagesSet
	var participants []entity.TripExpenseParticipant
	if replaceParticipants {
		selected := in.ParticipantUserIDs
		shares := in.CustomShares
		percentages := in.CustomPercentages
		var err error
		participants, err = calculateExpenseParticipants(&next, selected, shares, percentages, users, travelers)
		if err != nil {
			return nil, nil, false, err
		}
	}
	next.UpdatedByUserID = &actorID
	return &next, participants, replaceParticipants, nil
}

func normalizeExpenseInput(tripID, actorID uuid.UUID, in appdto.CreateExpenseInput) (*entity.TripExpense, error) {
	amount, currency, err := normalizeMoney(in.Amount.Amount, in.Amount.Currency)
	if err != nil {
		return nil, err
	}
	category, err := normalizeExpenseCategory(in.Category)
	if err != nil {
		return nil, err
	}
	splitType, err := normalizeExpenseSplitType(in.SplitType)
	if err != nil {
		return nil, err
	}
	expense := &entity.TripExpense{
		ID:                  uuid.New(),
		TripID:              tripID,
		Title:               strings.TrimSpace(in.Title),
		Description:         trimOptionalText(in.Description),
		Amount:              amount,
		Currency:            currency,
		Category:            category,
		ExpenseDate:         in.ExpenseDate,
		PaidByUserID:        in.PaidByUserID,
		SplitType:           splitType,
		LinkedRouteLegID:    trimOptionalText(in.LinkedRouteLegID),
		LinkedAccommodation: in.LinkedAccommodation,
		Notes:               trimOptionalText(in.Notes),
		Status:              entity.ExpenseStatusActive,
		Metadata:            in.Metadata,
		CreatedByUserID:     actorID,
		UpdatedByUserID:     &actorID,
	}
	if in.LinkedItinerary != nil {
		expense.LinkedDayNumber = &in.LinkedItinerary.DayNumber
		expense.LinkedItemIndex = &in.LinkedItinerary.ItemIndex
		expense.LinkedItemID = trimOptionalText(in.LinkedItinerary.ItemID)
	}
	if expense.Metadata == nil {
		expense.Metadata = map[string]any{}
	}
	if err := validateExpenseBasics(expense.Title, expense.Description, expense.Notes); err != nil {
		return nil, err
	}
	if expense.ExpenseDate.IsZero() {
		return nil, apperrs.NewInvalidInput("expenseDate is required")
	}
	return expense, nil
}

func validateExpenseBasics(title string, description, notes *string) error {
	runes := len([]rune(strings.TrimSpace(title)))
	if runes < 2 || runes > maxExpenseTitleLength {
		return apperrs.NewInvalidInput("title must be between 2 and %d characters", maxExpenseTitleLength)
	}
	if description != nil && len([]rune(*description)) > maxExpenseTextLength {
		return apperrs.NewInvalidInput("description must be at most %d characters", maxExpenseTextLength)
	}
	if notes != nil && len([]rune(*notes)) > maxExpenseTextLength {
		return apperrs.NewInvalidInput("notes must be at most %d characters", maxExpenseTextLength)
	}
	return nil
}

func normalizeMoney(amount float64, currency string) (float64, string, error) {
	if amount <= 0 {
		return 0, "", apperrs.NewInvalidInput("amount must be greater than 0")
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if !validCurrencyCode(currency) {
		return 0, "", apperrs.NewInvalidInput("currency must be a 3-letter code")
	}
	return round2(amount), currency, nil
}

func normalizeExpenseCategory(raw entity.ExpenseCategory) (entity.ExpenseCategory, error) {
	category := entity.ExpenseCategory(strings.ToLower(strings.TrimSpace(string(raw))))
	switch category {
	case entity.ExpenseCategoryTransport,
		entity.ExpenseCategoryAccommodation,
		entity.ExpenseCategoryFood,
		entity.ExpenseCategoryTickets,
		entity.ExpenseCategoryActivities,
		entity.ExpenseCategoryShopping,
		entity.ExpenseCategoryFuel,
		entity.ExpenseCategoryParking,
		entity.ExpenseCategoryTolls,
		entity.ExpenseCategoryCamping,
		entity.ExpenseCategoryGroceries,
		entity.ExpenseCategoryHealthSafety,
		entity.ExpenseCategoryOther:
		return category, nil
	default:
		return "", apperrs.NewInvalidInput("category is invalid")
	}
}

func normalizeExpenseSplitType(raw entity.ExpenseSplitType) (entity.ExpenseSplitType, error) {
	splitType := entity.ExpenseSplitType(strings.ToLower(strings.TrimSpace(string(raw))))
	if splitType == "" {
		return entity.ExpenseSplitSelectedEqual, nil
	}
	switch splitType {
	case entity.ExpenseSplitEqual,
		entity.ExpenseSplitSelectedEqual,
		entity.ExpenseSplitCustomAmounts,
		entity.ExpenseSplitCustomPercentages,
		entity.ExpenseSplitPayerOnly:
		return splitType, nil
	default:
		return "", apperrs.NewInvalidInput("splitType is invalid")
	}
}

func calculateExpenseParticipants(
	expense *entity.TripExpense,
	selected []uuid.UUID,
	customShares []appdto.ExpenseCustomAmount,
	customPercentages []appdto.ExpenseCustomPercentage,
	users map[uuid.UUID]expenseUser,
	travelers []entity.TripTraveler,
) ([]entity.TripExpenseParticipant, error) {
	var participantIDs []uuid.UUID
	shareCents := map[uuid.UUID]int{}
	percentages := map[uuid.UUID]float64{}
	totalCents := moneyCents(expense.Amount)
	switch expense.SplitType {
	case entity.ExpenseSplitEqual:
		participantIDs = linkedTravelerUserIDs(travelers)
		if len(participantIDs) == 0 {
			participantIDs = sortedUserIDs(users)
		}
		assignEqualShares(totalCents, participantIDs, shareCents)
	case entity.ExpenseSplitSelectedEqual:
		participantIDs = dedupeUUIDs(selected)
		if len(participantIDs) == 0 {
			return nil, apperrs.NewInvalidInput("participantUserIds is required")
		}
		assignEqualShares(totalCents, participantIDs, shareCents)
	case entity.ExpenseSplitCustomAmounts:
		if len(customShares) == 0 {
			return nil, apperrs.NewInvalidInput("customShares is required")
		}
		sum := 0
		for _, share := range customShares {
			if _, ok := users[share.UserID]; !ok {
				return nil, apperrs.NewInvalidInput("custom share references an unknown participant")
			}
			currency := strings.ToUpper(strings.TrimSpace(share.Currency))
			if currency != expense.Currency {
				return nil, apperrs.NewInvalidInput("custom share currency must match expense currency")
			}
			cents := moneyCents(share.Amount)
			if cents < 0 {
				return nil, apperrs.NewInvalidInput("custom share amount must be non-negative")
			}
			shareCents[share.UserID] += cents
			sum += cents
		}
		if sum != totalCents {
			return nil, apperrs.NewInvalidInput("customShares must sum to the expense amount")
		}
		participantIDs = sortedShareKeys(shareCents)
	case entity.ExpenseSplitCustomPercentages:
		if len(customPercentages) == 0 {
			return nil, apperrs.NewInvalidInput("customPercentages is required")
		}
		sum := 0.0
		participantIDs = make([]uuid.UUID, 0, len(customPercentages))
		for _, item := range customPercentages {
			if _, ok := users[item.UserID]; !ok {
				return nil, apperrs.NewInvalidInput("custom percentage references an unknown participant")
			}
			if item.Percentage < 0 {
				return nil, apperrs.NewInvalidInput("custom percentage must be non-negative")
			}
			participantIDs = append(participantIDs, item.UserID)
			percentages[item.UserID] += item.Percentage
			sum += item.Percentage
		}
		if math.Abs(sum-100) > 0.01 {
			return nil, apperrs.NewInvalidInput("customPercentages must sum to 100")
		}
		participantIDs = dedupeUUIDs(participantIDs)
		assignPercentageShares(totalCents, participantIDs, percentages, shareCents)
	case entity.ExpenseSplitPayerOnly:
		participantIDs = []uuid.UUID{expense.PaidByUserID}
		shareCents[expense.PaidByUserID] = totalCents
		percentages[expense.PaidByUserID] = 100
	default:
		return nil, apperrs.NewInvalidInput("splitType is invalid")
	}
	if len(participantIDs) == 0 {
		return nil, apperrs.NewInvalidInput("at least one participant is required")
	}
	for _, id := range participantIDs {
		if _, ok := users[id]; !ok {
			return nil, apperrs.NewInvalidInput("participantUserIds references a user without trip access")
		}
	}
	sort.Slice(participantIDs, func(i, j int) bool { return participantIDs[i].String() < participantIDs[j].String() })
	out := make([]entity.TripExpenseParticipant, 0, len(participantIDs))
	for _, id := range participantIDs {
		amount := centsMoney(shareCents[id])
		currency := expense.Currency
		percentage := percentages[id]
		if percentage == 0 && totalCents > 0 {
			percentage = round4((float64(shareCents[id]) / float64(totalCents)) * 100)
		}
		out = append(out, entity.TripExpenseParticipant{
			ID:              uuid.New(),
			ExpenseID:       expense.ID,
			TripID:          expense.TripID,
			UserID:          id,
			ShareAmount:     &amount,
			ShareCurrency:   &currency,
			SharePercentage: &percentage,
		})
	}
	return out, nil
}

func assignEqualShares(totalCents int, ids []uuid.UUID, out map[uuid.UUID]int) {
	if len(ids) == 0 {
		return
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	base := totalCents / len(ids)
	remainder := totalCents % len(ids)
	for index, id := range ids {
		out[id] = base
		if index < remainder {
			out[id]++
		}
	}
}

func assignPercentageShares(totalCents int, ids []uuid.UUID, percentages map[uuid.UUID]float64, out map[uuid.UUID]int) {
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	sum := 0
	for _, id := range ids {
		cents := int(math.Round(float64(totalCents) * percentages[id] / 100))
		out[id] = cents
		sum += cents
	}
	remainder := totalCents - sum
	for i := 0; remainder != 0 && len(ids) > 0; i++ {
		id := ids[i%len(ids)]
		if remainder > 0 {
			out[id]++
			remainder--
		} else if out[id] > 0 {
			out[id]--
			remainder++
		} else {
			remainder++
		}
	}
}

func (s *Service) expenseUsers(ctx context.Context, trip *entity.Trip, actor auth.AuthenticatedUser) (map[uuid.UUID]expenseUser, []entity.TripTraveler, error) {
	users := map[uuid.UUID]expenseUser{}
	if trip.UserID != nil {
		users[*trip.UserID] = expenseUser{ID: *trip.UserID, DisplayName: "Trip owner"}
	}
	collaborators, err := s.repo.ListTripCollaborators(ctx, trip.ID)
	if err != nil {
		return nil, nil, err
	}
	for _, collaborator := range collaborators {
		if collaborator.Status != entity.CollaboratorStatusAccepted {
			continue
		}
		addExpenseUser(users, collaborator.UserID, shortUserName(collaborator.UserID))
	}
	travelers, err := s.repo.ListActiveTripTravelersByTrip(ctx, trip.ID)
	if err != nil {
		return nil, nil, err
	}
	for _, traveler := range travelers {
		if traveler.LinkedUserID != nil {
			addExpenseUser(users, *traveler.LinkedUserID, traveler.Name)
		}
	}
	if trip.WorkspaceID != nil && s.workspacesEnabled && s.workspaceProvider != nil {
		members, err := s.workspaceProvider.ListMembers(ctx, *trip.WorkspaceID)
		if err != nil {
			return nil, nil, err
		}
		for _, member := range members {
			if member.Status == workspaces.MemberStatusActive {
				addExpenseUser(users, member.UserID, shortUserName(member.UserID))
			}
		}
	}
	if actor.ID != uuid.Nil {
		name := strings.TrimSpace(actor.Email)
		if name == "" {
			name = shortUserName(actor.ID)
		}
		addExpenseUser(users, actor.ID, name)
	}
	return users, travelers, nil
}

func addExpenseUser(users map[uuid.UUID]expenseUser, id uuid.UUID, displayName string) {
	if id == uuid.Nil {
		return
	}
	existing, ok := users[id]
	if ok && existing.DisplayName != "" && !strings.HasPrefix(existing.DisplayName, "User ") {
		return
	}
	if strings.TrimSpace(displayName) == "" {
		displayName = shortUserName(id)
	}
	users[id] = expenseUser{ID: id, DisplayName: displayName}
}

func validateExpenseLinks(trip *entity.Trip, expense *entity.TripExpense) error {
	itinerary := parseItineraryLenient(trip.Itinerary)
	if expense.LinkedDayNumber != nil || expense.LinkedItemIndex != nil || expense.LinkedItemID != nil {
		if expense.LinkedDayNumber == nil || expense.LinkedItemIndex == nil {
			return apperrs.NewInvalidInput("linkedItinerary requires dayNumber and itemIndex")
		}
		found := false
		for _, day := range itinerary.Days {
			if day.Day == *expense.LinkedDayNumber && *expense.LinkedItemIndex >= 0 && *expense.LinkedItemIndex < len(day.Items) {
				found = true
				break
			}
		}
		if !found {
			return apperrs.NewInvalidInput("linked itinerary item does not exist")
		}
	}
	if expense.LinkedRouteLegID != nil {
		if trip.Route == nil {
			return apperrs.NewInvalidInput("linked route leg does not exist")
		}
		found := false
		for _, leg := range trip.Route.Legs {
			if leg.ID == *expense.LinkedRouteLegID {
				found = true
				break
			}
		}
		if !found {
			return apperrs.NewInvalidInput("linked route leg does not exist")
		}
	}
	if expense.LinkedAccommodation && trip.Accommodation == nil {
		return apperrs.NewInvalidInput("linkedAccommodation requires an accommodation")
	}
	return nil
}

func (s *Service) expenseFinancials(ctx context.Context, trip *entity.Trip, actor auth.AuthenticatedUser, requestedCurrency string) (expenseFinancials, error) {
	currency, err := resolveExpenseSummaryCurrency(requestedCurrency, trip)
	if err != nil {
		return expenseFinancials{}, err
	}
	users, _, err := s.expenseUsers(ctx, trip, actor)
	if err != nil {
		return expenseFinancials{}, err
	}
	expenses, err := s.repo.ListTripExpensesByTrip(ctx, trip.ID, appdto.ListExpensesInput{})
	if err != nil {
		return expenseFinancials{}, err
	}
	participants, err := s.repo.ListExpenseParticipantsByTrip(ctx, trip.ID)
	if err != nil {
		return expenseFinancials{}, err
	}
	settlements, err := s.repo.ListTripSettlementsByTrip(ctx, trip.ID)
	if err != nil {
		return expenseFinancials{}, err
	}
	fin := expenseFinancials{
		Currency:       currency,
		Expenses:       expenses,
		Participants:   participants,
		Settlements:    settlements,
		Users:          users,
		OriginalTotals: map[string]float64{},
		ByCategory:     map[entity.ExpenseCategory]float64{},
		ByPayer:        map[uuid.UUID]float64{},
		Warnings:       []string{},
	}
	paid := map[uuid.UUID]float64{}
	share := map[uuid.UUID]float64{}
	expenseByID := map[uuid.UUID]entity.TripExpense{}
	for _, expense := range expenses {
		expenseByID[expense.ID] = expense
		amount, ok, reason, err := s.convertExpenseAmount(ctx, expense.Amount, expense.Currency, currency)
		if err != nil {
			return expenseFinancials{}, err
		}
		fin.OriginalTotals[expense.Currency] += expense.Amount
		if !ok {
			fin.Warnings = append(fin.Warnings, fmt.Sprintf("%s %s could not be converted to %s (%s).", formatExpenseAmount(expense.Amount), expense.Currency, currency, reason))
			continue
		}
		amount = round2(amount)
		fin.ActualTotal += amount
		fin.ByCategory[expense.Category] += amount
		fin.ByPayer[expense.PaidByUserID] += amount
		paid[expense.PaidByUserID] += amount
		addExpenseUser(fin.Users, expense.PaidByUserID, shortUserName(expense.PaidByUserID))
	}
	for _, participant := range participants {
		expense, ok := expenseByID[participant.ExpenseID]
		if !ok || participant.ShareAmount == nil {
			continue
		}
		shareCurrency := expense.Currency
		if participant.ShareCurrency != nil && strings.TrimSpace(*participant.ShareCurrency) != "" {
			shareCurrency = strings.ToUpper(strings.TrimSpace(*participant.ShareCurrency))
		}
		amount, ok, reason, err := s.convertExpenseAmount(ctx, *participant.ShareAmount, shareCurrency, currency)
		if err != nil {
			return expenseFinancials{}, err
		}
		if !ok {
			fin.Warnings = append(fin.Warnings, fmt.Sprintf("A participant share in %s could not be converted to %s (%s).", shareCurrency, currency, reason))
			continue
		}
		share[participant.UserID] += round2(amount)
		addExpenseUser(fin.Users, participant.UserID, shortUserName(participant.UserID))
	}
	netBefore := map[uuid.UUID]float64{}
	for id := range fin.Users {
		netBefore[id] = round2(paid[id] - share[id])
	}
	for _, settlement := range settlements {
		if settlement.Status != entity.SettlementStatusPaid {
			continue
		}
		amount, ok, reason, err := s.convertExpenseAmount(ctx, settlement.Amount, settlement.Currency, currency)
		if err != nil {
			return expenseFinancials{}, err
		}
		if !ok {
			fin.Warnings = append(fin.Warnings, fmt.Sprintf("A paid settlement in %s could not be converted to %s (%s).", settlement.Currency, currency, reason))
			continue
		}
		netBefore[settlement.FromUserID] += round2(amount)
		netBefore[settlement.ToUserID] -= round2(amount)
		addExpenseUser(fin.Users, settlement.FromUserID, shortUserName(settlement.FromUserID))
		addExpenseUser(fin.Users, settlement.ToUserID, shortUserName(settlement.ToUserID))
	}
	userIDs := sortedUserIDs(fin.Users)
	fin.Balances = make([]appdto.ExpenseBalance, 0, len(userIDs))
	for _, id := range userIDs {
		rawNet := round2(paid[id] - share[id])
		outstanding := round2(netBefore[id])
		settled := round2(outstanding - rawNet)
		status := "settled"
		if outstanding > settlementTolerance {
			status = "gets_back"
		} else if outstanding < -settlementTolerance {
			status = "owes"
		}
		fin.Balances = append(fin.Balances, appdto.ExpenseBalance{
			UserID:               id,
			DisplayName:          fin.Users[id].DisplayName,
			Paid:                 money(paid[id], currency),
			Share:                money(share[id], currency),
			Net:                  money(rawNet, currency),
			NetBeforeSettlements: money(rawNet, currency),
			SettledAmount:        money(settled, currency),
			NetOutstanding:       money(outstanding, currency),
			Status:               status,
		})
	}
	fin.ActualTotal = round2(fin.ActualTotal)
	fin.CalculationHash = calculationHash(fin.Expenses, fin.Participants, fin.Settlements, currency)
	return fin, nil
}

func (s *Service) convertExpenseAmount(ctx context.Context, amount float64, from string, to string) (float64, bool, string, error) {
	converted, _, ok, reason, err := s.convertCostSplitAmount(ctx, amount, from, to)
	return converted, ok, reason, err
}

func settlementSuggestions(tripID uuid.UUID, currency string, balances []appdto.ExpenseBalance, users map[uuid.UUID]expenseUser, hash string) []appdto.SettlementSuggestion {
	debtors := make([]settlementSide, 0)
	creditors := make([]settlementSide, 0)
	for _, balance := range balances {
		net := balance.NetOutstanding.Amount
		if net < -settlementTolerance {
			debtors = append(debtors, settlementSide{UserID: balance.UserID, Amount: round2(-net)})
		} else if net > settlementTolerance {
			creditors = append(creditors, settlementSide{UserID: balance.UserID, Amount: round2(net)})
		}
	}
	sortSides(debtors)
	sortSides(creditors)
	out := make([]appdto.SettlementSuggestion, 0)
	i, j := 0, 0
	for i < len(debtors) && j < len(creditors) {
		amount := round2(math.Min(debtors[i].Amount, creditors[j].Amount))
		if amount > settlementTolerance {
			fromID := debtors[i].UserID
			toID := creditors[j].UserID
			id := calculatedSettlementID(tripID, fromID, toID, amount, currency, hash)
			out = append(out, appdto.SettlementSuggestion{
				ID:              id,
				FromUserID:      fromID,
				FromDisplayName: displayName(users, fromID),
				ToUserID:        toID,
				ToDisplayName:   displayName(users, toID),
				Amount:          money(amount, currency),
				Status:          entity.SettlementStatusPending,
				Source:          entity.SettlementSourceCalculated,
				CalculationHash: hash,
			})
		}
		debtors[i].Amount = round2(debtors[i].Amount - amount)
		creditors[j].Amount = round2(creditors[j].Amount - amount)
		if debtors[i].Amount <= settlementTolerance {
			i++
		}
		if creditors[j].Amount <= settlementTolerance {
			j++
		}
	}
	return out
}

func calculationHash(expenses []entity.TripExpense, participants []entity.TripExpenseParticipant, settlements []entity.TripSettlement, currency string) string {
	parts := make([]string, 0, len(expenses)+len(participants)+len(settlements)+1)
	parts = append(parts, "currency="+currency)
	for _, expense := range expenses {
		parts = append(parts, fmt.Sprintf("e:%s:%s:%0.2f:%s:%s", expense.ID, expense.PaidByUserID, expense.Amount, expense.Currency, expense.UpdatedAt.UTC().Format(time.RFC3339Nano)))
	}
	for _, participant := range participants {
		amount := 0.0
		if participant.ShareAmount != nil {
			amount = *participant.ShareAmount
		}
		parts = append(parts, fmt.Sprintf("p:%s:%s:%0.2f", participant.ExpenseID, participant.UserID, amount))
	}
	for _, settlement := range settlements {
		if settlement.Status == entity.SettlementStatusPaid {
			parts = append(parts, fmt.Sprintf("s:%s:%s:%s:%0.2f:%s", settlement.ID, settlement.FromUserID, settlement.ToUserID, settlement.Amount, settlement.Currency))
		}
	}
	sort.Strings(parts)
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])[:20]
}

func calculatedSettlementID(tripID, fromID, toID uuid.UUID, amount float64, currency, hash string) string {
	return fmt.Sprintf("calculated:%s:%s:%s:%0.2f:%s:%s", tripID, fromID, toID, amount, currency, hash)
}

func sortSides(items []settlementSide) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].Amount != items[j].Amount {
			return items[i].Amount > items[j].Amount
		}
		return items[i].UserID.String() < items[j].UserID.String()
	})
}

func canMarkSettlementPaid(access TripAccess, actorID, fromID, toID uuid.UUID) bool {
	return access.CanEdit() || actorID == fromID || actorID == toID
}

func resolveExpenseSummaryCurrency(raw string, trip *entity.Trip) (string, error) {
	currency := strings.ToUpper(strings.TrimSpace(raw))
	if currency == "" {
		currency = strings.ToUpper(strings.TrimSpace(trip.BudgetCurrency))
	}
	if currency == "" {
		currency = budget.DefaultCurrency
	}
	if !validCurrencyCode(currency) {
		return "", apperrs.NewInvalidInput("currency must be a 3-letter code")
	}
	return currency, nil
}

func (s *Service) plannedExpenseTotal(ctx context.Context, trip *entity.Trip, currency string) (*float64, []string) {
	summary, err := s.budgetSummaryForTrip(ctx, trip)
	if err != nil {
		return nil, []string{"Estimated budget could not be calculated."}
	}
	if summary.TripBudget != nil {
		value := round2(*summary.TripBudget)
		return &value, nil
	}
	if summary.EstimatedTotal > 0 {
		value := round2(summary.EstimatedTotal)
		return &value, nil
	}
	return nil, []string{"No planned budget or estimated trip total is available."}
}

func expenseDTO(expense *entity.TripExpense, participants []entity.TripExpenseParticipant, users map[uuid.UUID]expenseUser) appdto.TripExpense {
	out := appdto.TripExpense{
		ID:                  expense.ID,
		TripID:              expense.TripID,
		Title:               expense.Title,
		Description:         expense.Description,
		Amount:              money(expense.Amount, expense.Currency),
		Category:            expense.Category,
		ExpenseDate:         expense.ExpenseDate.Format("2006-01-02"),
		PaidByUserID:        expense.PaidByUserID,
		PaidByDisplayName:   displayName(users, expense.PaidByUserID),
		SplitType:           expense.SplitType,
		Participants:        make([]appdto.ExpenseParticipant, 0, len(participants)),
		LinkedRouteLegID:    expense.LinkedRouteLegID,
		LinkedAccommodation: expense.LinkedAccommodation,
		Notes:               expense.Notes,
		Metadata:            expense.Metadata,
		CreatedByUserID:     expense.CreatedByUserID,
		CreatedAt:           expense.CreatedAt,
		UpdatedAt:           expense.UpdatedAt,
	}
	if expense.Metadata == nil {
		out.Metadata = map[string]any{}
	}
	if expense.LinkedDayNumber != nil && expense.LinkedItemIndex != nil {
		out.LinkedItinerary = &appdto.LinkedItineraryRef{
			DayNumber: *expense.LinkedDayNumber,
			ItemIndex: *expense.LinkedItemIndex,
			ItemID:    expense.LinkedItemID,
		}
	}
	sort.Slice(participants, func(i, j int) bool { return participants[i].UserID.String() < participants[j].UserID.String() })
	for _, participant := range participants {
		amount := 0.0
		if participant.ShareAmount != nil {
			amount = *participant.ShareAmount
		}
		currency := expense.Currency
		if participant.ShareCurrency != nil && *participant.ShareCurrency != "" {
			currency = *participant.ShareCurrency
		}
		out.Participants = append(out.Participants, appdto.ExpenseParticipant{
			UserID:          participant.UserID,
			DisplayName:     displayName(users, participant.UserID),
			ShareAmount:     money(amount, currency),
			SharePercentage: participant.SharePercentage,
		})
	}
	return out
}

func settlementDTO(settlement *entity.TripSettlement, users map[uuid.UUID]expenseUser, currency string) appdto.TripSettlement {
	return appdto.TripSettlement{
		ID:                settlement.ID,
		TripID:            settlement.TripID,
		FromUserID:        settlement.FromUserID,
		FromDisplayName:   displayName(users, settlement.FromUserID),
		ToUserID:          settlement.ToUserID,
		ToDisplayName:     displayName(users, settlement.ToUserID),
		Amount:            money(settlement.Amount, settlement.Currency),
		Status:            settlement.Status,
		Source:            settlement.Source,
		PaidAt:            settlement.PaidAt,
		PaidByUserID:      settlement.PaidByUserID,
		CancelledAt:       settlement.CancelledAt,
		CancelledByUserID: settlement.CancelledByUserID,
		Notes:             settlement.Notes,
		CreatedAt:         settlement.CreatedAt,
		UpdatedAt:         settlement.UpdatedAt,
	}
}

func expenseActivityMetadata(expense *entity.TripExpense, participantCount int) map[string]any {
	title := expense.Title
	if len([]rune(title)) > 80 {
		title = string([]rune(title)[:80])
	}
	return map[string]any{
		"expenseId":        expense.ID.String(),
		"expenseTitle":     title,
		"amount":           expense.Amount,
		"currency":         expense.Currency,
		"category":         string(expense.Category),
		"paidByUserId":     expense.PaidByUserID.String(),
		"participantCount": participantCount,
	}
}

func settlementActivityMetadata(settlement *entity.TripSettlement) map[string]any {
	return map[string]any{
		"settlementId": settlement.ID.String(),
		"fromUserId":   settlement.FromUserID.String(),
		"toUserId":     settlement.ToUserID.String(),
		"amount":       settlement.Amount,
		"currency":     settlement.Currency,
		"status":       string(settlement.Status),
	}
}

func (s *Service) notifyExpenseParticipants(ctx context.Context, trip *entity.Trip, actorID uuid.UUID, expense *entity.TripExpense, participants []entity.TripExpenseParticipant, users map[uuid.UUID]expenseUser) {
	if !s.notificationsEnabled || s.notifier == nil {
		return
	}
	inputs := make([]notifications.NotificationCreateInput, 0, len(participants))
	for _, participant := range participants {
		if participant.UserID == actorID || participant.UserID == uuid.Nil {
			continue
		}
		inputs = append(inputs, notifications.NotificationCreateInput{
			UserID:      participant.UserID,
			ActorUserID: &actorID,
			Type:        notifications.TypeExpenseAdded,
			Title:       "Expense added",
			Message:     fmt.Sprintf("%s added %s for %s.", displayName(users, actorID), expense.Title, tripDestination(trip)),
			EntityType:  activityEntityType(notifications.EntityTripExpense),
			EntityID:    activityEntityID(expense.ID),
			Metadata: map[string]any{
				"tripId":    trip.ID.String(),
				"expenseId": expense.ID.String(),
				"amount":    expense.Amount,
				"currency":  expense.Currency,
			},
		})
	}
	s.sendNotifications(ctx, inputs)
}

func (s *Service) notifySettlementPaid(ctx context.Context, trip *entity.Trip, actorID uuid.UUID, settlement *entity.TripSettlement, users map[uuid.UUID]expenseUser) {
	if !s.notificationsEnabled || s.notifier == nil {
		return
	}
	recipients := []uuid.UUID{settlement.FromUserID, settlement.ToUserID}
	inputs := make([]notifications.NotificationCreateInput, 0, 2)
	for _, recipient := range recipients {
		if recipient == actorID || recipient == uuid.Nil {
			continue
		}
		inputs = append(inputs, notifications.NotificationCreateInput{
			UserID:      recipient,
			ActorUserID: &actorID,
			Type:        notifications.TypeSettlementPaid,
			Title:       "Settlement marked paid",
			Message:     fmt.Sprintf("%s marked a %s settlement as paid.", displayName(users, actorID), tripDestination(trip)),
			EntityType:  activityEntityType(notifications.EntityTripSettlement),
			EntityID:    activityEntityID(settlement.ID),
			Metadata: map[string]any{
				"tripId":       trip.ID.String(),
				"settlementId": settlement.ID.String(),
				"amount":       settlement.Amount,
				"currency":     settlement.Currency,
			},
		})
	}
	s.sendNotifications(ctx, inputs)
}

func linkedTravelerUserIDs(travelers []entity.TripTraveler) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(travelers))
	for _, traveler := range travelers {
		if traveler.LinkedUserID != nil {
			ids = append(ids, *traveler.LinkedUserID)
		}
	}
	return dedupeUUIDs(ids)
}

func sortedUserIDs(users map[uuid.UUID]expenseUser) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(users))
	for id := range users {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	return ids
}

func sortedShareKeys(shares map[uuid.UUID]int) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(shares))
	for id := range shares {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	return ids
}

func money(amount float64, currency string) appdto.MoneyAmount {
	return appdto.MoneyAmount{Amount: round2(amount), Currency: currency}
}

func moneyCents(amount float64) int {
	return int(math.Round(amount * 100))
}

func centsMoney(cents int) float64 {
	return round2(float64(cents) / 100)
}

func round4(value float64) float64 {
	return math.Round(value*10000) / 10000
}

func formatExpenseAmount(value float64) string {
	if math.Abs(value-math.Round(value)) < 0.005 {
		return fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("%.2f", round2(value))
}

func validCurrencyCode(currency string) bool {
	if len(currency) != 3 {
		return false
	}
	for _, ch := range currency {
		if ch < 'A' || ch > 'Z' {
			return false
		}
	}
	return true
}

func displayName(users map[uuid.UUID]expenseUser, id uuid.UUID) string {
	if user, ok := users[id]; ok && strings.TrimSpace(user.DisplayName) != "" {
		return user.DisplayName
	}
	return shortUserName(id)
}

func shortUserName(id uuid.UUID) string {
	text := id.String()
	if len(text) <= 8 {
		return "User " + text
	}
	return "User " + text[:8]
}

func trimOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func categoryTotalsDTO(totals map[entity.ExpenseCategory]float64, currency string) []appdto.ExpenseCategoryTotal {
	order := []entity.ExpenseCategory{
		entity.ExpenseCategoryTransport,
		entity.ExpenseCategoryAccommodation,
		entity.ExpenseCategoryFood,
		entity.ExpenseCategoryTickets,
		entity.ExpenseCategoryActivities,
		entity.ExpenseCategoryShopping,
		entity.ExpenseCategoryFuel,
		entity.ExpenseCategoryParking,
		entity.ExpenseCategoryTolls,
		entity.ExpenseCategoryCamping,
		entity.ExpenseCategoryGroceries,
		entity.ExpenseCategoryHealthSafety,
		entity.ExpenseCategoryOther,
	}
	out := make([]appdto.ExpenseCategoryTotal, 0, len(totals))
	for _, category := range order {
		if round2(totals[category]) != 0 {
			out = append(out, appdto.ExpenseCategoryTotal{Category: category, Amount: money(totals[category], currency)})
		}
	}
	return out
}

func payerTotalsDTO(totals map[uuid.UUID]float64, users map[uuid.UUID]expenseUser, currency string) []appdto.ExpensePayerTotal {
	ids := make([]uuid.UUID, 0, len(totals))
	for id, amount := range totals {
		if round2(amount) != 0 {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		if totals[ids[i]] != totals[ids[j]] {
			return totals[ids[i]] > totals[ids[j]]
		}
		return ids[i].String() < ids[j].String()
	})
	out := make([]appdto.ExpensePayerTotal, 0, len(ids))
	for _, id := range ids {
		out = append(out, appdto.ExpensePayerTotal{UserID: id, DisplayName: displayName(users, id), Paid: money(totals[id], currency)})
	}
	return out
}

func originalTotalsDTO(totals map[string]float64) []appdto.MoneyAmount {
	currencies := make([]string, 0, len(totals))
	for currency, amount := range totals {
		if round2(amount) != 0 {
			currencies = append(currencies, currency)
		}
	}
	sort.Strings(currencies)
	out := make([]appdto.MoneyAmount, 0, len(currencies))
	for _, currency := range currencies {
		out = append(out, money(totals[currency], currency))
	}
	return out
}

func sumSuggestions(suggestions []appdto.SettlementSuggestion) float64 {
	total := 0.0
	for _, suggestion := range suggestions {
		total += suggestion.Amount.Amount
	}
	return round2(total)
}
