package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/approvalrisk"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	domainerrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspacepolicies"
)

// GetTripApprovalRisk returns a deterministic, live approval-risk score for a
// private trip. Personal trips return not_applicable. Public-share callers never
// reach this method because the route is mounted only in the authenticated group.
func (s *Service) GetTripApprovalRisk(
	ctx context.Context,
	tripID uuid.UUID,
) (approvalrisk.Response, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return approvalrisk.Response{}, err
	}
	trip, _, err := s.requireViewerEditorOrOwner(ctx, tripID, user.ID)
	if err != nil {
		return approvalrisk.Response{}, err
	}
	if trip.WorkspaceID == nil {
		return approvalrisk.NotApplicable(trip.ID, "personal_trip"), nil
	}
	return s.calculateApprovalRiskForTrip(ctx, user.ID, trip, true), nil
}

func (s *Service) calculateApprovalRiskForTrip(
	ctx context.Context,
	userID uuid.UUID,
	trip *entity.Trip,
	includeHeavySignals bool,
) approvalrisk.Response {
	if trip == nil {
		return approvalrisk.NotApplicable(uuid.Nil, "trip_not_found")
	}
	if trip.WorkspaceID == nil {
		return approvalrisk.NotApplicable(trip.ID, "personal_trip")
	}

	now := time.Now().UTC()
	itinerary := parseItineraryLenient(trip.Itinerary)
	signalsUnavailable := make([]string, 0)

	hasWorkspaceBudget, workspaceBudgetSignal := s.approvalRiskWorkspaceBudgetSignal(
		ctx,
		userID,
		trip.WorkspaceID,
		includeHeavySignals,
		&signalsUnavailable,
	)

	_, checklistInput, err := s.computeChecklistWithInput(ctx, trip, hasWorkspaceBudget)
	if err != nil {
		s.warn("approval risk: checklist signal unavailable",
			zap.String("trip_id", trip.ID.String()),
			zap.Error(err),
		)
		signalsUnavailable = append(signalsUnavailable, "approval checklist")
		counts := countItineraryItems(itinerary)
		checklistInput.ItineraryDayCount = len(itinerary.Days)
		checklistInput.ItineraryItemCount = counts.items
		checklistInput.HasTripBudget = trip.BudgetAmount != nil
		checklistInput.TripBudgetAmount = valueOrZeroFloat(trip.BudgetAmount)
		checklistInput.HasWorkspaceBudget = hasWorkspaceBudget
		checklistInput.BookableItemCount = counts.bookable
		checklistInput.AvailabilityUncheckedCount = counts.unchecked
		checklistInput.AvailabilityLowConfidenceCount = counts.lowConfidence
		checklistInput.AvailabilityUnavailableCount = counts.unavailable
		checklistInput.AvailabilityPriceChangedCount = counts.priceChanged
		checklistInput.AvailabilityFallbackCount = counts.fallback
	}

	var policyEvaluationPtr *workspacepolicies.Evaluation
	policyEvaluation, err := s.evaluateTripPolicyForTrip(ctx, trip)
	if err != nil {
		s.warn("approval risk: workspace policy signal unavailable",
			zap.String("trip_id", trip.ID.String()),
			zap.Error(err),
		)
		signalsUnavailable = append(signalsUnavailable, "workspace policy")
	} else {
		policyEvaluationPtr = &policyEvaluation
	}

	metadata := approvalrisk.MetadataSignal{}
	if includeHeavySignals {
		metadata = s.approvalRiskMetadataSignal(ctx, trip.ID, &signalsUnavailable)
	}

	return approvalrisk.Score(approvalrisk.Input{
		TripID:      trip.ID,
		WorkspaceID: trip.WorkspaceID,
		GeneratedAt: now,
		Trip: approvalrisk.TripContext{
			BudgetAmount:   trip.BudgetAmount,
			BudgetCurrency: trip.BudgetCurrency,
			Days:           int(trip.Days),
			Accommodation:  trip.Accommodation,
		},
		ChecklistInput:         checklistInput,
		PolicyEvaluation:       policyEvaluationPtr,
		Itinerary:              itinerary,
		WorkspaceBudget:        workspaceBudgetSignal,
		Metadata:               metadata,
		SignalUnavailableNames: dedupeSignalNames(signalsUnavailable),
	})
}

