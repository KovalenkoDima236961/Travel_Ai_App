package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/analytics"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/budget"
	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type CreateWorkspaceBudgetInput struct {
	Name        string
	Description *string
	Amount      float64
	Currency    string
	PeriodStart *time.Time
	PeriodEnd   *time.Time
	IsPrimary   *bool
}

type UpdateWorkspaceBudgetInput struct {
	Name           *string
	Description    *string
	DescriptionSet bool
	Amount         *float64
	Currency       *string
	PeriodStart    *time.Time
	PeriodStartSet bool
	PeriodEnd      *time.Time
	PeriodEndSet   bool
	IsPrimary      *bool
}

type WorkspaceBudgetResponse struct {
	ID               uuid.UUID                    `json:"id"`
	WorkspaceID      uuid.UUID                    `json:"workspaceId"`
	Name             string                       `json:"name"`
	Description      *string                      `json:"description,omitempty"`
	Amount           float64                      `json:"amount"`
	Currency         string                       `json:"currency"`
	PeriodStart      *string                      `json:"periodStart,omitempty"`
	PeriodEnd        *string                      `json:"periodEnd,omitempty"`
	Status           entity.WorkspaceBudgetStatus `json:"status"`
	IsPrimary        bool                         `json:"isPrimary"`
	CreatedByUserID  uuid.UUID                    `json:"createdByUserId"`
	ArchivedByUserID *uuid.UUID                   `json:"archivedByUserId,omitempty"`
	CreatedAt        time.Time                    `json:"createdAt"`
	UpdatedAt        time.Time                    `json:"updatedAt"`
	ArchivedAt       *time.Time                   `json:"archivedAt,omitempty"`
}

type WorkspaceBudgetEnvelope struct {
	Budget WorkspaceBudgetResponse `json:"budget"`
}

type WorkspaceBudgetsEnvelope struct {
	Budgets []WorkspaceBudgetResponse `json:"budgets"`
}

type WorkspaceBudgetSummaryResponse struct {
	Budget           WorkspaceBudgetResponse       `json:"budget"`
	GeneratedAt      time.Time                     `json:"generatedAt"`
	Summary          WorkspaceBudgetSummaryMetrics `json:"summary"`
	ByTrip           []WorkspaceBudgetTripSummary  `json:"byTrip"`
	ByCategory       []WorkspaceBudgetBreakdown    `json:"byCategory"`
	BySource         []WorkspaceBudgetBreakdown    `json:"bySource"`
	ExpensiveItems   []analytics.ExpensiveCostItem `json:"expensiveItems"`
	Insights         []analytics.CostInsight       `json:"insights"`
	Warnings         []string                      `json:"warnings"`
	ExchangeRateInfo *budget.ExchangeRateInfo      `json:"exchangeRateInfo,omitempty"`
}

type WorkspaceBudgetSummaryMetrics struct {
	TripCount              int     `json:"tripCount"`
	EstimatedTotal         float64 `json:"estimatedTotal"`
	RemainingAmount        float64 `json:"remainingAmount"`
	OverBudgetAmount       float64 `json:"overBudgetAmount"`
	UtilizationPercent     float64 `json:"utilizationPercent"`
	MissingEstimateCount   int     `json:"missingEstimateCount"`
	UncertainEstimateCount int     `json:"uncertainEstimateCount"`
	ConvertedItemCount     int     `json:"convertedItemCount"`
	UnconvertedItemCount   int     `json:"unconvertedItemCount"`
}

type WorkspaceBudgetTripSummary struct {
	TripID               uuid.UUID `json:"tripId"`
	Title                string    `json:"title"`
	Destination          string    `json:"destination"`
	StartDate            *string   `json:"startDate,omitempty"`
	EstimatedTotal       float64   `json:"estimatedTotal"`
	PercentageOfBudget   float64   `json:"percentageOfBudget"`
	MissingEstimateCount int       `json:"missingEstimateCount"`
	OverTripBudgetAmount *float64  `json:"overTripBudgetAmount,omitempty"`
}

type WorkspaceBudgetBreakdown struct {
	Category                   string  `json:"category,omitempty"`
	Source                     string  `json:"source,omitempty"`
	Amount                     float64 `json:"amount"`
	PercentageOfBudget         float64 `json:"percentageOfBudget,omitempty"`
	PercentageOfEstimatedTotal float64 `json:"percentageOfEstimatedTotal"`
	ItemCount                  int     `json:"itemCount"`
}

func NewWorkspaceBudgetResponse(b *entity.WorkspaceBudget) WorkspaceBudgetResponse {
	return WorkspaceBudgetResponse{
		ID:               b.ID,
		WorkspaceID:      b.WorkspaceID,
		Name:             b.Name,
		Description:      b.Description,
		Amount:           b.Amount,
		Currency:         b.Currency,
		PeriodStart:      dateString(b.PeriodStart),
		PeriodEnd:        dateString(b.PeriodEnd),
		Status:           b.Status,
		IsPrimary:        b.IsPrimary,
		CreatedByUserID:  b.CreatedByUserID,
		ArchivedByUserID: b.ArchivedByUserID,
		CreatedAt:        b.CreatedAt,
		UpdatedAt:        b.UpdatedAt,
		ArchivedAt:       b.ArchivedAt,
	}
}

func NewWorkspaceBudgetEnvelope(b *entity.WorkspaceBudget) WorkspaceBudgetEnvelope {
	return WorkspaceBudgetEnvelope{Budget: NewWorkspaceBudgetResponse(b)}
}

func NewWorkspaceBudgetsEnvelope(budgets []entity.WorkspaceBudget) WorkspaceBudgetsEnvelope {
	items := make([]WorkspaceBudgetResponse, 0, len(budgets))
	for i := range budgets {
		items = append(items, NewWorkspaceBudgetResponse(&budgets[i]))
	}
	return WorkspaceBudgetsEnvelope{Budgets: items}
}

func dateString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	out := value.Format("2006-01-02")
	return &out
}
