package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budgetconfidence"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	tripobs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/observability"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/triphealth"
)

var ErrTripHealthDisabled = errors.New("trip health is disabled")

func (s *Service) GetTripHealth(
	ctx context.Context,
	tripID uuid.UUID,
	options triphealth.Options,
) (triphealth.Response, error) {
	started := time.Now()
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return triphealth.Response{}, err
	}
	if !s.tripHealthConfig.Enabled {
		return triphealth.Response{}, ErrTripHealthDisabled
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return triphealth.Response{}, err
	}
	cacheKey := summaryCacheKey("trip_health", trip, user.ID, options.IncludeResolved, options.IncludeDebug)
	if cached, ok := s.summaryCache.get("trip_health", cacheKey); ok {
		if response, valid := cached.(triphealth.Response); valid {
			return response, nil
		}
	}

	snapshot := triphealth.Snapshot{
		Trip:      trip,
		Itinerary: parseItineraryLenient(trip.Itinerary),
		Now:       time.Now().UTC(),
		Config:    s.tripHealthConfig,
	}
	if s.verificationConfig.Enabled {
		verificationResponse := s.verificationForTrip(ctx, trip)
		snapshot.Verification = &verificationResponse
	}

	s.loadHealthBudget(ctx, trip, &snapshot)
	s.loadHealthBudgetConfidence(ctx, trip, &snapshot, options)
	s.loadHealthCollaboration(ctx, tripID, &snapshot)
	s.loadHealthChecklist(ctx, tripID, &snapshot)
	s.loadHealthReminders(ctx, tripID, &snapshot)
	s.loadHealthExpenses(ctx, tripID, &snapshot)
	if trip.WorkspaceID != nil {
		s.loadHealthPolicy(ctx, trip, &snapshot)
		s.loadHealthApproval(ctx, user.ID, trip, &snapshot)
	}

	response := triphealth.Evaluate(snapshot, options)
	s.log.Info("trip health evaluated",
		zap.String("trip_id", tripID.String()),
		zap.String("user_id", user.ID.String()),
		zap.Int("score", response.Score),
		zap.String("level", string(response.Level)),
		zap.Int("issue_count", len(response.Issues)),
		zap.Duration("duration", time.Since(started)),
		zap.Strings("subsystem_failures", snapshot.SubsystemFailures),
	)
	tripobs.RecordSummaryCompute("trip_health", time.Since(started))
	s.summaryCache.set("trip_health", cacheKey, response)
	return response, nil
}

func (s *Service) loadHealthBudget(ctx context.Context, trip *entity.Trip, snapshot *triphealth.Snapshot) {
	summary, err := s.budgetSummaryForTrip(ctx, trip)
	if err != nil {
		snapshot.BudgetLoadFailed = true
		snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "budget")
		fallback := budgetSummaryForTrip(trip)
		snapshot.Budget = &fallback
		return
	}
	snapshot.Budget = &summary
}

func (s *Service) loadHealthBudgetConfidence(
	ctx context.Context,
	trip *entity.Trip,
	snapshot *triphealth.Snapshot,
	options triphealth.Options,
) {
	if !s.budgetConfidenceConfig.Enabled {
		return
	}
	response := s.budgetConfidenceForTrip(ctx, trip, budgetconfidenceOptions(trip, options))
	snapshot.BudgetConfidence = &response
}

func budgetconfidenceOptions(trip *entity.Trip, options triphealth.Options) budgetconfidence.Options {
	currency := ""
	if trip != nil {
		currency = trip.BudgetCurrency
	}
	return budgetconfidence.Options{
		Currency:     currency,
		IncludeDebug: options.IncludeDebug,
	}
}

func (s *Service) loadHealthCollaboration(ctx context.Context, tripID uuid.UUID, snapshot *triphealth.Snapshot) {
	collaborators, err := s.repo.ListTripCollaborators(ctx, tripID)
	if err != nil {
		snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "collaboration")
	} else {
		snapshot.Collaborators = collaborators
	}
	availability, err := s.repo.ListTripAvailabilityResponsesByTrip(ctx, tripID)
	if err != nil {
		snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "availability")
	} else {
		snapshot.AvailabilityResponses = availability
	}
	polls, err := s.repo.ListTripPollsByTrip(ctx, tripID, false)
	if err != nil {
		snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "collaboration")
	} else {
		snapshot.Polls = polls
	}
}

