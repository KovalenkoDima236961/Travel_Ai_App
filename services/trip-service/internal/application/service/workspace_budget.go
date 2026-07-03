package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/analytics"
	appdto "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/dto"
	apperrs "github.com/KovalenkoDima236961/Travel_Ai_App/internal/application/errs"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/auth"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/notifications"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/workspaces"
)

const (
	maxWorkspaceBudgetNameLength        = 100
	minWorkspaceBudgetNameLength        = 2
	maxWorkspaceBudgetDescriptionLength = 500
)

type workspaceAnalyticsComputation struct {
	Analytics        analytics.WorkspaceCostAnalytics
	ExchangeRateInfo *budget.ExchangeRateInfo
	SkippedUndated   int
	TripLimitReached bool
}

func (s *Service) CreateWorkspaceBudget(
	ctx context.Context,
	workspaceID uuid.UUID,
	in appdto.CreateWorkspaceBudgetInput,
) (*entity.WorkspaceBudget, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.requireWorkspaceBudgetManageAccess(ctx, user.ID, workspaceID); err != nil {
		return nil, err
	}

	name, description, amount, currency, err := normalizeWorkspaceBudgetFields(
		in.Name,
		in.Description,
		in.Amount,
		in.Currency,
		in.PeriodStart,
		in.PeriodEnd,
	)
	if err != nil {
		return nil, err
	}

	isPrimary := false
	if in.IsPrimary != nil {
		isPrimary = *in.IsPrimary
	} else {
		status := entity.WorkspaceBudgetStatusActive
		count, err := s.repo.CountWorkspaceBudgets(ctx, workspaceID, &status)
		if err != nil {
			return nil, err
		}
		isPrimary = count == 0
	}

	created, err := s.repo.CreateWorkspaceBudget(ctx, &entity.WorkspaceBudget{
		ID:              uuid.New(),
		WorkspaceID:     workspaceID,
		Name:            name,
		Description:     description,
		Amount:          amount,
		Currency:        currency,
		PeriodStart:     normalizeDatePtr(in.PeriodStart),
		PeriodEnd:       normalizeDatePtr(in.PeriodEnd),
		Status:          entity.WorkspaceBudgetStatusActive,
		IsPrimary:       isPrimary,
		CreatedByUserID: user.ID,
	})
	if err != nil {
		return nil, err
	}

	s.log.Info("workspace budget created",
		zap.String("workspace_id", workspaceID.String()),
		zap.String("budget_id", created.ID.String()),
		zap.String("user_id", user.ID.String()),
	)
	s.notifyWorkspaceBudgetMutation(
		ctx,
		created,
		user.ID,
		notifications.TypeWorkspaceBudgetCreated,
		"Workspace budget created",
		fmt.Sprintf("%s was created for %.2f %s.", created.Name, created.Amount, created.Currency),
		"created",
	)
	return created, nil
}

func (s *Service) ListWorkspaceBudgets(
	ctx context.Context,
	workspaceID uuid.UUID,
	statusRaw string,
) ([]entity.WorkspaceBudget, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.requireWorkspaceBudgetViewAccess(ctx, user.ID, workspaceID); err != nil {
		return nil, err
	}
	status, err := parseWorkspaceBudgetStatus(statusRaw)
	if err != nil {
		return nil, err
	}
	return s.repo.ListWorkspaceBudgetsByWorkspace(ctx, workspaceID, status)
}

func (s *Service) GetWorkspaceBudget(
	ctx context.Context,
	workspaceID uuid.UUID,
	budgetID uuid.UUID,
) (*entity.WorkspaceBudget, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.requireWorkspaceBudgetViewAccess(ctx, user.ID, workspaceID); err != nil {
		return nil, err
	}
	return s.repo.GetWorkspaceBudgetByID(ctx, workspaceID, budgetID)
}

