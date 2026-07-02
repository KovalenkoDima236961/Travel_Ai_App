package budgetoptimization

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/KovalenkoDima236961/Travel_Ai_App/internal/domain/entity"
)

type ProposalEnvelope struct {
	Proposal ProposalResponse `json:"proposal"`
}

type ListResponse struct {
	Proposals []ProposalResponse `json:"proposals"`
	Limit     int                `json:"limit"`
}

type ProposalResponse struct {
	ID                        uuid.UUID                               `json:"id"`
	TripID                    uuid.UUID                               `json:"tripId"`
	JobID                     *uuid.UUID                              `json:"jobId"`
	CreatedByUserID           uuid.UUID                               `json:"createdByUserId"`
	Scope                     entity.BudgetOptimizationScope          `json:"scope"`
	DayNumber                 *int                                    `json:"dayNumber"`
	Status                    entity.BudgetOptimizationProposalStatus `json:"status"`
	ExpectedItineraryRevision int                                     `json:"expectedItineraryRevision"`
	BaseItineraryRevision     int                                     `json:"baseItineraryRevision"`
	Currency                  string                                  `json:"currency"`
	TargetReductionAmount     *float64                                `json:"targetReductionAmount"`
	EstimatedSavingsAmount    *float64                                `json:"estimatedSavingsAmount"`
	Proposal                  ProposalContent                         `json:"proposal"`
	AppliedItineraryRevision  *int                                    `json:"appliedItineraryRevision"`
	CreatedAt                 time.Time                               `json:"createdAt"`
	AppliedAt                 *time.Time                              `json:"appliedAt"`
	DiscardedAt               *time.Time                              `json:"discardedAt"`
	ExpiredAt                 *time.Time                              `json:"expiredAt"`
	UpdatedAt                 time.Time                               `json:"updatedAt"`
}

func NewProposalEnvelope(proposal *entity.BudgetOptimizationProposal) ProposalEnvelope {
	return ProposalEnvelope{Proposal: NewProposalResponse(proposal)}
}

func NewListResponse(proposals []entity.BudgetOptimizationProposal, limit int) ListResponse {
	items := make([]ProposalResponse, 0, len(proposals))
	for i := range proposals {
		items = append(items, NewProposalResponse(&proposals[i]))
	}
	return ListResponse{Proposals: items, Limit: limit}
}

func NewProposalResponse(proposal *entity.BudgetOptimizationProposal) ProposalResponse {
	content := ProposalContent{}
	if len(proposal.ProposalJSON) > 0 {
		_ = json.Unmarshal(proposal.ProposalJSON, &content)
	}
	return ProposalResponse{
		ID:                        proposal.ID,
		TripID:                    proposal.TripID,
		JobID:                     proposal.JobID,
		CreatedByUserID:           proposal.CreatedByUserID,
		Scope:                     proposal.Scope,
		DayNumber:                 proposal.DayNumber,
		Status:                    proposal.Status,
		ExpectedItineraryRevision: proposal.ExpectedItineraryRevision,
		BaseItineraryRevision:     proposal.BaseItineraryRevision,
		Currency:                  proposal.Currency,
		TargetReductionAmount:     proposal.TargetReductionAmount,
		EstimatedSavingsAmount:    proposal.EstimatedSavingsAmount,
		Proposal:                  content,
		AppliedItineraryRevision:  proposal.AppliedItineraryRevision,
		CreatedAt:                 proposal.CreatedAt,
		AppliedAt:                 proposal.AppliedAt,
		DiscardedAt:               proposal.DiscardedAt,
		ExpiredAt:                 proposal.ExpiredAt,
		UpdatedAt:                 proposal.UpdatedAt,
	}
}
