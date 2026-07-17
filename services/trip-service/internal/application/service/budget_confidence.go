package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetconfidence"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
)

var ErrBudgetConfidenceDisabled = errors.New("budget confidence is disabled")

func (s *Service) GetBudgetConfidence(
	ctx context.Context,
	tripID uuid.UUID,
	options budgetconfidence.Options,
) (budgetconfidence.Response, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return budgetconfidence.Response{}, err
	}
	if !s.budgetConfidenceConfig.Enabled {
		return budgetconfidence.Response{}, ErrBudgetConfidenceDisabled
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return budgetconfidence.Response{}, err
	}
	currency := strings.ToUpper(strings.TrimSpace(options.Currency))
	cacheKey := summaryCacheKey("budget_confidence", trip, user.ID, currency, options.IncludeDebug)
	if cached, ok := s.summaryCache.get("budget_confidence", cacheKey); ok {
		if response, valid := cached.(budgetconfidence.Response); valid {
			return response, nil
		}
	}
	response := s.budgetConfidenceForTrip(ctx, trip, options)
	s.summaryCache.set("budget_confidence", cacheKey, response)
	return response, nil
}

func (s *Service) budgetConfidenceForTrip(
	ctx context.Context,
	trip *entity.Trip,
	options budgetconfidence.Options,
) budgetconfidence.Response {
	cfg := s.budgetConfidenceConfig
	if cfg == (budgetconfidence.Config{}) {
		cfg = budgetconfidence.DefaultConfig()
	}
	itinerary := parseItineraryLenient(trip.Itinerary)
	currency := strings.ToUpper(strings.TrimSpace(options.Currency))
	if currency == "" {
		currency = strings.ToUpper(strings.TrimSpace(trip.BudgetCurrency))
	}

	var (
		summary           *budget.Summary
		summaryLoadFailed bool
		warnings          []string
	)
	if loaded, err := s.budgetSummaryForTrip(ctx, trip); err != nil {
		summaryLoadFailed = true
		warnings = append(warnings, "Budget summary could not be fully loaded.")
		if s.budgetConversionFailOpen {
			fallback := budgetSummaryForTrip(trip)
			summary = &fallback
		}
	} else {
		summary = &loaded
	}

	expenses, expenseFailed := s.loadBudgetConfidenceExpenses(ctx, trip.ID)
	receipts, receiptOCR, receiptFailed := s.loadBudgetConfidenceReceipts(ctx, trip.ID)
	if expenseFailed {
		warnings = append(warnings, "Actual expenses could not be loaded.")
	}
	if receiptFailed {
		warnings = append(warnings, "Receipt metadata could not be fully loaded.")
	}

	response := budgetconfidence.Compute(ctx, budgetconfidence.Input{
		Trip:                    trip,
		Itinerary:               itinerary,
		BudgetSummary:           summary,
		Expenses:                expenses,
		Receipts:                receipts,
		ReceiptOCR:              receiptOCR,
		Converter:               s.budgetConversionProvider,
		ConversionEnabled:       s.budgetConversionEnabled,
		ConversionFailOpen:      s.budgetConversionFailOpen,
		Currency:                currency,
		Now:                     time.Now().UTC(),
		Config:                  cfg,
		IncludeDebug:            options.IncludeDebug,
		ExpenseLoadFailed:       expenseFailed,
		ReceiptLoadFailed:       receiptFailed,
		BudgetSummaryLoadFailed: summaryLoadFailed,
		AdditionalWarnings:      warnings,
	})
	s.log.Info("budget confidence evaluated",
		zap.String("trip_id", trip.ID.String()),
		zap.Int("score", response.Score),
		zap.String("level", string(response.Level)),
		zap.String("risk_level", string(response.RiskLevel)),
		zap.Int("issue_count", len(response.Issues)),
	)
	return response
}

func (s *Service) loadBudgetConfidenceExpenses(ctx context.Context, tripID uuid.UUID) ([]entity.TripExpense, bool) {
	expenses, err := s.repo.ListTripExpensesByTrip(ctx, tripID, appdto.ListExpensesInput{})
	if err != nil {
		s.warn("budget confidence: failed to load expenses",
			zap.String("trip_id", tripID.String()),
			zap.Error(err),
		)
		return nil, true
	}
	return expenses, false
}

func (s *Service) loadBudgetConfidenceReceipts(
	ctx context.Context,
	tripID uuid.UUID,
) ([]entity.TripExpenseReceipt, map[uuid.UUID]*entity.ReceiptOCRResult, bool) {
	receipts, err := s.repo.ListTripExpenseReceipts(ctx, tripID, appdto.ListReceiptsInput{})
	if err != nil {
		s.warn("budget confidence: failed to load receipts",
			zap.String("trip_id", tripID.String()),
			zap.Error(err),
		)
		return nil, nil, true
	}
	ocrByReceipt := make(map[uuid.UUID]*entity.ReceiptOCRResult, len(receipts))
	failed := false
	for _, receipt := range receipts {
		if receipt.DeletedAt != nil {
			continue
		}
		ocr, err := s.repo.GetLatestReceiptOCRResult(ctx, tripID, receipt.ID)
		if err != nil {
			if !errors.Is(err, domainerrs.ErrNotFound) {
				failed = true
				s.warn("budget confidence: failed to load receipt OCR",
					zap.String("trip_id", tripID.String()),
					zap.String("receipt_id", receipt.ID.String()),
					zap.Error(err),
				)
			}
			continue
		}
		if ocr != nil {
			ocrByReceipt[receipt.ID] = ocr
		}
	}
	return receipts, ocrByReceipt, failed
}