func (s *Service) UpdateWorkspaceBudget(
	ctx context.Context,
	workspaceID uuid.UUID,
	budgetID uuid.UUID,
	in appdto.UpdateWorkspaceBudgetInput,
) (*entity.WorkspaceBudget, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.requireWorkspaceBudgetManageAccess(ctx, user.ID, workspaceID); err != nil {
		return nil, err
	}

	current, err := s.repo.GetWorkspaceBudgetByID(ctx, workspaceID, budgetID)
	if err != nil {
		return nil, err
	}
	if current.Status == entity.WorkspaceBudgetStatusArchived {
		return nil, apperrs.NewInvalidInput("archived budgets cannot be updated")
	}

	next := *current
	if in.Name != nil {
		next.Name = *in.Name
	}
	if in.DescriptionSet {
		next.Description = in.Description
	}
	if in.Amount != nil {
		next.Amount = *in.Amount
	}
	if in.Currency != nil {
		next.Currency = *in.Currency
	}
	if in.PeriodStartSet {
		next.PeriodStart = normalizeDatePtr(in.PeriodStart)
	}
	if in.PeriodEndSet {
		next.PeriodEnd = normalizeDatePtr(in.PeriodEnd)
	}
	if in.IsPrimary != nil {
		next.IsPrimary = *in.IsPrimary
	}

	name, description, amount, currency, err := normalizeWorkspaceBudgetFields(
		next.Name,
		next.Description,
		next.Amount,
		next.Currency,
		next.PeriodStart,
		next.PeriodEnd,
	)
	if err != nil {
		return nil, err
	}
	next.Name = name
	next.Description = description
	next.Amount = amount
	next.Currency = currency

	updated, err := s.repo.UpdateWorkspaceBudget(ctx, &next)
	if err != nil {
		return nil, err
	}
	s.log.Info("workspace budget updated",
		zap.String("workspace_id", workspaceID.String()),
		zap.String("budget_id", budgetID.String()),
		zap.String("user_id", user.ID.String()),
	)
	s.notifyWorkspaceBudgetMutation(
		ctx,
		updated,
		user.ID,
		notifications.TypeWorkspaceBudgetUpdated,
		"Workspace budget updated",
		fmt.Sprintf("%s was updated.", updated.Name),
		"updated",
	)
	return updated, nil
}

func (s *Service) ArchiveWorkspaceBudget(ctx context.Context, workspaceID, budgetID uuid.UUID) (*entity.WorkspaceBudget, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.requireWorkspaceBudgetManageAccess(ctx, user.ID, workspaceID); err != nil {
		return nil, err
	}
	archived, err := s.repo.ArchiveWorkspaceBudget(ctx, workspaceID, budgetID, user.ID)
	if err != nil {
		return nil, err
	}
	s.log.Info("workspace budget archived",
		zap.String("workspace_id", workspaceID.String()),
		zap.String("budget_id", budgetID.String()),
		zap.String("user_id", user.ID.String()),
	)
	s.notifyWorkspaceBudgetMutation(
		ctx,
		archived,
		user.ID,
		notifications.TypeWorkspaceBudgetArchived,
		"Workspace budget archived",
		fmt.Sprintf("%s was archived.", archived.Name),
		"archived",
	)
	return archived, nil
}

func (s *Service) MakeWorkspaceBudgetPrimary(ctx context.Context, workspaceID, budgetID uuid.UUID) (*entity.WorkspaceBudget, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.requireWorkspaceBudgetManageAccess(ctx, user.ID, workspaceID); err != nil {
		return nil, err
	}
	current, err := s.repo.GetWorkspaceBudgetByID(ctx, workspaceID, budgetID)
	if err != nil {
		return nil, err
	}
	if current.Status != entity.WorkspaceBudgetStatusActive {
		return nil, apperrs.NewInvalidInput("only active budgets can be primary")
	}
	updated, err := s.repo.SetWorkspaceBudgetPrimary(ctx, workspaceID, budgetID)
	if err != nil {
		return nil, err
	}
	s.log.Info("workspace budget primary changed",
		zap.String("workspace_id", workspaceID.String()),
		zap.String("budget_id", budgetID.String()),
		zap.String("user_id", user.ID.String()),
	)
	s.notifyWorkspaceBudgetMutation(
		ctx,
		updated,
		user.ID,
		notifications.TypeWorkspaceBudgetUpdated,
		"Primary workspace budget changed",
		fmt.Sprintf("%s is now the primary workspace budget.", updated.Name),
		"primary_changed",
	)
	return updated, nil
}