func (s *Service) approvalRiskWorkspaceBudgetSignal(
	ctx context.Context,
	userID uuid.UUID,
	workspaceID *uuid.UUID,
	includeHeavySignals bool,
	signalsUnavailable *[]string,
) (bool, *approvalrisk.WorkspaceBudgetSignal) {
	if workspaceID == nil {
		return false, nil
	}
	primary, err := s.repo.GetPrimaryWorkspaceBudget(ctx, *workspaceID)
	if err != nil {
		if !errors.Is(err, domainerrs.ErrNotFound) {
			s.warn("approval risk: failed to load primary workspace budget",
				zap.String("workspace_id", workspaceID.String()),
				zap.Error(err),
			)
			*signalsUnavailable = append(*signalsUnavailable, "workspace budget")
		}
		return false, nil
	}
	if !includeHeavySignals {
		return true, nil
	}
	summary, err := s.calculateWorkspaceBudgetSummary(ctx, userID, primary)
	if err != nil {
		s.warn("approval risk: failed to calculate workspace budget summary",
			zap.String("workspace_id", workspaceID.String()),
			zap.String("budget_id", primary.ID.String()),
			zap.Error(err),
		)
		*signalsUnavailable = append(*signalsUnavailable, "workspace budget")
		return true, nil
	}
	return true, &approvalrisk.WorkspaceBudgetSignal{
		Amount:             primary.Amount,
		Currency:           primary.Currency,
		EstimatedTotal:     summary.Summary.EstimatedTotal,
		OverBudgetAmount:   summary.Summary.OverBudgetAmount,
		UtilizationPercent: summary.Summary.UtilizationPercent,
	}
}

func (s *Service) approvalRiskMetadataSignal(
	ctx context.Context,
	tripID uuid.UUID,
	signalsUnavailable *[]string,
) approvalrisk.MetadataSignal {
	versions, err := s.repo.ListItineraryVersionsByTrip(ctx, tripID, 1, 0)
	if err != nil {
		s.warn("approval risk: failed to load itinerary version metadata",
			zap.String("trip_id", tripID.String()),
			zap.Error(err),
		)
		*signalsUnavailable = append(*signalsUnavailable, "AI/template metadata")
		return approvalrisk.MetadataSignal{}
	}
	if len(versions) == 0 {
		return approvalrisk.MetadataSignal{}
	}
	latest := versions[0]
	source := strings.TrimSpace(string(latest.Source))
	if value, ok := latest.Metadata["source"].(string); ok && strings.TrimSpace(value) != "" {
		source = strings.TrimSpace(value)
	}
	return approvalrisk.MetadataSignal{
		Source:               source,
		TemplateFallbackUsed: metadataBool(latest.Metadata, "fallbackUsed"),
		TemplateWarningCount: metadataStringSliceCount(latest.Metadata, "warnings"),
		ValidationRepairUsed: metadataBool(latest.Metadata, "repairUsed") ||
			metadataBool(latest.Metadata, "validationRepairUsed"),
	}
}

func metadataBool(metadata map[string]any, key string) bool {
	if metadata == nil {
		return false
	}
	switch value := metadata[key].(type) {
	case bool:
		return value
	case string:
		return strings.EqualFold(strings.TrimSpace(value), "true")
	default:
		return false
	}
}

func metadataStringSliceCount(metadata map[string]any, key string) int {
	if metadata == nil {
		return 0
	}
	switch value := metadata[key].(type) {
	case []string:
		return len(value)
	case []any:
		return len(value)
	default:
		return 0
	}
}

func dedupeSignalNames(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func (s *Service) warn(message string, fields ...zap.Field) {
	if s.log == nil {
		return
	}
	s.log.Warn(message, fields...)
}