func (s *Service) loadHealthChecklist(ctx context.Context, tripID uuid.UUID, snapshot *triphealth.Snapshot) {
	checklist, err := s.activeChecklistWithItems(ctx, tripID)
	if err != nil {
		if !errors.Is(err, domainerrs.ErrNotFound) {
			snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "checklist")
		}
		return
	}
	snapshot.Checklist = checklist
}

func (s *Service) loadHealthReminders(ctx context.Context, tripID uuid.UUID, snapshot *triphealth.Snapshot) {
	reminders, err := s.repo.ListTripRemindersByTrip(ctx, tripID, entity.TripReminderFilters{})
	if err != nil {
		snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "reminders")
		return
	}
	snapshot.Reminders = reminders
}

func (s *Service) loadHealthExpenses(ctx context.Context, tripID uuid.UUID, snapshot *triphealth.Snapshot) {
	expenses, err := s.repo.ListTripExpensesByTrip(ctx, tripID, appdto.ListExpensesInput{})
	if err != nil {
		snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "expenses")
	} else {
		snapshot.Expenses = expenses
	}
	settlements, err := s.repo.ListTripSettlementsByTrip(ctx, tripID)
	if err != nil {
		snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "settlements")
	} else {
		snapshot.Settlements = settlements
	}
	receipts, err := s.repo.ListTripExpenseReceipts(ctx, tripID, appdto.ListReceiptsInput{})
	if err != nil {
		snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "receipts")
		return
	}
	receiptIDs := make([]uuid.UUID, 0, len(receipts))
	for _, receipt := range receipts {
		if receipt.DeletedAt == nil {
			receiptIDs = append(receiptIDs, receipt.ID)
		}
	}
	ocrResults, err := s.repo.ListLatestReceiptOCRResults(ctx, tripID, receiptIDs)
	if err != nil {
		snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "receipt OCR")
	}
	ocrByReceipt := make(map[uuid.UUID]entity.ReceiptOCRResult, len(ocrResults))
	for _, result := range ocrResults {
		ocrByReceipt[result.ReceiptID] = result
	}
	receiptCountByExpense := map[uuid.UUID]int{}
	for _, receipt := range receipts {
		if receipt.DeletedAt != nil {
			continue
		}
		if receipt.ExpenseID != nil {
			receiptCountByExpense[*receipt.ExpenseID]++
		}
		if ocr, ok := ocrByReceipt[receipt.ID]; ok {
			snapshot.ReceiptOCRSignals = append(snapshot.ReceiptOCRSignals, triphealth.ReceiptOCRSignal{
				ReceiptID:  receipt.ID,
				Confidence: ocr.Confidence,
				Warnings:   append([]string(nil), ocr.Warnings...),
			})
		}
	}
	for expenseID, count := range receiptCountByExpense {
		snapshot.ExpenseReceiptSignals = append(snapshot.ExpenseReceiptSignals, triphealth.ExpenseReceiptSignal{
			ExpenseID:    expenseID,
			ReceiptCount: count,
		})
	}
}

func (s *Service) loadHealthPolicy(ctx context.Context, trip *entity.Trip, snapshot *triphealth.Snapshot) {
	evaluation, err := s.evaluateTripPolicyForTrip(ctx, trip)
	if err != nil {
		snapshot.PolicyLoadFailed = true
		snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "policy")
		return
	}
	snapshot.PolicyEvaluation = &evaluation
}

func (s *Service) loadHealthApproval(
	ctx context.Context,
	userID uuid.UUID,
	trip *entity.Trip,
	snapshot *triphealth.Snapshot,
) {
	fields, err := s.repo.GetTripApprovalFields(ctx, trip.ID)
	if err != nil {
		snapshot.ApprovalLoadFailed = true
		snapshot.SubsystemFailures = append(snapshot.SubsystemFailures, "approval")
	} else {
		snapshot.Approval = fields
	}
	risk := s.calculateApprovalRiskForTrip(ctx, userID, trip, true)
	snapshot.ApprovalRisk = &risk
}