func (s *Service) GetWorkspaceBudgetSummary(
	ctx context.Context,
	workspaceID uuid.UUID,
	budgetID uuid.UUID,
) (appdto.WorkspaceBudgetSummaryResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.WorkspaceBudgetSummaryResponse{}, err
	}
	if err := s.requireWorkspaceBudgetViewAccess(ctx, user.ID, workspaceID); err != nil {
		return appdto.WorkspaceBudgetSummaryResponse{}, err
	}
	budgetEntity, err := s.repo.GetWorkspaceBudgetByID(ctx, workspaceID, budgetID)
	if err != nil {
		return appdto.WorkspaceBudgetSummaryResponse{}, err
	}
	return s.calculateWorkspaceBudgetSummary(ctx, user.ID, budgetEntity)
}

func (s *Service) GetPrimaryWorkspaceBudgetSummary(
	ctx context.Context,
	workspaceID uuid.UUID,
) (appdto.WorkspaceBudgetSummaryResponse, error) {
	user, err := auth.MustUserFromContext(ctx)
	if err != nil {
		return appdto.WorkspaceBudgetSummaryResponse{}, err
	}
	if err := s.requireWorkspaceBudgetViewAccess(ctx, user.ID, workspaceID); err != nil {
		return appdto.WorkspaceBudgetSummaryResponse{}, err
	}
	budgetEntity, err := s.repo.GetPrimaryWorkspaceBudget(ctx, workspaceID)
	if err != nil {
		return appdto.WorkspaceBudgetSummaryResponse{}, err
	}
	return s.calculateWorkspaceBudgetSummary(ctx, user.ID, budgetEntity)
}

func (s *Service) calculateWorkspaceBudgetSummary(
	ctx context.Context,
	userID uuid.UUID,
	budgetEntity *entity.WorkspaceBudget,
) (appdto.WorkspaceBudgetSummaryResponse, error) {
	computed, err := s.calculateWorkspaceCostAnalytics(ctx, userID, budgetEntity.WorkspaceID, appdto.WorkspaceCostAnalyticsInput{
		Currency: budgetEntity.Currency,
		From:     budgetEntity.PeriodStart,
		To:       budgetEntity.PeriodEnd,
	})
	if err != nil {
		return appdto.WorkspaceBudgetSummaryResponse{}, err
	}
	workspaceAnalytics := computed.Analytics
	summary := appdto.WorkspaceBudgetSummaryMetrics{
		TripCount:              workspaceAnalytics.Summary.TripCount,
		EstimatedTotal:         workspaceAnalytics.Summary.EstimatedTotal,
		RemainingAmount:        roundMoney(budgetEntity.Amount - workspaceAnalytics.Summary.EstimatedTotal),
		OverBudgetAmount:       roundMoney(math.Max(0, workspaceAnalytics.Summary.EstimatedTotal-budgetEntity.Amount)),
		UtilizationPercent:     percentageOf(workspaceAnalytics.Summary.EstimatedTotal, budgetEntity.Amount),
		MissingEstimateCount:   workspaceAnalytics.Summary.MissingEstimateCount,
		UncertainEstimateCount: workspaceAnalytics.Summary.UncertainEstimateCount,
		ConvertedItemCount:     workspaceAnalytics.Summary.ConvertedItemCount,
		UnconvertedItemCount:   workspaceAnalytics.Summary.UnconvertedItemCount,
	}

	warnings := append([]string{}, workspaceAnalytics.Warnings...)
	if computed.SkippedUndated > 0 {
		warnings = append(warnings, fmt.Sprintf("%d trip(s) without a start date were excluded from this dated budget period.", computed.SkippedUndated))
	}

	return appdto.WorkspaceBudgetSummaryResponse{
		Budget:           appdto.NewWorkspaceBudgetResponse(budgetEntity),
		GeneratedAt:      workspaceAnalytics.GeneratedAt,
		Summary:          summary,
		ByTrip:           budgetTripSummaries(workspaceAnalytics.ByTrip, budgetEntity.Amount),
		ByCategory:       budgetBreakdowns(workspaceAnalytics.ByCategory, budgetEntity.Amount, "category"),
		BySource:         budgetBreakdowns(workspaceAnalytics.BySource, budgetEntity.Amount, "source"),
		ExpensiveItems:   workspaceAnalytics.ExpensiveItems,
		Insights:         workspaceBudgetInsights(budgetEntity, summary, workspaceAnalytics.ByTrip),
		Warnings:         uniqueWarnings(warnings),
		ExchangeRateInfo: computed.ExchangeRateInfo,
	}, nil
}

