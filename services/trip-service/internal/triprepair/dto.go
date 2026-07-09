package triprepair

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
	Proposals []ProposalSummaryResponse `json:"proposals"`
	Limit     int                       `json:"limit"`
}

type ProposalSummaryResponse struct {
	ID                    uuid.UUID                       `json:"id"`
	TripID                uuid.UUID                       `json:"tripId"`
	JobID                 *uuid.UUID                      `json:"jobId"`
	CreatedByUserID       uuid.UUID                       `json:"createdByUserId"`
	Status                entity.TripRepairProposalStatus `json:"status"`
	RepairMode            string                          `json:"repairMode"`
	BaseItineraryRevision int                             `json:"baseItineraryRevision"`
	BaseRiskScore         *int                            `json:"baseRiskScore"`
	ProposedRiskScore     *int                            `json:"proposedRiskScore"`
	BasePolicyStatus      *string                         `json:"basePolicyStatus"`
	ProposedPolicyStatus  *string                         `json:"proposedPolicyStatus"`
	Summary               Summary                         `json:"summary"`
	CreatedAt             time.Time                       `json:"createdAt"`
	UpdatedAt             time.Time                       `json:"updatedAt"`
	AppliedAt             *time.Time                      `json:"appliedAt"`
	AppliedByUserID       *uuid.UUID                      `json:"appliedByUserId"`
	DiscardedAt           *time.Time                      `json:"discardedAt"`
	DiscardedByUserID     *uuid.UUID                      `json:"discardedByUserId"`
	ExpiredAt             *time.Time                      `json:"expiredAt"`
}

type ProposalResponse struct {
	ProposalSummaryResponse
	Issues   []Issue         `json:"issues"`
	Proposal ProposalContent `json:"proposal"`
}

func NewListResponse(proposals []entity.TripRepairProposal, limit int) ListResponse {
	items := make([]ProposalSummaryResponse, 0, len(proposals))
	for i := range proposals {
		items = append(items, NewProposalSummaryResponse(&proposals[i]))
	}
	return ListResponse{Proposals: items, Limit: limit}
}

func NewProposalEnvelope(proposal *entity.TripRepairProposal) ProposalEnvelope {
	return ProposalEnvelope{Proposal: NewProposalResponse(proposal)}
}

func NewProposalSummaryResponse(proposal *entity.TripRepairProposal) ProposalSummaryResponse {
	content := ProposalContent{}
	if len(proposal.ProposalJSON) > 0 {
		_ = json.Unmarshal(proposal.ProposalJSON, &content)
	}
	return ProposalSummaryResponse{
		ID:                    proposal.ID,
		TripID:                proposal.TripID,
		JobID:                 proposal.JobID,
		CreatedByUserID:       proposal.CreatedByUserID,
		Status:                proposal.Status,
		RepairMode:            proposal.RepairMode,
		BaseItineraryRevision: proposal.BaseItineraryRevision,
		BaseRiskScore:         proposal.BaseRiskScore,
		ProposedRiskScore:     proposal.ProposedRiskScore,
		BasePolicyStatus:      proposal.BasePolicyStatus,
		ProposedPolicyStatus:  proposal.ProposedPolicyStatus,
		Summary:               content.RepairSummary,
		CreatedAt:             proposal.CreatedAt,
		UpdatedAt:             proposal.UpdatedAt,
		AppliedAt:             proposal.AppliedAt,
		AppliedByUserID:       proposal.AppliedByUserID,
		DiscardedAt:           proposal.DiscardedAt,
		DiscardedByUserID:     proposal.DiscardedByUserID,
		ExpiredAt:             proposal.ExpiredAt,
	}
}

func NewProposalResponse(proposal *entity.TripRepairProposal) ProposalResponse {
	content := ProposalContent{}
	if len(proposal.ProposalJSON) > 0 {
		_ = json.Unmarshal(proposal.ProposalJSON, &content)
	}
	issues := []Issue{}
	if len(proposal.IssuesJSON) > 0 {
		_ = json.Unmarshal(proposal.IssuesJSON, &issues)
	}
	return ProposalResponse{
		ProposalSummaryResponse: NewProposalSummaryResponse(proposal),
		Issues:                  issues,
		Proposal:                content,
	}
}