func (s *Service) calculateWorkspaceCostAnalytics(
	ctx context.Context,
	userID uuid.UUID,
	workspaceID uuid.UUID,
	in appdto.WorkspaceCostAnalyticsInput,
) (workspaceAnalyticsComputation, error) {
	targetCurrency := in.Currency
	if targetCurrency == "" {
		targetCurrency = budget.DefaultCurrency
	}

	trips, err := s.repo.ListAccessible(
		ctx,
		userID,
		[]uuid.UUID{workspaceID},
		appdto.TripListScopeWorkspace,
		&workspaceID,
		maxWorkspaceAnalyticsTrips,
		0,
	)
	if err != nil {
		return workspaceAnalyticsComputation{}, err
	}

	generatedAt := time.Now().UTC()
	items := make([]analytics.WorkspaceTripInput, 0, len(trips))
	var exchangeRateInfo *budget.ExchangeRateInfo
	skippedUndated := 0
	for _, trip := range trips {
		included, skippedMissingDate := tripInWorkspaceBudgetDateRange(trip, in.From, in.To)
		if skippedMissingDate {
			skippedUndated++
		}
		if !included {
			continue
		}
		tripAnalytics, err := s.tripCostAnalyticsForTrip(ctx, &trip, targetCurrency, generatedAt)
		if err != nil {
			return workspaceAnalyticsComputation{}, err
		}
		if exchangeRateInfo == nil && tripAnalytics.ExchangeRateInfo != nil {
			exchangeRateInfo = cloneBudgetExchangeRateInfo(tripAnalytics.ExchangeRateInfo)
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
	return workspaceAnalyticsComputation{
		Analytics:        result,
		ExchangeRateInfo: exchangeRateInfo,
		SkippedUndated:   skippedUndated,
		TripLimitReached: len(trips) >= maxWorkspaceAnalyticsTrips,
	}, nil
}

func (s *Service) requireWorkspaceBudgetViewAccess(ctx context.Context, userID, workspaceID uuid.UUID) error {
	access, err := s.workspaceAccess(ctx, userID, workspaceID)
	if err != nil {
		return err
	}
	switch access.Role {
	case workspaces.RoleOwner, workspaces.RoleAdmin, workspaces.RoleMember, workspaces.RoleViewer:
		return nil
	default:
		return apperrs.ErrForbidden
	}
}

func (s *Service) requireWorkspaceBudgetManageAccess(ctx context.Context, userID, workspaceID uuid.UUID) error {
	access, err := s.workspaceAccess(ctx, userID, workspaceID)
	if err != nil {
		return err
	}
	if access.WorkspaceArchived {
		return apperrs.ErrForbidden
	}
	switch access.Role {
	case workspaces.RoleOwner, workspaces.RoleAdmin:
		return nil
	default:
		return apperrs.ErrForbidden
	}
}

func (s *Service) workspaceAccess(ctx context.Context, userID, workspaceID uuid.UUID) (*workspaces.Access, error) {
	if !s.workspacesEnabled || s.workspaceProvider == nil {
		return nil, apperrs.ErrForbidden
	}
	access, err := s.workspaceProvider.AccessCheck(ctx, userID, workspaceID)
	if err != nil {
		return nil, err
	}
	if access == nil || !access.HasAccess {
		return nil, apperrs.ErrForbidden
	}
	return access, nil
}

func (s *Service) notifyWorkspaceBudgetMutation(
	ctx context.Context,
	budgetEntity *entity.WorkspaceBudget,
	actorID uuid.UUID,
	notificationType string,
	title string,
	message string,
	event string,
) {
	if !s.notificationsEnabled || s.notifier == nil || budgetEntity == nil {
		return
	}
	recipients := s.workspaceBudgetNotificationRecipients(ctx, budgetEntity.WorkspaceID, actorID)
	if len(recipients) == 0 {
		return
	}
	actor := actorID
	entityID := budgetEntity.ID
	inputs := make([]notifications.NotificationCreateInput, 0, len(recipients))
	for _, recipient := range recipients {
		inputs = append(inputs, notifications.NotificationCreateInput{
			UserID:      recipient,
			ActorUserID: &actor,
			Type:        notificationType,
			Title:       title,
			Message:     message,
			EntityType:  activityEntityType(notifications.EntityWorkspaceBudget),
			EntityID:    &entityID,
			Metadata: map[string]any{
				"event":       event,
				"workspaceId": budgetEntity.WorkspaceID.String(),
				"budgetId":    budgetEntity.ID.String(),
				"budgetName":  budgetEntity.Name,
				"amount":      budgetEntity.Amount,
				"currency":    budgetEntity.Currency,
				"status":      string(budgetEntity.Status),
				"isPrimary":   budgetEntity.IsPrimary,
				"periodStart": workspaceBudgetDateMetadata(budgetEntity.PeriodStart),
				"periodEnd":   workspaceBudgetDateMetadata(budgetEntity.PeriodEnd),
				"url":         fmt.Sprintf("/workspaces/%s/budgets/%s", budgetEntity.WorkspaceID, budgetEntity.ID),
			},
		})
	}
	s.sendNotifications(ctx, inputs)
}

func (s *Service) workspaceBudgetNotificationRecipients(ctx context.Context, workspaceID, actorID uuid.UUID) []uuid.UUID {
	if !s.workspacesEnabled || s.workspaceProvider == nil {
		return nil
	}
	members, err := s.workspaceProvider.ListMembers(ctx, workspaceID)
	if err != nil {
		s.log.Warn("failed to list workspace members for budget notification fan-out",
			zap.String("workspace_id", workspaceID.String()),
			zap.Error(err),
		)
		return nil
	}
	seen := map[uuid.UUID]struct{}{actorID: {}}
	recipients := make([]uuid.UUID, 0)
	for _, member := range members {
		if member.Status != workspaces.MemberStatusActive {
			continue
		}
		switch member.Role {
		case workspaces.RoleOwner, workspaces.RoleAdmin:
		default:
			continue
		}
		if member.UserID == uuid.Nil {
			continue
		}
		if _, ok := seen[member.UserID]; ok {
			continue
		}
		seen[member.UserID] = struct{}{}
		recipients = append(recipients, member.UserID)
	}
	return recipients
}

func workspaceBudgetDateMetadata(value *time.Time) *string {
	if value == nil {
		return nil
	}
	out := value.Format("2006-01-02")
	return &out
}

func normalizeWorkspaceBudgetFields(
	name string,
	description *string,
	amount float64,
	currency string,
	periodStart *time.Time,
	periodEnd *time.Time,
) (string, *string, float64, string, error) {
	name = strings.TrimSpace(name)
	if len(name) < minWorkspaceBudgetNameLength || len(name) > maxWorkspaceBudgetNameLength {
		return "", nil, 0, "", apperrs.NewInvalidInput("name must be between 2 and 100 characters")
	}
	if description != nil {
		trimmed := strings.TrimSpace(*description)
		if trimmed == "" {
			description = nil
		} else {
			if len(trimmed) > maxWorkspaceBudgetDescriptionLength {
				return "", nil, 0, "", apperrs.NewInvalidInput("description must be 500 characters or less")
			}
			description = &trimmed
		}
	}
	if amount < 0 {
		return "", nil, 0, "", apperrs.NewInvalidInput("amount must be greater than or equal to 0")
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if !validCurrency(currency) {
		return "", nil, 0, "", apperrs.NewInvalidInput("currency must be a 3-letter uppercase code")
	}
	if periodStart != nil && periodEnd != nil && truncateDate(*periodStart).After(truncateDate(*periodEnd)) {
		return "", nil, 0, "", apperrs.NewInvalidInput("periodStart must be before or equal to periodEnd")
	}
	return name, description, roundMoney(amount), currency, nil
}

func parseWorkspaceBudgetStatus(raw string) (*entity.WorkspaceBudgetStatus, error) {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return nil, nil
	}
	status := entity.WorkspaceBudgetStatus(raw)
	switch status {
	case entity.WorkspaceBudgetStatusActive, entity.WorkspaceBudgetStatusArchived:
		return &status, nil
	default:
		return nil, apperrs.NewInvalidInput("invalid budget status")
	}
}

func validCurrency(value string) bool {
	if len(value) != 3 {
		return false
	}
	for _, ch := range value {
		if ch < 'A' || ch > 'Z' {
			return false
		}
	}
	return true
}

func normalizeDatePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	date := truncateDate(*value)
	return &date
}

func tripInWorkspaceBudgetDateRange(trip entity.Trip, from, to *time.Time) (bool, bool) {
	if from == nil && to == nil {
		return true, false
	}
	if trip.StartDate == nil {
		return false, true
	}
	start := truncateDate(*trip.StartDate)
	if from != nil && start.Before(truncateDate(*from)) {
		return false, false
	}
	if to != nil && start.After(truncateDate(*to)) {
		return false, false
	}
	return true, false
}

func budgetTripSummaries(trips []analytics.TripCostSummary, amount float64) []appdto.WorkspaceBudgetTripSummary {
	out := make([]appdto.WorkspaceBudgetTripSummary, 0, len(trips))
	for _, trip := range trips {
		out = append(out, appdto.WorkspaceBudgetTripSummary{
			TripID:               trip.TripID,
			Title:                trip.Title,
			Destination:          trip.Destination,
			StartDate:            trip.StartDate,
			EstimatedTotal:       trip.EstimatedTotal,
			PercentageOfBudget:   percentageOf(trip.EstimatedTotal, amount),
			MissingEstimateCount: trip.MissingEstimateCount,
			OverTripBudgetAmount: trip.OverBudgetAmount,
		})
	}
	return out
}

func budgetBreakdowns(entries []analytics.CostAmountBreakdown, amount float64, kind string) []appdto.WorkspaceBudgetBreakdown {
	out := make([]appdto.WorkspaceBudgetBreakdown, 0, len(entries))
	for _, entry := range entries {
		item := appdto.WorkspaceBudgetBreakdown{
			Amount:                     entry.Amount,
			PercentageOfBudget:         percentageOf(entry.Amount, amount),
			PercentageOfEstimatedTotal: entry.Percentage,
			ItemCount:                  entry.ItemCount,
		}
		if kind == "category" {
			item.Category = entry.Category
		} else {
			item.Source = entry.Source
		}
		out = append(out, item)
	}
	return out
}

func workspaceBudgetInsights(
	b *entity.WorkspaceBudget,
	summary appdto.WorkspaceBudgetSummaryMetrics,
	byTrip []analytics.TripCostSummary,
) []analytics.CostInsight {
	insights := make([]analytics.CostInsight, 0)
	if summary.TripCount == 0 {
		insights = append(insights, analytics.CostInsight{
			Type:     "no_trips_in_period",
			Severity: analytics.InsightSeverityInfo,
			Title:    "No trips in this budget period",
			Message:  "No workspace trips currently match this budget period.",
		})
	}
	if summary.OverBudgetAmount > 0 {
		insights = append(insights, analytics.CostInsight{
			Type:     "workspace_budget_exceeded",
			Severity: analytics.InsightSeverityCritical,
			Title:    "Workspace budget is exceeded",
			Message:  fmt.Sprintf("Estimated costs are %.2f %s above the shared budget.", summary.OverBudgetAmount, b.Currency),
			Action:   &analytics.CostInsightAction{Type: analytics.ActionOpenWorkspaceAnalytics},
		})
	} else if summary.UtilizationPercent >= 80 && summary.UtilizationPercent <= 100 {
		insights = append(insights, analytics.CostInsight{
			Type:     "workspace_budget_nearing_limit",
			Severity: analytics.InsightSeverityWarning,
			Title:    fmt.Sprintf("Workspace budget is %.0f%% used", summary.UtilizationPercent),
			Message:  "Estimated costs are close to the shared budget limit.",
			Action:   &analytics.CostInsightAction{Type: analytics.ActionOpenWorkspaceAnalytics},
		})
	}
	if len(byTrip) > 0 && percentageOf(byTrip[0].EstimatedTotal, b.Amount) > 40 {
		tripID := byTrip[0].TripID
		insights = append(insights, analytics.CostInsight{
			Type:     "expensive_trip",
			Severity: analytics.InsightSeverityInfo,
			Title:    "One trip uses a large share",
			Message:  fmt.Sprintf("%s consumes %.0f%% of the shared budget.", byTrip[0].Title, percentageOf(byTrip[0].EstimatedTotal, b.Amount)),
			Action:   &analytics.CostInsightAction{Type: analytics.ActionOpenTrip, TripID: &tripID},
		})
	}
	if summary.MissingEstimateCount > 0 {
		insights = append(insights, analytics.CostInsight{
			Type:     "missing_estimates",
			Severity: analytics.InsightSeverityWarning,
			Title:    "Some estimates are missing",
			Message:  fmt.Sprintf("%d cost-relevant item(s) are missing estimates.", summary.MissingEstimateCount),
			Action:   &analytics.CostInsightAction{Type: analytics.ActionCheckMissingPrices},
		})
	}
	if summary.UnconvertedItemCount > 0 {
		insights = append(insights, analytics.CostInsight{
			Type:     "conversion_warnings",
			Severity: analytics.InsightSeverityWarning,
			Title:    "Some costs were not converted",
			Message:  fmt.Sprintf("%d cost(s) could not be converted into %s.", summary.UnconvertedItemCount, b.Currency),
		})
	}
	if b.PeriodStart == nil && b.PeriodEnd == nil {
		insights = append(insights, analytics.CostInsight{
			Type:     "budget_period_empty_or_open",
			Severity: analytics.InsightSeverityInfo,
			Title:    "Budget covers all workspace trips",
			Message:  "This shared budget has no start or end date.",
		})
	}
	if len(insights) == 0 {
		insights = append(insights, analytics.CostInsight{
			Type:     "export_budget_report",
			Severity: analytics.InsightSeverityInfo,
			Title:    "Budget report is ready",
			Message:  "Export this workspace budget summary for planning review.",
			Action:   &analytics.CostInsightAction{Type: analytics.ActionExportBudgetReport},
		})
	}
	return insights
}

func uniqueWarnings(values []string) []string {
	seen := make(map[string]struct{}, len(values)+1)
	out := make([]string, 0, len(values)+1)
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	if len(out) == 0 {
		out = append(out, analytics.PlanningDisclaimer)
	}
	return out
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}

func percentageOf(amount, total float64) float64 {
	if total <= 0 {
		return 0
	}
	return roundMoney(amount / total * 100)
}

func cloneBudgetExchangeRateInfo(info *budget.ExchangeRateInfo) *budget.ExchangeRateInfo {
	if info == nil {
		return nil
	}
	copyInfo := *info
	return &copyInfo
}

func activeBudgetUsageFromSummary(summary appdto.WorkspaceBudgetSummaryResponse) *analytics.ActiveWorkspaceBudget {
	return &analytics.ActiveWorkspaceBudget{
		ID:                 summary.Budget.ID,
		Name:               summary.Budget.Name,
		Amount:             summary.Budget.Amount,
		Currency:           summary.Budget.Currency,
		PeriodStart:        summary.Budget.PeriodStart,
		PeriodEnd:          summary.Budget.PeriodEnd,
		EstimatedTotal:     summary.Summary.EstimatedTotal,
		RemainingAmount:    summary.Summary.RemainingAmount,
		OverBudgetAmount:   summary.Summary.OverBudgetAmount,
		UtilizationPercent: summary.Summary.UtilizationPercent,
	}
}

func workspaceBudgetPeriodMatches(from, to *time.Time, b *entity.WorkspaceBudget) bool {
	return sameDatePtr(from, b.PeriodStart) && sameDatePtr(to, b.PeriodEnd)
}

func sameDatePtr(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return truncateDate(*a).Equal(truncateDate(*b))
}

func budgetInsightsForWorkspaceAnalytics(insights []analytics.CostInsight) []analytics.CostInsight {
	out := make([]analytics.CostInsight, 0, 2)
	for _, insight := range insights {
		switch insight.Type {
		case "workspace_budget_exceeded", "workspace_budget_nearing_limit":
			out = append(out, insight)
		}
	}
	return out
}
